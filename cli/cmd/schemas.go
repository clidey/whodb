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
	schemasConnection string
	schemasFormat     string
	schemasQuiet      bool
)

var schemasCmd = &cobra.Command{
	Use:           "schemas",
	Short:         "List database schemas",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `List all schemas in a database.

Prerequisites:
  Create and save a connection first via:
    whodb-cli connect --type <db> --host <host> --user <user> --database <db> --name <name>

Output formats:
  auto   - Table for terminals, plain for pipes (default)
  table  - Human-readable table with borders
  plain  - One schema per line for scripting
  json   - JSON array of schema names
  csv    - CSV format`,
	Example: `  # List schemas for a connection
  whodb-cli schemas --connection mydb

  # Output as JSON for scripting
  whodb-cli schemas --connection mydb --format json

  # Quiet mode (no informational messages)
  whodb-cli schemas --connection mydb --quiet`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		startTime := time.Now()

		format, err := output.ParseFormat(schemasFormat)
		if err != nil {
			return err
		}

		out := output.New(
			output.WithFormat(format),
			output.WithQuiet(schemasQuiet),
		)

		mgr, err := dbmgr.NewManager()
		if err != nil {
			return fmt.Errorf("cannot initialize database manager: %w", err)
		}

		var conn *dbmgr.Connection
		if schemasConnection != "" {
			conn, _, err = mgr.ResolveConnection(schemasConnection)
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
		if !schemasQuiet {
			spinner = output.NewSpinner(fmt.Sprintf("Connecting to %s...", conn.Type))
		}
		spinner.Start()
		if err := mgr.Connect(conn); err != nil {
			spinner.StopWithError("Connection failed")
			return fmt.Errorf("cannot connect to database: %w", err)
		}
		spinner.StopWithSuccess("Connected")
		defer mgr.Disconnect()

		if !schemasQuiet {
			spinner = output.NewSpinner("Fetching schemas...")
		}
		spinner.Start()
		schemas, err := mgr.GetSchemas()
		if err != nil {
			spinner.StopWithError("Failed to fetch schemas")
			return fmt.Errorf("failed to fetch schemas: %w", err)
		}
		spinner.Stop()

		analytics.TrackSchemasListed(ctx, conn.Type, len(schemas), time.Since(startTime).Milliseconds())

		// Convert schemas to QueryResult format for consistent output
		columns := []output.Column{{Name: "schema", Type: "string"}}
		rows := make([][]any, len(schemas))
		for i, schema := range schemas {
			rows[i] = []any{schema}
		}

		result := &output.QueryResult{
			Columns: columns,
			Rows:    rows,
		}

		return out.WriteQueryResult(result)
	},
}

func init() {
	rootCmd.AddCommand(schemasCmd)

	schemasCmd.Flags().StringVarP(&schemasConnection, "connection", "c", "", "connection name to use")
	schemasCmd.Flags().StringVarP(&schemasFormat, "format", "f", "auto", "output format: auto, table, plain, json, csv")
	schemasCmd.Flags().BoolVarP(&schemasQuiet, "quiet", "q", false, "suppress informational messages")
}
