package cli

import (
	"context"
	"fmt"

	libfossil "github.com/danmestas/libfossil"
	"github.com/spf13/cobra"
)

func newPushCommand() *cobra.Command {
	var user, pass string
	cmd := &cobra.Command{
		Use:   "push [url]",
		Short: "Push changes to a remote repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			transport := libfossil.NewHTTPTransport(args[0])
			result, err := r.Sync(ctx, transport, libfossil.SyncOpts{
				Push:     true,
				User:     user,
				Password: pass,
			})
			if err != nil {
				return fmt.Errorf("push failed: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Push complete\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  Rounds:   %d\n", result.Rounds)
			fmt.Fprintf(cmd.OutOrStdout(), "  Sent:     %d files (%d bytes)\n", result.FilesSent, result.BytesSent)
			fmt.Fprintf(cmd.OutOrStdout(), "  Received: %d files (%d bytes)\n", result.FilesRecvd, result.BytesRecvd)
			if len(result.Errors) > 0 {
				for _, e := range result.Errors {
					fmt.Fprintf(cmd.ErrOrStderr(), "  Error:    %s\n", e)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&user, "user", "u", "", "Username for push auth")
	cmd.Flags().StringVarP(&pass, "pass", "p", "", "Password for push auth")
	return cmd
}
