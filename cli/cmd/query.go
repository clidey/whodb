/*
 * Copyright 2026 Clidey, Inc.
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
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/pkg/analytics"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	queryConnection string
	queryFormat     string
	queryQuiet      bool
)

var queryCmd = &cobra.Command{
	Use:           "query [SQL]",
	Short:         "Execute a SQL query",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Execute a SQL query against a connection.

Prerequisites:
  Use a saved connection:
    whodb-cli connect --type <db> --host <host> --user <user> --database <db> --name <name>
  Or configure an environment profile, for example:
    WHODB_POSTGRES='[{"alias":"prod","host":"localhost","user":"user","password":"pass","database":"db","port":"5432"}]'
    WHODB_MYSQL_1='{"alias":"dev","host":"localhost","user":"user","password":"pass","database":"db","port":"3306"}'

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
  whodb-cli query "SELECT * FROM logs" | grep ERROR

  # Read SQL from stdin
  echo "SELECT * FROM users" | whodb-cli query -
  cat query.sql | whodb-cli query -`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("missing SQL query\n\nUsage: whodb-cli query [SQL]\n\nExample:\n  whodb-cli query \"SELECT * FROM users LIMIT 10\"\n\nRun 'whodb-cli query --help' for more options")
		}
		if len(args) > 1 {
			return fmt.Errorf("too many arguments (expected 1 SQL query, got %d)\n\nTip: Wrap your SQL in quotes:\n  whodb-cli query \"SELECT * FROM users\"", len(args))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		startTime := time.Now()
		sql := args[0]

		if sql == "-" {
			scanner := bufio.NewScanner(os.Stdin)
			var lines []string
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("reading from stdin: %w", err)
			}
			sql = strings.Join(lines, "\n")
			if strings.TrimSpace(sql) == "" {
				return fmt.Errorf("no SQL provided via stdin")
			}
		}

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
			conn, _, err = mgr.ResolveConnection(queryConnection)
			if err != nil {
				return err
			}
		} else {
			conns := mgr.ListAvailableConnections()
			if len(conns) == 0 {
				return fmt.Errorf("no connections available. Create one first:\n  whodb-cli connect --type postgres --host localhost --user myuser --database mydb --name myconn")
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
			analytics.TrackConnectError(ctx, conn.Type, "connection_failed", time.Since(startTime).Milliseconds())
			return fmt.Errorf("cannot connect to database: %w", err)
		}
		spinner.StopWithSuccess("Connected")
		defer mgr.Disconnect()

		if !queryQuiet {
			spinner = output.NewSpinner("Executing query...")
		}
		spinner.Start()
		queryStart := time.Now()
		result, err := mgr.ExecuteQuery(sql)
		if err != nil {
			spinner.StopWithError("Query failed")
			analytics.TrackQueryError(ctx, conn.Type, "execution_failed", time.Since(queryStart).Milliseconds())
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

		// Track successful query execution
		analytics.TrackQueryExecute(ctx, conn.Type, detectStatementType(sql), true,
			time.Since(queryStart).Milliseconds(), len(rows), map[string]any{
				"format": string(format),
			})

		queryResult := &output.QueryResult{
			Columns: columns,
			Rows:    rows,
		}

		return out.WriteQueryResult(queryResult)
	},
}

// detectStatementType returns the SQL statement type (SELECT, INSERT, etc.)
func detectStatementType(sql string) string {
	sql = strings.TrimSpace(strings.ToUpper(sql))
	switch {
	case strings.HasPrefix(sql, "SELECT"), strings.HasPrefix(sql, "WITH"):
		return "SELECT"
	case strings.HasPrefix(sql, "INSERT"):
		return "INSERT"
	case strings.HasPrefix(sql, "UPDATE"):
		return "UPDATE"
	case strings.HasPrefix(sql, "DELETE"):
		return "DELETE"
	case strings.HasPrefix(sql, "CREATE"):
		return "CREATE"
	case strings.HasPrefix(sql, "ALTER"):
		return "ALTER"
	case strings.HasPrefix(sql, "DROP"):
		return "DROP"
	case strings.HasPrefix(sql, "TRUNCATE"):
		return "TRUNCATE"
	case strings.HasPrefix(sql, "SHOW"):
		return "SHOW"
	case strings.HasPrefix(sql, "DESCRIBE"), strings.HasPrefix(sql, "DESC"):
		return "DESCRIBE"
	case strings.HasPrefix(sql, "EXPLAIN"):
		return "EXPLAIN"
	default:
		return "OTHER"
	}
}

func init() {
	rootCmd.AddCommand(queryCmd)

	queryCmd.Flags().StringVarP(&queryConnection, "connection", "c", "", "connection name to use")
	queryCmd.Flags().StringVarP(&queryFormat, "format", "f", "auto", "output format: auto, table, plain, json, csv")
	queryCmd.Flags().BoolVarP(&queryQuiet, "quiet", "q", false, "suppress informational messages")
}
