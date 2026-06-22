package cli

import (
	"fmt"
	"time"

	libfossil "github.com/danmestas/libfossil"
	"github.com/danmestas/libfossil/internal/auth"
	"github.com/spf13/cobra"
)

func newInviteCommand() *cobra.Command {
	var cap, url string
	var ttl time.Duration
	cmd := &cobra.Command{
		Use:   "invite <login>",
		Short: "Generate invite token for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			login := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			password, err := generatePassword()
			if err != nil {
				return err
			}

			if err := r.CreateUser(libfossil.UserOpts{
				Login:    login,
				Password: password,
				Caps:     cap,
			}); err != nil {
				return err
			}

			if ttl > 0 {
				expiry := time.Now().Add(ttl).Format("2006-01-02 15:04:05")
				r.Inner().DB().Exec("UPDATE user SET cexpire=? WHERE login=?", expiry, login)
			}

			u := url
			if u == "" {
				val, err := r.Config("last-sync-url")
				if err == nil {
					u = val
				}
			}

			tok := auth.InviteToken{
				URL:      u,
				Login:    login,
				Password: password,
				Caps:     cap,
			}

			encoded := tok.Encode()

			fmt.Printf("Invite for %q (capabilities: %s", login, cap)
			if ttl > 0 {
				fmt.Printf(", expires: %s", time.Now().Add(ttl).Format(time.RFC3339))
			}
			fmt.Println("):")
			fmt.Println()
			fmt.Printf("  fossil repo clone --invite %s\n", encoded)
			fmt.Println()
			fmt.Println("Share this command with the recipient. It contains credentials - treat it like a password.")
			return nil
		},
	}
	cmd.Flags().StringVar(&cap, "cap", "", "Capability string (e.g. oi)")
	cmd.Flags().StringVar(&url, "url", "", "Sync URL to embed in token")
	cmd.Flags().DurationVar(&ttl, "ttl", 0, "Token time-to-live (e.g. 24h)")
	cmd.MarkFlagRequired("cap")
	return cmd
}
