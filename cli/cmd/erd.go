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
	"sort"
	"strings"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/spf13/cobra"
)

var (
	erdConnection string
	erdSchema     string
	erdFormat     string
	erdQuiet      bool
)

type erdCommandOutput struct {
	Schema        string                  `json:"schema,omitempty"`
	StorageUnits  []erdStorageUnitOutput  `json:"storageUnits"`
	Relationships []erdRelationshipOutput `json:"relationships"`
}

type erdStorageUnitOutput struct {
	Name    string            `json:"name"`
	Kind    string            `json:"kind,omitempty"`
	Columns []erdColumnOutput `json:"columns,omitempty"`
}

type erdColumnOutput struct {
	Name             string `json:"name"`
	Type             string `json:"type,omitempty"`
	IsPrimary        bool   `json:"isPrimary,omitempty"`
	IsForeignKey     bool   `json:"isForeignKey,omitempty"`
	ReferencedTable  string `json:"referencedTable,omitempty"`
	ReferencedColumn string `json:"referencedColumn,omitempty"`
}

type erdRelationshipOutput struct {
	SourceStorageUnit string `json:"sourceStorageUnit"`
	SourceColumn      string `json:"sourceColumn,omitempty"`
	TargetStorageUnit string `json:"targetStorageUnit"`
	TargetColumn      string `json:"targetColumn,omitempty"`
	RelationshipType  string `json:"relationshipType,omitempty"`
}

var erdCmd = &cobra.Command{
	Use:           "erd",
	Short:         "Render schema relationships as text",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Render the same backend graph metadata used by the TUI ER diagram view.

The text output is optimized for terminal inspection, while JSON exposes the
storage units, columns, and normalized relationship edges for automation.`,
	Example: `  # Show a textual ERD summary
  whodb erd --connection mydb

  # Compare a specific schema
  whodb erd --connection mydb --schema public

  # Emit machine-readable JSON
  whodb erd --connection mydb --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := resolveERDFormat(erdFormat)
		if err != nil {
			return err
		}

		quiet := erdQuiet || format == output.FormatJSON
		out := newCommandOutput(cmd, format, quiet)

		mgr, err := dbmgr.NewManager()
		if err != nil {
			return fmt.Errorf("cannot initialize database manager: %w", err)
		}

		var conn *dbmgr.Connection
		if erdConnection != "" {
			conn, _, err = mgr.ResolveConnection(erdConnection)
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

		schemaName, err := mgr.ResolveSnapshotSchema(conn, erdSchema)
		if err != nil {
			return fmt.Errorf("resolve schema: %w", err)
		}

		if !quiet {
			spinner = output.NewSpinner("Loading graph metadata...")
			spinner.Start()
		}

		graphUnits, err := mgr.GetGraph(schemaName)
		if err != nil {
			if spinner != nil {
				spinner.StopWithError("Graph load failed")
			}
			return fmt.Errorf("load graph: %w", err)
		}

		result, err := buildERDCommandOutput(mgr, schemaName, graphUnits)
		if err != nil {
			if spinner != nil {
				spinner.StopWithError("ERD render failed")
			}
			return err
		}
		if spinner != nil {
			spinner.Stop()
		}

		if format == output.FormatJSON {
			return writeAutomationEnvelope(cmd, "erd", result)
		}

		_, err = fmt.Fprint(cmd.OutOrStdout(), renderERDCommandText(result))
		return err
	},
}

func resolveERDFormat(value string) (output.Format, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "table", "text":
		return output.FormatTable, nil
	case "json":
		return output.FormatJSON, nil
	default:
		return "", fmt.Errorf("invalid --format %q (expected text or json)", value)
	}
}

func buildERDCommandOutput(mgr *dbmgr.Manager, schema string, graphUnits []engine.GraphUnit) (*erdCommandOutput, error) {
	relationships, relationshipTargets := buildERDRelationships(graphUnits)

	storageUnitNames := make([]string, 0, len(graphUnits))
	for _, graphUnit := range graphUnits {
		storageUnitNames = append(storageUnitNames, graphUnit.Unit.Name)
	}

	columnsByStorageUnit, err := mgr.GetColumnsForStorageUnits(schema, storageUnitNames)
	if err != nil {
		return nil, err
	}

	storageUnits := make([]erdStorageUnitOutput, 0, len(graphUnits))
	for _, graphUnit := range graphUnits {
		columns := columnsByStorageUnit[graphUnit.Unit.Name]
		columnOutputs := make([]erdColumnOutput, 0, len(columns))
		for _, column := range columns {
			columnOutput := erdColumnOutput{
				Name:      column.Name,
				Type:      column.Type,
				IsPrimary: column.IsPrimary,
			}

			if target, ok := relationshipTargets[graphUnit.Unit.Name][column.Name]; ok {
				columnOutput.IsForeignKey = true
				columnOutput.ReferencedTable = target.TargetStorageUnit
				columnOutput.ReferencedColumn = target.TargetColumn
			} else if column.IsForeignKey {
				columnOutput.IsForeignKey = true
				if column.ReferencedTable != nil {
					columnOutput.ReferencedTable = *column.ReferencedTable
				}
				if column.ReferencedColumn != nil {
					columnOutput.ReferencedColumn = *column.ReferencedColumn
				}
			}

			columnOutputs = append(columnOutputs, columnOutput)
		}

		sort.Slice(columnOutputs, func(i, j int) bool {
			return columnOutputs[i].Name < columnOutputs[j].Name
		})

		storageUnits = append(storageUnits, erdStorageUnitOutput{
			Name:    graphUnit.Unit.Name,
			Columns: columnOutputs,
		})
	}

	sort.Slice(storageUnits, func(i, j int) bool {
		return storageUnits[i].Name < storageUnits[j].Name
	})
	sort.Slice(relationships, func(i, j int) bool {
		left := relationships[i].SourceStorageUnit + "." + relationships[i].SourceColumn + "->" + relationships[i].TargetStorageUnit
		right := relationships[j].SourceStorageUnit + "." + relationships[j].SourceColumn + "->" + relationships[j].TargetStorageUnit
		return left < right
	})

	return &erdCommandOutput{
		Schema:        schema,
		StorageUnits:  storageUnits,
		Relationships: relationships,
	}, nil
}

func buildERDRelationships(graphUnits []engine.GraphUnit) ([]erdRelationshipOutput, map[string]map[string]erdRelationshipOutput) {
	relationships := make([]erdRelationshipOutput, 0)
	targets := make(map[string]map[string]erdRelationshipOutput)

	for _, graphUnit := range graphUnits {
		for _, relation := range graphUnit.Relations {
			sourceStorageUnit := graphUnit.Unit.Name
			targetStorageUnit := relation.Name
			sourceColumn := derefERDValue(relation.SourceColumn)
			targetColumn := derefERDValue(relation.TargetColumn)

			if relation.RelationshipType == "OneToMany" {
				sourceStorageUnit = relation.Name
				targetStorageUnit = graphUnit.Unit.Name
			}

			relationship := erdRelationshipOutput{
				SourceStorageUnit: sourceStorageUnit,
				SourceColumn:      sourceColumn,
				TargetStorageUnit: targetStorageUnit,
				TargetColumn:      targetColumn,
				RelationshipType:  string(relation.RelationshipType),
			}
			relationships = append(relationships, relationship)

			if sourceColumn == "" {
				continue
			}
			if _, ok := targets[sourceStorageUnit]; !ok {
				targets[sourceStorageUnit] = make(map[string]erdRelationshipOutput)
			}
			targets[sourceStorageUnit][sourceColumn] = relationship
		}
	}

	return relationships, targets
}

func derefERDValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func renderERDCommandText(result *erdCommandOutput) string {
	if result == nil {
		return ""
	}

	var b strings.Builder

	b.WriteString("Schema Graph\n")
	if result.Schema != "" {
		fmt.Fprintf(&b, "  Schema: %s\n", result.Schema)
	}
	fmt.Fprintf(&b, "  Storage units: %d\n", len(result.StorageUnits))
	fmt.Fprintf(&b, "  Relationships: %d\n\n", len(result.Relationships))

	for _, storageUnit := range result.StorageUnits {
		header := storageUnit.Name
		if storageUnit.Kind != "" {
			header += " (" + storageUnit.Kind + ")"
		}
		b.WriteString(header)
		b.WriteString("\n")
		for _, column := range storageUnit.Columns {
			var traits []string
			if column.IsPrimary {
				traits = append(traits, "pk")
			}
			if column.IsForeignKey && column.ReferencedTable != "" {
				traits = append(traits, fmt.Sprintf("fk -> %s.%s", column.ReferencedTable, column.ReferencedColumn))
			}

			line := fmt.Sprintf("  - %s", column.Name)
			if column.Type != "" {
				line += " " + column.Type
			}
			if len(traits) > 0 {
				line += " [" + strings.Join(traits, ", ") + "]"
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if len(result.Relationships) > 0 {
		b.WriteString("Relationships\n")
		for _, relationship := range result.Relationships {
			line := fmt.Sprintf("  - %s", relationship.SourceStorageUnit)
			if relationship.SourceColumn != "" {
				line += "." + relationship.SourceColumn
			}
			line += " -> " + relationship.TargetStorageUnit
			if relationship.TargetColumn != "" {
				line += "." + relationship.TargetColumn
			}
			if relationship.RelationshipType != "" {
				line += " [" + relationship.RelationshipType + "]"
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	return b.String()
}

func init() {
	rootCmd.AddCommand(erdCmd)

	erdCmd.Flags().StringVarP(&erdConnection, "connection", "c", "", "connection name to use")
	erdCmd.Flags().StringVarP(&erdSchema, "schema", "s", "", "schema to render")
	erdCmd.Flags().StringVarP(&erdFormat, "format", "f", "text", "output format: text or json")
	erdCmd.Flags().BoolVarP(&erdQuiet, "quiet", "q", false, "suppress informational messages")

	erdCmd.RegisterFlagCompletionFunc("connection", completeConnectionNames)
	erdCmd.RegisterFlagCompletionFunc("format", completeAuditFormats)
}
