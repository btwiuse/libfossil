package cli

import (
	"fmt"

	libfossil "github.com/danmestas/libfossil"
	"github.com/spf13/cobra"
)

func newAnnotateCommand() *cobra.Command {
	var version string
	cmd := &cobra.Command{
		Use:   "annotate <file>",
		Short: "Annotate file lines with version history",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			file := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			startRID, err := resolveRID(r, version)
			if err != nil {
				return err
			}

			lines, err := r.Annotate(libfossil.AnnotateOpts{
				FilePath: file,
				StartRID: startRID,
			})
			if err != nil {
				return err
			}

			for _, l := range lines {
				uuid := l.UUID
				if len(uuid) > 10 {
					uuid = uuid[:10]
				}
				fmt.Printf("%s %8s %s | %s\n",
					uuid, l.User, l.Date.Format("2006-01-02"), l.Text)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&version, "version", "", "Starting version (default: tip)")
	return cmd
}

func newBlameCommand() *cobra.Command {
	return newAnnotateCommand()
}
