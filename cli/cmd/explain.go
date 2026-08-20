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
	"fmt"
	"os"
	"strings"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	explainConnection string
	explainFormat     string
	explainQuiet      bool
)

var explainCmd = &cobra.Command{
	Use:           "explain [SQL]",
	Short:         "Run EXPLAIN on a query",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Run EXPLAIN using the active database's native explain mode.

Like the TUI explain view, this command reuses the backend explain path so the
database-specific explain prefix stays aligned with the current plugin.`,
	Example: `  # Explain a query with a saved connection
  whodb explain --connection mydb "SELECT * FROM users LIMIT 10"

  # Emit JSON
  whodb explain --connection mydb --format json "SELECT * FROM users"

  # Read SQL from stdin
  cat query.sql | whodb explain --connection mydb -`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("missing SQL query")
		}
		if len(args) > 1 {
			return fmt.Errorf("too many arguments (expected 1 SQL query, got %d)", len(args))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		sql, err := readSQLArgument(args[0])
		if err != nil {
			return err
		}

		format, err := output.ParseFormat(explainFormat)
		if err != nil {
			return err
		}

		quiet := explainQuiet || shouldSuppressInformationalOutput(cmd, format)
		out := newCommandOutput(cmd, format, quiet)

		mgr, err := dbmgr.NewManager()
		if err != nil {
			return fmt.Errorf("cannot initialize database manager: %w", err)
		}

		var conn *dbmgr.Connection
		if explainConnection != "" {
			conn, _, err = mgr.ResolveConnection(explainConnection)
			if err != nil {
				return err
			}
		} else {
			conns := mgr.ListAvailableConnections()
			if len(conns) == 0 {
				return fmt.Errorf("no connections available. Create one first:\n  whodb connect --type postgres --host localhost --user myuser --database mydb --name myconn")
			}
			conn = &conns[0]
			out.Info("Using connection: %s", conn.Name)
		}

		var spinner *output.Spinner
		if !quiet {
			spinner = output.NewSpinner(fmt.Sprintf("Connecting to %s...", conn.Type))
			spinner.Start()
		}
		if err := mgr.Connect(conn); err != nil {
			if spinner != nil {
				spinner.StopWithError("Connection failed")
			}
			return fmt.Errorf("cannot connect to database: %w", err)
		}
		if spinner != nil {
			spinner.Stop()
		}
		defer mgr.Disconnect()

		if !quiet {
			spinner = output.NewSpinner("Running EXPLAIN...")
			spinner.Start()
		}
		result, err := mgr.ExecuteExplain(sql)
		if err != nil {
			if spinner != nil {
				spinner.StopWithError("Explain failed")
			}
			return fmt.Errorf("explain failed: %w", err)
		}
		if spinner != nil {
			spinner.Stop()
		}

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

		return out.WriteQueryResult(&output.QueryResult{
			Columns: columns,
			Rows:    rows,
		})
	},
}

func readSQLArgument(value string) (string, error) {
	if value != "-" {
		return value, nil
	}

	scanner := bufio.NewScanner(os.Stdin)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading from stdin: %w", err)
	}

	sql := strings.Join(lines, "\n")
	if strings.TrimSpace(sql) == "" {
		return "", fmt.Errorf("no SQL provided via stdin")
	}

	return sql, nil
}

func init() {
	rootCmd.AddCommand(explainCmd)

	explainCmd.Flags().StringVarP(&explainConnection, "connection", "c", "", "connection name to use")
	explainCmd.Flags().StringVarP(&explainFormat, "format", "f", "auto", "output format: auto, table, plain, json, ndjson, or csv")
	explainCmd.Flags().BoolVarP(&explainQuiet, "quiet", "q", false, "suppress informational messages")

	explainCmd.RegisterFlagCompletionFunc("connection", completeConnectionNames)
	explainCmd.RegisterFlagCompletionFunc("format", completeOutputFormats)
}
