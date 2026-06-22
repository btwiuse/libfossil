package cli

import (
	"fmt"

	libfossil "github.com/danmestas/libfossil"
	"github.com/spf13/cobra"
)

func newTimelineCommand() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "timeline",
		Short: "Show repository history",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			tipRid, err := resolveRID(r, "")
			if err != nil {
				return err
			}

			entries, err := r.Timeline(libfossil.LogOpts{Start: tipRid, Limit: limit})
			if err != nil {
				return err
			}

			for _, e := range entries {
				uuid := e.UUID
				if len(uuid) > 10 {
					uuid = uuid[:10]
				}
				fmt.Printf("%s  %s  %s  %s\n", uuid, e.Time.Format("2006-01-02 15:04"), e.User, e.Comment)
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Number of entries")
	return cmd
}
