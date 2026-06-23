package cli_test

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	libfossil "github.com/danmestas/libfossil"
	"github.com/danmestas/libfossil/cli"

	_ "github.com/danmestas/libfossil/internal/testdriver"
)

func TestPushCommandWithServer(t *testing.T) {
	repoPath := filepath.Join(t.TempDir(), "server.fossil")
	serverRepo, err := libfossil.Create(repoPath, libfossil.CreateOpts{User: "admin"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, _, err := serverRepo.Commit(libfossil.CommitOpts{
		Files:   []libfossil.FileToCommit{{Name: "base.txt", Content: []byte("base\n")}},
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

	// Clone so we have a local repo to push from.
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
	parentID, err := clientRepo.ResolveVersion("tip")
	if err != nil {
		t.Fatalf("ResolveVersion(client): %v", err)
	}
	if _, _, err := clientRepo.Commit(libfossil.CommitOpts{
		Files: []libfossil.FileToCommit{
			{Name: "base.txt", Content: []byte("base\n")},
			{Name: "pushed.txt", Content: []byte("pushed from command\n")},
		},
		Comment:  "cli push test",
		User:     "alice",
		ParentID: parentID,
	}); err != nil {
		t.Fatalf("Commit(client): %v", err)
	}
	clientRepo.Close()

	// Now invoke the push CLI command.
	pushCmd := cli.NewRootCommand()
	pushCmd.SetOut(io.Discard)
	pushCmd.SetErr(io.Discard)
	pushCmd.SetArgs([]string{
		"--repo", clientPath,
		"push", fmt.Sprintf("http://%s", addr),
		"--user", "alice",
		"--pass", "secret",
	})
	if err := pushCmd.Execute(); err != nil {
		t.Fatalf("push command: %v", err)
	}

	// Reopen the server and verify the pushed commit arrived.
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
		if f.Name == "pushed.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("server files = %+v, want pushed.txt", files)
	}
}
