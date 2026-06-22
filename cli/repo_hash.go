package cli

import (
	"fmt"
	"os"

	"github.com/danmestas/libfossil/internal/hash"
	"github.com/spf13/cobra"
)

func newHashCommand() *cobra.Command {
	var sha3 bool
	cmd := &cobra.Command{
		Use:   "hash <files...>",
		Short: "Hash files (SHA1 or SHA3)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			for _, path := range args {
				data, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("reading %s: %w", path, err)
				}
				var h string
				if sha3 {
					h = hash.SHA3(data)
				} else {
					h = hash.SHA1(data)
				}
				fmt.Printf("%s  %s\n", h, path)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&sha3, "sha3", false, "Use SHA3-256 instead of SHA1")
	return cmd
}
