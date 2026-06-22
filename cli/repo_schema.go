package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/danmestas/libfossil/internal/repo"
	"github.com/spf13/cobra"
)

func newSchemaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Synced table schema operations",
	}

	cmd.AddCommand(newSchemaAddCommand())
	cmd.AddCommand(newSchemaListCommand())
	cmd.AddCommand(newSchemaShowCommand())
	cmd.AddCommand(newSchemaRemoveCommand())

	return cmd
}

func newSchemaAddCommand() *cobra.Command {
	var columns, conflict string
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Register a new synced table",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			inner := r.Inner()
			if err := repo.EnsureSyncSchema(inner.DB()); err != nil {
				return err
			}

			colDefs, err := parseColumnDefs(columns)
			if err != nil {
				return err
			}

			def := repo.TableDef{
				Columns:  colDefs,
				Conflict: conflict,
			}

			mtime := time.Now().Unix()
			if err := repo.RegisterSyncedTable(inner.DB(), name, def, mtime); err != nil {
				return err
			}

			fmt.Printf("Registered table %q with %d columns (%s)\n", name, len(colDefs), conflict)
			return nil
		},
	}
	cmd.Flags().StringVar(&columns, "columns", "", "Column defs: name:type[:pk],... (e.g. peer_id:text:pk,addr:text)")
	cmd.Flags().StringVar(&conflict, "conflict", "mtime-wins", "Conflict strategy")
	cmd.MarkFlagRequired("columns")
	return cmd
}

func parseColumnDefs(s string) ([]repo.ColumnDef, error) {
	if s == "" {
		return nil, fmt.Errorf("column definitions must not be empty")
	}

	parts := strings.Split(s, ",")
	var cols []repo.ColumnDef

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		fields := strings.Split(part, ":")
		if len(fields) < 2 {
			return nil, fmt.Errorf("column definition %q must have at least name:type", part)
		}

		col := repo.ColumnDef{
			Name: strings.TrimSpace(fields[0]),
			Type: strings.TrimSpace(fields[1]),
			PK:   false,
		}

		if len(fields) >= 3 && strings.TrimSpace(fields[2]) == "pk" {
			col.PK = true
		}

		cols = append(cols, col)
	}

	if len(cols) == 0 {
		return nil, fmt.Errorf("no valid columns found in definition")
	}

	return cols, nil
}

func newSchemaListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered synced tables",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			inner := r.Inner()
			if err := repo.EnsureSyncSchema(inner.DB()); err != nil {
				return err
			}

			tables, err := repo.ListSyncedTables(inner.DB())
			if err != nil {
				return err
			}

			if len(tables) == 0 {
				fmt.Println("(no synced tables registered)")
				return nil
			}

			fmt.Printf("%-20s %-15s %s\n", "TABLE", "CONFLICT", "COLUMNS")
			fmt.Printf("%-20s %-15s %s\n", strings.Repeat("-", 20), strings.Repeat("-", 15), strings.Repeat("-", 7))
			for _, t := range tables {
				fmt.Printf("%-20s %-15s %d\n", t.Name, t.Def.Conflict, len(t.Def.Columns))
			}
			return nil
		},
	}
	return cmd
}

func newSchemaShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Show table schema details",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			inner := r.Inner()
			if err := repo.EnsureSyncSchema(inner.DB()); err != nil {
				return err
			}

			tables, err := repo.ListSyncedTables(inner.DB())
			if err != nil {
				return err
			}

			var found *repo.TableInfo
			for _, t := range tables {
				if t.Name == name {
					t := t
					found = &t
					break
				}
			}

			if found == nil {
				return fmt.Errorf("table %q not found", name)
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(found)
		},
	}
	return cmd
}

func newSchemaRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a synced table registration",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			inner := r.Inner()
			if err := repo.EnsureSyncSchema(inner.DB()); err != nil {
				return err
			}

			if err := repo.ValidateTableName(name); err != nil {
				return err
			}

			tables, err := repo.ListSyncedTables(inner.DB())
			if err != nil {
				return err
			}
			found := false
			for _, t := range tables {
				if t.Name == name {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("table %q not registered", name)
			}

			if _, err := inner.DB().Exec("DELETE FROM _sync_schema WHERE name=?", name); err != nil {
				return fmt.Errorf("delete from _sync_schema: %w", err)
			}

			dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS x_%s", name)
			if _, err := inner.DB().Exec(dropSQL); err != nil {
				return fmt.Errorf("drop table: %w", err)
			}

			fmt.Printf("Removed table %q and dropped x_%s\n", name, name)
			return nil
		},
	}
	return cmd
}
