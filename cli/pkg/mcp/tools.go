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
	"errors"
	"fmt"
	"sync"
	"time"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// QueryInput is the input for the whodb_query tool.
type QueryInput struct {
	// Connection is the name of a saved connection or environment profile.
	Connection string `json:"connection"`
	// Query is the SQL query to execute
	Query string `json:"query"`
}

// QueryOutput is the output for the whodb_query tool.
type QueryOutput struct {
	Columns              []string `json:"columns"`
	Rows                 [][]any  `json:"rows"`
	Error                string   `json:"error,omitempty"`
	Warning              string   `json:"warning,omitempty"`
	ConfirmationRequired bool     `json:"confirmation_required,omitempty"`
	ConfirmationToken    string   `json:"confirmation_token,omitempty"`
	ConfirmationQuery    string   `json:"confirmation_query,omitempty"`
}

// SchemasInput is the input for the whodb_schemas tool.
type SchemasInput struct {
	// Connection is the name of a saved connection or environment profile.
	Connection string `json:"connection"`
}

// SchemasOutput is the output for the whodb_schemas tool.
type SchemasOutput struct {
	Schemas []string `json:"schemas"`
	Error   string   `json:"error,omitempty"`
}

// TablesInput is the input for the whodb_tables tool.
type TablesInput struct {
	// Connection is the name of a saved connection or environment profile.
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
	// Connection is the name of a saved connection or environment profile.
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

// ConnectionInfo represents a connection (without sensitive data).
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

// ConfirmInput is the input for the whodb_confirm tool.
type ConfirmInput struct {
	// Token is the confirmation token from a previous query response
	Token string `json:"token"`
}

// ConfirmOutput is the output for the whodb_confirm tool.
type ConfirmOutput struct {
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
	Error   string   `json:"error,omitempty"`
	Message string   `json:"message,omitempty"`
}

// PendingConfirmation stores a query awaiting user confirmation
type PendingConfirmation struct {
	Token      string
	Query      string
	Connection string
	ExpiresAt  time.Time
}

// pendingConfirmations stores queries awaiting confirmation
var (
	pendingConfirmations = make(map[string]*PendingConfirmation)
	pendingMutex         sync.RWMutex
)

// generateConfirmationToken creates a secure random token
func generateConfirmationToken() string {
	return uuid.New().String()
}

// storePendingConfirmation stores a query for later confirmation
func storePendingConfirmation(query, connection string) string {
	token := generateConfirmationToken()

	pendingMutex.Lock()
	defer pendingMutex.Unlock()

	// Clean up expired confirmations
	now := time.Now()
	for k, v := range pendingConfirmations {
		if v.ExpiresAt.Before(now) {
			delete(pendingConfirmations, k)
		}
	}

	pendingConfirmations[token] = &PendingConfirmation{
		Token:      token,
		Query:      query,
		Connection: connection,
		ExpiresAt:  now.Add(60 * time.Second), // 60 second expiry
	}

	return token
}

// getPendingConfirmation retrieves and removes a pending confirmation
func getPendingConfirmation(token string) (*PendingConfirmation, error) {
	pendingMutex.Lock()
	defer pendingMutex.Unlock()

	pending, ok := pendingConfirmations[token]
	if !ok {
		return nil, errors.New("confirmation token not found or expired")
	}

	if pending.ExpiresAt.Before(time.Now()) {
		delete(pendingConfirmations, token)
		return nil, errors.New("confirmation token has expired")
	}

	// Remove after use (one-time use)
	delete(pendingConfirmations, token)
	return pending, nil
}

// countAvailableConnections returns the number of available database connections.
func countAvailableConnections() int {
	conns, _ := ListAvailableConnections()
	return len(conns)
}

// Query execution helpers

// executeQuery runs a query with optional timeout and returns the result
func executeQuery(ctx context.Context, mgr *dbmgr.Manager, query string, timeout time.Duration) (*engine.GetRowsResult, error) {
	if timeout > 0 {
		queryCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return mgr.ExecuteQueryWithContext(queryCtx, query)
	}
	return mgr.ExecuteQuery(query)
}

// convertColumns extracts column names from the result
func convertColumns(result *engine.GetRowsResult) []string {
	columns := make([]string, len(result.Columns))
	for i, col := range result.Columns {
		columns[i] = col.Name
	}
	return columns
}

// convertRows converts result rows to [][]any with optional row limit
// Returns the converted rows and whether results were truncated
func convertRows(result *engine.GetRowsResult, maxRows int) ([][]any, bool) {
	totalRows := len(result.Rows)
	rowLimit := totalRows
	if maxRows > 0 && totalRows > maxRows {
		rowLimit = maxRows
	}

	rows := make([][]any, rowLimit)
	for i := 0; i < rowLimit; i++ {
		row := result.Rows[i]
		rows[i] = make([]any, len(row))
		for j, val := range row {
			rows[i][j] = val
		}
	}

	return rows, totalRows > rowLimit
}

// Tool handlers

// HandleQuery executes a SQL query against the specified connection with security validation.
func HandleQuery(ctx context.Context, req *mcp.CallToolRequest, input QueryInput, secOpts *SecurityOptions) (*mcp.CallToolResult, QueryOutput, error) {
	// Validate input
	connCount := countAvailableConnections()
	if err := ValidateQueryInput(&input, connCount); err != nil {
		return nil, QueryOutput{Error: err.Error()}, nil
	}

	// Detect statement type first
	stmtType := DetectStatementType(input.Query)

	// In confirm-writes mode, write operations need confirmation before execution
	if secOpts.ConfirmWrites && IsWriteStatement(stmtType) {
		// Validate the query first (check for other issues like multi-statement, dangerous functions)
		// Pass allowWrite=true and allowDestructive=true since user will confirm
		err := ValidateSQLStatement(input.Query, true, secOpts.SecurityLevel, secOpts.AllowMultiStatement, true)
		if err != nil {
			// Query has other issues beyond just being a write operation
			return nil, QueryOutput{Error: fmt.Sprintf("query blocked: %v", err)}, nil
		}

		// Create a pending confirmation for the write operation
		token := storePendingConfirmation(input.Query, input.Connection)

		return nil, QueryOutput{
			ConfirmationRequired: true,
			ConfirmationToken:    token,
			ConfirmationQuery:    input.Query,
			Warning:              fmt.Sprintf("This %s operation requires your approval before it runs. You have 60 seconds to confirm or cancel.", stmtType),
		}, nil
	}

	// Validate the query against security settings
	// allowDestructive: only if --allow-drop flag is set (confirm-writes handled above)
	allowWrite := !secOpts.ReadOnly
	allowDestructive := secOpts.AllowDrop
	err := ValidateSQLStatement(input.Query, allowWrite, secOpts.SecurityLevel, secOpts.AllowMultiStatement, allowDestructive)
	if err != nil {
		return nil, QueryOutput{Error: fmt.Sprintf("query blocked: %v", err)}, nil
	}

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

	result, err := executeQuery(ctx, mgr, input.Query, secOpts.QueryTimeout)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, QueryOutput{Error: fmt.Sprintf("query timed out after %v", secOpts.QueryTimeout)}, nil
		}
		return nil, QueryOutput{Error: fmt.Sprintf("query failed: %v", err)}, nil
	}

	columns := convertColumns(result)
	rows, truncated := convertRows(result, secOpts.MaxRows)

	output := QueryOutput{Columns: columns, Rows: rows}
	if truncated {
		output.Warning = fmt.Sprintf("Results truncated to %d rows. Use LIMIT in your query for more control.", secOpts.MaxRows)
	}

	return nil, output, nil
}

// HandleConfirm confirms and executes a pending write operation.
func HandleConfirm(ctx context.Context, req *mcp.CallToolRequest, input ConfirmInput, secOpts *SecurityOptions) (*mcp.CallToolResult, ConfirmOutput, error) {
	// Validate input
	if err := ValidateConfirmInput(&input); err != nil {
		return nil, ConfirmOutput{Error: err.Error()}, nil
	}

	// Get the pending confirmation
	pending, err := getPendingConfirmation(input.Token)
	if err != nil {
		return nil, ConfirmOutput{Error: err.Error()}, nil
	}

	// Resolve connection
	conn, err := ResolveConnectionOrDefault(pending.Connection)
	if err != nil {
		return nil, ConfirmOutput{Error: err.Error()}, nil
	}

	mgr, err := dbmgr.NewManager()
	if err != nil {
		return nil, ConfirmOutput{Error: fmt.Sprintf("cannot initialize database manager: %v", err)}, nil
	}

	if err := mgr.Connect(conn); err != nil {
		return nil, ConfirmOutput{Error: fmt.Sprintf("cannot connect to database: %v", err)}, nil
	}
	defer mgr.Disconnect()

	result, err := executeQuery(ctx, mgr, pending.Query, secOpts.QueryTimeout)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, ConfirmOutput{Error: fmt.Sprintf("query timed out after %v", secOpts.QueryTimeout)}, nil
		}
		return nil, ConfirmOutput{Error: fmt.Sprintf("query failed: %v", err)}, nil
	}

	columns := convertColumns(result)
	rows, _ := convertRows(result, 0) // No row limit for confirmed writes

	stmtType := DetectStatementType(pending.Query)
	return nil, ConfirmOutput{
		Columns: columns,
		Rows:    rows,
		Message: fmt.Sprintf("%s operation completed successfully", stmtType),
	}, nil
}

// HandleSchemas lists all schemas in the database.
func HandleSchemas(ctx context.Context, req *mcp.CallToolRequest, input SchemasInput) (*mcp.CallToolResult, SchemasOutput, error) {
	// Validate input
	connCount := countAvailableConnections()
	if err := ValidateSchemasInput(&input, connCount); err != nil {
		return nil, SchemasOutput{Error: err.Error()}, nil
	}

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
	// Validate input
	connCount := countAvailableConnections()
	if err := ValidateTablesInput(&input, connCount); err != nil {
		return nil, TablesOutput{Error: err.Error()}, nil
	}

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
	// Validate input
	connCount := countAvailableConnections()
	if err := ValidateColumnsInput(&input, connCount); err != nil {
		return nil, ColumnsOutput{Error: err.Error()}, nil
	}

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

	mgr, err := dbmgr.NewManager()
	if err != nil {
		return nil, ConnectionsOutput{Error: fmt.Sprintf("cannot initialize database manager: %v", err)}, nil
	}

	for _, info := range mgr.ListConnectionsWithSource() {
		conn := info.Connection
		connections = append(connections, ConnectionInfo{
			Name:     conn.Name,
			Type:     conn.Type,
			Host:     conn.Host,
			Port:     conn.Port,
			Database: conn.Database,
			Schema:   conn.Schema,
			Source:   info.Source,
		})
	}

	return nil, ConnectionsOutput{Connections: connections}, nil
}
