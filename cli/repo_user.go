package cli

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	libfossil "github.com/danmestas/libfossil"
	"github.com/spf13/cobra"
)

func newUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "User management",
	}
	cmd.AddCommand(newUserAddCommand())
	cmd.AddCommand(newUserListCommand())
	cmd.AddCommand(newUserUpdateCommand())
	cmd.AddCommand(newUserRmCommand())
	cmd.AddCommand(newUserPasswdCommand())
	return cmd
}

func newUserAddCommand() *cobra.Command {
	var cap string
	cmd := &cobra.Command{
		Use:   "add <login>",
		Short: "Create a new user",
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
			fmt.Printf("Created user %q (caps: %s)\n", login, cap)
			fmt.Printf("Password: %s\n", password)
			return nil
		},
	}
	cmd.Flags().StringVar(&cap, "cap", "", "Capability string (e.g. oi)")
	cmd.MarkFlagRequired("cap")
	return cmd
}

func newUserListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all users",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			users, err := r.ListUsers()
			if err != nil {
				return err
			}
			fmt.Printf("%-20s %-20s\n", "LOGIN", "CAPABILITIES")
			for _, u := range users {
				fmt.Printf("%-20s %-20s\n", u.Login, u.Caps)
			}
			return nil
		},
	}
	return cmd
}

func newUserUpdateCommand() *cobra.Command {
	var cap string
	cmd := &cobra.Command{
		Use:   "update <login>",
		Short: "Update user capabilities",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			login := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			if err := r.SetCaps(login, cap); err != nil {
				return err
			}
			fmt.Printf("Updated %q capabilities: %s\n", login, cap)
			return nil
		},
	}
	cmd.Flags().StringVar(&cap, "cap", "", "New capability string")
	cmd.MarkFlagRequired("cap")
	return cmd
}

func newUserRmCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm <login>",
		Short: "Delete a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			login := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			if err := r.DeleteUser(login); err != nil {
				return err
			}
			fmt.Printf("Deleted user %q\n", login)
			return nil
		},
	}
	return cmd
}

func newUserPasswdCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "passwd <login>",
		Short: "Reset user password",
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

			if err := r.SetPassword(login, password); err != nil {
				return err
			}
			fmt.Printf("New password for %q: %s\n", login, password)
			return nil
		},
	}
	return cmd
}

func generatePassword() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating password: %w", err)
	}
	return hex.EncodeToString(b), nil
}
