package cli

import (
	"fmt"
	"os"

	"github.com/danmestas/libfossil/internal/blob"
	"github.com/danmestas/libfossil/internal/content"
	"github.com/danmestas/libfossil/internal/fsltype"
	"github.com/spf13/cobra"
)

func newCatCommand() *cobra.Command {
	var raw bool
	cmd := &cobra.Command{
		Use:   "cat <artifact>",
		Short: "Output artifact content",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			artifact := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			rid, err := resolveRID(r, artifact)
			if err != nil {
				return err
			}

			db := r.Inner().DB()
			var data []byte
			if raw {
				data, err = blob.Load(db, fsltype.FslID(rid))
			} else {
				data, err = content.Expand(db, fsltype.FslID(rid))
			}
			if err != nil {
				return fmt.Errorf("reading artifact: %w", err)
			}

			os.Stdout.Write(data)
			return nil
		},
	}
	cmd.Flags().BoolVar(&raw, "raw", false, "Output raw blob (no delta expansion)")
	return cmd
}
