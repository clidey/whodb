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
	// EnabledTools specifies which tools to enable. If empty, all tools are enabled.
	// Valid values: "query", "schemas", "tables", "columns", "connections", "confirm"
	EnabledTools []string
	// DisabledTools specifies which tools to disable. Takes precedence over EnabledTools.
	// Valid values: "query", "schemas", "tables", "columns", "connections", "confirm"
	DisabledTools []string
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

	// Create tool enablement from server options
	toolEnablement := &ToolEnablement{
		EnabledTools:  opts.EnabledTools,
		DisabledTools: opts.DisabledTools,
	}

	// Register tools with security options and enablement
	registerTools(server, secOpts, toolEnablement)

	// Register prompts for AI assistant guidance
	registerPrompts(server)

	// Register resources
	registerResources(server)

	return server
}

// ToolEnablement tracks which tools should be registered.
type ToolEnablement struct {
	EnabledTools  []string
	DisabledTools []string
}

// isToolEnabled checks if a tool should be registered based on enabled/disabled lists.
func (te *ToolEnablement) isToolEnabled(toolName string) bool {
	// If disabled list contains this tool, it's disabled
	for _, t := range te.DisabledTools {
		if t == toolName {
			return false
		}
	}
	// If enabled list is empty, all tools are enabled by default
	if len(te.EnabledTools) == 0 {
		return true
	}
	// If enabled list is specified, only those tools are enabled
	for _, t := range te.EnabledTools {
		if t == toolName {
			return true
		}
	}
	return false
}

// boolPtr returns a pointer to a bool value (helper for ToolAnnotations).
func boolPtr(b bool) *bool {
	return &b
}

// registerTools registers all database tools with the server.
func registerTools(server *mcp.Server, secOpts *SecurityOptions, toolEnablement *ToolEnablement) {
	if toolEnablement == nil {
		toolEnablement = &ToolEnablement{}
	}

	// Build query tool description based on security settings
	queryDesc := buildQueryDescription(secOpts)

	// whodb_query - Execute SQL queries (with security validation)
	// Hints: Can modify data (not read-only), potentially destructive (DELETE/DROP),
	// not idempotent (same INSERT twice = duplicates), closed world (database only)
	if toolEnablement.isToolEnabled("query") {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "whodb_query",
			Description: queryDesc,
			Annotations: &mcp.ToolAnnotations{
				Title:           "Execute SQL Query",
				ReadOnlyHint:    secOpts.ReadOnly, // If server is read-only, this tool is read-only
				DestructiveHint: boolPtr(!secOpts.ReadOnly && !secOpts.ConfirmWrites), // Destructive only if writes allowed without confirmation
				IdempotentHint:  false,            // Queries can have side effects
				OpenWorldHint:   boolPtr(false),   // Closed world (database only)
			},
		}, createQueryHandler(secOpts))
	}

	// whodb_schemas - List database schemas
	// Hints: Read-only, idempotent, closed world
	if toolEnablement.isToolEnabled("schemas") {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "whodb_schemas",
			Description: descSchemas,
			Annotations: &mcp.ToolAnnotations{
				Title:          "List Database Schemas",
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  boolPtr(false),
			},
		}, HandleSchemas)
	}

	// whodb_tables - List tables in a schema
	// Hints: Read-only, idempotent, closed world
	if toolEnablement.isToolEnabled("tables") {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "whodb_tables",
			Description: descTables,
			Annotations: &mcp.ToolAnnotations{
				Title:          "List Database Tables",
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  boolPtr(false),
			},
		}, HandleTables)
	}

	// whodb_columns - Describe table columns
	// Hints: Read-only, idempotent, closed world
	if toolEnablement.isToolEnabled("columns") {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "whodb_columns",
			Description: descColumns,
			Annotations: &mcp.ToolAnnotations{
				Title:          "Describe Table Columns",
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  boolPtr(false),
			},
		}, HandleColumns)
	}

	// whodb_connections - List available connections
	// Hints: Read-only, idempotent, closed world
	if toolEnablement.isToolEnabled("connections") {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "whodb_connections",
			Description: descConnections,
			Annotations: &mcp.ToolAnnotations{
				Title:          "List Database Connections",
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  boolPtr(false),
			},
		}, HandleConnections)
	}

	// whodb_confirm - Confirm pending write operations (only registered if confirm-writes is enabled)
	// Hints: Not read-only (executes writes), potentially destructive, not idempotent (tokens are single-use)
	if secOpts.ConfirmWrites && toolEnablement.isToolEnabled("confirm") {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "whodb_confirm",
			Description: descConfirm,
			Annotations: &mcp.ToolAnnotations{
				Title:           "Confirm Write Operation",
				ReadOnlyHint:    false,
				DestructiveHint: boolPtr(true), // Can execute destructive operations
				IdempotentHint:  false,         // Tokens are single-use
				OpenWorldHint:   boolPtr(false),
			},
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

// registerPrompts registers MCP prompts for AI assistant guidance.
func registerPrompts(server *mcp.Server) {
	// query_help - Guidance on writing SQL queries
	server.AddPrompt(&mcp.Prompt{
		Name:        "query_help",
		Title:       "SQL Query Help",
		Description: "Get guidance on writing SQL queries with WhoDB, including best practices and examples.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "database_type",
				Title:       "Database Type",
				Description: "The type of database (postgres, mysql, sqlite, etc.) for dialect-specific help.",
				Required:    false,
			},
			{
				Name:        "query_type",
				Title:       "Query Type",
				Description: "The type of query you need help with (select, insert, update, delete, join, aggregate).",
				Required:    false,
			},
		},
	}, handleQueryHelpPrompt)

	// schema_exploration_help - Guidance on exploring database structure
	server.AddPrompt(&mcp.Prompt{
		Name:        "schema_exploration_help",
		Title:       "Schema Exploration Help",
		Description: "Learn how to effectively explore database schemas, tables, and relationships using WhoDB tools.",
	}, handleSchemaExplorationHelpPrompt)

	// workflow_help - Common database workflows
	server.AddPrompt(&mcp.Prompt{
		Name:        "workflow_help",
		Title:       "Database Workflow Help",
		Description: "Get guidance on common database workflows like data analysis, debugging queries, or understanding table relationships.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "workflow",
				Title:       "Workflow Type",
				Description: "The workflow you need help with (analysis, debugging, relationships, migration).",
				Required:    false,
			},
		},
	}, handleWorkflowHelpPrompt)
}

// handleQueryHelpPrompt returns guidance for writing SQL queries.
func handleQueryHelpPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	dbType := ""
	queryType := ""
	if req.Params.Arguments != nil {
		if v, ok := req.Params.Arguments["database_type"]; ok {
			dbType = v
		}
		if v, ok := req.Params.Arguments["query_type"]; ok {
			queryType = v
		}
	}

	content := buildQueryHelpContent(dbType, queryType)
	return &mcp.GetPromptResult{
		Description: "SQL query guidance for WhoDB",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: content},
			},
		},
	}, nil
}

// handleSchemaExplorationHelpPrompt returns guidance for exploring database structure.
func handleSchemaExplorationHelpPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	content := `I need help exploring a database schema. Please guide me through the WhoDB workflow:

## Recommended Exploration Workflow

1. **Start with connections** - Use whodb_connections to see available databases
2. **List schemas** - Use whodb_schemas to discover namespaces (public, analytics, etc.)
3. **Explore tables** - Use whodb_tables to see tables in a schema with metadata
4. **Understand structure** - Use whodb_columns to see columns, types, and relationships
5. **Query data** - Use whodb_query for actual data exploration

## Tips for Effective Exploration

- Always check foreign key relationships in whodb_columns output
- Use row counts from whodb_tables to identify large tables before querying
- Look for naming patterns (e.g., *_id columns often indicate relationships)
- Check for audit columns (created_at, updated_at) to understand data lifecycle

## Example Exploration Session

` + "```" + `
# Step 1: What databases are available?
whodb_connections → [{name: "mydb", type: "postgres", ...}]

# Step 2: What schemas exist?
whodb_schemas(connection="mydb") → ["public", "analytics"]

# Step 3: What tables are in public schema?
whodb_tables(connection="mydb", schema="public") → [{name: "users", ...}, {name: "orders", ...}]

# Step 4: What's the structure of the users table?
whodb_columns(connection="mydb", table="users") → [{name: "id", is_primary: true}, ...]

# Step 5: Sample some data
whodb_query(connection="mydb", query="SELECT * FROM users LIMIT 5")
` + "```" + `

Please help me explore my database using this workflow.`

	return &mcp.GetPromptResult{
		Description: "Database schema exploration guidance",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: content},
			},
		},
	}, nil
}

// handleWorkflowHelpPrompt returns guidance for common database workflows.
func handleWorkflowHelpPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	workflow := "general"
	if req.Params.Arguments != nil {
		if v, ok := req.Params.Arguments["workflow"]; ok {
			workflow = v
		}
	}

	content := buildWorkflowHelpContent(workflow)
	return &mcp.GetPromptResult{
		Description: "Database workflow guidance",
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: content},
			},
		},
	}, nil
}

// buildQueryHelpContent builds context-aware query help.
func buildQueryHelpContent(dbType, queryType string) string {
	base := `I need help writing SQL queries with WhoDB. Please provide guidance on:

## Query Best Practices

1. **Always use LIMIT** for exploratory queries to avoid overwhelming results
2. **Select specific columns** instead of SELECT * for better performance
3. **Check schema first** with whodb_columns before writing complex queries
4. **Use parameterized patterns** when constructing dynamic queries

## Security Notes

- WhoDB validates queries before execution
- Multi-statement queries (separated by ;) are blocked by default
- Dangerous functions may be blocked depending on security level
- Write operations may require confirmation depending on server mode

`

	if dbType != "" {
		base += fmt.Sprintf("\n## %s-Specific Tips\n\n", dbType)
		switch dbType {
		case "postgres":
			base += `- Use ILIKE for case-insensitive pattern matching
- Use jsonb operators (->>, @>) for JSON columns
- Use CTEs (WITH clause) for complex queries
- Use EXPLAIN ANALYZE to understand query performance
`
		case "mysql":
			base += `- Use LIKE with BINARY for case-sensitive matching
- Use JSON_EXTRACT() for JSON columns
- Use SHOW CREATE TABLE to see table definitions
- Use EXPLAIN to understand query execution
`
		case "sqlite":
			base += `- SQLite is type-flexible, but be consistent
- Use json_extract() for JSON columns
- Use .schema command equivalent via whodb_columns
- VACUUM periodically for performance
`
		}
	}

	if queryType != "" {
		base += fmt.Sprintf("\n## %s Query Examples\n\n", queryType)
		switch queryType {
		case "select":
			base += "```sql\n-- Basic select with filtering\nSELECT id, name, email FROM users WHERE active = true LIMIT 10;\n\n-- With ordering\nSELECT * FROM orders ORDER BY created_at DESC LIMIT 20;\n```\n"
		case "insert":
			base += "```sql\n-- Single row insert\nINSERT INTO users (name, email) VALUES ('John', 'john@example.com');\n\n-- Multiple rows\nINSERT INTO users (name, email) VALUES ('A', 'a@x.com'), ('B', 'b@x.com');\n```\n"
		case "update":
			base += "```sql\n-- Update with WHERE clause (always use WHERE!)\nUPDATE users SET active = false WHERE last_login < '2024-01-01';\n\n-- Update single row by ID\nUPDATE orders SET status = 'shipped' WHERE id = 123;\n```\n"
		case "delete":
			base += "```sql\n-- Delete with WHERE clause (always use WHERE!)\nDELETE FROM sessions WHERE expires_at < NOW();\n\n-- Delete by ID\nDELETE FROM users WHERE id = 456;\n```\n"
		case "join":
			base += "```sql\n-- Inner join\nSELECT u.name, o.total FROM users u\nJOIN orders o ON u.id = o.user_id\nWHERE o.status = 'completed' LIMIT 10;\n\n-- Left join to include users without orders\nSELECT u.name, COUNT(o.id) as order_count\nFROM users u LEFT JOIN orders o ON u.id = o.user_id\nGROUP BY u.id, u.name;\n```\n"
		case "aggregate":
			base += "```sql\n-- Count with grouping\nSELECT status, COUNT(*) as count FROM orders GROUP BY status;\n\n-- Sum with filtering\nSELECT user_id, SUM(total) as total_spent\nFROM orders WHERE created_at > '2024-01-01'\nGROUP BY user_id HAVING SUM(total) > 100;\n```\n"
		}
	}

	base += "\nPlease help me write effective SQL queries following these guidelines."
	return base
}

// buildWorkflowHelpContent builds workflow-specific guidance.
func buildWorkflowHelpContent(workflow string) string {
	switch workflow {
	case "analysis":
		return `I need help with data analysis using WhoDB. Guide me through:

## Data Analysis Workflow

1. **Understand the data** - Explore schema to find relevant tables
2. **Sample the data** - Run small queries to understand data patterns
3. **Aggregate and summarize** - Use GROUP BY, COUNT, SUM, AVG
4. **Filter and refine** - Add WHERE clauses based on findings
5. **Export or report** - Format results for further use

## Useful Analysis Patterns

` + "```sql" + `
-- Distribution analysis
SELECT column, COUNT(*) as count
FROM table GROUP BY column ORDER BY count DESC;

-- Time-based analysis
SELECT DATE(created_at) as date, COUNT(*) as daily_count
FROM events GROUP BY DATE(created_at) ORDER BY date;

-- Percentile/quantile analysis (PostgreSQL)
SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY value) as median
FROM measurements;
` + "```" + `

Please help me analyze my data using these patterns.`

	case "debugging":
		return `I need help debugging database queries with WhoDB. Guide me through:

## Query Debugging Workflow

1. **Start simple** - Run the simplest version of your query
2. **Add complexity gradually** - Add one clause at a time
3. **Check data types** - Use whodb_columns to verify column types
4. **Verify relationships** - Check foreign keys before JOINs
5. **Inspect intermediate results** - Break complex queries into steps

## Common Issues to Check

- NULL handling (use IS NULL, not = NULL)
- Data type mismatches in comparisons
- Missing indexes causing slow queries
- Incorrect JOIN conditions causing duplicates
- Case sensitivity in string comparisons

## Debugging Steps

` + "```" + `
# Step 1: Check table structure
whodb_columns(table="problematic_table")

# Step 2: Sample raw data
SELECT * FROM table LIMIT 5;

# Step 3: Check for NULLs
SELECT COUNT(*), COUNT(column) FROM table;

# Step 4: Verify JOIN data exists
SELECT COUNT(*) FROM table1 t1
JOIN table2 t2 ON t1.id = t2.foreign_id;
` + "```" + `

Please help me debug my query using this approach.`

	case "relationships":
		return `I need help understanding table relationships with WhoDB.

## Finding Relationships

1. **Check foreign keys** - whodb_columns shows is_foreign_key and referenced_table
2. **Look for naming patterns** - Columns like user_id, order_id suggest relationships
3. **Verify with data** - Query to confirm relationships exist

## Relationship Discovery Workflow

` + "```" + `
# Step 1: Get columns with FK info
whodb_columns(table="orders")
# Look for: is_foreign_key: true, referenced_table: "users"

# Step 2: Verify the relationship
SELECT o.id, u.name
FROM orders o JOIN users u ON o.user_id = u.id LIMIT 5;

# Step 3: Check cardinality
SELECT user_id, COUNT(*) as order_count
FROM orders GROUP BY user_id ORDER BY order_count DESC LIMIT 10;
` + "```" + `

## Common Relationship Types

- **One-to-Many**: user_id in orders (one user has many orders)
- **Many-to-Many**: Usually via junction table (user_roles with user_id and role_id)
- **One-to-One**: Rare, usually same primary key in both tables

Please help me map out the relationships in my database.`

	default:
		return `I need help with database operations using WhoDB.

## Available Workflows

1. **Schema Exploration** - Discover tables, columns, and relationships
2. **Data Analysis** - Aggregate, summarize, and understand data
3. **Query Debugging** - Troubleshoot slow or incorrect queries
4. **Relationship Mapping** - Understand how tables connect

## General Best Practices

- Always start with whodb_connections to see available databases
- Use whodb_schemas → whodb_tables → whodb_columns before querying
- Always use LIMIT for exploratory queries
- Check column types and relationships before writing JOINs

Please guide me through the appropriate workflow for my task.`
	}
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
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
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

Available prompts (for guidance):
- query_help: Get SQL query writing guidance (supports database_type and query_type args)
- schema_exploration_help: Learn how to explore database structure effectively
- workflow_help: Get guidance on common database workflows (analysis, debugging, relationships)

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
