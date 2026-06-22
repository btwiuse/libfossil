package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	libfossil "github.com/danmestas/libfossil"
	"github.com/spf13/cobra"
)

func NewCiCommand() *cobra.Command {
	var message, user, parent, branch string
	cmd := &cobra.Command{
		Use:   "ci <files...>",
		Short: "Checkin file changes",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			files := args
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			var parentRid int64
			if parent != "" {
				parentRid, err = resolveRID(r, parent)
				if err != nil {
					return fmt.Errorf("resolving parent: %w", err)
				}
			} else {
				parentRid, _ = resolveRID(r, "")
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			commitFiles := make([]libfossil.FileToCommit, len(files))
			for i, path := range files {
				data, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("reading %s: %w", path, err)
				}
				absPath, err := filepath.Abs(path)
				if err != nil {
					return fmt.Errorf("resolving %s: %w", path, err)
				}
				name, err := filepath.Rel(cwd, absPath)
				if err != nil {
					return fmt.Errorf("resolving %s: %w", path, err)
				}
				name = filepath.Clean(name)
				if name == "." || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
					return fmt.Errorf("%s is outside current directory", path)
				}
				commitFiles[i] = libfossil.FileToCommit{
					Name:    filepath.ToSlash(name),
					Content: data,
				}
			}

			u := user
			if u == "" {
				u = currentUser()
			}

			var tags []libfossil.TagSpec
			if branch != "" {
				tags = append(tags,
					libfossil.TagSpec{Name: "branch", Value: branch},
					libfossil.TagSpec{Name: "sym-" + branch},
				)
			}

			rid, uuid, err := r.Commit(libfossil.CommitOpts{
				Files:    commitFiles,
				Comment:  message,
				User:     u,
				ParentID: parentRid,
				Time:     time.Now().UTC(),
				Tags:     tags,
			})
			if err != nil {
				return err
			}

			fmt.Printf("checkin %s (rid=%d)\n", uuid[:10], rid)
			return nil
		},
	}
	cmd.Flags().StringVarP(&message, "message", "m", "", "Checkin comment")
	cmd.Flags().StringVar(&user, "user", "", "Checkin user (default: OS username)")
	cmd.Flags().StringVar(&parent, "parent", "", "Parent version UUID (default: tip)")
	cmd.Flags().StringVar(&branch, "branch", "", "Branch name for this checkin")
	cmd.MarkFlagRequired("message")
	return cmd
}
