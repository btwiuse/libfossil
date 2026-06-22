package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRmCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "rm <files...>",
		Short: "Stage files for removal",
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

			for _, name := range args {
				var id int64
				err := ckout.QueryRow("SELECT id FROM vfile WHERE pathname=? AND vid=?", name, vid).Scan(&id)
				if err != nil {
					return fmt.Errorf("%s: not tracked in checkout", name)
				}
				ckout.Exec("UPDATE vfile SET deleted=1 WHERE id=?", id)
				fmt.Printf("REMOVED  %s\n", name)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}
