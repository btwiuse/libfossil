package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newTagCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Tag operations",
	}
	cmd.AddCommand(newTagLsCommand())
	cmd.AddCommand(newTagAddCommand())
	return cmd
}

func newTagLsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls [version]",
		Short: "List tags on an artifact",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			version := ""
			if len(args) > 0 {
				version = args[0]
			}
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			rid, err := resolveRID(r, version)
			if err != nil {
				return err
			}

			db := r.Inner().DB()
			rows, err := db.Query(`
				SELECT t.tagname, tx.value,
				       CASE tx.tagtype WHEN 1 THEN '+' WHEN 2 THEN '*' ELSE '-' END as type
				FROM tagxref tx
				JOIN tag t ON t.tagid=tx.tagid
				WHERE tx.rid=? AND tx.tagtype>0
				ORDER BY t.tagname`, rid)
			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				var name, value, tagtype string
				rows.Scan(&name, &value, &tagtype)
				if value != "" {
					fmt.Printf("%s%s=%s\n", tagtype, name, value)
				} else {
					fmt.Printf("%s%s\n", tagtype, name)
				}
			}
			return rows.Err()
		},
	}
	return cmd
}

func newTagAddCommand() *cobra.Command {
	var version string
	cmd := &cobra.Command{
		Use:   "add <tag> [value]",
		Short: "Add a tag to an artifact",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(_ *cobra.Command, args []string) error {
			tagName, value := args[0], ""
			if len(args) > 1 {
				value = args[1]
			}
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			rid, err := resolveRID(r, version)
			if err != nil {
				return err
			}

			db := r.Inner().DB()

			symName := "sym-" + tagName
			var tagID int64
			err = db.QueryRow("SELECT tagid FROM tag WHERE tagname=?", symName).Scan(&tagID)
			if err != nil {
				result, err := db.Exec("INSERT INTO tag(tagname) VALUES(?)", symName)
				if err != nil {
					return fmt.Errorf("creating tag: %w", err)
				}
				tagID, _ = result.LastInsertId()
			}

			_, err = db.Exec(`
				INSERT OR REPLACE INTO tagxref(tagid, tagtype, srcid, rid, mtime, value)
				VALUES(?, 1, 0, ?, julianday('now'), ?)`,
				tagID, rid, value)
			if err != nil {
				return fmt.Errorf("applying tag: %w", err)
			}

			var uuid string
			db.QueryRow("SELECT uuid FROM blob WHERE rid=?", rid).Scan(&uuid)
			if len(uuid) > 10 {
				uuid = uuid[:10]
			}
			fmt.Printf("tagged %s with %s\n", uuid, tagName)
			return nil
		},
	}
	cmd.Flags().StringVar(&version, "version", "", "Version to tag (default: tip)")
	return cmd
}
