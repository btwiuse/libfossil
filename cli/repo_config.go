package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Repository configuration",
	}
	cmd.AddCommand(newConfigLsCommand())
	cmd.AddCommand(newConfigGetCommand())
	cmd.AddCommand(newConfigSetCommand())
	return cmd
}

func newConfigLsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List all config entries",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			db := r.Inner().DB()
			rows, err := db.Query("SELECT name, value FROM config ORDER BY name")
			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				var name, value string
				rows.Scan(&name, &value)
				fmt.Printf("%-20s %s\n", name, value)
			}
			return rows.Err()
		},
	}
	return cmd
}

func newConfigGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			key := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			val, err := r.Config(key)
			if err != nil {
				return fmt.Errorf("config key %q not found", key)
			}
			fmt.Println(val)
			return nil
		},
	}
	return cmd
}

func newConfigSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			if err := r.SetConfig(key, value); err != nil {
				return err
			}
			fmt.Printf("%s = %s\n", key, value)
			return nil
		},
	}
	return cmd
}
