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
	"time"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/pkg/analytics"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	tablesConnection string
	tablesSchema     string
	tablesFormat     string
	tablesQuiet      bool
)

var tablesCmd = &cobra.Command{
	Use:           "tables",
	Short:         "List tables in a schema",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `List all tables (storage units) in a database schema.

Prerequisites:
  Create and save a connection first via:
    whodb-cli connect --type <db> --host <host> --user <user> --database <db> --name <name>

Output formats:
  auto   - Table for terminals, plain for pipes (default)
  table  - Human-readable table with borders
  plain  - Tab-separated values for scripting
  json   - JSON array of table objects
  csv    - CSV format`,
	Example: `  # List tables in the default/first schema
  whodb-cli tables --connection mydb

  # List tables in a specific schema
  whodb-cli tables --connection mydb --schema public

  # Output as JSON
  whodb-cli tables --connection mydb --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		startTime := time.Now()

		format, err := output.ParseFormat(tablesFormat)
		if err != nil {
			return err
		}

		out := output.New(
			output.WithFormat(format),
			output.WithQuiet(tablesQuiet),
		)

		mgr, err := dbmgr.NewManager()
		if err != nil {
			return fmt.Errorf("cannot initialize database manager: %w", err)
		}

		var conn *dbmgr.Connection
		if tablesConnection != "" {
			conn, _, err = mgr.ResolveConnection(tablesConnection)
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
		if !tablesQuiet {
			spinner = output.NewSpinner(fmt.Sprintf("Connecting to %s...", conn.Type))
		}
		spinner.Start()
		if err := mgr.Connect(conn); err != nil {
			spinner.StopWithError("Connection failed")
			return fmt.Errorf("cannot connect to database: %w", err)
		}
		spinner.StopWithSuccess("Connected")
		defer mgr.Disconnect()

		// Get schema - use provided, connection default, or first available
		schema := tablesSchema
		if schema == "" && conn.Schema != "" {
			schema = conn.Schema
		}
		if schema == "" {
			// Get first schema; schema-less databases (SQLite, Redis, etc.) return an error here
			schemas, err := mgr.GetSchemas()
			if err != nil {
				schemas = []string{}
			}
			if len(schemas) > 0 {
				schema = schemas[0]
				out.Info("Using schema: %s", schema)
			}
		}

		if !tablesQuiet {
			spinner = output.NewSpinner("Fetching tables...")
		}
		spinner.Start()
		tables, err := mgr.GetStorageUnits(schema)
		if err != nil {
			spinner.StopWithError("Failed to fetch tables")
			return fmt.Errorf("failed to fetch tables: %w", err)
		}
		spinner.Stop()

		analytics.TrackTablesListed(ctx, conn.Type, len(tables), time.Since(startTime).Milliseconds())

		// Collect all attribute keys from all tables
		attrKeys := make(map[string]bool)
		for _, t := range tables {
			for _, attr := range t.Attributes {
				attrKeys[attr.Key] = true
			}
		}

		// Sort attribute keys for consistent output
		attrNames := make([]string, 0, len(attrKeys))
		for key := range attrKeys {
			attrNames = append(attrNames, key)
		}

		// Convert to QueryResult format
		columns := []output.Column{
			{Name: "name", Type: "string"},
		}
		for _, name := range attrNames {
			columns = append(columns, output.Column{Name: name, Type: "string"})
		}

		rows := make([][]any, len(tables))
		for i, t := range tables {
			row := []any{t.Name}
			// Add attributes in consistent order
			attrMap := make(map[string]string)
			for _, attr := range t.Attributes {
				attrMap[attr.Key] = attr.Value
			}
			for _, name := range attrNames {
				row = append(row, attrMap[name])
			}
			rows[i] = row
		}

		result := &output.QueryResult{
			Columns: columns,
			Rows:    rows,
		}

		return out.WriteQueryResult(result)
	},
}

func init() {
	rootCmd.AddCommand(tablesCmd)

	tablesCmd.Flags().StringVarP(&tablesConnection, "connection", "c", "", "connection name to use")
	tablesCmd.Flags().StringVarP(&tablesSchema, "schema", "s", "", "schema to list tables from")
	tablesCmd.Flags().StringVarP(&tablesFormat, "format", "f", "auto", "output format: auto, table, plain, json, csv")
	tablesCmd.Flags().BoolVarP(&tablesQuiet, "quiet", "q", false, "suppress informational messages")
}
