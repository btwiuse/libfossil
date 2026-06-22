package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func newUVCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uv",
		Short: "Unversioned file operations",
	}
	cmd.AddCommand(newUVLsCommand())
	cmd.AddCommand(newUVPutCommand())
	cmd.AddCommand(newUVGetCommand())
	cmd.AddCommand(newUVDeleteCommand())
	return cmd
}

func newUVLsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List unversioned files",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			entries, err := r.UVList()
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				fmt.Println("(no unversioned files)")
				return nil
			}

			for _, e := range entries {
				ts := e.Mtime.UTC().Format("2006-01-02 15:04:05")
				if e.Hash == "" {
					fmt.Printf("%-40s %s  [deleted]\n", e.Name, ts)
				} else {
					hash := e.Hash
					if len(hash) > 10 {
						hash = hash[:10]
					}
					fmt.Printf("%-40s %s  %6d  %s\n", e.Name, ts, e.Size, hash)
				}
			}
			return nil
		},
	}
	return cmd
}

func newUVPutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "put <name> <file>",
		Short: "Add or update an unversioned file",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			name, file := args[0], args[1]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("read %s: %w", file, err)
			}

			if err := r.UVWrite(name, data, time.Now()); err != nil {
				return err
			}

			fmt.Printf("wrote %s (%d bytes)\n", name, len(data))
			return nil
		},
	}
	return cmd
}

func newUVGetCommand() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Retrieve an unversioned file",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			data, _, hash, err := r.UVRead(name)
			if err != nil {
				return err
			}
			if data == nil && hash == "" {
				return fmt.Errorf("unversioned file %q not found", name)
			}
			if hash == "" {
				return fmt.Errorf("unversioned file %q has been deleted", name)
			}

			if output != "" {
				return os.WriteFile(output, data, 0o644)
			}
			_, err = os.Stdout.Write(data)
			return err
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file (default: stdout)")
	return cmd
}

func newUVDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an unversioned file (creates tombstone)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			if err := r.UVWrite(name, nil, time.Now()); err != nil {
				return err
			}

			fmt.Printf("deleted %s\n", name)
			return nil
		},
	}
	return cmd
}
