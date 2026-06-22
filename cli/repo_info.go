package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Repository statistics",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			db := r.Inner().DB()

			var blobCount, deltaCount, phantomCount int
			var totalSize int64
			var projectCode, serverCode string

			db.QueryRow("SELECT count(*) FROM blob WHERE size >= 0").Scan(&blobCount)
			db.QueryRow("SELECT count(*) FROM delta").Scan(&deltaCount)
			db.QueryRow("SELECT count(*) FROM phantom").Scan(&phantomCount)
			db.QueryRow("SELECT coalesce(sum(size),0) FROM blob WHERE size >= 0").Scan(&totalSize)
			db.QueryRow("SELECT value FROM config WHERE name='project-code'").Scan(&projectCode)
			db.QueryRow("SELECT value FROM config WHERE name='server-code'").Scan(&serverCode)

			fmt.Printf("project-code:  %s\n", projectCode)
			fmt.Printf("server-code:   %s\n", serverCode)
			fmt.Printf("blobs:         %d\n", blobCount)
			fmt.Printf("deltas:        %d\n", deltaCount)
			fmt.Printf("phantoms:      %d\n", phantomCount)
			fmt.Printf("total size:    %d bytes\n", totalSize)
			return nil
		},
	}
	return cmd
}
