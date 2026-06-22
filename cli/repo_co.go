package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/danmestas/libfossil/internal/blob"
	"github.com/danmestas/libfossil/internal/content"
	"github.com/spf13/cobra"
)

func newCoCommand() *cobra.Command {
	var dir string
	var force bool
	cmd := &cobra.Command{
		Use:   "co [version]",
		Short: "Checkout a version",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			version := ""
			if len(args) > 0 {
				version = args[0]
			}
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			rid, err := resolveRID(r, version)
			if err != nil {
				return err
			}

			files, err := r.ListFiles(rid)
			if err != nil {
				return err
			}

			db := r.Inner().DB()
			for _, f := range files {
				fileRid, ok := blob.Exists(db, f.UUID)
				if !ok {
					return fmt.Errorf("blob %s not found for file %s", f.UUID, f.Name)
				}
				data, err := content.Expand(db, fileRid)
				if err != nil {
					return fmt.Errorf("expanding %s: %w", f.Name, err)
				}

				outPath := filepath.Join(dir, f.Name)
				if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
					return err
				}

				if !force {
					if _, err := os.Stat(outPath); err == nil {
						return fmt.Errorf("file exists: %s (use --force to overwrite)", outPath)
					}
				}

				perm := os.FileMode(0o644)
				if f.Perm == "x" {
					perm = 0o755
				}
				if err := os.WriteFile(outPath, data, perm); err != nil {
					return err
				}

				fmt.Printf("  %s\n", f.Name)
			}

			fmt.Printf("checked out %d files\n", len(files))
			return nil
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Output directory (default: current dir)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files")
	return cmd
}
