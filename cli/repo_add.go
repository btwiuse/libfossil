package cli

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newAddCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "add <files...>",
		Short: "Stage files for addition",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			ckout, err := openCheckout(dir)
			if err != nil {
				return err
			}
			defer ckout.Close()

			vid, err := checkoutVid(ckout)
			if err != nil {
				return err
			}

			for _, path := range args {
				absPath, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				relPath, err := filepath.Rel(dir, absPath)
				if err != nil {
					return fmt.Errorf("file %s is not within checkout directory", path)
				}

				info, err := os.Stat(path)
				if err != nil {
					return fmt.Errorf("%s: %w", path, err)
				}
				if info.IsDir() {
					return fmt.Errorf("%s is a directory", path)
				}

				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				h := sha1.Sum(data)
				mhash := hex.EncodeToString(h[:])

				isExe := info.Mode()&0o111 != 0

				var existingID int64
				err = ckout.QueryRow("SELECT id FROM vfile WHERE pathname=? AND vid=?", relPath, vid).Scan(&existingID)
				if err == nil {
					ckout.Exec("UPDATE vfile SET chnged=1, deleted=0, mhash=? WHERE id=?", mhash, existingID)
					fmt.Printf("UPDATED  %s\n", relPath)
				} else {
					ckout.Exec(`INSERT INTO vfile(vid, chnged, deleted, isexe, islink, rid, mrid, mtime, pathname, mhash)
						VALUES(?, 1, 0, ?, 0, 0, 0, ?, ?, ?)`,
						vid, isExe, info.ModTime().Unix(), relPath, mhash)
					fmt.Printf("ADDED    %s\n", relPath)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}
