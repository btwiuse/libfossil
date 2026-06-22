package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/danmestas/libfossil/internal/blob"
	"github.com/danmestas/libfossil/internal/content"
	"github.com/danmestas/libfossil/internal/merge"
	"github.com/danmestas/libfossil/internal/repo"
	"github.com/spf13/cobra"
)

func newConflictsCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "conflicts",
		Short: "List/manage unresolved conflicts",
		RunE: func(_ *cobra.Command, _ []string) error {
			return listConflicts(dir)
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")

	cmd.AddCommand(newConflictsLsCommand())
	cmd.AddCommand(newConflictsShowCommand())
	cmd.AddCommand(newConflictsPickCommand())
	cmd.AddCommand(newConflictsMergeCommand())
	cmd.AddCommand(newConflictsExtractCommand())

	return cmd
}

func newConflictsLsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List all conflicts",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return listConflicts(".")
		},
	}
}

func listConflicts(dir string) error {
	found := 0

	ckout, err := openCheckout(dir)
	if err == nil {
		defer ckout.Close()
		vid, _ := checkoutVid(ckout)
		rows, err := ckout.Query("SELECT pathname FROM vfile WHERE chnged=5 AND vid=?", vid)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var name string
				rows.Scan(&name)
				fmt.Printf("CONFLICT  %s\n", name)
				found++
			}
		}
	}

	r, err := OpenRepo()
	if err == nil {
		defer r.Close()
		inner := r.Inner()
		entries, err := listConflictForkDetails(inner)
		if err == nil {
			for _, e := range entries {
				fmt.Printf("FORK      %s  (base=%d local=%d remote=%d)\n",
					e.filename, e.baseRid, e.localRid, e.remoteRid)
				found++
			}
		}
	}

	if found == 0 {
		fmt.Println("no conflicts")
	}
	return nil
}

type conflictForkEntry struct {
	filename  string
	baseRid   int64
	localRid  int64
	remoteRid int64
}

func listConflictForkDetails(r *repo.Repo) ([]conflictForkEntry, error) {
	var count int
	if r.DB().QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='conflict'").Scan(&count); count == 0 {
		return nil, nil
	}
	rows, err := r.DB().Query("SELECT filename, base_rid, local_rid, remote_rid FROM conflict ORDER BY mtime DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []conflictForkEntry
	for rows.Next() {
		var e conflictForkEntry
		rows.Scan(&e.filename, &e.baseRid, &e.localRid, &e.remoteRid)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func loadConflictFork(r *repo.Repo, filename string) (*conflictForkEntry, error) {
	var count int
	if r.DB().QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='conflict'").Scan(&count); count == 0 {
		return nil, fmt.Errorf("%s: no conflict-fork entries", filename)
	}
	var e conflictForkEntry
	e.filename = filename
	err := r.DB().QueryRow("SELECT base_rid, local_rid, remote_rid FROM conflict WHERE filename=?", filename).
		Scan(&e.baseRid, &e.localRid, &e.remoteRid)
	if err != nil {
		return nil, fmt.Errorf("%s: not found in conflict table", filename)
	}
	return &e, nil
}

func expandForkFile(r *repo.Repo, checkinRid int64, filename string) ([]byte, error) {
	if checkinRid <= 0 {
		return nil, nil
	}
	rows, err := r.DB().Query(`
		SELECT f.uuid FROM mlink m
		JOIN filename fn ON fn.fnid = m.fnid
		JOIN blob f ON f.rid = m.fid
		WHERE m.mid = ? AND fn.name = ?`, checkinRid, filename)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var uuid string
		rows.Scan(&uuid)
		frid, ok := blob.Exists(r.DB(), uuid)
		if !ok {
			return nil, fmt.Errorf("blob %s not found", uuid)
		}
		return content.Expand(r.DB(), frid)
	}
	return expandForkFileFallback(r, checkinRid, filename)
}

func expandForkFileFallback(r *repo.Repo, checkinRid int64, filename string) ([]byte, error) {
	files, err := r.DB().Query(`
		SELECT b2.uuid FROM event e
		JOIN blob b ON b.rid = e.objid
		WHERE e.objid = ?`, checkinRid)
	if err != nil || files == nil {
		return nil, fmt.Errorf("file %s not found in checkin %d", filename, checkinRid)
	}
	defer files.Close()
	return nil, fmt.Errorf("file %s not found in checkin %d", filename, checkinRid)
}

func newConflictsShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <file>",
		Short: "Show all versions of a conflicted file",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			file := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			inner := r.Inner()
			entry, err := loadConflictFork(inner, file)
			if err != nil {
				return err
			}

			base, _ := expandForkFile(inner, entry.baseRid, file)
			local, _ := expandForkFile(inner, entry.localRid, file)
			remote, _ := expandForkFile(inner, entry.remoteRid, file)

			fmt.Printf("=== BASE (ancestor, rid=%d) ===\n", entry.baseRid)
			os.Stdout.Write(base)
			fmt.Printf("\n=== LOCAL (your version, rid=%d) ===\n", entry.localRid)
			os.Stdout.Write(local)
			fmt.Printf("\n=== REMOTE (their version, rid=%d) ===\n", entry.remoteRid)
			os.Stdout.Write(remote)
			fmt.Println()
			return nil
		},
	}
	return cmd
}

func newConflictsPickCommand() *cobra.Command {
	var local, remote, base bool
	var dir string
	cmd := &cobra.Command{
		Use:   "pick <file>",
		Short: "Resolve by picking one version",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			file := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			inner := r.Inner()
			entry, err := loadConflictFork(inner, file)
			if err != nil {
				return err
			}

			var picked []byte
			var label string
			switch {
			case remote:
				picked, _ = expandForkFile(inner, entry.remoteRid, file)
				label = "remote"
			case base:
				picked, _ = expandForkFile(inner, entry.baseRid, file)
				label = "base"
			default:
				picked, _ = expandForkFile(inner, entry.localRid, file)
				label = "local"
			}

			outPath := filepath.Join(dir, file)
			os.MkdirAll(filepath.Dir(outPath), 0o755)
			if err := os.WriteFile(outPath, picked, 0o644); err != nil {
				return err
			}

			merge.ResolveConflictFork(inner, file)
			fmt.Printf("resolved: %s (picked %s)\n", file, label)
			return nil
		},
	}
	cmd.Flags().BoolVar(&local, "local", false, "Keep local version")
	cmd.Flags().BoolVar(&remote, "remote", false, "Keep remote version")
	cmd.Flags().BoolVar(&base, "base", false, "Revert to base version")
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}

func newConflictsMergeCommand() *cobra.Command {
	var strategy, dir string
	cmd := &cobra.Command{
		Use:   "merge <file>",
		Short: "Resolve by re-merging with a different strategy",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			file := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			inner := r.Inner()
			entry, err := loadConflictFork(inner, file)
			if err != nil {
				return err
			}

			strat, ok := merge.StrategyByName(strategy)
			if !ok {
				return fmt.Errorf("unknown strategy: %s", strategy)
			}

			base, _ := expandForkFile(inner, entry.baseRid, file)
			local, _ := expandForkFile(inner, entry.localRid, file)
			remote, _ := expandForkFile(inner, entry.remoteRid, file)

			result, err := strat.Merge(base, local, remote)
			if err != nil {
				return err
			}

			outPath := filepath.Join(dir, file)
			os.MkdirAll(filepath.Dir(outPath), 0o755)
			if err := os.WriteFile(outPath, result.Content, 0o644); err != nil {
				return err
			}

			if result.Clean {
				merge.ResolveConflictFork(inner, file)
				fmt.Printf("resolved: %s (merged with %s, clean)\n", file, strategy)
			} else {
				os.WriteFile(outPath+".LOCAL", local, 0o644)
				os.WriteFile(outPath+".BASELINE", base, 0o644)
				os.WriteFile(outPath+".MERGE", remote, 0o644)
				fmt.Printf("merged: %s (%s, %d conflicts remain -- edit and run mark-resolved)\n",
					file, strategy, len(result.Conflicts))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&strategy, "strategy", "three-way", "Strategy to use")
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}

func newConflictsExtractCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "extract <file>",
		Short: "Extract all versions to disk for manual editing",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			file := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			inner := r.Inner()
			entry, err := loadConflictFork(inner, file)
			if err != nil {
				return err
			}

			base, _ := expandForkFile(inner, entry.baseRid, file)
			local, _ := expandForkFile(inner, entry.localRid, file)
			remote, _ := expandForkFile(inner, entry.remoteRid, file)

			os.MkdirAll(dir, 0o755)

			basePath := filepath.Join(dir, file+".BASE")
			localPath := filepath.Join(dir, file+".LOCAL")
			remotePath := filepath.Join(dir, file+".REMOTE")

			os.MkdirAll(filepath.Dir(basePath), 0o755)
			os.WriteFile(basePath, base, 0o644)
			os.WriteFile(localPath, local, 0o644)
			os.WriteFile(remotePath, remote, 0o644)

			fmt.Printf("  %s\n", basePath)
			fmt.Printf("  %s\n", localPath)
			fmt.Printf("  %s\n", remotePath)
			return nil
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Output directory")
	return cmd
}
