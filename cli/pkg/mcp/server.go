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
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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
		Description: descSchemas,
	}, HandleSchemas)

	// whodb_tables - List tables in a schema
	mcp.AddTool(server, &mcp.Tool{
		Name:        "whodb_tables",
		Description: descTables,
	}, HandleTables)

	// whodb_columns - Describe table columns
	mcp.AddTool(server, &mcp.Tool{
		Name:        "whodb_columns",
		Description: descColumns,
	}, HandleColumns)

	// whodb_connections - List available connections
	mcp.AddTool(server, &mcp.Tool{
		Name:        "whodb_connections",
		Description: descConnections,
	}, HandleConnections)

	// whodb_confirm - Confirm pending write operations (only registered if confirm-writes is enabled)
	if secOpts.ConfirmWrites {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "whodb_confirm",
			Description: descConfirm,
		}, createConfirmHandler(secOpts))
	}
}

// buildQueryDescription creates the tool description based on security settings
func buildQueryDescription(secOpts *SecurityOptions) string {
	base := `Execute a SQL query against a database connection.

**Best for:** Running SQL SELECT, INSERT, UPDATE, DELETE statements when you need to query or modify data.
**Not recommended for:** Schema exploration (use whodb_schemas, whodb_tables, whodb_columns instead for faster, structured results).
**Common mistakes:** Running queries without specifying connection when multiple exist; using SELECT * instead of specific columns; forgetting LIMIT on large tables.

**Usage Example:**
` + "```json" + `
{
  "name": "whodb_query",
  "arguments": {
    "connection": "mydb",
    "query": "SELECT id, name, email FROM users WHERE active = true LIMIT 10"
  }
}
` + "```" + `

**Best practices:**
- Always use LIMIT for exploratory queries to avoid overwhelming results
- Prefer specific column selection over SELECT *
- Check schema structure with whodb_columns before writing complex queries
- Use parameterized values in your queries when possible`

	if secOpts.ReadOnly {
		return base + `

**Security Mode: READ-ONLY**
Only SELECT, SHOW, DESCRIBE, and EXPLAIN queries are allowed. Write operations (INSERT, UPDATE, DELETE, DROP, etc.) are blocked.`
	}
	if secOpts.ConfirmWrites {
		return base + `

**Security Mode: CONFIRM-WRITES (Default)**
Write operations (INSERT, UPDATE, DELETE, etc.) require user confirmation. When you submit a write query:
1. The query is validated but NOT executed
2. You receive a confirmation_token
3. Explain to the user what the query will do
4. Call whodb_confirm with the token after user approves
5. The query executes and returns results`
	}
	return base + `

**Security Mode: ALLOW-WRITE**
Full write access enabled. All queries execute immediately. Use with caution in production.`
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

// TransportType specifies the transport mechanism for the MCP server.
type TransportType string

const (
	// TransportStdio uses stdin/stdout for communication (default, for CLI integration).
	TransportStdio TransportType = "stdio"
	// TransportHTTP runs as an HTTP server with streaming support.
	TransportHTTP TransportType = "http"
)

// HTTPOptions configures the HTTP transport.
type HTTPOptions struct {
	// Host to bind to (default: "localhost").
	Host string
	// Port to listen on (default: 3000).
	Port int
	// RateLimit configures rate limiting for the HTTP transport.
	// Rate limiting is disabled by default.
	RateLimit RateLimitOptions
}

// Run starts the MCP server with stdio transport.
func Run(ctx context.Context, server *mcp.Server) error {
	return server.Run(ctx, &mcp.StdioTransport{})
}

// RunHTTP starts the MCP server as an HTTP service.
func RunHTTP(ctx context.Context, server *mcp.Server, opts *HTTPOptions, logger *slog.Logger) error {
	if opts == nil {
		opts = &HTTPOptions{}
	}
	if opts.Host == "" {
		opts.Host = "localhost"
	}
	if opts.Port == 0 {
		opts.Port = 3000
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	// Warn if binding to non-localhost (network exposure)
	if opts.Host != "localhost" && opts.Host != "127.0.0.1" {
		logger.Warn("MCP server binding to network interface - no authentication is configured",
			"host", opts.Host,
			"recommendation", "Use --host=localhost for local development, or add authentication for production")
	}

	// Create HTTP handler for MCP
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil)

	// Wrap with rate limiter if enabled
	var handler http.Handler = mcpHandler
	var rateLimiter *RateLimiter
	if opts.RateLimit.Enabled {
		rateLimiter = NewRateLimiter(opts.RateLimit)
		handler = rateLimiter.Middleware(mcpHandler)
		logger.Info("Rate limiting enabled",
			"qps", opts.RateLimit.QPS,
			"daily", opts.RateLimit.Daily,
			"bypass", opts.RateLimit.BypassToken != "")

		// Start cleanup goroutine to prevent memory growth
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					rateLimiter.Cleanup(10 * time.Minute)
				}
			}
		}()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]any{"status": "ok"}
		if rateLimiter != nil {
			response["rate_limit"] = rateLimiter.Stats()
		}
		_ = json.NewEncoder(w).Encode(response)
	})
	mux.Handle("/mcp", handler)

	addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	logger.Info("MCP server listening", "transport", "http", "address", addr, "endpoint", "/mcp", "health", "/health")
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("HTTP server error: %w", err)
	}
	return nil
}

// Tool descriptions with usage guidance

const descSchemas = `List all schemas (namespaces) in a database.

**Best for:** Discovering what schemas exist in a database; understanding database organization before exploring tables.
**Not recommended for:** When you already know the schema name (proceed directly to whodb_tables).
**Common mistakes:** Calling this repeatedly - schema lists rarely change during a session.

**Usage Example:**
` + "```json" + `
{
  "name": "whodb_schemas",
  "arguments": {
    "connection": "mydb"
  }
}
` + "```" + `

**Returns:** Array of schema names (e.g., ["public", "analytics", "audit"]).
**Typical workflow:** whodb_schemas → whodb_tables → whodb_columns → whodb_query`

const descTables = `List all tables in a database schema.

**Best for:** Discovering what tables exist in a schema; getting table metadata like row counts.
**Not recommended for:** When you already know the table name (proceed directly to whodb_columns or whodb_query).
**Common mistakes:** Not specifying schema when the database has multiple schemas with same-named tables.

**Usage Example:**
` + "```json" + `
{
  "name": "whodb_tables",
  "arguments": {
    "connection": "mydb",
    "schema": "public"
  }
}
` + "```" + `

**Returns:** Array of table objects with name and attributes (row count, size, etc.).
**Note:** If schema is omitted, uses the connection's default schema or the first available schema.`

const descColumns = `Describe the columns in a database table.

**Best for:** Understanding table structure before writing queries; discovering primary keys and foreign key relationships.
**Not recommended for:** When you need actual data (use whodb_query with SELECT).
**Common mistakes:** Forgetting to specify the table name; not using this before writing INSERT/UPDATE queries.

**Usage Example:**
` + "```json" + `
{
  "name": "whodb_columns",
  "arguments": {
    "connection": "mydb",
    "table": "users",
    "schema": "public"
  }
}
` + "```" + `

**Returns:** Array of column objects with:
- name: Column name
- type: Data type (varchar, integer, timestamp, etc.)
- is_primary: Whether this is a primary key
- is_foreign_key: Whether this references another table
- referenced_table/referenced_column: Foreign key target (if applicable)

**Pro tip:** Always call this before writing INSERT queries to ensure correct column names and types.`

const descConnections = `List all available database connections.

**Best for:** Discovering what databases are configured; choosing which connection to use.
**Not recommended for:** When you already know the connection name.
**Common mistakes:** Not calling this first when connection name is unknown.

**Usage Example:**
` + "```json" + `
{
  "name": "whodb_connections",
  "arguments": {}
}
` + "```" + `

**Returns:** Array of connection objects with:
- name: Connection identifier to use in other tools
- type: Database type (postgres, mysql, sqlite, etc.)
- host/port/database: Connection details (passwords are never exposed)
- source: "saved" (from CLI config) or "env" (from environment variables)

**Note:** If only one connection exists, other tools will use it automatically when connection is omitted.`

const descConfirm = `Confirm and execute a pending write operation.

**Best for:** Executing write queries after user approval in confirm-writes mode.
**Not recommended for:** Read queries (they execute immediately without confirmation).
**Common mistakes:** Using an expired token (tokens expire after 60 seconds); not explaining the query to the user before confirming.

**Usage Example:**
` + "```json" + `
{
  "name": "whodb_confirm",
  "arguments": {
    "token": "550e8400-e29b-41d4-a716-446655440000"
  }
}
` + "```" + `

**Workflow:**
1. Call whodb_query with a write operation (INSERT, UPDATE, DELETE, etc.)
2. Receive confirmation_required=true and a confirmation_token
3. Explain to the user what the query will do in plain language
4. After user approves, call whodb_confirm with the token
5. Query executes and returns results

**Important:** Tokens are single-use and expire after 60 seconds. If expired, re-submit the original query to get a new token.`

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
