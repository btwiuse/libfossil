package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newLsCommand() *cobra.Command {
	var long bool
	cmd := &cobra.Command{
		Use:   "ls [version]",
		Short: "List files in a version",
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

			for _, f := range files {
				if long {
					fmt.Printf("%s  %s  %s\n", f.UUID[:10], f.Perm, f.Name)
				} else {
					fmt.Println(f.Name)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&long, "long", "l", false, "Show sizes and hashes")
	return cmd
}
