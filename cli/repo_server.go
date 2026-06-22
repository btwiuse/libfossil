package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newServerCommand() *cobra.Command {
	var addr string
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Serve a repository over Fossil HTTP sync",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Serving %s on http://%s\n", r.Path(), addr)
			return r.ServeHTTP(ctx, addr)
		},
	}
	cmd.Flags().StringVarP(&addr, "addr", "a", "127.0.0.1:8080", "Listen address")
	return cmd
}
