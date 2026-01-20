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
	"log/slog"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version is set at build time.
var Version = "dev"

// ServerOptions configures the MCP server.
type ServerOptions struct {
	// Logger for server messages (defaults to stderr).
	Logger *slog.Logger
	// Instructions provides guidance to LLMs on how to use this server.
	Instructions string
	// ReadOnly prevents INSERT, UPDATE, DELETE, DROP, CREATE, ALTER, TRUNCATE operations.
	// Default: true
	ReadOnly bool
	// ConfirmWrites enables human-in-the-loop confirmation for write operations.
	// When enabled, write operations return a confirmation token that must be approved.
	// Default: false
	ConfirmWrites bool
	// SecurityLevel controls the strictness of query validation.
	// Options: "strict", "standard", "minimal". Default: "standard"
	SecurityLevel SecurityLevel
	// QueryTimeout is the maximum time a query can run before being cancelled.
	// Default: 30 seconds
	QueryTimeout time.Duration
	// MaxRows limits the number of rows returned by queries.
	// Default: 0 (unlimited). Set via --max-rows to enable truncation.
	MaxRows int
	// AllowMultiStatement permits multiple SQL statements in one query (separated by semicolons).
	// WARNING: Enabling this increases SQL injection risk.
	// Default: false
	AllowMultiStatement bool
	// AllowDrop permits DROP/TRUNCATE operations even in allow-write mode.
	// Without this, DROP is blocked unless --confirm-writes is used
	// Default: false
	AllowDrop bool
}

// SecurityOptions contains runtime security settings for query execution
type SecurityOptions struct {
	ReadOnly            bool
	ConfirmWrites       bool
	SecurityLevel       SecurityLevel
	QueryTimeout        time.Duration
	MaxRows             int
	AllowMultiStatement bool
	AllowDrop           bool
}

// NewServer creates a new WhoDB MCP server with all tools registered.
func NewServer(opts *ServerOptions) *mcp.Server {
	if opts == nil {
		opts = &ServerOptions{ConfirmWrites: true} // Safe default
	}

	// Fill in zero-value defaults
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}
	if opts.Instructions == "" {
		opts.Instructions = defaultInstructions
	}
	if opts.SecurityLevel == "" {
		opts.SecurityLevel = SecurityLevelStandard
	}
	if opts.QueryTimeout == 0 {
		opts.QueryTimeout = 30 * time.Second
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "whodb",
		Version: Version,
	}, &mcp.ServerOptions{
		Instructions: opts.Instructions,
		Logger:       opts.Logger,
	})

	// Create security options from server options
	secOpts := &SecurityOptions{
		ReadOnly:            opts.ReadOnly,
		ConfirmWrites:       opts.ConfirmWrites,
		SecurityLevel:       opts.SecurityLevel,
		QueryTimeout:        opts.QueryTimeout,
		MaxRows:             opts.MaxRows,
		AllowMultiStatement: opts.AllowMultiStatement,
		AllowDrop:           opts.AllowDrop,
	}

	// Register tools with security options
	registerTools(server, secOpts)

	// Register resources
	registerResources(server)

	return server
}

// registerTools registers all database tools with the server.
func registerTools(server *mcp.Server, secOpts *SecurityOptions) {
	// Build query tool description based on security settings
	queryDesc := buildQueryDescription(secOpts)

	// whodb_query - Execute SQL queries (with security validation)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "whodb_query",
		Description: queryDesc,
	}, createQueryHandler(secOpts))

	// whodb_schemas - List database schemas
	mcp.AddTool(server, &mcp.Tool{
		Name:        "whodb_schemas",
		Description: "List all schemas (namespaces) in a database. Schemas organize tables and other objects.",
	}, HandleSchemas)

	// whodb_tables - List tables in a schema
	mcp.AddTool(server, &mcp.Tool{
		Name:        "whodb_tables",
		Description: "List all tables in a database schema. Returns table names and metadata like row counts and sizes.",
	}, HandleTables)

	// whodb_columns - Describe table columns
	mcp.AddTool(server, &mcp.Tool{
		Name:        "whodb_columns",
		Description: "Describe the columns in a database table. Returns column names, types, primary keys, and foreign key relationships.",
	}, HandleColumns)

	// whodb_connections - List available connections
	mcp.AddTool(server, &mcp.Tool{
		Name:        "whodb_connections",
		Description: "List all available database connections from saved configurations and environment variables.",
	}, HandleConnections)

	// whodb_confirm - Confirm pending write operations (only registered if confirm-writes is enabled)
	if secOpts.ConfirmWrites {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "whodb_confirm",
			Description: "Confirm a pending write operation. Use this after whodb_query returns a confirmation request for write operations.",
		}, createConfirmHandler(secOpts))
	}
}

// buildQueryDescription creates the tool description based on security settings
func buildQueryDescription(secOpts *SecurityOptions) string {
	if secOpts.ReadOnly {
		return "Execute a SQL query against a database connection. READ-ONLY MODE: Only SELECT, SHOW, DESCRIBE, and EXPLAIN queries are allowed. Write operations (INSERT, UPDATE, DELETE, etc.) are blocked."
	}
	if secOpts.ConfirmWrites {
		return "Execute a SQL query against a database connection. Write operations (INSERT, UPDATE, DELETE, etc.) require confirmation via the whodb_confirm tool."
	}
	return "Execute a SQL query against a database connection. Use this for SELECT, INSERT, UPDATE, DELETE, and other SQL operations."
}

// createQueryHandler creates a query handler with security options
func createQueryHandler(secOpts *SecurityOptions) func(ctx context.Context, req *mcp.CallToolRequest, input QueryInput) (*mcp.CallToolResult, QueryOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input QueryInput) (*mcp.CallToolResult, QueryOutput, error) {
		return HandleQuery(ctx, req, input, secOpts)
	}
}

// createConfirmHandler creates a confirmation handler with security options
func createConfirmHandler(secOpts *SecurityOptions) func(ctx context.Context, req *mcp.CallToolRequest, input ConfirmInput) (*mcp.CallToolResult, ConfirmOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ConfirmInput) (*mcp.CallToolResult, ConfirmOutput, error) {
		return HandleConfirm(ctx, req, input, secOpts)
	}
}

// registerResources registers MCP resources for the server.
func registerResources(server *mcp.Server) {
	// Resource: Available connections
	server.AddResource(&mcp.Resource{
		Name:        "connections",
		URI:         "whodb://connections",
		Description: "List of available database connections",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		conns, err := ListAvailableConnections()
		if err != nil {
			return nil, err
		}

		data, _ := json.MarshalIndent(conns, "", "  ")
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{Text: string(data)},
			},
		}, nil
	})
}

// Run starts the MCP server with the given transport.
func Run(ctx context.Context, server *mcp.Server) error {
	return server.Run(ctx, &mcp.StdioTransport{})
}

const defaultInstructions = `WhoDB MCP Server - Database Management Tools

Available tools:
- whodb_query: Execute SQL queries against a database
- whodb_schemas: List database schemas/namespaces
- whodb_tables: List tables in a schema
- whodb_columns: Describe columns in a table
- whodb_connections: List available database connections
- whodb_confirm: Confirm pending write operations (enabled by default)

SECURITY MODE:
This server runs with confirm-writes enabled by default. When you execute write operations
(INSERT, UPDATE, DELETE, etc.), the query will be shown to the user for approval before
executing. This keeps the user in control while giving you full database functionality.

IMPORTANT: When a write operation requires confirmation, inform the user clearly:
- Tell them what query you're proposing to run
- Explain what it will do in plain language
- Let them know they'll see a confirmation prompt

If the user wants different security settings, they can restart with:
- --read-only: No writes allowed at all
- --allow-write: Full write access without confirmation (not recommended for production)

Connection names reference either:
1. Environment profiles, for example:
   - WHODB_POSTGRES='[{"alias":"prod","host":"localhost","user":"user","password":"pass","database":"db","port":"5432"}]'
   - WHODB_MYSQL_1='{"alias":"staging","host":"localhost","user":"user","password":"pass","database":"db","port":"3306"}'
2. Saved connections from WhoDB CLI configuration
Saved connections take precedence when names collide.

Example workflow:
1. List connections: whodb_connections
2. Explore schema: whodb_schemas(connection="mydb")
3. List tables: whodb_tables(connection="mydb", schema="public")
4. Describe table: whodb_columns(connection="mydb", table="users")
5. Query data: whodb_query(connection="mydb", query="SELECT * FROM users LIMIT 10")
6. Write data: whodb_query(connection="mydb", query="INSERT INTO...") -> user confirms -> whodb_confirm(token="...")

Best practices:
- Send ONE query at a time (multi-statement queries are blocked for security)
- Always use LIMIT for exploratory queries
- Check schema structure before writing queries
- Prefer specific column selection over SELECT *
- For writes, explain to the user what will happen before proposing the query
`
