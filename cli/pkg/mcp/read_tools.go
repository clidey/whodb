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

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/internal/schemadiff"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ExplainInput is the input for the whodb_explain tool.
type ExplainInput struct {
	// Connection is the name of a saved connection or environment profile.
	Connection string `json:"connection" jsonschema:"Connection name (optional if only one exists)"`
	// Query is the SQL query to explain.
	Query string `json:"query" jsonschema:"SQL query to explain"`
}

// ExplainOutput is the output for the whodb_explain tool.
type ExplainOutput struct {
	Columns     []string `json:"columns"`
	ColumnTypes []string `json:"column_types,omitempty"`
	Rows        [][]any  `json:"rows"`
	Error       string   `json:"error,omitempty"`
	RequestID   string   `json:"request_id,omitempty"`
}

// MarshalJSON ensures nil slices are serialized as [] instead of null.
func (o ExplainOutput) MarshalJSON() ([]byte, error) {
	if o.Columns == nil {
		o.Columns = []string{}
	}
	if o.Rows == nil {
		o.Rows = [][]any{}
	}
	type Alias ExplainOutput
	return json.Marshal(Alias(o))
}

// SchemaDiffInput is the input for the whodb_diff tool.
type SchemaDiffInput struct {
	// FromConnection is the base connection to compare from.
	FromConnection string `json:"from_connection" jsonschema:"Source connection name"`
	// ToConnection is the target connection to compare against.
	ToConnection string `json:"to_connection" jsonschema:"Target connection name"`
	// FromSchema optionally overrides the source schema/database name.
	FromSchema string `json:"from_schema,omitempty" jsonschema:"Source schema override"`
	// ToSchema optionally overrides the target schema/database name.
	ToSchema string `json:"to_schema,omitempty" jsonschema:"Target schema override"`
}

// SchemaDiffOutput is the output for the whodb_diff tool.
type SchemaDiffOutput struct {
	Result    *schemadiff.Result `json:"result,omitempty"`
	Error     string             `json:"error,omitempty"`
	RequestID string             `json:"request_id,omitempty"`
}

// ERDInput is the input for the whodb_erd tool.
type ERDInput struct {
	// Connection is the name of a saved connection or environment profile.
	Connection string `json:"connection" jsonschema:"Connection name (optional if only one exists)"`
	// Schema optionally overrides the schema/database to inspect.
	Schema string `json:"schema,omitempty" jsonschema:"Schema or database name override"`
}

// ERDStorageUnit describes one storage unit in the whodb_erd response.
type ERDStorageUnit struct {
	Name    string      `json:"name"`
	Kind    string      `json:"kind,omitempty"`
	Columns []ERDColumn `json:"columns,omitempty"`
}

// ERDColumn describes one column in the whodb_erd response.
type ERDColumn struct {
	Name             string `json:"name"`
	Type             string `json:"type,omitempty"`
	IsPrimary        bool   `json:"is_primary,omitempty"`
	IsForeignKey     bool   `json:"is_foreign_key,omitempty"`
	ReferencedTable  string `json:"referenced_table,omitempty"`
	ReferencedColumn string `json:"referenced_column,omitempty"`
}

// ERDRelationship describes a normalized relationship edge in the whodb_erd response.
type ERDRelationship struct {
	SourceStorageUnit string `json:"source_storage_unit"`
	SourceColumn      string `json:"source_column,omitempty"`
	TargetStorageUnit string `json:"target_storage_unit"`
	TargetColumn      string `json:"target_column,omitempty"`
	RelationshipType  string `json:"relationship_type,omitempty"`
}

// ERDOutput is the output for the whodb_erd tool.
type ERDOutput struct {
	Schema        string            `json:"schema,omitempty"`
	StorageUnits  []ERDStorageUnit  `json:"storage_units"`
	Relationships []ERDRelationship `json:"relationships"`
	Error         string            `json:"error,omitempty"`
	RequestID     string            `json:"request_id,omitempty"`
}

// MarshalJSON ensures nil slices are serialized as [] instead of null.
func (o ERDOutput) MarshalJSON() ([]byte, error) {
	if o.StorageUnits == nil {
		o.StorageUnits = []ERDStorageUnit{}
	}
	if o.Relationships == nil {
		o.Relationships = []ERDRelationship{}
	}
	type Alias ERDOutput
	return json.Marshal(Alias(o))
}

// AuditInput is the input for the whodb_audit tool.
type AuditInput struct {
	// Connection is the name of a saved connection or environment profile.
	Connection string `json:"connection" jsonschema:"Connection name (optional if only one exists)"`
	// Schema optionally overrides the schema/database to inspect.
	Schema string `json:"schema,omitempty" jsonschema:"Schema or database name override"`
	// Table restricts the audit to a single table.
	Table string `json:"table,omitempty" jsonschema:"Optional table name"`
	// NullWarning sets the warning threshold for null rates.
	NullWarning float64 `json:"null_warning,omitempty" jsonschema:"Warning threshold for null percentage"`
	// NullError sets the error threshold for null rates.
	NullError float64 `json:"null_error,omitempty" jsonschema:"Error threshold for null percentage"`
}

// AuditSummary reports the aggregate audit counts returned by whodb_audit.
type AuditSummary struct {
	TablesScanned int `json:"tables_scanned"`
	IssuesFound   int `json:"issues_found"`
}

// AuditOutput is the output for the whodb_audit tool.
type AuditOutput struct {
	Summary   AuditSummary        `json:"summary"`
	Results   []*dbmgr.TableAudit `json:"results"`
	Error     string              `json:"error,omitempty"`
	RequestID string              `json:"request_id,omitempty"`
}

// MarshalJSON ensures nil slices are serialized as [] instead of null.
func (o AuditOutput) MarshalJSON() ([]byte, error) {
	if o.Results == nil {
		o.Results = []*dbmgr.TableAudit{}
	}
	type Alias AuditOutput
	return json.Marshal(Alias(o))
}

// SuggestionsInput is the input for the whodb_suggestions tool.
type SuggestionsInput struct {
	// Connection is the name of a saved connection or environment profile.
	Connection string `json:"connection" jsonschema:"Connection name (optional if only one exists)"`
	// Schema optionally overrides the schema/database to inspect.
	Schema string `json:"schema,omitempty" jsonschema:"Schema or database name override"`
}

// SuggestionsOutput is the output for the whodb_suggestions tool.
type SuggestionsOutput struct {
	Suggestions []dbmgr.QuerySuggestion `json:"suggestions"`
	Error       string                  `json:"error,omitempty"`
	RequestID   string                  `json:"request_id,omitempty"`
}

// MarshalJSON ensures nil slices are serialized as [] instead of null.
func (o SuggestionsOutput) MarshalJSON() ([]byte, error) {
	if o.Suggestions == nil {
		o.Suggestions = []dbmgr.QuerySuggestion{}
	}
	type Alias SuggestionsOutput
	return json.Marshal(Alias(o))
}

// HandleExplain runs EXPLAIN for a SQL query against the specified connection.
func HandleExplain(ctx context.Context, req *mcp.CallToolRequest, input ExplainInput) (*mcp.CallToolResult, ExplainOutput, error) {
	requestID := generateRequestID("explain")
	startTime := time.Now()

	resolver, err := newConnectionResolver(true)
	if err != nil {
		TrackToolCall(ctx, "explain", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "connection"})
		return nil, ExplainOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	if strings.TrimSpace(input.Query) == "" {
		TrackToolCall(ctx, "explain", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "validation"})
		return nil, ExplainOutput{Error: "query is required", RequestID: requestID}, nil
	}
	if resolver.Count() > 1 && strings.TrimSpace(input.Connection) == "" {
		TrackToolCall(ctx, "explain", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "validation"})
		return nil, ExplainOutput{Error: "connection is required when multiple connections are available", RequestID: requestID}, nil
	}

	mgr, conn, err := newConnectedManagerFromResolver(resolver, input.Connection)
	if err != nil {
		TrackToolCall(ctx, "explain", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "connection"})
		return nil, ExplainOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	defer mgr.Disconnect()

	result, err := mgr.ExecuteExplain(input.Query)
	if err != nil {
		TrackToolCall(ctx, "explain", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "execution", "db_type": conn.Type})
		return nil, ExplainOutput{Error: fmt.Sprintf("explain failed: %v", err), RequestID: requestID}, nil
	}

	output := ExplainOutput{
		Columns:     convertColumns(result),
		ColumnTypes: convertColumnTypes(result),
		Rows:        mustConvertRows(result),
		RequestID:   requestID,
	}
	TrackToolCall(ctx, "explain", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{"row_count": len(output.Rows), "db_type": conn.Type})
	return nil, output, nil
}

// HandleSchemaDiff compares schema metadata between two saved connections.
func HandleSchemaDiff(ctx context.Context, req *mcp.CallToolRequest, input SchemaDiffInput) (*mcp.CallToolResult, SchemaDiffOutput, error) {
	requestID := generateRequestID("diff")
	startTime := time.Now()

	if strings.TrimSpace(input.FromConnection) == "" || strings.TrimSpace(input.ToConnection) == "" {
		TrackToolCall(ctx, "diff", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "validation"})
		return nil, SchemaDiffOutput{Error: "from_connection and to_connection are required", RequestID: requestID}, nil
	}
	if input.FromConnection == input.ToConnection && strings.TrimSpace(input.FromSchema) == strings.TrimSpace(input.ToSchema) {
		TrackToolCall(ctx, "diff", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "validation"})
		return nil, SchemaDiffOutput{Error: "diff requires different connections or schema overrides", RequestID: requestID}, nil
	}

	resolver, err := newConnectionResolver(true)
	if err != nil {
		TrackToolCall(ctx, "diff", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "connection_resolve"})
		return nil, SchemaDiffOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	fromConn, err := resolver.ResolveOrDefault(input.FromConnection)
	if err != nil {
		TrackToolCall(ctx, "diff", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "connection_resolve"})
		return nil, SchemaDiffOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	toConn, err := resolver.ResolveOrDefault(input.ToConnection)
	if err != nil {
		TrackToolCall(ctx, "diff", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "connection_resolve"})
		return nil, SchemaDiffOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	result, err := schemadiff.CompareConnections(fromConn, toConn, strings.TrimSpace(input.FromSchema), strings.TrimSpace(input.ToSchema))
	if err != nil {
		TrackToolCall(ctx, "diff", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "compare"})
		return nil, SchemaDiffOutput{Error: fmt.Sprintf("schema diff failed: %v", err), RequestID: requestID}, nil
	}

	TrackToolCall(ctx, "diff", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{
		"changed_storage_units": result.Summary.ChangedStorageUnits,
		"changed_columns":       result.Summary.ChangedColumns,
		"changed_relationships": result.Summary.ChangedRelationships,
	})
	return nil, SchemaDiffOutput{Result: result, RequestID: requestID}, nil
}

// HandleERD loads backend graph metadata for a connection.
func HandleERD(ctx context.Context, req *mcp.CallToolRequest, input ERDInput) (*mcp.CallToolResult, ERDOutput, error) {
	requestID := generateRequestID("erd")
	startTime := time.Now()

	resolver, err := newConnectionResolver(true)
	if err != nil {
		TrackToolCall(ctx, "erd", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "connection"})
		return nil, ERDOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	if resolver.Count() > 1 && strings.TrimSpace(input.Connection) == "" {
		TrackToolCall(ctx, "erd", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "validation"})
		return nil, ERDOutput{Error: "connection is required when multiple connections are available", RequestID: requestID}, nil
	}

	mgr, conn, err := newConnectedManagerFromResolver(resolver, input.Connection)
	if err != nil {
		TrackToolCall(ctx, "erd", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "connection"})
		return nil, ERDOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	defer mgr.Disconnect()

	schemaName, err := mgr.ResolveSnapshotSchema(conn, input.Schema)
	if err != nil {
		TrackToolCall(ctx, "erd", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "schema", "db_type": conn.Type})
		return nil, ERDOutput{Error: fmt.Sprintf("resolve schema: %v", err), RequestID: requestID}, nil
	}

	graphUnits, err := mgr.GetGraph(schemaName)
	if err != nil {
		TrackToolCall(ctx, "erd", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "graph", "db_type": conn.Type})
		return nil, ERDOutput{Error: fmt.Sprintf("load graph: %v", err), RequestID: requestID}, nil
	}

	output, err := buildERDOutput(mgr, schemaName, graphUnits)
	if err != nil {
		TrackToolCall(ctx, "erd", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "columns", "db_type": conn.Type})
		return nil, ERDOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	output.RequestID = requestID

	TrackToolCall(ctx, "erd", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{
		"storage_units":   len(output.StorageUnits),
		"relationships":   len(output.Relationships),
		"resolved_schema": output.Schema,
		"db_type":         conn.Type,
	})
	return nil, *output, nil
}

// HandleAudit runs data-quality checks across one schema or one table.
func HandleAudit(ctx context.Context, req *mcp.CallToolRequest, input AuditInput) (*mcp.CallToolResult, AuditOutput, error) {
	requestID := generateRequestID("audit")
	startTime := time.Now()

	resolver, err := newConnectionResolver(true)
	if err != nil {
		TrackToolCall(ctx, "audit", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "connection"})
		return nil, AuditOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	if resolver.Count() > 1 && strings.TrimSpace(input.Connection) == "" {
		TrackToolCall(ctx, "audit", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "validation"})
		return nil, AuditOutput{Error: "connection is required when multiple connections are available", RequestID: requestID}, nil
	}

	mgr, conn, err := newConnectedManagerFromResolver(resolver, input.Connection)
	if err != nil {
		TrackToolCall(ctx, "audit", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "connection"})
		return nil, AuditOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	defer mgr.Disconnect()

	schemaName, err := mgr.ResolveSnapshotSchema(conn, input.Schema)
	if err != nil {
		TrackToolCall(ctx, "audit", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "schema", "db_type": conn.Type})
		return nil, AuditOutput{Error: fmt.Sprintf("resolve schema: %v", err), RequestID: requestID}, nil
	}

	config := dbmgr.DefaultAuditConfig()
	if input.NullWarning > 0 {
		config.NullWarningPct = input.NullWarning
	}
	if input.NullError > 0 {
		config.NullErrorPct = input.NullError
	}

	var results []*dbmgr.TableAudit
	if strings.TrimSpace(input.Table) != "" {
		result, err := mgr.AuditTable(schemaName, input.Table, config)
		if err != nil {
			TrackToolCall(ctx, "audit", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "audit", "db_type": conn.Type})
			return nil, AuditOutput{Error: fmt.Sprintf("audit failed: %v", err), RequestID: requestID}, nil
		}
		results = []*dbmgr.TableAudit{result}
	} else {
		results, err = mgr.AuditSchema(schemaName, config)
		if err != nil {
			TrackToolCall(ctx, "audit", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "audit", "db_type": conn.Type})
			return nil, AuditOutput{Error: fmt.Sprintf("audit failed: %v", err), RequestID: requestID}, nil
		}
	}

	output := AuditOutput{
		Summary:   buildAuditSummary(results),
		Results:   results,
		RequestID: requestID,
	}
	TrackToolCall(ctx, "audit", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{
		"tables_scanned": output.Summary.TablesScanned,
		"issues_found":   output.Summary.IssuesFound,
		"db_type":        conn.Type,
	})
	return nil, output, nil
}

// HandleSuggestions loads backend-generated query suggestions for a connection.
func HandleSuggestions(ctx context.Context, req *mcp.CallToolRequest, input SuggestionsInput) (*mcp.CallToolResult, SuggestionsOutput, error) {
	requestID := generateRequestID("suggestions")
	startTime := time.Now()

	resolver, err := newConnectionResolver(true)
	if err != nil {
		TrackToolCall(ctx, "suggestions", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "connection"})
		return nil, SuggestionsOutput{Error: err.Error(), RequestID: requestID}, nil
	}

	if resolver.Count() > 1 && strings.TrimSpace(input.Connection) == "" {
		TrackToolCall(ctx, "suggestions", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "validation"})
		return nil, SuggestionsOutput{Error: "connection is required when multiple connections are available", RequestID: requestID}, nil
	}

	mgr, conn, err := newConnectedManagerFromResolver(resolver, input.Connection)
	if err != nil {
		TrackToolCall(ctx, "suggestions", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "connection"})
		return nil, SuggestionsOutput{Error: err.Error(), RequestID: requestID}, nil
	}
	defer mgr.Disconnect()

	schemaName, err := mgr.ResolveSnapshotSchema(conn, input.Schema)
	if err != nil {
		TrackToolCall(ctx, "suggestions", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "schema", "db_type": conn.Type})
		return nil, SuggestionsOutput{Error: fmt.Sprintf("resolve schema: %v", err), RequestID: requestID}, nil
	}

	suggestions, err := mgr.GetQuerySuggestions(schemaName)
	if err != nil {
		TrackToolCall(ctx, "suggestions", requestID, false, time.Since(startTime).Milliseconds(), map[string]any{"error_type": "suggestions", "db_type": conn.Type})
		return nil, SuggestionsOutput{Error: fmt.Sprintf("load suggestions: %v", err), RequestID: requestID}, nil
	}

	TrackToolCall(ctx, "suggestions", requestID, true, time.Since(startTime).Milliseconds(), map[string]any{
		"suggestion_count": len(suggestions),
		"db_type":          conn.Type,
	})
	return nil, SuggestionsOutput{Suggestions: suggestions, RequestID: requestID}, nil
}

func newConnectedManagerFromResolver(resolver *connectionResolver, connection string) (*dbmgr.Manager, *dbmgr.Connection, error) {
	conn, err := resolver.ResolveOrDefault(connection)
	if err != nil {
		return nil, nil, err
	}

	mgr, err := connectManager(conn)
	if err != nil {
		return nil, nil, err
	}

	return mgr, conn, nil
}

func connectManager(conn *dbmgr.Connection) (*dbmgr.Manager, error) {
	mgr, err := dbmgr.NewManager()
	if err != nil {
		return nil, fmt.Errorf("cannot initialize database manager: %w", err)
	}

	if err := mgr.Connect(conn); err != nil {
		return nil, fmt.Errorf("cannot connect to database: %w", err)
	}

	return mgr, nil
}

func mustConvertRows(result *engine.GetRowsResult) [][]any {
	rows, _ := convertRows(result, 0)
	return rows
}

func buildAuditSummary(results []*dbmgr.TableAudit) AuditSummary {
	totalIssues := 0
	for _, result := range results {
		totalIssues += len(result.Issues)
	}

	return AuditSummary{
		TablesScanned: len(results),
		IssuesFound:   totalIssues,
	}
}

func buildERDOutput(mgr *dbmgr.Manager, schema string, graphUnits []engine.GraphUnit) (*ERDOutput, error) {
	relationships, relationshipTargets := buildERDRelationships(graphUnits)

	storageUnitNames := make([]string, 0, len(graphUnits))
	for _, graphUnit := range graphUnits {
		storageUnitNames = append(storageUnitNames, graphUnit.Unit.Name)
	}

	columnsByStorageUnit, err := mgr.GetColumnsForStorageUnits(schema, storageUnitNames)
	if err != nil {
		return nil, err
	}

	storageUnits := make([]ERDStorageUnit, 0, len(graphUnits))
	for _, graphUnit := range graphUnits {
		columns := columnsByStorageUnit[graphUnit.Unit.Name]
		columnOutputs := make([]ERDColumn, 0, len(columns))
		for _, column := range columns {
			columnOutput := ERDColumn{
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

		storageUnit := ERDStorageUnit{
			Name:    graphUnit.Unit.Name,
			Columns: columnOutputs,
		}
		for _, attribute := range graphUnit.Unit.Attributes {
			if strings.EqualFold(attribute.Key, "type") {
				storageUnit.Kind = attribute.Value
				break
			}
		}

		storageUnits = append(storageUnits, storageUnit)
	}

	sort.Slice(storageUnits, func(i, j int) bool {
		return storageUnits[i].Name < storageUnits[j].Name
	})

	sort.Slice(relationships, func(i, j int) bool {
		if relationships[i].SourceStorageUnit != relationships[j].SourceStorageUnit {
			return relationships[i].SourceStorageUnit < relationships[j].SourceStorageUnit
		}
		if relationships[i].TargetStorageUnit != relationships[j].TargetStorageUnit {
			return relationships[i].TargetStorageUnit < relationships[j].TargetStorageUnit
		}
		return relationships[i].SourceColumn < relationships[j].SourceColumn
	})

	return &ERDOutput{
		Schema:        schema,
		StorageUnits:  storageUnits,
		Relationships: relationships,
	}, nil
}

func buildERDRelationships(graphUnits []engine.GraphUnit) ([]ERDRelationship, map[string]map[string]ERDRelationship) {
	relationships := make([]ERDRelationship, 0)
	targets := make(map[string]map[string]ERDRelationship)

	for _, unit := range graphUnits {
		for _, relation := range unit.Relations {
			relationship := ERDRelationship{
				SourceStorageUnit: unit.Unit.Name,
				TargetStorageUnit: relation.Name,
				RelationshipType:  string(relation.RelationshipType),
			}
			if relation.SourceColumn != nil {
				relationship.SourceColumn = *relation.SourceColumn
			}
			if relation.TargetColumn != nil {
				relationship.TargetColumn = *relation.TargetColumn
			}

			relationships = append(relationships, relationship)

			if relationship.SourceColumn != "" {
				if _, ok := targets[unit.Unit.Name]; !ok {
					targets[unit.Unit.Name] = make(map[string]ERDRelationship)
				}
				targets[unit.Unit.Name][relationship.SourceColumn] = relationship
			}
		}
	}

	return relationships, targets
}
