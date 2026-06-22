package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/danmestas/libfossil/internal/blob"
	"github.com/danmestas/libfossil/internal/content"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/spf13/cobra"
)

func newDiffCommand() *cobra.Command {
	var version, dir string
	var unified int
	cmd := &cobra.Command{
		Use:   "diff [version]",
		Short: "Show changes vs a version",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			v := version
			if len(args) > 0 {
				v = args[0]
			}
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			rid, err := resolveRID(r, v)
			if err != nil {
				return err
			}

			files, err := r.ListFiles(rid)
			if err != nil {
				return err
			}

			db := r.Inner().DB()
			hasDiff := false

			for _, f := range files {
				diskPath := filepath.Join(dir, f.Name)
				diskData, err := os.ReadFile(diskPath)
				if err != nil {
					if os.IsNotExist(err) {
						fmt.Printf("--- a/%s\n+++ /dev/null\n@@ deleted @@\n", f.Name)
						hasDiff = true
						continue
					}
					return err
				}

				fileRid, ok := blob.Exists(db, f.UUID)
				if !ok {
					return fmt.Errorf("blob %s not found for %s", f.UUID, f.Name)
				}
				repoContent, err := content.Expand(db, fileRid)
				if err != nil {
					return fmt.Errorf("%s: %w", f.Name, err)
				}

				repoStr := string(repoContent)
				diskStr := string(diskData)

				if repoStr == diskStr {
					continue
				}

				edits := myers.ComputeEdits(span.URIFromPath(f.Name), repoStr, diskStr)
				diff := fmt.Sprint(gotextdiff.ToUnified("a/"+f.Name, "b/"+f.Name, repoStr, edits))
				if diff != "" {
					fmt.Print(diff)
					hasDiff = true
				}
			}

			if !hasDiff {
				fmt.Println("no changes")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Working directory to compare")
	cmd.Flags().IntVarP(&unified, "unified", "U", 3, "Lines of context")
	return cmd
}
