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
	"strings"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/internal/schemadiff"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	diffFromConnection string
	diffToConnection   string
	diffSchema         string
	diffFromSchema     string
	diffToSchema       string
	diffFormat         string
	diffQuiet          bool
)

var diffCmd = &cobra.Command{
	Use:           "diff",
	Short:         "Compare schema metadata between two connections",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Compare schema metadata between two connections.

The diff command compares:
  - Added, removed, and changed storage units
  - Added, removed, and changed columns
  - Column properties such as type, nullability, uniqueness, defaults, and foreign keys
  - Relationship graph changes such as FK edge additions, removals, and type changes

By default, the CLI compares each connection's configured schema or database
scope. Use --schema to compare the same schema name on both sides, or use
--from-schema and --to-schema when the namespace names differ.`,
	Example: `  # Compare two saved connections using their default schemas
  whodb diff --from staging --to prod

  # Compare the same schema name on both connections
  whodb diff --from staging --to prod --schema public

  # Compare Postgres to MySQL by using each connection's configured namespace
  whodb diff --from dev-e2e_postgres-1 --to dev-e2e_mysql-1

  # Compare two different schema names
  whodb diff --from dev --to prod --from-schema app_dev --to-schema app_prod

  # Emit machine-readable JSON
  whodb diff --from staging --to prod --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(diffFromConnection) == "" {
			return fmt.Errorf("--from is required")
		}
		if strings.TrimSpace(diffToConnection) == "" {
			return fmt.Errorf("--to is required")
		}

		format, err := resolveSchemaDiffFormat(diffFormat)
		if err != nil {
			return err
		}

		quiet := diffQuiet || format == output.FormatJSON
		mgr, err := dbmgr.NewManager()
		if err != nil {
			return fmt.Errorf("cannot initialize database manager: %w", err)
		}

		fromConn, _, err := mgr.ResolveConnection(diffFromConnection)
		if err != nil {
			return err
		}
		toConn, _, err := mgr.ResolveConnection(diffToConnection)
		if err != nil {
			return err
		}

		fromSchemaName := diffFromSchema
		if fromSchemaName == "" {
			fromSchemaName = diffSchema
		}
		toSchemaName := diffToSchema
		if toSchemaName == "" {
			toSchemaName = diffSchema
		}

		var spinner *output.Spinner
		if !quiet {
			spinner = output.NewSpinner("Comparing schema metadata...")
			spinner.Start()
		}

		result, err := schemadiff.CompareConnections(fromConn, toConn, fromSchemaName, toSchemaName)
		if err != nil {
			if spinner != nil {
				spinner.StopWithError("Schema diff failed")
			}
			return err
		}
		if spinner != nil {
			spinner.Stop()
		}

		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "diff", result)
		}

		_, err = fmt.Fprint(cmd.OutOrStdout(), schemadiff.RenderText(result))
		return err
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)

	diffCmd.Flags().StringVar(&diffFromConnection, "from", "", "source connection name (required)")
	diffCmd.Flags().StringVar(&diffToConnection, "to", "", "target connection name (required)")
	diffCmd.Flags().StringVar(&diffSchema, "schema", "", "schema name to compare on both sides")
	diffCmd.Flags().StringVar(&diffFromSchema, "from-schema", "", "source schema name")
	diffCmd.Flags().StringVar(&diffToSchema, "to-schema", "", "target schema name")
	diffCmd.Flags().StringVarP(&diffFormat, "format", "f", "table", "output format: table or json")
	diffCmd.Flags().BoolVarP(&diffQuiet, "quiet", "q", false, "suppress informational messages")

	diffCmd.RegisterFlagCompletionFunc("from", completeConnectionNames)
	diffCmd.RegisterFlagCompletionFunc("to", completeConnectionNames)
	diffCmd.RegisterFlagCompletionFunc("format", completeAuditFormats)
}

func resolveSchemaDiffFormat(value string) (output.Format, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "table":
		return output.FormatTable, nil
	case "json":
		return output.FormatJSON, nil
	default:
		return "", fmt.Errorf("invalid --format %q (expected table or json)", value)
	}
}
