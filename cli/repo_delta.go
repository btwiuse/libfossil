package cli

import (
	"fmt"
	"os"

	"github.com/danmestas/libfossil/internal/delta"
	"github.com/spf13/cobra"
)

func newDeltaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delta",
		Short: "Delta create/apply operations",
	}
	cmd.AddCommand(newDeltaCreateCommand())
	cmd.AddCommand(newDeltaApplyCommand())
	return cmd
}

func newDeltaCreateCommand() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "create <source> <target>",
		Short: "Create a delta between two files",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			source, target := args[0], args[1]
			src, err := os.ReadFile(source)
			if err != nil {
				return fmt.Errorf("reading source: %w", err)
			}
			tgt, err := os.ReadFile(target)
			if err != nil {
				return fmt.Errorf("reading target: %w", err)
			}

			d := delta.Create(src, tgt)

			if output != "" {
				return os.WriteFile(output, d, 0o644)
			}
			os.Stdout.Write(d)
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file (default: stdout)")
	return cmd
}

func newDeltaApplyCommand() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "apply <source> <delta>",
		Short: "Apply a delta to a source file",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			source, deltaFile := args[0], args[1]
			src, err := os.ReadFile(source)
			if err != nil {
				return fmt.Errorf("reading source: %w", err)
			}
			d, err := os.ReadFile(deltaFile)
			if err != nil {
				return fmt.Errorf("reading delta: %w", err)
			}

			result, err := delta.Apply(src, d)
			if err != nil {
				return fmt.Errorf("applying delta: %w", err)
			}

			if output != "" {
				return os.WriteFile(output, result, 0o644)
			}
			os.Stdout.Write(result)
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file (default: stdout)")
	return cmd
}
