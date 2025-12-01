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

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	queryConnection string
	queryFormat     string
	queryQuiet      bool
)

var queryCmd = &cobra.Command{
	Use:   "query [SQL]",
	Short: "Execute a SQL query",
	Long: `Execute a SQL query against a saved connection.

Prerequisites:
  Create and save a connection first via:
    whodb-cli connect --type <db> --host <host> --user <user> --database <db> --name <name>

Output formats:
  auto   - Table for terminals, plain for pipes (default)
  table  - Human-readable table with borders
  plain  - Tab-separated values for grep/awk
  json   - JSON array of objects
  csv    - RFC 4180 CSV format`,
	Example: `  # Query with a named connection
  whodb-cli query --connection mydb "SELECT id, name FROM users LIMIT 5"

  # Pipe JSON to jq
  whodb-cli query --format json "SELECT * FROM users" | jq '.[].name'

  # Export to CSV file
  whodb-cli query --format csv "SELECT * FROM orders" > orders.csv

  # Use with grep (auto-selects plain format when piped)
  whodb-cli query "SELECT * FROM logs" | grep ERROR`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sql := args[0]

		format, err := output.ParseFormat(queryFormat)
		if err != nil {
			return err
		}

		out := output.New(
			output.WithFormat(format),
			output.WithQuiet(queryQuiet),
		)

		mgr, err := dbmgr.NewManager()
		if err != nil {
			return fmt.Errorf("cannot initialize database manager: %w", err)
		}

		var conn *dbmgr.Connection
		if queryConnection != "" {
			conn, err = mgr.GetConnection(queryConnection)
			if err != nil {
				return fmt.Errorf("connection %q not found", queryConnection)
			}
		} else {
			conns := mgr.ListConnections()
			if len(conns) == 0 {
				return fmt.Errorf("no saved connections. Create one first:\n  whodb-cli connect --type postgres --host localhost --user myuser --database mydb --name myconn")
			}
			conn = &conns[0]
			out.Info("Using connection: %s", conn.Name)
		}

		var spinner *output.Spinner
		if !queryQuiet {
			spinner = output.NewSpinner(fmt.Sprintf("Connecting to %s...", conn.Type))
		}
		spinner.Start()
		if err := mgr.Connect(conn); err != nil {
			spinner.StopWithError("Connection failed")
			return fmt.Errorf("cannot connect to database: %w", err)
		}
		spinner.StopWithSuccess("Connected")
		defer mgr.Disconnect()

		if !queryQuiet {
			spinner = output.NewSpinner("Executing query...")
		}
		spinner.Start()
		result, err := mgr.ExecuteQuery(sql)
		if err != nil {
			spinner.StopWithError("Query failed")
			return fmt.Errorf("query failed: %w", err)
		}
		spinner.Stop()

		columns := make([]output.Column, len(result.Columns))
		for i, col := range result.Columns {
			columns[i] = output.Column{Name: col.Name, Type: col.Type}
		}

		rows := make([][]any, len(result.Rows))
		for i, row := range result.Rows {
			rows[i] = make([]any, len(row))
			for j, cell := range row {
				rows[i][j] = cell
			}
		}

		queryResult := &output.QueryResult{
			Columns: columns,
			Rows:    rows,
		}

		return out.WriteQueryResult(queryResult)
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)

	queryCmd.Flags().StringVarP(&queryConnection, "connection", "c", "", "saved connection name to use")
	queryCmd.Flags().StringVarP(&queryFormat, "format", "f", "auto", "output format: auto, table, plain, json, csv")
	queryCmd.Flags().BoolVarP(&queryQuiet, "quiet", "q", false, "suppress informational messages")
}
