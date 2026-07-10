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
	"io"
	"strings"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	auditConnection  string
	auditSchema      string
	auditTable       string
	auditFormat      string
	auditNullWarning float64
	auditNullError   float64
	auditQuiet       bool
	auditType        string
	auditDatabase    string
	auditHost        string
	auditPort        int
	auditUser        string
)

type auditSummaryOutput struct {
	TablesScanned int `json:"tablesScanned"`
	IssuesFound   int `json:"issuesFound"`
}

type auditCommandOutput struct {
	Summary auditSummaryOutput  `json:"summary"`
	Results []*dbmgr.TableAudit `json:"results"`
}

var auditCmd = &cobra.Command{
	Use:           "audit",
	Short:         "Run data quality checks on tables",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Run data quality audit checks on one or more database tables.

Checks performed:
  - Null rate per column (configurable thresholds)
  - Distinct value count (low cardinality detection)
  - Duplicate detection on unique-looking columns
  - Orphaned foreign key references
  - Missing primary key detection
  - Type mismatch warnings (e.g. _id column with TEXT type)`,
	Example: `  # Audit a SQLite database directly
  whodb-cli audit --type sqlite3 --database ./app.db

  # Audit a Postgres database
  whodb-cli audit --type postgres --host localhost --user alice --database mydb

  # Audit using a saved connection
  whodb-cli audit --connection mydb

  # Audit a specific table
  whodb-cli audit --type sqlite3 --database ./app.db --table users

  # Output as JSON
  whodb-cli audit --connection mydb --format json

  # Custom null thresholds
  whodb-cli audit --connection mydb --null-warning 15 --null-error 60`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := resolveAuditFormat(auditFormat)
		if err != nil {
			return err
		}
		quiet := auditQuiet || format == output.FormatJSON
		out := newCommandOutput(cmd, format, quiet)

		mgr, err := dbmgr.NewManager()
		if err != nil {
			return fmt.Errorf("cannot initialize database manager: %w", err)
		}

		resolvedType, typeKnown := lookupDatabaseType(auditType)
		if auditType != "" && !typeKnown {
			return fmt.Errorf("unsupported database type %q", auditType)
		}
		if auditType != "" && typeKnown && isConnectionFieldRequired(resolvedType.ID, "Database") && auditDatabase == "" && auditConnection == "" {
			return fmt.Errorf("--database is required for %s", resolvedType.Label)
		}

		var conn *dbmgr.Connection
		if typeKnown && (auditDatabase != "" || !isConnectionFieldRequired(resolvedType.ID, "Database")) {
			// Inline connection from flags
			h := auditHost
			if h == "" {
				if isFileBasedDatabaseType(resolvedType.ID) {
					h = auditDatabase
				} else {
					h = "localhost"
				}
			}
			p := auditPort
			if p == 0 {
				p = getDefaultPort(resolvedType.ID)
			}
			conn = &dbmgr.Connection{
				Type:     resolvedType.ID,
				Host:     h,
				Port:     p,
				Username: auditUser,
				Database: auditDatabase,
			}
		} else if auditConnection != "" {
			conn, _, err = mgr.ResolveConnection(auditConnection)
			if err != nil {
				return err
			}
		} else {
			conns := mgr.ListAvailableConnections()
			if len(conns) == 0 {
				return fmt.Errorf("provide --type and --database, or --connection:\n  whodb-cli audit --type sqlite3 --database ./app.db\n  whodb-cli audit --connection mydb")
			}
			conn = &conns[0]
			out.Info("Using connection: %s", conn.Name)
		}

		var spinner *output.Spinner
		if !quiet {
			spinner = output.NewSpinner(fmt.Sprintf("Connecting to %s...", conn.Type))
		}
		if spinner != nil {
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

		// Resolve schema
		schema := auditSchema
		if schema == "" && conn.Schema != "" {
			schema = conn.Schema
		}
		if schema == "" {
			schemas, err := mgr.GetSchemas()
			if err != nil {
				schemas = []string{}
			}
			if len(schemas) > 0 {
				schema = schemas[0]
				out.Info("Using schema: %s", schema)
			}
		}

		config := dbmgr.DefaultAuditConfig()
		if auditNullWarning > 0 {
			config.NullWarningPct = auditNullWarning
		}
		if auditNullError > 0 {
			config.NullErrorPct = auditNullError
		}

		if !quiet {
			spinner = output.NewSpinner("Running audit...")
		}
		if spinner != nil {
			spinner.Start()
		}

		var results []*dbmgr.TableAudit
		if auditTable != "" {
			result, err := mgr.AuditTable(schema, auditTable, config)
			if err != nil {
				if spinner != nil {
					spinner.StopWithError("Audit failed")
				}
				return fmt.Errorf("audit failed: %w", err)
			}
			results = []*dbmgr.TableAudit{result}
		} else {
			results, err = mgr.AuditSchema(schema, config)
			if err != nil {
				if spinner != nil {
					spinner.StopWithError("Audit failed")
				}
				return fmt.Errorf("audit failed: %w", err)
			}
		}

		if spinner != nil {
			spinner.StopWithSuccess("Audit complete")
		}

		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "audit", buildAuditCommandOutput(results))
		}
		printAuditTable(cmd.OutOrStdout(), results)
		return nil
	},
}

func resolveAuditFormat(value string) (output.Format, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "table":
		return output.FormatTable, nil
	case "json":
		return output.FormatJSON, nil
	default:
		return "", fmt.Errorf("invalid --format %q (expected table or json)", value)
	}
}

func buildAuditCommandOutput(results []*dbmgr.TableAudit) auditCommandOutput {
	totalIssues := 0
	for _, t := range results {
		totalIssues += len(t.Issues)
	}

	return auditCommandOutput{
		Summary: auditSummaryOutput{
			TablesScanned: len(results),
			IssuesFound:   totalIssues,
		},
		Results: results,
	}
}

// printAuditTable outputs audit results as a human-readable table.
func printAuditTable(out io.Writer, results []*dbmgr.TableAudit) {
	totalIssues := 0
	for _, t := range results {
		totalIssues += len(t.Issues)
	}
	fmt.Fprintf(out, "\n%d tables scanned, %d issues found\n\n", len(results), totalIssues)

	for _, tbl := range results {
		icon := tableSeverityIcon(tbl)
		fmt.Fprintf(out, "%s %s (%d rows)\n", icon, tbl.TableName, tbl.RowCount)

		if !tbl.HasPrimaryKey {
			fmt.Fprintf(out, "  [!] No primary key\n")
		}

		// Column summary
		for _, col := range tbl.Columns {
			colIcon := columnSeverityIcon(col.Severity)
			pkLabel := ""
			if col.IsPrimary {
				pkLabel = " [PK]"
			}
			fmt.Fprintf(out, "  %s %-20s %-12s nulls:%.0f%% distinct:%d%s\n",
				colIcon, col.Name, col.Type, col.NullPct, col.DistinctCount, pkLabel)
			for _, issue := range col.Issues {
				fmt.Fprintf(out, "      %s\n", issue)
			}
		}

		// FK results
		for _, fk := range tbl.ForeignKeys {
			fkIcon := columnSeverityIcon(fk.Severity)
			fkLine := fmt.Sprintf("  %s FK %s.%s -> %s.%s",
				fkIcon, fk.SourceTable, fk.SourceColumn, fk.TargetTable, fk.TargetColumn)
			if fk.OrphanCount > 0 {
				fkLine += fmt.Sprintf(" (%d orphans)", fk.OrphanCount)
			}
			fmt.Fprintln(out, fkLine)
		}

		// Duplicates
		for _, dup := range tbl.Duplicates {
			fmt.Fprintf(out, "  [!] Duplicates in %s: %d groups, %d rows\n",
				strings.Join(dup.Columns, ", "), dup.DuplicateCount, dup.TotalDuplicateRows)
		}

		// Issues
		for _, issue := range tbl.Issues {
			issueIcon := columnSeverityIcon(issue.Severity)
			fmt.Fprintf(out, "  %s %s\n", issueIcon, issue.Message)
		}

		fmt.Fprintln(out)
	}
}

// tableSeverityIcon returns a text icon for the worst severity in a table audit.
func tableSeverityIcon(tbl *dbmgr.TableAudit) string {
	worst := dbmgr.SeverityOK
	for _, issue := range tbl.Issues {
		if issue.Severity == dbmgr.SeverityError {
			worst = dbmgr.SeverityError
			break
		}
		if issue.Severity == dbmgr.SeverityWarning {
			worst = dbmgr.SeverityWarning
		}
	}
	return columnSeverityIcon(worst)
}

// columnSeverityIcon returns a text icon for the given severity.
func columnSeverityIcon(severity dbmgr.AuditSeverity) string {
	switch severity {
	case dbmgr.SeverityOK:
		return "[ok]"
	case dbmgr.SeverityWarning:
		return "[!!]"
	case dbmgr.SeverityError:
		return "[XX]"
	default:
		return "[--]"
	}
}

func init() {
	rootCmd.AddCommand(auditCmd)

	auditCmd.Flags().StringVarP(&auditConnection, "connection", "c", "", "saved connection name")
	auditCmd.Flags().StringVar(&auditType, "type", "", "database type (postgres, mysql, sqlite3, etc.)")
	auditCmd.Flags().StringVar(&auditDatabase, "database", "", "database name or file path")
	auditCmd.Flags().StringVar(&auditHost, "host", "", "database host (default: localhost)")
	auditCmd.Flags().IntVar(&auditPort, "port", 0, "database port (default: auto)")
	auditCmd.Flags().StringVar(&auditUser, "user", "", "database username")
	auditCmd.Flags().StringVarP(&auditSchema, "schema", "s", "", "schema to audit")
	auditCmd.Flags().StringVarP(&auditTable, "table", "t", "", "specific table to audit (default: all tables)")
	auditCmd.Flags().StringVarP(&auditFormat, "format", "f", "", "output format: table or json (default: table)")
	auditCmd.Flags().Float64Var(&auditNullWarning, "null-warning", 0, "null percentage warning threshold (default: 10)")
	auditCmd.Flags().Float64Var(&auditNullError, "null-error", 0, "null percentage error threshold (default: 50)")
	auditCmd.Flags().BoolVarP(&auditQuiet, "quiet", "q", false, "suppress informational messages")

	auditCmd.RegisterFlagCompletionFunc("connection", completeConnectionNames)
	auditCmd.RegisterFlagCompletionFunc("type", completeDatabaseTypes)
	auditCmd.RegisterFlagCompletionFunc("format", completeAuditFormats)
}
