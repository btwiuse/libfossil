package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newQueryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query <sql>",
		Short: "Execute SQL against repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			sql := args[0]
			r, err := OpenRepo()
			if err != nil {
				return err
			}
			defer r.Close()

			db := r.Inner().DB()

			upper := strings.ToUpper(strings.TrimSpace(sql))
			if strings.HasPrefix(upper, "SELECT") || strings.HasPrefix(upper, "PRAGMA") {
				rows, err := db.Query(sql)
				if err != nil {
					return err
				}
				defer rows.Close()

				cols, err := rows.Columns()
				if err != nil {
					return err
				}

				fmt.Println(strings.Join(cols, "|"))

				vals := make([]any, len(cols))
				ptrs := make([]any, len(cols))
				for i := range vals {
					ptrs[i] = &vals[i]
				}
				for rows.Next() {
					rows.Scan(ptrs...)
					parts := make([]string, len(cols))
					for i, v := range vals {
						parts[i] = fmt.Sprintf("%v", v)
					}
					fmt.Println(strings.Join(parts, "|"))
				}
				return rows.Err()
			}

			result, err := db.Exec(sql)
			if err != nil {
				return err
			}
			affected, _ := result.RowsAffected()
			fmt.Printf("%d rows affected\n", affected)
			return nil
		},
	}
	return cmd
}
