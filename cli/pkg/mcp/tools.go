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

package mcp

import (
	"context"
	"fmt"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool input types - JSON tags define the schema

// QueryInput is the input for the whodb_query tool.
type QueryInput struct {
	// Connection is the name of a saved connection or environment variable
	// (e.g., 'prod' resolves to WHODB_PROD_URI)
	Connection string `json:"connection"`
	// Query is the SQL query to execute
	Query string `json:"query"`
}

// QueryOutput is the output for the whodb_query tool.
type QueryOutput struct {
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
	Error   string   `json:"error,omitempty"`
}

// SchemasInput is the input for the whodb_schemas tool.
type SchemasInput struct {
	// Connection is the name of a saved connection or environment variable
	Connection string `json:"connection"`
}

// SchemasOutput is the output for the whodb_schemas tool.
type SchemasOutput struct {
	Schemas []string `json:"schemas"`
	Error   string   `json:"error,omitempty"`
}

// TablesInput is the input for the whodb_tables tool.
type TablesInput struct {
	// Connection is the name of a saved connection or environment variable
	Connection string `json:"connection"`
	// Schema to list tables from (uses default if not specified)
	Schema string `json:"schema,omitempty"`
}

// TableInfo represents information about a database table.
type TableInfo struct {
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// TablesOutput is the output for the whodb_tables tool.
type TablesOutput struct {
	Tables []TableInfo `json:"tables"`
	Schema string      `json:"schema"`
	Error  string      `json:"error,omitempty"`
}

// ColumnsInput is the input for the whodb_columns tool.
type ColumnsInput struct {
	// Connection is the name of a saved connection or environment variable
	Connection string `json:"connection"`
	// Schema containing the table
	Schema string `json:"schema,omitempty"`
	// Table name to describe
	Table string `json:"table"`
}

// ColumnInfo represents information about a database column.
type ColumnInfo struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	IsPrimary        bool   `json:"is_primary"`
	IsForeignKey     bool   `json:"is_foreign_key"`
	ReferencedTable  string `json:"referenced_table,omitempty"`
	ReferencedColumn string `json:"referenced_column,omitempty"`
}

// ColumnsOutput is the output for the whodb_columns tool.
type ColumnsOutput struct {
	Columns []ColumnInfo `json:"columns"`
	Table   string       `json:"table"`
	Schema  string       `json:"schema"`
	Error   string       `json:"error,omitempty"`
}

// ConnectionsInput is the input for the whodb_connections tool.
type ConnectionsInput struct{}

// ConnectionInfo represents a saved connection (without sensitive data).
type ConnectionInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Database string `json:"database,omitempty"`
	Schema   string `json:"schema,omitempty"`
	Source   string `json:"source"` // "env" or "saved"
}

// ConnectionsOutput is the output for the whodb_connections tool.
type ConnectionsOutput struct {
	Connections []ConnectionInfo `json:"connections"`
	Error       string           `json:"error,omitempty"`
}

// Tool handlers

// HandleQuery executes a SQL query against the specified connection.
func HandleQuery(ctx context.Context, req *mcp.CallToolRequest, input QueryInput) (*mcp.CallToolResult, QueryOutput, error) {
	conn, err := ResolveConnectionOrDefault(input.Connection)
	if err != nil {
		return nil, QueryOutput{Error: err.Error()}, nil
	}

	mgr, err := dbmgr.NewManager()
	if err != nil {
		return nil, QueryOutput{Error: fmt.Sprintf("cannot initialize database manager: %v", err)}, nil
	}

	if err := mgr.Connect(conn); err != nil {
		return nil, QueryOutput{Error: fmt.Sprintf("cannot connect to database: %v", err)}, nil
	}
	defer mgr.Disconnect()

	result, err := mgr.ExecuteQuery(input.Query)
	if err != nil {
		return nil, QueryOutput{Error: fmt.Sprintf("query failed: %v", err)}, nil
	}

	// Convert columns
	columns := make([]string, len(result.Columns))
	for i, col := range result.Columns {
		columns[i] = col.Name
	}

	// Convert rows from [][]string to [][]any
	rows := make([][]any, len(result.Rows))
	for i, row := range result.Rows {
		rows[i] = make([]any, len(row))
		for j, val := range row {
			rows[i][j] = val
		}
	}

	return nil, QueryOutput{Columns: columns, Rows: rows}, nil
}

// HandleSchemas lists all schemas in the database.
func HandleSchemas(ctx context.Context, req *mcp.CallToolRequest, input SchemasInput) (*mcp.CallToolResult, SchemasOutput, error) {
	conn, err := ResolveConnectionOrDefault(input.Connection)
	if err != nil {
		return nil, SchemasOutput{Error: err.Error()}, nil
	}

	mgr, err := dbmgr.NewManager()
	if err != nil {
		return nil, SchemasOutput{Error: fmt.Sprintf("cannot initialize database manager: %v", err)}, nil
	}

	if err := mgr.Connect(conn); err != nil {
		return nil, SchemasOutput{Error: fmt.Sprintf("cannot connect to database: %v", err)}, nil
	}
	defer mgr.Disconnect()

	schemas, err := mgr.GetSchemas()
	if err != nil {
		return nil, SchemasOutput{Error: fmt.Sprintf("failed to fetch schemas: %v", err)}, nil
	}

	return nil, SchemasOutput{Schemas: schemas}, nil
}

// HandleTables lists all tables in a schema.
func HandleTables(ctx context.Context, req *mcp.CallToolRequest, input TablesInput) (*mcp.CallToolResult, TablesOutput, error) {
	conn, err := ResolveConnectionOrDefault(input.Connection)
	if err != nil {
		return nil, TablesOutput{Error: err.Error()}, nil
	}

	mgr, err := dbmgr.NewManager()
	if err != nil {
		return nil, TablesOutput{Error: fmt.Sprintf("cannot initialize database manager: %v", err)}, nil
	}

	if err := mgr.Connect(conn); err != nil {
		return nil, TablesOutput{Error: fmt.Sprintf("cannot connect to database: %v", err)}, nil
	}
	defer mgr.Disconnect()

	// Determine schema
	schema := input.Schema
	if schema == "" && conn.Schema != "" {
		schema = conn.Schema
	}
	if schema == "" {
		schemas, err := mgr.GetSchemas()
		if err != nil {
			return nil, TablesOutput{Error: fmt.Sprintf("failed to fetch schemas: %v", err)}, nil
		}
		if len(schemas) == 0 {
			return nil, TablesOutput{Error: "no schemas found in database"}, nil
		}
		schema = schemas[0]
	}

	tables, err := mgr.GetStorageUnits(schema)
	if err != nil {
		return nil, TablesOutput{Error: fmt.Sprintf("failed to fetch tables: %v", err)}, nil
	}

	// Convert to output format
	tableInfos := make([]TableInfo, len(tables))
	for i, t := range tables {
		attrs := make(map[string]string)
		for _, attr := range t.Attributes {
			attrs[attr.Key] = attr.Value
		}
		tableInfos[i] = TableInfo{
			Name:       t.Name,
			Attributes: attrs,
		}
	}

	return nil, TablesOutput{Tables: tableInfos, Schema: schema}, nil
}

// HandleColumns describes columns in a table.
func HandleColumns(ctx context.Context, req *mcp.CallToolRequest, input ColumnsInput) (*mcp.CallToolResult, ColumnsOutput, error) {
	conn, err := ResolveConnectionOrDefault(input.Connection)
	if err != nil {
		return nil, ColumnsOutput{Error: err.Error()}, nil
	}

	mgr, err := dbmgr.NewManager()
	if err != nil {
		return nil, ColumnsOutput{Error: fmt.Sprintf("cannot initialize database manager: %v", err)}, nil
	}

	if err := mgr.Connect(conn); err != nil {
		return nil, ColumnsOutput{Error: fmt.Sprintf("cannot connect to database: %v", err)}, nil
	}
	defer mgr.Disconnect()

	// Determine schema
	schema := input.Schema
	if schema == "" && conn.Schema != "" {
		schema = conn.Schema
	}
	if schema == "" {
		schemas, err := mgr.GetSchemas()
		if err != nil {
			return nil, ColumnsOutput{Error: fmt.Sprintf("failed to fetch schemas: %v", err)}, nil
		}
		if len(schemas) == 0 {
			return nil, ColumnsOutput{Error: "no schemas found in database"}, nil
		}
		schema = schemas[0]
	}

	columns, err := mgr.GetColumns(schema, input.Table)
	if err != nil {
		return nil, ColumnsOutput{Error: fmt.Sprintf("failed to fetch columns: %v", err)}, nil
	}

	// Convert to output format
	columnInfos := make([]ColumnInfo, len(columns))
	for i, col := range columns {
		info := ColumnInfo{
			Name:         col.Name,
			Type:         col.Type,
			IsPrimary:    col.IsPrimary,
			IsForeignKey: col.IsForeignKey,
		}
		if col.ReferencedTable != nil {
			info.ReferencedTable = *col.ReferencedTable
		}
		if col.ReferencedColumn != nil {
			info.ReferencedColumn = *col.ReferencedColumn
		}
		columnInfos[i] = info
	}

	return nil, ColumnsOutput{Columns: columnInfos, Table: input.Table, Schema: schema}, nil
}

// HandleConnections lists all available connections.
func HandleConnections(ctx context.Context, req *mcp.CallToolRequest, input ConnectionsInput) (*mcp.CallToolResult, ConnectionsOutput, error) {
	var connections []ConnectionInfo

	// Get saved connections
	mgr, err := dbmgr.NewManager()
	if err == nil {
		for _, conn := range mgr.ListConnections() {
			connections = append(connections, ConnectionInfo{
				Name:     conn.Name,
				Type:     conn.Type,
				Host:     conn.Host,
				Port:     conn.Port,
				Database: conn.Database,
				Schema:   conn.Schema,
				Source:   "saved",
			})
		}
	}

	// Check for environment variable connections
	envConns, _ := ListAvailableConnections()
	savedNames := make(map[string]bool)
	for _, c := range connections {
		savedNames[c.Name] = true
	}

	for _, name := range envConns {
		if !savedNames[name] {
			// This is an env-only connection
			conn, err := ResolveConnection(name)
			if err == nil {
				connections = append(connections, ConnectionInfo{
					Name:     name,
					Type:     conn.Type,
					Host:     conn.Host,
					Port:     conn.Port,
					Database: conn.Database,
					Schema:   conn.Schema,
					Source:   "env",
				})
			}
		}
	}

	return nil, ConnectionsOutput{Connections: connections}, nil
}
