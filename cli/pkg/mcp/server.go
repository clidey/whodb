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
	"encoding/json"
	"log/slog"
	"os"

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
}

// NewServer creates a new WhoDB MCP server with all tools registered.
func NewServer(opts *ServerOptions) *mcp.Server {
	if opts == nil {
		opts = &ServerOptions{}
	}

	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	if opts.Instructions == "" {
		opts.Instructions = defaultInstructions
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "whodb",
		Version: Version,
	}, &mcp.ServerOptions{
		Instructions: opts.Instructions,
		Logger:       opts.Logger,
	})

	// Register tools
	registerTools(server)

	// Register resources
	registerResources(server)

	return server
}

// registerTools registers all database tools with the server.
func registerTools(server *mcp.Server) {
	// whodb_query - Execute SQL queries
	mcp.AddTool(server, &mcp.Tool{
		Name:        "whodb_query",
		Description: "Execute a SQL query against a database connection. Use this for SELECT, INSERT, UPDATE, DELETE, and other SQL operations.",
	}, HandleQuery)

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

Connection names reference either:
1. Environment variables: WHODB_{NAME}_URI (e.g., "prod" -> WHODB_PROD_URI)
2. Saved connections from WhoDB CLI configuration

Example workflow:
1. List connections: whodb_connections
2. Explore schema: whodb_schemas(connection="mydb")
3. List tables: whodb_tables(connection="mydb", schema="public")
4. Describe table: whodb_columns(connection="mydb", table="users")
5. Query data: whodb_query(connection="mydb", query="SELECT * FROM users LIMIT 10")

Best practices:
- Always use LIMIT for exploratory queries
- Check schema structure before writing queries
- Use parameterized queries when possible
- Prefer specific column selection over SELECT *
`
