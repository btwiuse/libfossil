package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUndoCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "undo",
		Short: "Undo last operation",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()
			if err := r.Undo(dir); err != nil {
				return err
			}
			fmt.Println("undone")
			return nil
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}

func newRedoCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "redo",
		Short: "Redo undone operation",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()
			if err := r.Redo(dir); err != nil {
				return err
			}
			fmt.Println("redone")
			return nil
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}
