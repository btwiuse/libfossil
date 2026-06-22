package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStashCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stash",
		Short: "Stash working changes",
	}
	cmd.AddCommand(newStashSaveCommand())
	cmd.AddCommand(newStashPopCommand())
	cmd.AddCommand(newStashApplyCommand())
	cmd.AddCommand(newStashLsCommand())
	cmd.AddCommand(newStashDropCommand())
	cmd.AddCommand(newStashClearCommand())
	return cmd
}

func newStashSaveCommand() *cobra.Command {
	var message, dir string
	cmd := &cobra.Command{
		Use:   "save",
		Short: "Stash working changes",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()
			return r.StashSave(dir, message)
		},
	}
	cmd.Flags().StringVarP(&message, "message", "m", "", "Stash message")
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}

func newStashPopCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "pop",
		Short: "Apply top stash and drop it",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()
			return r.StashPop(dir)
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}

func newStashApplyCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "apply [id]",
		Short: "Apply stash without dropping",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			var id int64
			if len(args) > 0 {
				fmt.Sscanf(args[0], "%d", &id)
			}
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()
			return r.StashApply(dir, id)
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}

func newStashLsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List stash entries",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			entries, err := r.StashList()
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Println("no stash entries")
				return nil
			}
			for _, e := range entries {
				fmt.Printf("%3d: %s  %s\n", e.ID, e.Time, e.Comment)
			}
			return nil
		},
	}
	return cmd
}

func newStashDropCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drop <id>",
		Short: "Remove stash entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			var id int64
			fmt.Sscanf(args[0], "%d", &id)
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()
			return r.StashDrop(id)
		},
	}
	return cmd
}

func newStashClearCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Remove all stash entries",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()
			return r.StashClear()
		},
	}
	return cmd
}
