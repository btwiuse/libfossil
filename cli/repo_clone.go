package cli

import (
	"context"
	"fmt"
	"time"

	libfossil "github.com/danmestas/libfossil"
	"github.com/danmestas/libfossil/internal/auth"
	"github.com/spf13/cobra"
)

func newCloneCommand() *cobra.Command {
	var user, pass, invite string
	cmd := &cobra.Command{
		Use:   "clone [url] [path]",
		Short: "Clone a remote repository",
		Args:  cobra.RangeArgs(0, 2),
		RunE: func(_ *cobra.Command, args []string) error {
			url := ""
			path := ""
			switch len(args) {
			case 2:
				url, path = args[0], args[1]
			case 1:
				url = args[0]
			}

			if invite != "" {
				token, err := auth.DecodeInviteToken(invite)
				if err != nil {
					return fmt.Errorf("invalid invite token: %w", err)
				}
				if url == "" {
					url = token.URL
				}
				if user == "" {
					user = token.Login
				}
				if pass == "" {
					pass = token.Password
				}
			}
			if url == "" {
				return fmt.Errorf("URL required (provide as argument or via --invite token)")
			}
			if path == "" {
				return fmt.Errorf("path required")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			transport := libfossil.NewHTTPTransport(url)
			opts := libfossil.CloneOpts{
				User:     user,
				Password: pass,
			}

			r, result, err := libfossil.Clone(ctx, path, transport, opts)
			if err != nil {
				return fmt.Errorf("clone failed: %w", err)
			}
			defer r.Close()

			fmt.Printf("Cloned into %s\n", path)
			fmt.Printf("  Rounds:       %d\n", result.Rounds)
			fmt.Printf("  Blobs:        %d\n", result.BlobsRecvd)
			fmt.Printf("  Project-code: %s\n", result.ProjectCode)
			return nil
		},
	}
	cmd.Flags().StringVarP(&user, "user", "u", "", "Username for clone auth")
	cmd.Flags().StringVarP(&pass, "pass", "p", "", "Password for clone auth")
	cmd.Flags().StringVar(&invite, "invite", "", "Invite token (from fossil invite)")
	return cmd
}
