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
    whodb connect --type <db> --host <host> --user <user> --database <db> --name <name>

Output formats:
  auto   - Table for terminals, plain for pipes (default)
  table  - Human-readable table with borders
  plain  - One schema per line for scripting
  json   - JSON array of schema names
  ndjson - One JSON object per line
  csv    - CSV format`,
	Example: `  # List schemas for a connection
  whodb schemas --connection mydb

  # Output as JSON for scripting
  whodb schemas --connection mydb --format json

  # Quiet mode (no informational messages)
  whodb schemas --connection mydb --quiet`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		startTime := time.Now()

		format, err := output.ParseFormat(schemasFormat)
		if err != nil {
			return err
		}

		quiet := schemasQuiet || shouldSuppressInformationalOutput(cmd, format)
		out := newCommandOutput(cmd, format, quiet)

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
			spinner = output.NewSpinner("Fetching schemas...")
			spinner.Start()
		}
		schemas, err := mgr.GetSchemas()
		if err != nil {
			if spinner != nil {
				spinner.StopWithError("Failed to fetch schemas")
			}
			return fmt.Errorf("failed to fetch schemas: %w", err)
		}
		if spinner != nil {
			spinner.Stop()
		}

		analytics.TrackSchemasListed(ctx, conn.Type, len(schemas), time.Since(startTime).Milliseconds())

		// Convert schemas to StringQueryResult for consistent output without
		// materializing [][]any.
		columns := []output.Column{{Name: "schema", Type: "string"}}
		rows := make([][]string, len(schemas))
		for i, schema := range schemas {
			rows[i] = []string{schema}
		}

		result := &output.StringQueryResult{
			Columns: columns,
			Rows:    rows,
		}

		return out.WriteStringQueryResult(result)
	},
}

func init() {
	rootCmd.AddCommand(schemasCmd)

	schemasCmd.Flags().StringVarP(&schemasConnection, "connection", "c", "", "connection name to use")
	schemasCmd.Flags().StringVarP(&schemasFormat, "format", "f", "auto", "output format: auto, table, plain, json, ndjson, csv")
	schemasCmd.Flags().BoolVarP(&schemasQuiet, "quiet", "q", false, "suppress informational messages")

	schemasCmd.RegisterFlagCompletionFunc("connection", completeConnectionNames)
	schemasCmd.RegisterFlagCompletionFunc("format", completeOutputFormats)
}
