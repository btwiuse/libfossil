package cli

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/danmestas/libfossil/internal/blob"
	"github.com/danmestas/libfossil/internal/content"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show working directory changes",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			tipRid, err := resolveRID(r, "tip")
			if err != nil {
				fmt.Println("empty repository -- no checkins")
				return nil
			}

			manifestFiles, err := r.ListFiles(tipRid)
			if err != nil {
				return err
			}

			expected := make(map[string]string)
			for _, f := range manifestFiles {
				expected[f.Name] = f.UUID
			}

			db := r.Inner().DB()

			seen := make(map[string]bool)
			err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					name := d.Name()
					if name == ".fslckout" || name == "_FOSSIL_" || name == ".fsl" || strings.HasPrefix(name, ".") {
						return filepath.SkipDir
					}
					return nil
				}

				relPath, err := filepath.Rel(dir, path)
				if err != nil {
					return err
				}
				if strings.HasPrefix(filepath.Base(relPath), ".") {
					return nil
				}

				seen[relPath] = true

				uuid, inManifest := expected[relPath]
				if !inManifest {
					fmt.Printf("EXTRA    %s\n", relPath)
					return nil
				}

				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				h := sha1.Sum(data)
				diskHash := hex.EncodeToString(h[:])

				if diskHash != uuid {
					fileRid, ok := blob.Exists(db, uuid)
					if ok {
						blobData, err := content.Expand(db, fileRid)
						if err == nil {
							bh := sha1.Sum(blobData)
							blobHash := hex.EncodeToString(bh[:])
							if blobHash == diskHash {
								return nil
							}
						}
					}
					fmt.Printf("EDITED   %s\n", relPath)
				}

				return nil
			})
			if err != nil {
				return err
			}

			for name := range expected {
				if !seen[name] {
					fmt.Printf("MISSING  %s\n", name)
				}
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory to scan")
	return cmd
}
