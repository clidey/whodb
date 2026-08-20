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
	"github.com/clidey/whodb/cli/pkg/mcp"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/spf13/cobra"
)

var (
	queryConnection string
	queryFormat     string
	queryQuiet      bool
	queryStream     bool
)

var queryCmd = &cobra.Command{
	Use:           "query [SQL]",
	Short:         "Execute a SQL query",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Execute a SQL query against a connection.

Prerequisites:
  Use a saved connection:
    whodb connect --type <db> --host <host> --user <user> --database <db> --name <name>
  Or configure an environment profile, for example:
    WHODB_POSTGRES='[{"alias":"prod","host":"localhost","user":"user","password":"pass","database":"db","port":"5432"}]'
    WHODB_MYSQL_1='{"alias":"dev","host":"localhost","user":"user","password":"pass","database":"db","port":"3306"}'

Output formats:
  auto   - Table for terminals, plain for pipes (default)
  table  - Human-readable table with borders
  plain  - Tab-separated values for grep/awk
  json   - JSON array of objects
  ndjson - One JSON object per line
  csv    - RFC 4180 CSV format

Streaming:
  Use --stream for row-by-row query output without buffering the full result set.
  Supported streaming formats: plain, json, ndjson, csv.`,
	Example: `  # Query with a named connection
  whodb query --connection mydb "SELECT id, name FROM users LIMIT 5"

  # Pipe JSON to jq
  whodb query --format json "SELECT * FROM users" | jq '.[].name'

  # Export to CSV file
  whodb query --format csv "SELECT * FROM orders" > orders.csv

  # Use with grep (auto-selects plain format when piped)
  whodb query "SELECT * FROM logs" | grep ERROR

  # Stream NDJSON for large result sets
  whodb query --stream --format ndjson "SELECT * FROM audit_log"

  # Read SQL from stdin
  echo "SELECT * FROM users" | whodb query -
  cat query.sql | whodb query -`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("missing SQL query\n\nUsage: whodb query [SQL]\n\nExample:\n  whodb query \"SELECT * FROM users LIMIT 10\"\n\nRun 'whodb query --help' for more options")
		}
		if len(args) > 1 {
			return fmt.Errorf("too many arguments (expected 1 SQL query, got %d)\n\nTip: Wrap your SQL in quotes:\n  whodb query \"SELECT * FROM users\"", len(args))
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

		quiet := queryQuiet || shouldSuppressInformationalOutput(cmd, format)
		out := newCommandOutput(cmd, format, quiet)

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
			analytics.TrackConnectError(ctx, conn.Type, "connection_failed", time.Since(startTime).Milliseconds())
			return fmt.Errorf("cannot connect to database: %w", err)
		}
		if spinner != nil {
			spinner.Stop()
		}
		defer mgr.Disconnect()

		if !quiet {
			spinner = output.NewSpinner("Executing query...")
			spinner.Start()
		}
		queryStart := time.Now()

		if queryStream {
			streamWriter := &outputQueryStreamWriter{writer: out}
			rowCount, err := mgr.ExecuteQueryStream(ctx, sql, streamWriter)
			if err != nil {
				if spinner != nil {
					spinner.StopWithError("Query failed")
				}
				analytics.TrackQueryError(ctx, conn.Type, "execution_failed", time.Since(queryStart).Milliseconds())
				return fmt.Errorf("query failed: %w", err)
			}
			if spinner != nil {
				spinner.Stop()
			}
			if err := streamWriter.Close(); err != nil {
				return fmt.Errorf("finalizing streamed output: %w", err)
			}

			analytics.TrackQueryExecute(ctx, conn.Type, string(mcp.DetectStatementType(sql)), true,
				time.Since(queryStart).Milliseconds(), rowCount, map[string]any{
					"format":    string(format),
					"streaming": true,
				})
			return nil
		}

		result, err := mgr.ExecuteQuery(sql)
		if err != nil {
			if spinner != nil {
				spinner.StopWithError("Query failed")
			}
			analytics.TrackQueryError(ctx, conn.Type, "execution_failed", time.Since(queryStart).Milliseconds())
			return fmt.Errorf("query failed: %w", err)
		}
		if spinner != nil {
			spinner.Stop()
		}

		columns := make([]output.Column, len(result.Columns))
		for i, col := range result.Columns {
			columns[i] = output.Column{Name: col.Name, Type: col.Type}
		}

		// Track successful query execution
		analytics.TrackQueryExecute(ctx, conn.Type, string(mcp.DetectStatementType(sql)), true,
			time.Since(queryStart).Milliseconds(), len(result.Rows), map[string]any{
				"format":    string(format),
				"streaming": false,
			})

		queryResult := &output.StringQueryResult{
			Columns: columns,
			Rows:    result.Rows,
		}

		return out.WriteStringQueryResult(queryResult)
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)

	queryCmd.Flags().StringVarP(&queryConnection, "connection", "c", "", "connection name to use")
	queryCmd.Flags().StringVarP(&queryFormat, "format", "f", "auto", "output format: auto, table, plain, json, ndjson, csv")
	queryCmd.Flags().BoolVar(&queryStream, "stream", false, "stream query results row by row (plain/json/ndjson/csv only)")
	queryCmd.Flags().BoolVarP(&queryQuiet, "quiet", "q", false, "suppress informational messages")

	queryCmd.RegisterFlagCompletionFunc("connection", completeConnectionNames)
	queryCmd.RegisterFlagCompletionFunc("format", completeOutputFormats)
}

type outputQueryStreamWriter struct {
	writer *output.Writer
	stream output.QueryStream
}

func (w *outputQueryStreamWriter) WriteColumns(columns []engine.Column) error {
	if w.stream != nil {
		return nil
	}

	outputColumns := make([]output.Column, len(columns))
	for i, col := range columns {
		outputColumns[i] = output.Column{Name: col.Name, Type: col.Type}
	}

	stream, err := w.writer.BeginQueryStream(outputColumns)
	if err != nil {
		return err
	}
	w.stream = stream
	return nil
}

func (w *outputQueryStreamWriter) WriteRow(row []string) error {
	if w.stream == nil {
		return fmt.Errorf("stream output is not initialized")
	}
	return w.stream.WriteRow(row)
}

func (w *outputQueryStreamWriter) Close() error {
	if w.stream == nil {
		return nil
	}
	return w.stream.Close()
}
