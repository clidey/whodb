/*
 * Copyright 2025 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/spf13/cobra"
)

var (
	queryConnection string
	queryFormat     string
	queryOutput     string
)

var queryCmd = &cobra.Command{
	Use:   "query [SQL]",
	Short: "Execute a SQL query",
	Long: `Execute a SQL query against a saved connection.

Prerequisites:
  - Create and save a connection first via: whodb-cli connect --type <db> --host <host> --user <user> --database <db> --name <name>
  - Enter the password when prompted. The query command does not prompt for passwords and uses saved credentials.

Flags:
  --connection <name>  Use a specific saved connection (defaults to the first saved connection if omitted)
  --format <table|json|csv>  Output format (default: table). Note: json/csv currently print placeholders.
  --output <path>  Output file (currently unused; prints to stdout)`,
	Example: `
  # Use a named connection
  whodb-cli query --connection app-local "SELECT id, name FROM users LIMIT 5;"

  # Use the first saved connection
  whodb-cli query "SELECT 1;"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sql := args[0]

		mgr, err := dbmgr.NewManager()
		if err != nil {
			return fmt.Errorf("error creating database manager: %w", err)
		}

		var conn *dbmgr.Connection
		if queryConnection != "" {
			conn, err = mgr.GetConnection(queryConnection)
			if err != nil {
				return fmt.Errorf("error getting connection: %w", err)
			}
		} else {
			conns := mgr.ListConnections()
			if len(conns) == 0 {
				return fmt.Errorf("no connections configured. Use 'whodb-cli connect' first")
			}
			conn = &conns[0]
		}

		if err := mgr.Connect(conn); err != nil {
			return fmt.Errorf("error connecting to database: %w", err)
		}
		defer mgr.Disconnect()

		result, err := mgr.ExecuteQuery(sql)
		if err != nil {
			return fmt.Errorf("error executing query: %w", err)
		}

		switch queryFormat {
		case "table":
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

			for i, col := range result.Columns {
				fmt.Fprint(w, col.Name)
				if i < len(result.Columns)-1 {
					fmt.Fprint(w, "\t")
				}
			}
			fmt.Fprintln(w)

			for i := range result.Columns {
				fmt.Fprint(w, "---")
				if i < len(result.Columns)-1 {
					fmt.Fprint(w, "\t")
				}
			}
			fmt.Fprintln(w)

			for _, row := range result.Rows {
				for i, cell := range row {
					fmt.Fprint(w, cell)
					if i < len(row)-1 {
						fmt.Fprint(w, "\t")
					}
				}
				fmt.Fprintln(w)
			}
			w.Flush()

		case "json":
			fmt.Println("JSON output not yet implemented")

		case "csv":
			fmt.Println("CSV output not yet implemented")

		default:
			return fmt.Errorf("unknown output format: %s", queryFormat)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)

	queryCmd.Flags().StringVar(&queryConnection, "connection", "", "connection name to use")
	queryCmd.Flags().StringVar(&queryFormat, "format", "table", "output format (table, json, csv)")
	queryCmd.Flags().StringVar(&queryOutput, "output", "", "output file (default: stdout)")
}
