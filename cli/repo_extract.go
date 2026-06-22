package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/danmestas/libfossil/internal/blob"
	"github.com/danmestas/libfossil/internal/content"
	"github.com/spf13/cobra"
)

func newExtractCommand() *cobra.Command {
	var version, dir string
	cmd := &cobra.Command{
		Use:   "extract [files...]",
		Short: "Extract files from a version",
		Args:  cobra.ArbitraryArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			rid, err := resolveRID(r, version)
			if err != nil {
				return err
			}

			allFiles, err := r.ListFiles(rid)
			if err != nil {
				return err
			}

			db := r.Inner().DB()

			type fileInfo struct {
				Name string
				UUID string
				Perm string
			}
			var files []fileInfo
			if len(args) > 0 {
				wanted := make(map[string]bool)
				for _, f := range args {
					wanted[f] = true
				}
				for _, f := range allFiles {
					if wanted[f.Name] {
						files = append(files, fileInfo{f.Name, f.UUID, f.Perm})
					}
				}
				if len(files) == 0 {
					return fmt.Errorf("no matching files found")
				}
			} else {
				for _, f := range allFiles {
					files = append(files, fileInfo{f.Name, f.UUID, f.Perm})
				}
			}

			for _, f := range files {
				fileRid, ok := blob.Exists(db, f.UUID)
				if !ok {
					return fmt.Errorf("blob %s not found for %s", f.UUID, f.Name)
				}
				data, err := content.Expand(db, fileRid)
				if err != nil {
					return fmt.Errorf("expanding %s: %w", f.Name, err)
				}

				outPath := filepath.Join(dir, f.Name)
				if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
					return err
				}

				perm := os.FileMode(0o644)
				if f.Perm == "x" {
					perm = 0o755
				}
				if err := os.WriteFile(outPath, data, perm); err != nil {
					return err
				}
				fmt.Println(f.Name)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&version, "version", "", "Version to extract from (default: tip)")
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Output directory")
	return cmd
}
