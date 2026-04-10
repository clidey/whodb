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
	"path/filepath"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	importConnection  string
	importFile        string
	importTable       string
	importSchema      string
	importDelimiter   string
	importHeader      bool
	importCreateTable bool
	importQuiet       bool
)

var importCmd = &cobra.Command{
	Use:           "import",
	Short:         "Import data from CSV or Excel",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Import data from a CSV or Excel file into a database table.

The file format is auto-detected from the extension (.csv, .xlsx).
CSV delimiter is auto-detected unless specified with --delimiter.

Examples:
  # Import CSV with headers into existing table
  whodb-cli import --connection mydb --file data.csv --table users

  # Import CSV and create the table automatically
  whodb-cli import --connection mydb --file data.csv --table users --create-table

  # Import Excel file
  whodb-cli import --connection mydb --file data.xlsx --table orders

  # Import CSV without headers
  whodb-cli import --connection mydb --file data.csv --table raw_data --no-header

  # Specify tab delimiter for TSV files
  whodb-cli import --connection mydb --file data.txt --table logs --delimiter "\t"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		w := output.New(output.WithQuiet(importQuiet))

		if importFile == "" {
			return fmt.Errorf("--file is required")
		}
		if importTable == "" {
			return fmt.Errorf("--table is required")
		}

		// Resolve connection
		mgr, err := dbmgr.NewManager()
		if err != nil {
			return fmt.Errorf("init: %w", err)
		}

		conn, _, err := mgr.ResolveConnection(importConnection)
		if err != nil {
			return err
		}

		if err := mgr.Connect(conn); err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer mgr.Disconnect()

		// Parse delimiter
		var delimiter rune
		switch importDelimiter {
		case "":
			delimiter = 0 // auto-detect
		case "\\t", "tab":
			delimiter = '\t'
		default:
			if len(importDelimiter) == 1 {
				delimiter = rune(importDelimiter[0])
			} else {
				return fmt.Errorf("delimiter must be a single character")
			}
		}

		absPath, err := filepath.Abs(importFile)
		if err != nil {
			absPath = importFile
		}

		opts := dbmgr.ImportOptions{
			HasHeader:   importHeader,
			Delimiter:   delimiter,
			CreateTable: importCreateTable,
			BatchSize:   500,
		}

		w.Info("Reading %s...", filepath.Base(absPath))

		headers, rows, err := dbmgr.ReadFileForImport(absPath, opts)
		if err != nil {
			return err
		}

		w.Info("Read %d rows, %d columns", len(rows), len(headers))

		if len(rows) == 0 {
			w.Info("File has no data rows to import")
			return nil
		}

		w.Info("Importing %d rows into %s...", len(rows), importTable)

		result, err := mgr.ImportData(importSchema, importTable, headers, rows, opts)
		if err != nil {
			return err
		}

		w.Success("Imported %d rows into %s", result.RowsImported, importTable)

		if result.TableCreated {
			w.Info("Table %s created", importTable)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&importConnection, "connection", "c", "", "Connection name")
	importCmd.Flags().StringVarP(&importFile, "file", "f", "", "Path to CSV or Excel file (required)")
	importCmd.Flags().StringVarP(&importTable, "table", "t", "", "Target table name (required)")
	importCmd.Flags().StringVarP(&importSchema, "schema", "s", "", "Target schema (optional)")
	importCmd.Flags().StringVarP(&importDelimiter, "delimiter", "d", "", "CSV delimiter (auto-detected if omitted)")
	importCmd.Flags().BoolVar(&importHeader, "header", true, "First row contains column headers")
	importCmd.Flags().BoolVar(&importCreateTable, "create-table", false, "Create the table if it doesn't exist")
	importCmd.Flags().BoolVarP(&importQuiet, "quiet", "q", false, "Suppress informational messages")
}
