package cli_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	libfossil "github.com/danmestas/libfossil"
	"github.com/danmestas/libfossil/cli"

	_ "github.com/danmestas/libfossil/internal/testdriver"
)

func TestUICommandServesHTTP(t *testing.T) {
	repoPath := filepath.Join(t.TempDir(), "ui.fossil")
	r, err := libfossil.Create(repoPath, libfossil.CreateOpts{User: "admin"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, _, err := r.Commit(libfossil.CommitOpts{
		Files:   []libfossil.FileToCommit{{Name: "f.txt", Content: []byte("data\n")}},
		Comment: "initial",
		User:    "admin",
	}); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	r.Close()

	addr := freeAddr(t)
	ctx, cancel, errCh := setupUICommand(t, repoPath, addr)
	defer cancel()

	// Hit the timeline endpoint — the web UI handler serves it.
	url := fmt.Sprintf("http://%s/timeline", addr)
	deadline := time.Now().Add(5 * time.Second)
	var resp *http.Response
	for time.Now().Before(deadline) {
		resp, err = http.Get(url)
		if err == nil {
			defer resp.Body.Close()
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		// Check if the command already failed.
		select {
		case cmdErr := <-errCh:
			t.Fatalf("ui command failed before responding: %v", cmdErr)
		default:
		}
		t.Fatalf("http.Get: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("status = %d, body = %s", resp.StatusCode, string(body))
	}
	resp.Body.Close()

	// Make sure context cancellation shuts down cleanly.
	cancel()
	<-ctx.Done()
}

func setupUICommand(t *testing.T, repoPath, addr string) (context.Context, context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cmd := cli.NewRootCommand()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--repo", repoPath, "ui", "--addr", addr, "--no-browser"})
	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Execute()
	}()
	time.Sleep(50 * time.Millisecond)
	select {
	case err := <-errCh:
		t.Fatalf("ui command exited early: %v", err)
	default:
	}
	return ctx, cancel, errCh
}
