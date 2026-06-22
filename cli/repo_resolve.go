package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newResolveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve <name>",
		Short: "Resolve symbolic name to UUID",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			rid, err := resolveRID(r, name)
			if err != nil {
				return err
			}

			var uuid string
			r.Inner().DB().QueryRow("SELECT uuid FROM blob WHERE rid=?", rid).Scan(&uuid)

			fmt.Printf("rid:  %d\n", rid)
			fmt.Printf("uuid: %s\n", uuid)
			return nil
		},
	}
	return cmd
}
