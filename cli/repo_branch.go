package cli

import (
	"fmt"
	"time"

	libfossil "github.com/danmestas/libfossil"
	"github.com/danmestas/libfossil/internal/blob"
	"github.com/danmestas/libfossil/internal/content"
	"github.com/danmestas/libfossil/internal/fsltype"
	"github.com/danmestas/libfossil/internal/tag"
	"github.com/spf13/cobra"
)

func newBranchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Branch operations",
	}
	cmd.AddCommand(newBranchLsCommand())
	cmd.AddCommand(newBranchNewCommand())
	cmd.AddCommand(newBranchCloseCommand())
	return cmd
}

func newBranchLsCommand() *cobra.Command {
	var closed, all bool
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List branches",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			db := r.Inner().DB()

			var filter string
			switch {
			case all:
			case closed:
				filter = " AND tx.tagtype = 0"
			default:
				filter = " AND tx.tagtype > 0"
			}

			query := fmt.Sprintf(`
				SELECT DISTINCT substr(t.tagname, 5) AS branch, b.uuid,
				       datetime(e.mtime), e.user
				FROM tag t
				JOIN tagxref tx ON tx.tagid = t.tagid
				JOIN blob b ON b.rid = tx.rid
				LEFT JOIN event e ON e.objid = tx.rid
				WHERE t.tagname LIKE 'sym-%%'%s
				ORDER BY e.mtime DESC`, filter)

			rows, err := db.Query(query)
			if err != nil {
				return err
			}
			defer rows.Close()

			count := 0
			for rows.Next() {
				var branch, uuid string
				var mtime, user *string
				if err := rows.Scan(&branch, &uuid, &mtime, &user); err != nil {
					return err
				}
				short := uuid
				if len(short) > 10 {
					short = short[:10]
				}
				mt := ""
				if mtime != nil {
					mt = *mtime
				}
				u := ""
				if user != nil {
					u = *user
				}
				fmt.Printf("%-20s %s  %s  %s\n", branch, short, mt, u)
				count++
			}
			if err := rows.Err(); err != nil {
				return err
			}
			if count == 0 {
				fmt.Println("no branches found")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&closed, "closed", false, "Show only closed branches")
	cmd.Flags().BoolVar(&all, "all", false, "Show all branches (open and closed)")
	return cmd
}

func newBranchNewCommand() *cobra.Command {
	var from, message, user string
	cmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create new branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			parentRid, err := resolveRID(r, from)
			if err != nil {
				return fmt.Errorf("resolving parent: %w", err)
			}

			entries, err := r.ListFiles(parentRid)
			if err != nil {
				return fmt.Errorf("listing parent files: %w", err)
			}

			db := r.Inner().DB()
			files := make([]libfossil.FileToCommit, 0, len(entries))
			for _, e := range entries {
				fileRid, ok := blob.Exists(db, e.UUID)
				if !ok {
					return fmt.Errorf("blob %s not found for %s", e.UUID, e.Name)
				}
				data, err := content.Expand(db, fileRid)
				if err != nil {
					return fmt.Errorf("expanding %s: %w", e.Name, err)
				}
				files = append(files, libfossil.FileToCommit{
					Name:    e.Name,
					Content: data,
					Perm:    e.Perm,
				})
			}

			u := user
			if u == "" {
				u = currentUser()
			}

			comment := message
			if comment == "" {
				comment = "Create branch " + name
			}

			rid, uuid, err := r.Commit(libfossil.CommitOpts{
				Files:    files,
				Comment:  comment,
				User:     u,
				ParentID: parentRid,
				Time:     time.Now().UTC(),
				Tags: []libfossil.TagSpec{
					{Name: "branch", Value: name},
					{Name: "sym-" + name},
				},
			})
			if err != nil {
				return err
			}

			short := uuid
			if len(short) > 10 {
				short = short[:10]
			}
			fmt.Printf("branch %s created at %s (rid=%d)\n", name, short, rid)
			return nil
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "Parent version (default: tip)")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Checkin comment")
	cmd.Flags().StringVar(&user, "user", "", "Checkin user (default: OS username)")
	return cmd
}

func newBranchCloseCommand() *cobra.Command {
	var user string
	cmd := &cobra.Command{
		Use:   "close <name>",
		Short: "Close a branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			db := r.Inner().DB()
			tagName := "sym-" + name
			var tipRID int64
			err = db.QueryRow(`
				SELECT tx.rid FROM tagxref tx
				JOIN tag t ON t.tagid = tx.tagid
				WHERE t.tagname = ? AND tx.tagtype > 0
				ORDER BY tx.mtime DESC LIMIT 1`, tagName).Scan(&tipRID)
			if err != nil {
				return fmt.Errorf("branch %q not found or already closed", name)
			}

			u := user
			if u == "" {
				u = currentUser()
			}

			inner := r.Inner()
			fslID := fsltype.FslID(tipRID)

			if _, err := tag.AddTag(inner, tag.TagOpts{
				TargetRID: fslID,
				TagName:   tagName,
				TagType:   tag.TagCancel,
				User:      u,
			}); err != nil {
				return fmt.Errorf("cancelling branch tag: %w", err)
			}

			if _, err := tag.AddTag(inner, tag.TagOpts{
				TargetRID: fslID,
				TagName:   "closed",
				TagType:   tag.TagSingleton,
				User:      u,
			}); err != nil {
				return fmt.Errorf("adding closed tag: %w", err)
			}

			fmt.Printf("branch %s closed\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&user, "user", "", "User for control artifacts (default: OS username)")
	return cmd
}
