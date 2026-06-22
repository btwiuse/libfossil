package cli_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	libfossil "github.com/danmestas/libfossil"
	"github.com/danmestas/libfossil/cli"

	_ "github.com/danmestas/libfossil/internal/testdriver"
)

func TestServerCommandSupportsClone(t *testing.T) {
	repoPath := filepath.Join(t.TempDir(), "server.fossil")
	serverRepo, err := libfossil.Create(repoPath, libfossil.CreateOpts{User: "admin"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, _, err := serverRepo.Commit(libfossil.CommitOpts{
		Files: []libfossil.FileToCommit{{Name: "hello.txt", Content: []byte("hello\n")}},
		Comment: "initial",
		User:    "admin",
	}); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := serverRepo.CreateUser(libfossil.UserOpts{
		Login:    "alice",
		Password: "secret",
		Caps:     "gio",
	}); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if err := serverRepo.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	addr := freeAddr(t)
	cancel, errCh := startServerCommand(t, repoPath, addr)
	defer func() {
		cancel()
		if err := <-errCh; err != nil {
			t.Fatalf("server command: %v", err)
		}
	}()

	clientPath := filepath.Join(t.TempDir(), "client.fossil")
	clientRepo, result, err := libfossil.Clone(
		context.Background(),
		clientPath,
		libfossil.NewHTTPTransport(fmt.Sprintf("http://%s", addr)),
		libfossil.CloneOpts{User: "alice", Password: "secret"},
	)
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	defer clientRepo.Close()
	if result == nil || result.BlobsRecvd == 0 {
		t.Fatalf("clone result = %+v, want blobs received", result)
	}
	rid, err := clientRepo.ResolveVersion("tip")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	files, err := clientRepo.ListFiles(rid)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 1 || files[0].Name != "hello.txt" {
		t.Fatalf("files = %+v, want hello.txt", files)
	}
}

func TestServerCommandSupportsPush(t *testing.T) {
	repoPath := filepath.Join(t.TempDir(), "server.fossil")
	serverRepo, err := libfossil.Create(repoPath, libfossil.CreateOpts{User: "admin"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, _, err := serverRepo.Commit(libfossil.CommitOpts{
		Files: []libfossil.FileToCommit{{Name: "base.txt", Content: []byte("base\n")}},
		Comment: "initial",
		User:    "admin",
	}); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := serverRepo.CreateUser(libfossil.UserOpts{
		Login:    "alice",
		Password: "secret",
		Caps:     "gio",
	}); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if err := serverRepo.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	addr := freeAddr(t)
	cancel, errCh := startServerCommand(t, repoPath, addr)
	defer func() {
		cancel()
		if err := <-errCh; err != nil {
			t.Fatalf("server command: %v", err)
		}
	}()

	clientPath := filepath.Join(t.TempDir(), "client.fossil")
	clientRepo, _, err := libfossil.Clone(
		context.Background(),
		clientPath,
		libfossil.NewHTTPTransport(fmt.Sprintf("http://%s", addr)),
		libfossil.CloneOpts{User: "alice", Password: "secret"},
	)
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	defer clientRepo.Close()

	parentID, err := clientRepo.ResolveVersion("tip")
	if err != nil {
		t.Fatalf("ResolveVersion(client): %v", err)
	}
	if _, _, err := clientRepo.Commit(libfossil.CommitOpts{
		Files: []libfossil.FileToCommit{
			{Name: "base.txt", Content: []byte("base\n")},
			{Name: "push.txt", Content: []byte("pushed\n")},
		},
		Comment:  "push",
		User:     "alice",
		ParentID: parentID,
	}); err != nil {
		t.Fatalf("Commit(client): %v", err)
	}

	res, err := clientRepo.Sync(
		context.Background(),
		libfossil.NewHTTPTransport(fmt.Sprintf("http://%s", addr)),
		libfossil.SyncOpts{Push: true, User: "alice", Password: "secret"},
	)
	if err != nil {
		t.Fatalf("Sync(push): %v", err)
	}
	if res == nil || res.FilesSent == 0 {
		t.Fatalf("push result = %+v, want files sent", res)
	}

	updatedServer, err := libfossil.Open(repoPath)
	if err != nil {
		t.Fatalf("Open(server): %v", err)
	}
	defer updatedServer.Close()
	rid, err := updatedServer.ResolveVersion("tip")
	if err != nil {
		t.Fatalf("ResolveVersion(server): %v", err)
	}
	files, err := updatedServer.ListFiles(rid)
	if err != nil {
		t.Fatalf("ListFiles(server): %v", err)
	}
	found := false
	for _, f := range files {
		if f.Name == "push.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("server files = %+v, want push.txt", files)
	}
}

func startServerCommand(t *testing.T, repoPath, addr string) (context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cmd := cli.NewRootCommand()
	cmd.SetContext(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--repo", repoPath, "server", "--addr", addr})
	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Execute()
	}()
	waitForServer(t, addr)
	return cancel, errCh
}

func waitForServer(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	url := fmt.Sprintf("http://%s/", addr)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("server at %s did not become ready", addr)
}

func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()
	return ln.Addr().String()
}
