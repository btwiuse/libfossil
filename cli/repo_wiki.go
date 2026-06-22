package cli

import (
	"fmt"
	"os"

	"github.com/danmestas/libfossil/internal/content"
	"github.com/danmestas/libfossil/internal/deck"
	"github.com/danmestas/libfossil/internal/fsltype"
	"github.com/spf13/cobra"
)

func newWikiCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wiki",
		Short: "Wiki page operations",
	}
	cmd.AddCommand(newWikiLsCommand())
	cmd.AddCommand(newWikiExportCommand())
	return cmd
}

func newWikiLsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List wiki pages",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			db := r.Inner().DB()
			rows, err := db.Query(`
				SELECT DISTINCT b.uuid, b.rid FROM event e
				JOIN blob b ON b.rid=e.objid
				WHERE e.type='w'
				ORDER BY e.mtime DESC`)
			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				var uuid string
				var rid int64
				rows.Scan(&uuid, &rid)

				data, err := content.Expand(db, fsltype.FslID(rid))
				if err != nil {
					continue
				}
				d, err := deck.Parse(data)
				if err != nil {
					continue
				}
				if d.L != "" {
					fmt.Printf("%-40s %s\n", d.L, uuid[:10])
				}
			}
			return rows.Err()
		},
	}
	return cmd
}

func newWikiExportCommand() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "export <page>",
		Short: "Export a wiki page",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			page := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			db := r.Inner().DB()
			var data []byte

			rid, err := resolveRID(r, page)
			if err == nil {
				data, err = content.Expand(db, fsltype.FslID(rid))
				if err != nil {
					return err
				}
			} else {
				rows, err := db.Query(`
					SELECT b.rid FROM event e
					JOIN blob b ON b.rid=e.objid
					WHERE e.type='w'
					ORDER BY e.mtime DESC`)
				if err != nil {
					return err
				}
				defer rows.Close()

				found := false
				for rows.Next() {
					var rid int64
					rows.Scan(&rid)
					d, err := content.Expand(db, fsltype.FslID(rid))
					if err != nil {
						continue
					}
					parsed, err := deck.Parse(d)
					if err != nil {
						continue
					}
					if parsed.L == page {
						data = parsed.W
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("wiki page %q not found", page)
				}
			}

			if data != nil && rid > 0 {
				parsed, err := deck.Parse(data)
				if err == nil && len(parsed.W) > 0 {
					data = parsed.W
				}
			}

			if output != "" {
				return os.WriteFile(output, data, 0o644)
			}
			os.Stdout.Write(data)
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file (default: stdout)")
	return cmd
}
