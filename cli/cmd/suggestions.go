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
	"fmt"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	suggestionsConnection string
	suggestionsSchema     string
	suggestionsFormat     string
	suggestionsQuiet      bool
)

var suggestionsCmd = &cobra.Command{
	Use:           "suggestions",
	Short:         "Show backend-generated query suggestions",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Show backend-generated query suggestions for a connection.

Suggestions are built from the actual storage units in the resolved schema or
database, so they stay aligned with the same backend logic used by the app.`,
	Example: `  # Show suggestions for the configured/default schema
  whodb suggestions --connection mydb

  # Show suggestions for a specific schema
  whodb suggestions --connection mydb --schema public

  # Emit machine-readable output
  whodb suggestions --connection mydb --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := output.ParseFormat(suggestionsFormat)
		if err != nil {
			return err
		}

		quiet := suggestionsQuiet || shouldSuppressInformationalOutput(cmd, format)
		out := newCommandOutput(cmd, format, quiet)

		mgr, err := dbmgr.NewManager()
		if err != nil {
			return fmt.Errorf("cannot initialize database manager: %w", err)
		}

		var conn *dbmgr.Connection
		if suggestionsConnection != "" {
			conn, _, err = mgr.ResolveConnection(suggestionsConnection)
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

		if err := mgr.Connect(conn); err != nil {
			return fmt.Errorf("cannot connect to database: %w", err)
		}
		defer mgr.Disconnect()

		schemaName, err := mgr.ResolveSnapshotSchema(conn, suggestionsSchema)
		if err != nil {
			return fmt.Errorf("resolve schema: %w", err)
		}

		suggestions, err := mgr.GetQuerySuggestions(schemaName)
		if err != nil {
			return fmt.Errorf("load suggestions: %w", err)
		}

		if effectiveCommandOutputFormat(cmd, format) == output.FormatJSON {
			return writeCommandJSON(cmd, suggestions)
		}

		columns := []output.Column{
			{Name: "category", Type: "string"},
			{Name: "description", Type: "string"},
		}
		rows := make([][]any, len(suggestions))
		for i, suggestion := range suggestions {
			rows[i] = []any{suggestion.Category, suggestion.Description}
		}

		return out.WriteQueryResult(&output.QueryResult{
			Columns: columns,
			Rows:    rows,
		})
	},
}

func init() {
	rootCmd.AddCommand(suggestionsCmd)

	suggestionsCmd.Flags().StringVarP(&suggestionsConnection, "connection", "c", "", "connection name to use")
	suggestionsCmd.Flags().StringVarP(&suggestionsSchema, "schema", "s", "", "schema to use when generating suggestions")
	suggestionsCmd.Flags().StringVarP(&suggestionsFormat, "format", "f", "table", "output format: table, plain, json, ndjson, or csv")
	suggestionsCmd.Flags().BoolVarP(&suggestionsQuiet, "quiet", "q", false, "suppress informational messages")

	suggestionsCmd.RegisterFlagCompletionFunc("connection", completeConnectionNames)
	suggestionsCmd.RegisterFlagCompletionFunc("format", completeOutputFormats)
}
