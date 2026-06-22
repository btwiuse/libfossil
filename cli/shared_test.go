package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	libfossil "github.com/danmestas/libfossil"
	"github.com/danmestas/libfossil/cli"
	libdb "github.com/danmestas/libfossil/db"

	_ "github.com/danmestas/libfossil/internal/testdriver"
)

func TestGlobalsOpenRepo(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "test.fossil")

	r, err := libfossil.Create(repoPath, libfossil.CreateOpts{User: "test"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	r.Close()

	cli.Repo = repoPath
	opened, err := cli.OpenRepo()
	if err != nil {
		t.Fatalf("OpenRepo: %v", err)
	}
	defer opened.Close()

	if opened.Path() != repoPath {
		t.Errorf("Path() = %q, want %q", opened.Path(), repoPath)
	}
}

func TestGlobalsOpenRepoNotFound(t *testing.T) {
	cli.Repo = "/nonexistent/repo.fossil"
	_, err := cli.OpenRepo()
	if err == nil {
		t.Fatal("expected error for nonexistent repo")
	}
}

func TestGlobalsOpenRepoAutoFind(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "auto.fossil")

	r, err := libfossil.Create(repoPath, libfossil.CreateOpts{User: "test"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	r.Close()

	// Change to the temp dir so findRepo can discover the .fossil file.
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmp)

	cli.Repo = ""
	opened, err := cli.OpenRepo()
	if err != nil {
		t.Fatalf("OpenRepo auto-find: %v", err)
	}
	defer opened.Close()
}

func TestRepoCiPreservesNestedRelativePaths(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "nested.fossil")

	r, err := libfossil.Create(repoPath, libfossil.CreateOpts{User: "test"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	r.Close()

	work := filepath.Join(tmp, "work")
	nested := filepath.Join(work, "src", "app.txt")
	if err := os.MkdirAll(filepath.Dir(nested), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nested, []byte("nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}

	cmd := cli.NewCiCommand()
	cli.Repo = repoPath
	cmd.SetArgs([]string{"-m", "initial nested", "--user", "test", filepath.Join("src", "app.txt")})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ci command: %v", err)
	}

	opened, err := libfossil.Open(repoPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer opened.Close()
	rid, err := opened.ResolveVersion("tip")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	files, err := opened.ListFiles(rid)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 1 || files[0].Name != "src/app.txt" {
		t.Fatalf("files = %+v, want one src/app.txt entry", files)
	}
}

func TestRepoOpenPopulatesVFileFromTip(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "open.fossil")

	r, err := libfossil.Create(repoPath, libfossil.CreateOpts{User: "test"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, _, err := r.Commit(libfossil.CommitOpts{
		Files: []libfossil.FileToCommit{
			{Name: "hello.txt", Content: []byte("hello\n")},
		},
		Comment: "initial",
		User:    "test",
	}); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	r.Close()

	checkoutDir := filepath.Join(tmp, "checkout")
	cmd := cli.NewOpenCommand()
	cli.Repo = repoPath
	cmd.SetArgs([]string{checkoutDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("open command: %v", err)
	}

	ckdb, err := libdb.OpenSQL(filepath.Join(checkoutDir, ".fslckout"), libdb.OpenConfig{}, nil)
	if err != nil {
		t.Fatalf("open checkout db: %v", err)
	}
	defer ckdb.Close()
	var count int
	if err := ckdb.QueryRow("SELECT count(*) FROM vfile WHERE pathname='hello.txt' AND rid > 0").Scan(&count); err != nil {
		t.Fatalf("query vfile: %v", err)
	}
	if count != 1 {
		t.Fatalf("vfile count = %d, want 1", count)
	}
}

func TestRepoCiRejectsOutsideCurrentDirectory(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "outside.fossil")
	r, err := libfossil.Create(repoPath, libfossil.CreateOpts{User: "test"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	r.Close()

	work := filepath.Join(tmp, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(tmp, "outside.txt")
	if err := os.WriteFile(outside, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}

	cmd := cli.NewCiCommand()
	cli.Repo = repoPath
	cmd.SetArgs([]string{"-m", "outside", "--user", "test", filepath.Join("..", "outside.txt")})
	if err := cmd.Execute(); err == nil {
		t.Fatal("ci command accepted path outside current directory")
	}
}
