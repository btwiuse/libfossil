package cli

import (
	"fmt"

	libfossil "github.com/danmestas/libfossil"
	"github.com/spf13/cobra"
)

func newNewCommand() *cobra.Command {
	var user string
	cmd := &cobra.Command{
		Use:   "new <path>",
		Short: "Create a new repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path := args[0]
			if user == "" {
				user = currentUser()
			}
			r, err := libfossil.Create(path, libfossil.CreateOpts{User: user})
			if err != nil {
				return err
			}
			r.Close()
			fmt.Printf("created repository: %s\n", path)
			return nil
		},
	}
	cmd.Flags().StringVar(&user, "user", "", "Default user name")
	return cmd
}
