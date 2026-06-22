package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVerifyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify repository integrity",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			if err := r.Verify(); err != nil {
				return fmt.Errorf("verification failed: %w", err)
			}
			fmt.Println("repository integrity verified")
			return nil
		},
	}
	return cmd
}
