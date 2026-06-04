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
	"context"
	"fmt"
	"strconv"
	"time"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/pkg/analytics"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	columnsConnection string
	columnsSchema     string
	columnsTable      string
	columnsFormat     string
	columnsQuiet      bool
)

var columnsCmd = &cobra.Command{
	Use:           "columns",
	Short:         "Describe table columns",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `List all columns in a database table with their types and attributes.

Prerequisites:
  Create and save a connection first via:
    whodb-cli connect --type <db> --host <host> --user <user> --database <db> --name <name>

Output formats:
  auto   - Table for terminals, plain for pipes (default)
  table  - Human-readable table with borders
  plain  - Tab-separated values for scripting
  json   - JSON array of column objects
  ndjson - One JSON object per line
  csv    - CSV format`,
	Example: `  # Describe columns in a table
  whodb-cli columns --connection mydb --table users

  # Specify schema explicitly
  whodb-cli columns --connection mydb --schema public --table users

  # Output as JSON for automation
  whodb-cli columns --connection mydb --table users --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		startTime := time.Now()

		if columnsTable == "" {
			return fmt.Errorf("--table flag is required")
		}

		format, err := output.ParseFormat(columnsFormat)
		if err != nil {
			return err
		}

		quiet := columnsQuiet || shouldSuppressInformationalOutput(cmd, format)
		out := newCommandOutput(cmd, format, quiet)

		mgr, err := dbmgr.NewManager()
		if err != nil {
			return fmt.Errorf("cannot initialize database manager: %w", err)
		}

		var conn *dbmgr.Connection
		if columnsConnection != "" {
			conn, _, err = mgr.ResolveConnection(columnsConnection)
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
		defer mgr.Disconnect() //nolint:errcheck

		// Get schema
		schema := columnsSchema
		if schema == "" && conn.Schema != "" {
			schema = conn.Schema
		}
		if schema == "" {
			// Schema-less databases (SQLite, Redis, etc.) don't support schemas
			schemas, err := mgr.GetSchemas()
			if err != nil {
				schemas = []string{}
			}
			if len(schemas) > 0 {
				schema = schemas[0]
				out.Info("Using schema: %s", schema)
			}
		}

		if !quiet {
			spinner = output.NewSpinner("Fetching columns...")
			spinner.Start()
		}
		columns, err := mgr.GetColumns(schema, columnsTable)
		if err != nil {
			if spinner != nil {
				spinner.StopWithError("Failed to fetch columns")
			}
			return fmt.Errorf("failed to fetch columns: %w", err)
		}
		if spinner != nil {
			spinner.Stop()
		}

		analytics.TrackColumnsListed(ctx, conn.Type, len(columns), time.Since(startTime).Milliseconds())

		// Convert to StringQueryResult to avoid materializing [][]any.
		outputColumns := []output.Column{
			{Name: "name", Type: "string"},
			{Name: "type", Type: "string"},
			{Name: "is_primary", Type: "bool"},
			{Name: "is_foreign_key", Type: "bool"},
			{Name: "referenced_table", Type: "string"},
			{Name: "referenced_column", Type: "string"},
		}

		rows := make([][]string, len(columns))
		for i, col := range columns {
			refTable := ""
			refColumn := ""
			if col.ReferencedTable != nil {
				refTable = *col.ReferencedTable
			}
			if col.ReferencedColumn != nil {
				refColumn = *col.ReferencedColumn
			}
			rows[i] = []string{
				col.Name,
				col.Type,
				strconv.FormatBool(col.IsPrimary),
				strconv.FormatBool(col.IsForeignKey),
				refTable,
				refColumn,
			}
		}

		result := &output.StringQueryResult{
			Columns: outputColumns,
			Rows:    rows,
		}

		return out.WriteStringQueryResult(result)
	},
}

func init() {
	rootCmd.AddCommand(columnsCmd)

	columnsCmd.Flags().StringVarP(&columnsConnection, "connection", "c", "", "connection name to use")
	columnsCmd.Flags().StringVarP(&columnsSchema, "schema", "s", "", "schema containing the table")
	columnsCmd.Flags().StringVarP(&columnsTable, "table", "t", "", "table to describe (required)")
	columnsCmd.Flags().StringVarP(&columnsFormat, "format", "f", "auto", "output format: auto, table, plain, json, ndjson, csv")
	columnsCmd.Flags().BoolVarP(&columnsQuiet, "quiet", "q", false, "suppress informational messages")

	columnsCmd.RegisterFlagCompletionFunc("connection", completeConnectionNames)
	columnsCmd.RegisterFlagCompletionFunc("format", completeOutputFormats)
}
