package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRenameCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "rename <from> <to>",
		Short: "Rename a tracked file",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			from, to := args[0], args[1]
			ckout, err := openCheckout(dir)
			if err != nil {
				return err
			}
			defer ckout.Close()

			vid, err := checkoutVid(ckout)
			if err != nil {
				return err
			}

			var id int64
			err = ckout.QueryRow("SELECT id FROM vfile WHERE pathname=? AND vid=?", from, vid).Scan(&id)
			if err != nil {
				return fmt.Errorf("%s: not tracked in checkout", from)
			}

			var existing int64
			if ckout.QueryRow("SELECT id FROM vfile WHERE pathname=? AND vid=?", to, vid).Scan(&existing) == nil {
				return fmt.Errorf("%s: already tracked in checkout", to)
			}

			_, err = ckout.Exec("UPDATE vfile SET pathname=?, origname=?, chnged=1 WHERE id=?", to, from, id)
			if err != nil {
				return err
			}
			fmt.Printf("RENAMED  %s -> %s\n", from, to)
			return nil
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}
