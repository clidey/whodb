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
	"path/filepath"
	"strings"
	"time"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/pkg/analytics"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	exportConnection string
	exportSchema     string
	exportTable      string
	exportQuery      string
	exportFormat     string
	exportOutput     string
	exportDelimiter  string
	exportQuiet      bool
)

var exportCmd = &cobra.Command{
	Use:           "export",
	Short:         "Export data to CSV or Excel",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Export table data or query results to CSV or Excel format.

You can export either:
  - An entire table using --table
  - Query results using --query

The output format is determined by:
  1. The --format flag (csv or excel)
  2. The file extension of --output (.csv or .xlsx)
  3. Default: csv`,
	Example: `  # Export a table to CSV
  whodb-cli export --connection mydb --table users --output users.csv

  # Export to Excel
  whodb-cli export --connection mydb --table users --output users.xlsx

  # Export query results
  whodb-cli export --connection mydb --query "SELECT * FROM users WHERE active = true" --output active_users.csv

  # Custom CSV delimiter
  whodb-cli export --connection mydb --table users --output users.csv --delimiter ";"

  # Specify schema
  whodb-cli export --connection mydb --schema public --table users --output users.csv`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		startTime := time.Now()

		if exportTable == "" && exportQuery == "" {
			return fmt.Errorf("either --table or --query is required")
		}
		if exportTable != "" && exportQuery != "" {
			return fmt.Errorf("cannot use both --table and --query")
		}
		if exportOutput == "" {
			return fmt.Errorf("--output is required")
		}

		out := output.New(output.WithQuiet(exportQuiet))

		mgr, err := dbmgr.NewManager()
		if err != nil {
			return fmt.Errorf("cannot initialize database manager: %w", err)
		}

		var conn *dbmgr.Connection
		if exportConnection != "" {
			conn, _, err = mgr.ResolveConnection(exportConnection)
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
		if !exportQuiet {
			spinner = output.NewSpinner(fmt.Sprintf("Connecting to %s...", conn.Type))
		}
		spinner.Start()
		if err := mgr.Connect(conn); err != nil {
			spinner.StopWithError("Connection failed")
			return fmt.Errorf("cannot connect to database: %w", err)
		}
		spinner.StopWithSuccess("Connected")
		defer mgr.Disconnect()

		// Determine export format
		format := exportFormat
		if format == "" {
			ext := strings.ToLower(filepath.Ext(exportOutput))
			switch ext {
			case ".xlsx", ".xls":
				format = "excel"
			default:
				format = "csv"
			}
		}

		if format != "csv" && format != "excel" {
			return fmt.Errorf("unsupported format %q (use csv or excel)", format)
		}

		// Get schema
		schema := exportSchema
		if schema == "" && conn.Schema != "" {
			schema = conn.Schema
		}
		if schema == "" && exportTable != "" {
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

		if !exportQuiet {
			if exportTable != "" {
				spinner = output.NewSpinner(fmt.Sprintf("Exporting table %s...", exportTable))
			} else {
				spinner = output.NewSpinner("Exporting query results...")
			}
		}
		spinner.Start()

		var exportErr error
		if exportTable != "" {
			// Export table
			if format == "excel" {
				exportErr = mgr.ExportToExcel(schema, exportTable, exportOutput)
			} else {
				delimiter := exportDelimiter
				if delimiter == "" {
					delimiter = ","
				}
				exportErr = mgr.ExportToCSV(schema, exportTable, exportOutput, delimiter)
			}
		} else {
			// Export query results
			result, err := mgr.ExecuteQuery(exportQuery)
			if err != nil {
				spinner.StopWithError("Query failed")
				return fmt.Errorf("query failed: %w", err)
			}

			if format == "excel" {
				exportErr = mgr.ExportResultsToExcel(result, exportOutput)
			} else {
				delimiter := exportDelimiter
				if delimiter == "" {
					delimiter = ","
				}
				exportErr = mgr.ExportResultsToCSV(result, exportOutput, delimiter)
			}
		}

		if exportErr != nil {
			spinner.StopWithError("Export failed")
			return fmt.Errorf("export failed: %w", exportErr)
		}

		// Track export (row count is not always available, use -1 when unknown)
		analytics.TrackExport(ctx, conn.Type, format, -1, time.Since(startTime).Milliseconds())

		spinner.StopWithSuccess("Export complete")
		out.Success("Exported to %s", exportOutput)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&exportConnection, "connection", "c", "", "connection name to use")
	exportCmd.Flags().StringVarP(&exportSchema, "schema", "s", "", "schema containing the table")
	exportCmd.Flags().StringVarP(&exportTable, "table", "t", "", "table to export")
	exportCmd.Flags().StringVarP(&exportQuery, "query", "Q", "", "SQL query to export results from")
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "", "output format: csv or excel (default: auto-detect from filename)")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file path (required)")
	exportCmd.Flags().StringVarP(&exportDelimiter, "delimiter", "d", ",", "CSV delimiter (default: comma)")
	exportCmd.Flags().BoolVarP(&exportQuiet, "quiet", "q", false, "suppress informational messages")
}
