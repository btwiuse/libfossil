package cli

import (
	"fmt"
	"os"
	"path/filepath"

	libfossil "github.com/danmestas/libfossil"
	"github.com/danmestas/libfossil/db"
	"github.com/danmestas/libfossil/internal/blob"
	"github.com/danmestas/libfossil/internal/content"
	"github.com/danmestas/libfossil/internal/fsltype"
	"github.com/danmestas/libfossil/internal/merge"
	"github.com/spf13/cobra"
)

func newMergeCommand() *cobra.Command {
	var strategy, dir string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "merge <version>",
		Short: "Merge a divergent version",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			version := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			localRid, err := resolveRID(r, "tip")
			if err != nil {
				return fmt.Errorf("resolving local tip: %w", err)
			}
			remoteRid, err := resolveRID(r, version)
			if err != nil {
				return fmt.Errorf("resolving remote version: %w", err)
			}

			inner := r.Inner()
			idb := inner.DB()

			ancestorRid, err := r.FindCommonAncestor(localRid, remoteRid)
			if err != nil {
				return fmt.Errorf("finding common ancestor: %w", err)
			}

			var ancestorUUID string
			idb.QueryRow("SELECT uuid FROM blob WHERE rid=?", ancestorRid).Scan(&ancestorUUID)
			if len(ancestorUUID) > 10 {
				ancestorUUID = ancestorUUID[:10]
			}
			fmt.Printf("common ancestor: %s (rid=%d)\n", ancestorUUID, ancestorRid)

			baseFiles, err := r.ListFiles(ancestorRid)
			if err != nil {
				return fmt.Errorf("listing ancestor files: %w", err)
			}
			localFiles, err := r.ListFiles(localRid)
			if err != nil {
				return fmt.Errorf("listing local files: %w", err)
			}
			remoteFiles, err := r.ListFiles(remoteRid)
			if err != nil {
				return fmt.Errorf("listing remote files: %w", err)
			}

			baseMap := toFileMap(baseFiles)
			localMap := toFileMap(localFiles)
			remoteMap := toFileMap(remoteFiles)

			resolver := merge.LoadResolver(inner, fsltype.FslID(localRid))

			ckout, err := openCheckout(dir)
			if err != nil && !dryRun {
				return err
			}
			if ckout != nil {
				defer ckout.Close()
			}
			vid, _ := checkoutVid(ckout)

			merged, conflicts := 0, 0

			allFiles := make(map[string]bool)
			for name := range localMap {
				allFiles[name] = true
			}
			for name := range remoteMap {
				allFiles[name] = true
			}

			for name := range allFiles {
				localUUID := localMap[name]
				remoteUUID := remoteMap[name]
				baseUUID := baseMap[name]

				if localUUID == remoteUUID {
					continue
				}

				stratName := strategy
				if stratName == "" {
					stratName = resolver.Resolve(name)
				}
				strat, ok := merge.StrategyByName(stratName)
				if !ok {
					return fmt.Errorf("unknown strategy %q for %s", stratName, name)
				}

				baseContent := loadBlobByUUID(idb, baseUUID)
				localContent := loadBlobByUUID(idb, localUUID)
				remoteContent := loadBlobByUUID(idb, remoteUUID)

				if dryRun {
					fmt.Printf("  [%s] %s\n", stratName, name)
					continue
				}

				result, err := strat.Merge(baseContent, localContent, remoteContent)
				if err != nil {
					return fmt.Errorf("merging %s: %w", name, err)
				}

				outPath := filepath.Join(dir, name)
				os.MkdirAll(filepath.Dir(outPath), 0o755)

				if result.Clean {
					os.WriteFile(outPath, result.Content, 0o644)
					if ckout != nil {
						ckout.Exec("UPDATE vfile SET chnged=1 WHERE pathname=? AND vid=?", name, vid)
					}
					fmt.Printf("  [merged]   %s (%s)\n", name, stratName)
					merged++
				} else if strat.Name() == "conflict-fork" {
					merge.EnsureConflictTable(inner)
					var baseRID, localRID, remoteRID int64
					idb.QueryRow("SELECT rid FROM blob WHERE uuid=?", baseUUID).Scan(&baseRID)
					idb.QueryRow("SELECT rid FROM blob WHERE uuid=?", localUUID).Scan(&localRID)
					idb.QueryRow("SELECT rid FROM blob WHERE uuid=?", remoteUUID).Scan(&remoteRID)
					merge.RecordConflictFork(inner, name, baseRID, localRID, remoteRID)
					fmt.Printf("  [fork]     %s (conflict-fork: all versions preserved)\n", name)
					conflicts++
				} else {
					os.WriteFile(outPath, result.Content, 0o644)
					os.WriteFile(outPath+".LOCAL", localContent, 0o644)
					os.WriteFile(outPath+".BASELINE", baseContent, 0o644)
					os.WriteFile(outPath+".MERGE", remoteContent, 0o644)
					if ckout != nil {
						ckout.Exec("UPDATE vfile SET chnged=5 WHERE pathname=? AND vid=?", name, vid)
					}
					fmt.Printf("  [CONFLICT] %s (%s, %d regions)\n", name, stratName, len(result.Conflicts))
					conflicts++
				}
			}

			fmt.Printf("\n%d files merged, %d conflicts\n", merged, conflicts)
			return nil
		},
	}
	cmd.Flags().StringVar(&strategy, "strategy", "", "Override merge strategy for all files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be merged without writing")
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}

func newMarkResolvedCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "mark-resolved <file>",
		Short: "Mark a conflict as resolved",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			file := args[0]
			resolved := false

			ckout, err := openCheckout(dir)
			if err == nil {
				defer ckout.Close()
				vid, _ := checkoutVid(ckout)
				result, err := ckout.Exec("UPDATE vfile SET chnged=1 WHERE pathname=? AND vid=? AND chnged=5", file, vid)
				if err == nil {
					affected, _ := result.RowsAffected()
					if affected > 0 {
						base := filepath.Join(dir, file)
						os.Remove(base + ".LOCAL")
						os.Remove(base + ".BASELINE")
						os.Remove(base + ".MERGE")
						fmt.Printf("resolved: %s (conflict markers)\n", file)
						resolved = true
					}
				}
			}

			r, err := OpenRepo()
			if err == nil {
				defer r.Close()
				err := r.ResolveConflictFork(file)
				if err == nil {
					fmt.Printf("resolved: %s (conflict-fork)\n", file)
					resolved = true
				}
			}

			if !resolved {
				return fmt.Errorf("%s: no conflict found", file)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}

func toFileMap(files []libfossil.FileEntry) map[string]string {
	m := make(map[string]string)
	for _, f := range files {
		m[f.Name] = f.UUID
	}
	return m
}

func loadBlobByUUID(d *db.DB, uuid string) []byte {
	if uuid == "" {
		return nil
	}
	rid, ok := blob.Exists(d, uuid)
	if !ok {
		return nil
	}
	data, err := content.Expand(d, rid)
	if err != nil {
		return nil
	}
	return data
}
