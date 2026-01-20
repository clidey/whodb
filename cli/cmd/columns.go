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
  csv    - CSV format`,
	Example: `  # Describe columns in a table
  whodb-cli columns --connection mydb --table users

  # Specify schema explicitly
  whodb-cli columns --connection mydb --schema public --table users

  # Output as JSON for automation
  whodb-cli columns --connection mydb --table users --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if columnsTable == "" {
			return fmt.Errorf("--table flag is required")
		}

		format, err := output.ParseFormat(columnsFormat)
		if err != nil {
			return err
		}

		out := output.New(
			output.WithFormat(format),
			output.WithQuiet(columnsQuiet),
		)

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
		if !columnsQuiet {
			spinner = output.NewSpinner(fmt.Sprintf("Connecting to %s...", conn.Type))
		}
		spinner.Start()
		if err := mgr.Connect(conn); err != nil {
			spinner.StopWithError("Connection failed")
			return fmt.Errorf("cannot connect to database: %w", err)
		}
		spinner.StopWithSuccess("Connected")
		defer mgr.Disconnect()

		// Get schema
		schema := columnsSchema
		if schema == "" && conn.Schema != "" {
			schema = conn.Schema
		}
		if schema == "" {
			schemas, err := mgr.GetSchemas()
			if err != nil {
				return fmt.Errorf("failed to fetch schemas: %w", err)
			}
			if len(schemas) == 0 {
				return fmt.Errorf("no schemas found in database")
			}
			schema = schemas[0]
			out.Info("Using schema: %s", schema)
		}

		if !columnsQuiet {
			spinner = output.NewSpinner("Fetching columns...")
		}
		spinner.Start()
		columns, err := mgr.GetColumns(schema, columnsTable)
		if err != nil {
			spinner.StopWithError("Failed to fetch columns")
			return fmt.Errorf("failed to fetch columns: %w", err)
		}
		spinner.Stop()

		// Convert to QueryResult format
		outputColumns := []output.Column{
			{Name: "name", Type: "string"},
			{Name: "type", Type: "string"},
			{Name: "is_primary", Type: "bool"},
			{Name: "is_foreign_key", Type: "bool"},
			{Name: "referenced_table", Type: "string"},
			{Name: "referenced_column", Type: "string"},
		}

		rows := make([][]any, len(columns))
		for i, col := range columns {
			refTable := ""
			refColumn := ""
			if col.ReferencedTable != nil {
				refTable = *col.ReferencedTable
			}
			if col.ReferencedColumn != nil {
				refColumn = *col.ReferencedColumn
			}
			rows[i] = []any{
				col.Name,
				col.Type,
				col.IsPrimary,
				col.IsForeignKey,
				refTable,
				refColumn,
			}
		}

		result := &output.QueryResult{
			Columns: outputColumns,
			Rows:    rows,
		}

		return out.WriteQueryResult(result)
	},
}

func init() {
	rootCmd.AddCommand(columnsCmd)

	columnsCmd.Flags().StringVarP(&columnsConnection, "connection", "c", "", "connection name to use")
	columnsCmd.Flags().StringVarP(&columnsSchema, "schema", "s", "", "schema containing the table")
	columnsCmd.Flags().StringVarP(&columnsTable, "table", "t", "", "table to describe (required)")
	columnsCmd.Flags().StringVarP(&columnsFormat, "format", "f", "auto", "output format: auto, table, plain, json, csv")
	columnsCmd.Flags().BoolVarP(&columnsQuiet, "quiet", "q", false, "suppress informational messages")
}
