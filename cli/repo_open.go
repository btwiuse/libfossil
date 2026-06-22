package cli

import (
	"fmt"
	"os"
	"path/filepath"

	libfossil "github.com/danmestas/libfossil"
	"github.com/spf13/cobra"
)

func NewOpenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open [dir]",
		Short: "Open a checkout in a directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			if Repo == "" {
				return fmt.Errorf("repository required (use -R <path>)")
			}

			absRepo, err := filepath.Abs(Repo)
			if err != nil {
				return err
			}

			ckoutPath := filepath.Join(dir, ".fslckout")
			if _, err := os.Stat(ckoutPath); err == nil {
				return fmt.Errorf("checkout already exists: %s", ckoutPath)
			}

			Repo = absRepo
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			co, err := r.CreateCheckout(dir, libfossil.CheckoutCreateOpts{})
			if err != nil {
				os.Remove(ckoutPath)
				return err
			}
			defer co.Close()

			tipRid, tipHash, err := co.Version()
			if err != nil {
				os.Remove(ckoutPath)
				return err
			}

			fmt.Printf("opened checkout in %s (repo: %s)\n", dir, absRepo)
			if tipRid > 0 {
				fmt.Printf("checked out version %s\n", tipHash[:10])
			} else {
				fmt.Println("empty repository -- no checkins yet")
			}
			return nil
		},
	}
	return cmd
}
