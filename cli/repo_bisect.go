package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newBisectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bisect",
		Short: "Binary search for bugs",
	}
	cmd.AddCommand(newBisectGoodCommand())
	cmd.AddCommand(newBisectBadCommand())
	cmd.AddCommand(newBisectSkipCommand())
	cmd.AddCommand(newBisectNextCommand())
	cmd.AddCommand(newBisectResetCommand())
	cmd.AddCommand(newBisectLsCommand())
	cmd.AddCommand(newBisectStatusCommand())
	return cmd
}

func newBisectGoodCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "good [version]",
		Short: "Mark version as good",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("bisect: not yet implemented (requires checkout DB)")
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}

func newBisectBadCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "bad [version]",
		Short: "Mark version as bad",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("bisect: not yet implemented (requires checkout DB)")
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}

func newBisectSkipCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "skip [version]",
		Short: "Skip current version",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("bisect: not yet implemented (requires checkout DB)")
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}

func newBisectNextCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "next",
		Short: "Check out midpoint version",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("bisect: not yet implemented (requires checkout DB)")
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}

func newBisectResetCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Clear bisect state",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("bisect: not yet implemented (requires checkout DB)")
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}

func newBisectLsCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "Show bisect path",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("bisect: not yet implemented (requires checkout DB)")
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}

func newBisectStatusCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show bisect state",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("bisect: not yet implemented (requires checkout DB)")
		},
	}
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Checkout directory")
	return cmd
}
