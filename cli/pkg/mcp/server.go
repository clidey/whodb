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
	"strings"
	"time"

	"github.com/clidey/whodb/cli/internal/agentmanifest"
	"github.com/clidey/whodb/cli/pkg/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
	// AllowWrite permits write operations without confirmation.
	// Default: false
	AllowWrite bool
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
	// Valid values: "query", "schemas", "tables", "columns", "connections", "confirm",
	// "pending", "explain", "diff", "erd", "audit", "suggestions"
	EnabledTools []string
	// DisabledTools specifies which tools to disable. Takes precedence over EnabledTools.
	// Valid values: "query", "schemas", "tables", "columns", "connections", "confirm",
	// "pending", "explain", "diff", "erd", "audit", "suggestions"
	DisabledTools []string
	// DefaultConnection is the connection to use when none is specified.
	// This simplifies AI interaction when working with a single database.
	DefaultConnection string
	// AllowedConnections restricts which connections can be used.
	// When set, only these connections are visible and accessible.
	// If DefaultConnection is not set, the first allowed connection becomes the default.
	AllowedConnections []string
	// PlatformEnabled runs hosted WhoDB platform mode.
	// When enabled, only hosted platform tools are registered.
	PlatformEnabled bool
}

// SecurityOptions contains runtime security settings for query execution
type SecurityOptions struct {
	ReadOnly            bool
	ConfirmWrites       bool
	AllowWrite          bool
	SecurityLevel       SecurityLevel
	QueryTimeout        time.Duration
	MaxRows             int
	AllowMultiStatement bool
	AllowDrop           bool
	DefaultConnection   string   // Injected connection when not specified
	AllowedConnections  []string // If set, only these connections are accessible
}

// mapWhoDBLogLevel maps WHODB_LOG_LEVEL to slog.Level for the MCP server's default logger.
func mapWhoDBLogLevel() slog.Level {
	switch strings.ToLower(os.Getenv("WHODB_LOG_LEVEL")) {
	case "debug":
		return slog.LevelDebug
	case "warning", "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "none", "off", "disabled":
		// slog doesn't have "off" — use a level higher than any real level
		return slog.Level(12)
	default:
		return slog.LevelInfo
	}
}

// NewServer creates a new WhoDB MCP server with all tools registered.
func NewServer(opts *ServerOptions) *mcp.Server {
	if opts == nil {
		opts = &ServerOptions{ConfirmWrites: true} // Safe default
	}

	// Fill in zero-value defaults
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: mapWhoDBLogLevel(),
		}))
	}
	if !opts.ReadOnly && !opts.ConfirmWrites && !opts.AllowWrite {
		opts.ConfirmWrites = true
	}
	if opts.Instructions == "" && opts.PlatformEnabled {
		opts.Instructions = platformInstructions
	} else if opts.Instructions == "" {
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
		Version: version.Version,
	}, &mcp.ServerOptions{
		Instructions: opts.Instructions,
		Logger:       opts.Logger,
	})

	// Determine default connection: explicit flag takes precedence, then first allowed connection
	defaultConn := opts.DefaultConnection
	if defaultConn == "" && len(opts.AllowedConnections) > 0 {
		defaultConn = opts.AllowedConnections[0]
	}

	// Create security options from server options
	secOpts := &SecurityOptions{
		ReadOnly:            opts.ReadOnly,
		ConfirmWrites:       opts.ConfirmWrites,
		AllowWrite:          opts.AllowWrite,
		SecurityLevel:       opts.SecurityLevel,
		QueryTimeout:        opts.QueryTimeout,
		MaxRows:             opts.MaxRows,
		AllowMultiStatement: opts.AllowMultiStatement,
		AllowDrop:           opts.AllowDrop,
		DefaultConnection:   defaultConn,
		AllowedConnections:  opts.AllowedConnections,
	}

	// Create tool enablement from server options
	toolEnablement := &ToolEnablement{
		EnabledTools:  opts.EnabledTools,
		DisabledTools: opts.DisabledTools,
	}

	if opts.PlatformEnabled {
		registerPlatformTools(server, secOpts)
		registerPlatformPrompts(server)
		registerPlatformResources(server, secOpts)
		return server
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

// isConnectionAllowed checks if a connection name is in the allowed list.
// Returns true if AllowedConnections is empty (no restrictions) or if the connection is in the list.
func (so *SecurityOptions) isConnectionAllowed(connection string) bool {
	if len(so.AllowedConnections) == 0 {
		return true // No restrictions
	}
	for _, allowed := range so.AllowedConnections {
		if allowed == connection {
			return true
		}
	}
	return false
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
				ReadOnlyHint:    secOpts.ReadOnly,                                     // If server is read-only, this tool is read-only
				DestructiveHint: boolPtr(!secOpts.ReadOnly && !secOpts.ConfirmWrites), // Destructive only if writes allowed without confirmation
				IdempotentHint:  false,                                                // Queries can have side effects
				OpenWorldHint:   boolPtr(false),                                       // Closed world (database only)
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
		}, createSchemasHandler(secOpts))
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
		}, createTablesHandler(secOpts))
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
		}, createColumnsHandler(secOpts))
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
		}, createConnectionsHandler(secOpts))
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

	// whodb_pending - List pending confirmations (only registered if confirm-writes is enabled)
	// Hints: Read-only, idempotent, closed world
	if secOpts.ConfirmWrites && toolEnablement.isToolEnabled("pending") {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "whodb_pending",
			Description: descPending,
			Annotations: &mcp.ToolAnnotations{
				Title:          "List Pending Confirmations",
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  boolPtr(false),
			},
		}, createPendingHandler(secOpts))
	}

	// whodb_explain - Run EXPLAIN for a SQL query.
	if toolEnablement.isToolEnabled("explain") {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "whodb_explain",
			Description: descExplain,
			Annotations: &mcp.ToolAnnotations{
				Title:          "Explain SQL Query",
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  boolPtr(false),
			},
		}, createExplainHandler(secOpts))
	}

	// whodb_diff - Compare schema metadata between two connections.
	if toolEnablement.isToolEnabled("diff") {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "whodb_diff",
			Description: descDiff,
			Annotations: &mcp.ToolAnnotations{
				Title:          "Compare Schemas",
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  boolPtr(false),
			},
		}, createSchemaDiffHandler(secOpts))
	}

	// whodb_erd - Load backend graph metadata.
	if toolEnablement.isToolEnabled("erd") {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "whodb_erd",
			Description: descERD,
			Annotations: &mcp.ToolAnnotations{
				Title:          "Load ER Diagram Metadata",
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  boolPtr(false),
			},
		}, createERDHandler(secOpts))
	}

	// whodb_audit - Run data quality audits.
	if toolEnablement.isToolEnabled("audit") {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "whodb_audit",
			Description: descAudit,
			Annotations: &mcp.ToolAnnotations{
				Title:          "Run Data Quality Audit",
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  boolPtr(false),
			},
		}, createAuditHandler(secOpts))
	}

	// whodb_suggestions - Load backend-generated query suggestions.
	if toolEnablement.isToolEnabled("suggestions") {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "whodb_suggestions",
			Description: descSuggestions,
			Annotations: &mcp.ToolAnnotations{
				Title:          "Load Query Suggestions",
				ReadOnlyHint:   true,
				IdempotentHint: true,
				OpenWorldHint:  boolPtr(false),
			},
		}, createSuggestionsHandler(secOpts))
	}
}

// buildQueryDescription creates the tool description based on security settings
func buildQueryDescription(secOpts *SecurityOptions) string {
	base := `Execute a SQL query against a database connection.

**Best for:** Running SQL SELECT, INSERT, UPDATE, DELETE statements when you need to query or modify data.
**Not recommended for:** Schema exploration (use whodb_schemas, whodb_tables, whodb_columns instead for faster, structured results).
**Common mistakes:** Running queries without specifying connection when multiple exist; using SELECT * instead of specific columns; forgetting LIMIT on large tables.

**Usage Example (simple query):**
` + "```json" + `
{
  "name": "whodb_query",
  "arguments": {
    "connection": "mydb",
    "query": "SELECT id, name, email FROM users WHERE active = true LIMIT 10"
  }
}
` + "```" + `

**Usage Example (parameterized query - RECOMMENDED for user input):**
` + "```json" + `
{
  "name": "whodb_query",
  "arguments": {
    "connection": "mydb",
    "query": "SELECT * FROM users WHERE id = $1 AND status = $2",
    "parameters": [123, "active"]
  }
}
` + "```" + `
**Placeholder syntax by database:** PostgreSQL uses $1, $2, $3; MySQL/SQLite/DuckDB/ClickHouse use ?

**Best practices:**
- **Use parameterized queries** when incorporating user-provided values - this prevents SQL injection
- Always use LIMIT for exploratory queries to avoid overwhelming results
- Prefer specific column selection over SELECT *
- Check schema structure with whodb_columns before writing complex queries`

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

// createPendingHandler creates a handler for listing pending confirmations
func createPendingHandler(secOpts *SecurityOptions) func(ctx context.Context, req *mcp.CallToolRequest, input PendingInput) (*mcp.CallToolResult, PendingOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input PendingInput) (*mcp.CallToolResult, PendingOutput, error) {
		return HandlePending(ctx, req, input, secOpts)
	}
}

// createSchemasHandler creates a schemas handler with connection injection and validation
func createSchemasHandler(secOpts *SecurityOptions) func(ctx context.Context, req *mcp.CallToolRequest, input SchemasInput) (*mcp.CallToolResult, SchemasOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input SchemasInput) (*mcp.CallToolResult, SchemasOutput, error) {
		// Inject default connection if not specified
		if input.Connection == "" && secOpts.DefaultConnection != "" {
			input.Connection = secOpts.DefaultConnection
		}
		// Validate connection is allowed
		if !secOpts.isConnectionAllowed(input.Connection) {
			return nil, SchemasOutput{Error: fmt.Sprintf("connection %q is not allowed", input.Connection)}, nil
		}
		return HandleSchemas(ctx, req, input)
	}
}

// createTablesHandler creates a tables handler with connection injection and validation
func createTablesHandler(secOpts *SecurityOptions) func(ctx context.Context, req *mcp.CallToolRequest, input TablesInput) (*mcp.CallToolResult, TablesOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input TablesInput) (*mcp.CallToolResult, TablesOutput, error) {
		// Inject default connection if not specified
		if input.Connection == "" && secOpts.DefaultConnection != "" {
			input.Connection = secOpts.DefaultConnection
		}
		// Validate connection is allowed
		if !secOpts.isConnectionAllowed(input.Connection) {
			return nil, TablesOutput{Error: fmt.Sprintf("connection %q is not allowed", input.Connection)}, nil
		}
		return HandleTables(ctx, req, input)
	}
}

// createColumnsHandler creates a columns handler with connection injection and validation
func createColumnsHandler(secOpts *SecurityOptions) func(ctx context.Context, req *mcp.CallToolRequest, input ColumnsInput) (*mcp.CallToolResult, ColumnsOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ColumnsInput) (*mcp.CallToolResult, ColumnsOutput, error) {
		// Inject default connection if not specified
		if input.Connection == "" && secOpts.DefaultConnection != "" {
			input.Connection = secOpts.DefaultConnection
		}
		// Validate connection is allowed
		if !secOpts.isConnectionAllowed(input.Connection) {
			return nil, ColumnsOutput{Error: fmt.Sprintf("connection %q is not allowed", input.Connection)}, nil
		}
		return HandleColumns(ctx, req, input)
	}
}

// createConnectionsHandler creates a connections handler that filters by allowed connections
func createConnectionsHandler(secOpts *SecurityOptions) func(ctx context.Context, req *mcp.CallToolRequest, input ConnectionsInput) (*mcp.CallToolResult, ConnectionsOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ConnectionsInput) (*mcp.CallToolResult, ConnectionsOutput, error) {
		result, output, err := HandleConnections(ctx, req, input)
		if err != nil || output.Error != "" {
			return result, output, err
		}
		// Filter connections if AllowedConnections is set
		if len(secOpts.AllowedConnections) > 0 {
			filtered := make([]ConnectionInfo, 0)
			for _, conn := range output.Connections {
				if secOpts.isConnectionAllowed(conn.Name) {
					filtered = append(filtered, conn)
				}
			}
			output.Connections = filtered
		}
		return result, output, err
	}
}

// createExplainHandler creates an explain handler with connection injection and validation.
func createExplainHandler(secOpts *SecurityOptions) func(ctx context.Context, req *mcp.CallToolRequest, input ExplainInput) (*mcp.CallToolResult, ExplainOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ExplainInput) (*mcp.CallToolResult, ExplainOutput, error) {
		if input.Connection == "" && secOpts.DefaultConnection != "" {
			input.Connection = secOpts.DefaultConnection
		}
		if !secOpts.isConnectionAllowed(input.Connection) {
			return nil, ExplainOutput{Error: fmt.Sprintf("connection %q is not allowed", input.Connection)}, nil
		}
		return HandleExplain(ctx, req, input)
	}
}

// createSchemaDiffHandler creates a diff handler with dual-connection validation.
func createSchemaDiffHandler(secOpts *SecurityOptions) func(ctx context.Context, req *mcp.CallToolRequest, input SchemaDiffInput) (*mcp.CallToolResult, SchemaDiffOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input SchemaDiffInput) (*mcp.CallToolResult, SchemaDiffOutput, error) {
		if input.FromConnection == "" && secOpts.DefaultConnection != "" {
			input.FromConnection = secOpts.DefaultConnection
		}
		if input.ToConnection == "" && secOpts.DefaultConnection != "" {
			input.ToConnection = secOpts.DefaultConnection
		}
		if !secOpts.isConnectionAllowed(input.FromConnection) {
			return nil, SchemaDiffOutput{Error: fmt.Sprintf("connection %q is not allowed", input.FromConnection)}, nil
		}
		if !secOpts.isConnectionAllowed(input.ToConnection) {
			return nil, SchemaDiffOutput{Error: fmt.Sprintf("connection %q is not allowed", input.ToConnection)}, nil
		}
		return HandleSchemaDiff(ctx, req, input)
	}
}

// createERDHandler creates an ERD handler with connection injection and validation.
func createERDHandler(secOpts *SecurityOptions) func(ctx context.Context, req *mcp.CallToolRequest, input ERDInput) (*mcp.CallToolResult, ERDOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ERDInput) (*mcp.CallToolResult, ERDOutput, error) {
		if input.Connection == "" && secOpts.DefaultConnection != "" {
			input.Connection = secOpts.DefaultConnection
		}
		if !secOpts.isConnectionAllowed(input.Connection) {
			return nil, ERDOutput{Error: fmt.Sprintf("connection %q is not allowed", input.Connection)}, nil
		}
		return HandleERD(ctx, req, input)
	}
}

// createAuditHandler creates an audit handler with connection injection and validation.
func createAuditHandler(secOpts *SecurityOptions) func(ctx context.Context, req *mcp.CallToolRequest, input AuditInput) (*mcp.CallToolResult, AuditOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input AuditInput) (*mcp.CallToolResult, AuditOutput, error) {
		if input.Connection == "" && secOpts.DefaultConnection != "" {
			input.Connection = secOpts.DefaultConnection
		}
		if !secOpts.isConnectionAllowed(input.Connection) {
			return nil, AuditOutput{Error: fmt.Sprintf("connection %q is not allowed", input.Connection)}, nil
		}
		return HandleAudit(ctx, req, input)
	}
}

// createSuggestionsHandler creates a suggestions handler with connection injection and validation.
func createSuggestionsHandler(secOpts *SecurityOptions) func(ctx context.Context, req *mcp.CallToolRequest, input SuggestionsInput) (*mcp.CallToolResult, SuggestionsOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input SuggestionsInput) (*mcp.CallToolResult, SuggestionsOutput, error) {
		if input.Connection == "" && secOpts.DefaultConnection != "" {
			input.Connection = secOpts.DefaultConnection
		}
		if !secOpts.isConnectionAllowed(input.Connection) {
			return nil, SuggestionsOutput{Error: fmt.Sprintf("connection %q is not allowed", input.Connection)}, nil
		}
		return HandleSuggestions(ctx, req, input)
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

	server.AddResource(&mcp.Resource{
		Name:        "agent-schema",
		URI:         "whodb://agent/schema",
		Description: "Machine-readable WhoDB agent capability manifest",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		data, err := json.MarshalIndent(agentmanifest.Build(), "", "  ")
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{Text: string(data)},
			},
		}, nil
	})
}

func registerPlatformResources(server *mcp.Server, secOpts *SecurityOptions) {
	server.AddResource(&mcp.Resource{
		Name:        "platform-schema",
		URI:         "whodb://platform/schema",
		Description: "Machine-readable hosted WhoDB platform MCP contract and enabled platform tools",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		manifest := agentmanifest.Build()
		tools := make([]agentmanifest.MCPTool, 0)
		for _, tool := range manifest.MCPTools {
			if strings.HasPrefix(tool.Name, manifest.PlatformMCP.ToolPrefix) && platformToolEnabledForMode(tool.Name, secOpts) {
				tools = append(tools, tool)
			}
		}
		return jsonResource("whodb://platform/schema", platformSchemaResource{
			Name:        manifest.Name,
			Version:     manifest.Version,
			PlatformMCP: manifest.PlatformMCP,
			Tools:       tools,
			Resources: []string{
				"whodb://platform/schema",
				"whodb://platform/workspace",
				"whodb://platform/tool-guide",
			},
			Prompts: []string{
				"whodb_platform_overview",
				"whodb_platform_read_workflow",
				"whodb_platform_write_safety",
				"whodb_platform_source_workflow",
			},
		})
	})

	server.AddResource(&mcp.Resource{
		Name:        "platform-workspace",
		URI:         "whodb://platform/workspace",
		Description: "Current hosted WhoDB login and selected workspace metadata",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		_, output, err := HandlePlatformStatus(ctx, nil, PlatformStatusInput{})
		if err != nil {
			return nil, err
		}
		return jsonResource("whodb://platform/workspace", output)
	})

	server.AddResource(&mcp.Resource{
		Name:        "platform-tool-guide",
		URI:         "whodb://platform/tool-guide",
		Description: "Hosted WhoDB platform MCP tool categories, read/write behavior, and field projection guidance",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return jsonResource("whodb://platform/tool-guide", buildPlatformToolGuide(secOpts))
	})
}

type platformSchemaResource struct {
	Name        string                    `json:"name"`
	Version     string                    `json:"version"`
	PlatformMCP agentmanifest.PlatformMCP `json:"platform_mcp"`
	Tools       []agentmanifest.MCPTool   `json:"tools"`
	Resources   []string                  `json:"resources"`
	Prompts     []string                  `json:"prompts"`
}

type platformToolGuideResource struct {
	Mode              string                      `json:"mode"`
	FieldProjection   string                      `json:"field_projection"`
	WriteBehavior     string                      `json:"write_behavior"`
	PermissionModel   string                      `json:"permission_model"`
	WorkspaceBehavior string                      `json:"workspace_behavior"`
	Categories        []platformToolGuideCategory `json:"categories"`
}

type platformToolGuideCategory struct {
	Name            string                      `json:"name"`
	Description     string                      `json:"description"`
	RecommendedUse  string                      `json:"recommended_use"`
	DefaultFields   []string                    `json:"default_fields,omitempty"`
	Tools           []platformToolGuideTool     `json:"tools"`
	SupportedWrites []platformToolGuideMutation `json:"supported_writes,omitempty"`
}

type platformToolGuideTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ReadOnly    bool   `json:"read_only"`
}

type platformToolGuideMutation struct {
	Tool      string   `json:"tool"`
	Resources []string `json:"resources,omitempty"`
	Actions   []string `json:"actions,omitempty"`
}

func jsonResource(uri string, value any) (*mcp.ReadResourceResult, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{URI: uri, MIMEType: "application/json", Text: string(data)},
		},
	}, nil
}

func buildPlatformToolGuide(secOpts *SecurityOptions) platformToolGuideResource {
	toolByName := map[string]platformToolGuideTool{}
	for _, tool := range platformToolDefinitions() {
		if !platformToolEnabledForMode(tool.Name, secOpts) {
			continue
		}
		readOnly := true
		if tool.Annotations != nil {
			readOnly = tool.Annotations.ReadOnlyHint
		}
		toolByName[tool.Name] = platformToolGuideTool{
			Name:        tool.Name,
			Description: tool.Description,
			ReadOnly:    readOnly,
		}
	}

	categories := []platformToolGuideCategory{
		platformToolCategory(toolByName, "workspace", "Login, organization, and project discovery.", "Start here before any project-scoped read or write.", nil, nil,
			"whodb_platform_status", "whodb_platform_orgs", "whodb_platform_projects"),
		platformToolCategory(toolByName, "sources", "Hosted source discovery, connection metadata, data previews, and source writes.", "List sources with id/name/type first; inspect config only when needed because secrets are redacted.", []string{"id", "name", "type"}, []platformToolGuideMutation{
			{Tool: "whodb_platform_source_create", Resources: []string{"source"}},
			{Tool: "whodb_platform_source_update", Resources: []string{"source"}},
			{Tool: "whodb_platform_source_delete", Resources: []string{"source"}},
		},
			"whodb_platform_sources", "whodb_platform_source_types", "whodb_platform_source_fields", "whodb_platform_source_objects", "whodb_platform_source_columns", "whodb_platform_source_rows", "whodb_platform_source_constraints", "whodb_platform_source_content", "whodb_platform_source_config", "whodb_platform_source_test", "whodb_platform_source_create", "whodb_platform_source_update", "whodb_platform_source_delete"),
		platformToolCategory(toolByName, "secrets", "Secret metadata only. Secret values are not returned.", "Use for secret names, ids, provider references, and configuration shape; never expect secret values.", []string{"id", "name", "type"}, []platformToolGuideMutation{
			{Tool: "whodb_platform_create", Resources: []string{"secret"}},
			{Tool: "whodb_platform_update", Resources: []string{"secret"}},
			{Tool: "whodb_platform_delete", Resources: []string{"secret"}},
		},
			"whodb_platform_secrets"),
		platformToolCategory(toolByName, "ai_providers", "AI provider metadata and provider model discovery. API keys are not returned.", "Use provider lists before selecting models or making provider changes.", []string{"id", "name", "providerType"}, []platformToolGuideMutation{
			{Tool: "whodb_platform_create", Resources: []string{"ai_provider"}},
			{Tool: "whodb_platform_delete", Resources: []string{"ai_provider"}},
		},
			"whodb_platform_ai_providers", "whodb_platform_ai_provider_models"),
		platformToolCategory(toolByName, "ontology", "Ontology object types, fast lookups, rows, and linked records.", "List ontologies first, then inspect one ontology or preview rows only when needed.", []string{"id", "apiName", "displayName"}, []platformToolGuideMutation{
			{Tool: "whodb_platform_create", Resources: []string{"ontology", "ontology_fast_lookup"}},
			{Tool: "whodb_platform_update", Resources: []string{"ontology"}},
			{Tool: "whodb_platform_delete", Resources: []string{"ontology", "ontology_fast_lookup"}},
		},
			"whodb_platform_ontologies", "whodb_platform_ontology", "whodb_platform_ontology_fast_lookups", "whodb_platform_ontology_fast_lookup_suggestions", "whodb_platform_ontology_rows", "whodb_platform_ontology_follow_link"),
		platformToolCategory(toolByName, "datasets", "Dataset metadata and dataset row previews.", "List datasets first; preview rows only when needed for the user request.", []string{"id", "name"}, []platformToolGuideMutation{
			{Tool: "whodb_platform_create", Resources: []string{"dataset"}},
			{Tool: "whodb_platform_update", Resources: []string{"dataset"}},
			{Tool: "whodb_platform_delete", Resources: []string{"dataset"}},
		},
			"whodb_platform_datasets", "whodb_platform_dataset", "whodb_platform_dataset_rows"),
		platformToolCategory(toolByName, "lineage", "Project and node lineage graph inspection.", "Use project lineage for broad context and neighbor/root lineage for focused graph expansion.", nil, nil,
			"whodb_platform_lineage", "whodb_platform_lineage_neighbors", "whodb_platform_project_lineage"),
		platformToolCategory(toolByName, "transforms", "Transform metadata, runs, and run actions.", "List transforms before run history; run transforms only after explicit user approval.", []string{"id", "name"}, []platformToolGuideMutation{
			{Tool: "whodb_platform_create", Resources: []string{"transform"}},
			{Tool: "whodb_platform_update", Resources: []string{"transform"}},
			{Tool: "whodb_platform_delete", Resources: []string{"transform"}},
			{Tool: "whodb_platform_action", Resources: []string{"transform"}, Actions: []string{"run"}},
		},
			"whodb_platform_transforms", "whodb_platform_transform_runs"),
		platformToolCategory(toolByName, "functions", "Function metadata, function files, deploy and redeploy actions.", "List functions with narrow fields; request files/content only when needed.", []string{"id", "name"}, []platformToolGuideMutation{
			{Tool: "whodb_platform_create", Resources: []string{"function"}},
			{Tool: "whodb_platform_update", Resources: []string{"function"}},
			{Tool: "whodb_platform_delete", Resources: []string{"function"}},
			{Tool: "whodb_platform_action", Resources: []string{"function"}, Actions: []string{"deploy", "redeploy"}},
		},
			"whodb_platform_functions", "whodb_platform_function"),
		platformToolCategory(toolByName, "files", "Project file browsing, previews, search, tabular file discovery, and storage usage.", "Search or list files first; preview file contents only when required.", []string{"id", "name", "isTabular"}, []platformToolGuideMutation{
			{Tool: "whodb_platform_create", Resources: []string{"folder"}},
			{Tool: "whodb_platform_delete", Resources: []string{"file", "folder"}},
			{Tool: "whodb_platform_action", Resources: []string{"file", "folder"}, Actions: []string{"upload", "rename", "move", "promote_to_dataset"}},
		},
			"whodb_platform_files", "whodb_platform_file_preview", "whodb_platform_file_search", "whodb_platform_tabular_files", "whodb_platform_storage_usage"),
		platformToolCategory(toolByName, "generic_writes", "Generic hosted create, update, delete, and action tools for supported platform resources.", "Use only after reading current state and obtaining user approval for the exact confirmation preview.", nil, []platformToolGuideMutation{
			{Tool: "whodb_platform_create", Resources: []string{"secret", "ai_provider", "ontology", "ontology_fast_lookup", "dataset", "transform", "folder", "function", "source_object"}},
			{Tool: "whodb_platform_update", Resources: []string{"secret", "ontology", "dataset", "transform", "function", "source_object"}},
			{Tool: "whodb_platform_delete", Resources: []string{"secret", "ai_provider", "ontology", "ontology_fast_lookup", "dataset", "transform", "file", "folder", "function", "source_object"}},
			{Tool: "whodb_platform_action", Resources: []string{"transform", "file", "folder", "function"}, Actions: []string{"run", "upload", "rename", "move", "promote_to_dataset", "deploy", "redeploy"}},
		},
			"whodb_platform_create", "whodb_platform_update", "whodb_platform_delete", "whodb_platform_action", "whodb_platform_pending", "whodb_platform_confirm"),
	}

	return platformToolGuideResource{
		Mode:              platformResourceMode(secOpts),
		FieldProjection:   "Use fields on supported read tools to request only the top-level fields needed, then call again with more fields if needed.",
		WriteBehavior:     platformResourceWriteBehavior(secOpts),
		PermissionModel:   "The hosted platform is authoritative for permissions. Workspace selection only scopes requests after token-backed authorization.",
		WorkspaceBehavior: "Use whodb_platform_status first. If no workspace is selected, use whodb_platform_orgs and whodb_platform_projects, then run whodb-cli use --org <org> --project <project>.",
		Categories:        filterPlatformToolGuideCategories(categories),
	}
}

func filterPlatformToolGuideCategories(categories []platformToolGuideCategory) []platformToolGuideCategory {
	result := make([]platformToolGuideCategory, 0, len(categories))
	for _, category := range categories {
		if len(category.Tools) > 0 || len(category.SupportedWrites) > 0 {
			result = append(result, category)
		}
	}
	return result
}

func platformToolCategory(toolByName map[string]platformToolGuideTool, name, description, recommendedUse string, defaultFields []string, writes []platformToolGuideMutation, toolNames ...string) platformToolGuideCategory {
	tools := make([]platformToolGuideTool, 0, len(toolNames))
	for _, toolName := range toolNames {
		if tool, ok := toolByName[toolName]; ok {
			tools = append(tools, tool)
		}
	}
	return platformToolGuideCategory{
		Name:            name,
		Description:     description,
		RecommendedUse:  recommendedUse,
		DefaultFields:   defaultFields,
		Tools:           tools,
		SupportedWrites: filterPlatformGuideWrites(toolByName, writes),
	}
}

func filterPlatformGuideWrites(toolByName map[string]platformToolGuideTool, writes []platformToolGuideMutation) []platformToolGuideMutation {
	result := make([]platformToolGuideMutation, 0, len(writes))
	for _, write := range writes {
		if _, ok := toolByName[write.Tool]; ok {
			result = append(result, write)
		}
	}
	return result
}

func platformToolEnabledForMode(name string, secOpts *SecurityOptions) bool {
	if secOpts == nil {
		secOpts = &SecurityOptions{ConfirmWrites: true}
	}
	switch name {
	case "whodb_platform_source_create", "whodb_platform_source_update", "whodb_platform_source_delete",
		"whodb_platform_create", "whodb_platform_update", "whodb_platform_delete", "whodb_platform_action":
		return !secOpts.ReadOnly
	case "whodb_platform_pending", "whodb_platform_confirm":
		return secOpts.ConfirmWrites
	default:
		return true
	}
}

func platformResourceMode(secOpts *SecurityOptions) string {
	switch {
	case secOpts != nil && secOpts.ReadOnly:
		return "read_only"
	case secOpts != nil && secOpts.AllowWrite:
		return "allow_write"
	case secOpts != nil && secOpts.ConfirmWrites:
		return "confirm_writes"
	default:
		return "confirm_writes"
	}
}

func platformResourceWriteBehavior(secOpts *SecurityOptions) string {
	switch platformResourceMode(secOpts) {
	case "read_only":
		return "Write tools are hidden."
	case "allow_write":
		return "Write tools execute immediately. Ask the user for confirmation before calling a mutating tool."
	default:
		return "Write tools return confirmation_required, confirmation_token, confirmation_preview, and confirmation_expiry. Call whodb_platform_confirm only after explicit user approval."
	}
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
				Description: "The type of database (postgres, mysql, sqlite, duckdb, etc.) for dialect-specific help.",
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

func registerPlatformPrompts(server *mcp.Server) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "whodb_platform_overview",
		Title:       "WhoDB Platform Overview",
		Description: "Understand hosted WhoDB platform MCP mode, workspace selection, permissions, and field projection.",
	}, handlePlatformOverviewPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "whodb_platform_read_workflow",
		Title:       "WhoDB Platform Read Workflow",
		Description: "Use hosted WhoDB platform read tools safely and efficiently.",
	}, handlePlatformReadWorkflowPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "whodb_platform_write_safety",
		Title:       "WhoDB Platform Write Safety",
		Description: "Handle hosted WhoDB platform write confirmations and destructive actions safely.",
	}, handlePlatformWriteSafetyPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "whodb_platform_source_workflow",
		Title:       "WhoDB Platform Source Workflow",
		Description: "Manage hosted WhoDB platform sources from discovery through create, update, and delete.",
	}, handlePlatformSourceWorkflowPrompt)
}

func platformPromptResult(description, content string) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: description,
		Messages: []*mcp.PromptMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: content},
			},
		},
	}, nil
}

func handlePlatformOverviewPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	const content = `You are using WhoDB hosted platform MCP mode.

Only whodb_platform_* tools are available. Local database tools such as whodb_query and whodb_connections are not exposed in this mode.

Use the active whodb-cli hosted login and selected workspace. Start with whodb_platform_status to confirm host, signed-in user, organization, and project. If no workspace is selected, call whodb_platform_orgs and whodb_platform_projects, then ask the user to run whodb-cli use --org <org> --project <project>.

Backend permissions are authoritative. The CLI may select a workspace, but the hosted platform decides what the signed-in user can read or change.

For read tools that accept fields, request only fields needed for the current answer, then request more fields only if needed.`

	return platformPromptResult("WhoDB hosted platform MCP mode guidance", content)
}

func handlePlatformReadWorkflowPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	const content = `Use this read workflow for WhoDB hosted platform data.

1. Call whodb_platform_status.
2. If needed, discover workspace with whodb_platform_orgs and whodb_platform_projects.
3. Use narrow list tools first:
   - whodb_platform_sources for sources
   - whodb_platform_datasets for datasets
   - whodb_platform_ontologies for ontology types
   - whodb_platform_transforms for transforms
   - whodb_platform_functions for functions
   - whodb_platform_files for files
   - whodb_platform_ai_providers for AI provider metadata
   - whodb_platform_secrets for secret metadata only
4. Use detail tools only after selecting a specific id/name.
5. Use fields to request only what you need. Avoid file contents, function files, source content, and row previews unless the user asks for them or they are required.
6. Never treat absence of returned data as proof the resource does not exist unless the hosted platform returned a successful response for the selected workspace.`

	return platformPromptResult("WhoDB hosted platform read workflow", content)
}

func handlePlatformWriteSafetyPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	const content = `Use this safety workflow before hosted platform writes.

Hosted create, update, delete, and action tools are mutating operations. In default mode, they return confirmation_required, confirmation_token, confirmation_preview, and confirmation_expiry. They do not execute until whodb_platform_confirm is called.

Before confirming:
1. Explain the confirmation_preview to the user.
2. Call out destructive or high-impact actions explicitly, including delete, move, deploy, redeploy, run, upload, source changes, source object changes, secret changes, and AI provider changes.
3. Do not call whodb_platform_confirm unless the user clearly approves that exact preview.
4. If the user changes their mind or asks for edits, rerun the write tool to produce a new confirmation token.
5. Use whodb_platform_pending only to recover pending tokens; do not use it as approval.

In --read-only or --safe-mode, hosted write tools are hidden. In --allow-write, writes execute immediately, so ask for user confirmation before calling the mutating tool itself.`

	return platformPromptResult("WhoDB hosted platform write safety workflow", content)
}

func handlePlatformSourceWorkflowPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	const content = `Use this workflow for hosted source management.

For discovery:
1. Call whodb_platform_sources with fields such as ["id", "name", "type"].
2. Use whodb_platform_source_config only when you need saved connection shape; secrets are redacted.
3. Use whodb_platform_source_objects, whodb_platform_source_columns, and whodb_platform_source_rows to inspect data.

For creating a source:
1. Call whodb_platform_source_types.
2. Call whodb_platform_source_fields for the chosen source_type.
3. Ask the user for required connection fields. Treat password, token, secret, key, and private-key fields as sensitive.
4. Optionally call whodb_platform_source_test with draft config.
5. Call whodb_platform_source_create.
6. Explain the confirmation_preview and call whodb_platform_confirm only after approval.

For updating a source:
1. Call whodb_platform_sources to select the source.
2. Call whodb_platform_source_config if you need to preserve existing redacted fields.
3. Send only changed fields to whodb_platform_source_update.
4. Confirm only after user approval.

For deleting a source:
1. Call whodb_platform_sources to verify id/name.
2. Call whodb_platform_source_delete.
3. Explain that deletion is destructive and confirm only after user approval.`

	return platformPromptResult("WhoDB hosted platform source workflow", content)
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
2. **Get tables with columns** - Use whodb_tables(include_columns=true) to get all tables AND their column details in one call
3. **Query data** - Use whodb_query for actual data exploration (results include column_types)

For multi-schema databases, use whodb_schemas(include_tables=true) first to see all schemas and their tables.

## Tips for Effective Exploration

- Use include_columns=true and include_tables=true to minimize round-trips
- Query results include column_types alongside column names — no separate whodb_columns call needed
- Always check foreign key relationships in column details
- Look for naming patterns (e.g., *_id columns often indicate relationships)
- Check for audit columns (created_at, updated_at) to understand data lifecycle

## Example Exploration Session

` + "```" + `
# Step 1: What databases are available?
whodb_connections → [{name: "mydb", type: "postgres", ...}]

# Step 2: Get all tables and their columns in one call
whodb_tables(connection="mydb", include_columns=true) → [
  {name: "users", columns: [{name: "id", type: "integer", is_primary: true}, ...]},
  {name: "orders", columns: [{name: "user_id", type: "integer", is_foreign_key: true, referenced_table: "users"}, ...]}
]

# Step 3: Query with full type awareness
whodb_query(connection="mydb", query="SELECT * FROM users LIMIT 5")
→ {columns: ["id", "name"], column_types: ["integer", "varchar"], rows: [...]}
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
		case "duckdb":
			base += `- DuckDB uses ? for query placeholders
- Use information_schema for metadata queries
- DuckDB supports INTERVAL, HUGEINT, and JSON types natively
- Use duckdb_constraints() for constraint introspection
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
- Use whodb_tables(include_columns=true) to get tables and columns in one call
- Query results include column_types — no separate whodb_columns call needed
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
**Typical workflow:** whodb_schemas → whodb_tables(include_columns=true) → whodb_query

**Optional parameter:** Set "include_tables": true to also return all tables within each schema in a single call. This populates a "details" array with schema names and their tables, saving you a separate whodb_tables call per schema.`

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
**Note:** If schema is omitted, uses the connection's default schema or the first available schema.

**Optional parameter:** Set "include_columns": true to also return column details (name, type, primary key, foreign keys) for each table. This saves you separate whodb_columns calls and gives you everything needed to write queries in a single round-trip.`

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
- type: Database type (postgres, mysql, sqlite, duckdb, etc.)
- host/port/database: Connection details (passwords are never exposed)
- source: "saved" (from CLI config) or "env" (from environment variables)

**Note:** If only one connection exists, other tools will use it automatically when connection is omitted.`

const descConfirm = `Confirm and execute a pending write operation.

**Best for:** Executing write queries after user approval in confirm-writes mode.
**Not recommended for:** Read queries (they execute immediately without confirmation).
**Common mistakes:** Using an expired token (tokens expire after 5 minutes); not explaining the query to the user before confirming.

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
2. Receive confirmation_required=true, a confirmation_token, and confirmation_expiry
3. Explain to the user what the query will do in plain language
4. After user approves, call whodb_confirm with the token
5. Query executes and returns results

**Token behavior:** Tokens are valid for 5 minutes (expiry time is in the response). If confirmation fails due to a connection error or timeout, you can retry with the same token — it is only consumed after successful execution. Use whodb_pending to list active tokens if you lose track.`

const descPending = `List all pending write confirmations that are waiting for approval.

**Best for:** Recovering lost confirmation tokens; checking what operations are pending.
**Not recommended for:** Anything else — this is a utility tool for the confirm-writes workflow.

**Usage Example:**
` + "```json" + `
{
  "name": "whodb_pending",
  "arguments": {}
}
` + "```" + `

**Returns:** Array of pending confirmations with token, query, connection, and expiry time.

**Important:** Tokens are single-use and expire after 60 seconds. If expired, re-submit the original query to get a new token.`

const descExplain = `Run EXPLAIN for a SQL query using the database's native explain mode.

**Best for:** Understanding query plans; checking whether a query will scan too much data before you run the real query.
**Not recommended for:** Fetching actual data (use whodb_query for that).
**Common mistakes:** Passing a non-SQL string; forgetting that EXPLAIN output is database-specific.

**Usage Example:**
` + "```json" + `
{
  "name": "whodb_explain",
  "arguments": {
    "connection": "mydb",
    "query": "SELECT * FROM users WHERE email LIKE '%@example.com' LIMIT 10"
  }
}
` + "```" + `

**Returns:** The database-native EXPLAIN output with columns and rows, ready for follow-up analysis.`

const descDiff = `Compare schema metadata between two database connections.

**Best for:** Spotting drift between environments; comparing staging vs production; reviewing storage-unit, column, and relationship changes.
**Not recommended for:** Row-level data comparison.
**Common mistakes:** Forgetting to specify both connections; comparing the same connection and schema without overrides.

**Usage Example:**
` + "```json" + `
{
  "name": "whodb_diff",
  "arguments": {
    "from_connection": "staging",
    "to_connection": "prod",
    "from_schema": "public",
    "to_schema": "public"
  }
}
` + "```" + `

**Returns:** A structured schema diff with storage-unit, column, and relationship summaries plus per-object changes.`

const descERD = `Load backend graph metadata for a schema or database.

**Best for:** Understanding how tables relate before writing joins; inspecting primary/foreign key relationships programmatically.
**Not recommended for:** Query execution.
**Common mistakes:** Expecting row data instead of metadata.

**Usage Example:**
` + "```json" + `
{
  "name": "whodb_erd",
  "arguments": {
    "connection": "mydb",
    "schema": "public"
  }
}
` + "```" + `

**Returns:** Storage units with columns plus normalized relationship edges sourced from backend graph metadata.`

const descAudit = `Run data-quality checks on one schema or table.

**Best for:** Finding null-rate spikes, missing primary keys, low-cardinality issues, duplicate rows, and orphaned foreign keys.
**Not recommended for:** Replacing a full observability or data-governance system.
**Common mistakes:** Forgetting to scope the audit to one table when you only need one table.

**Usage Example:**
` + "```json" + `
{
  "name": "whodb_audit",
  "arguments": {
    "connection": "mydb",
    "schema": "public",
    "table": "orders",
    "null_warning": 15,
    "null_error": 60
  }
}
` + "```" + `

**Returns:** Audit results per table, including issue summaries and the underlying table/column findings.`

const descSuggestions = `Load backend-generated starter queries for a schema or database.

**Best for:** Quickly orienting yourself in an unfamiliar database; suggesting first queries for exploration.
**Not recommended for:** Exhaustive SQL tutoring.
**Common mistakes:** Treating the suggestions as guaranteed-valid business logic rather than onboarding hints.

**Usage Example:**
` + "```json" + `
{
  "name": "whodb_suggestions",
  "arguments": {
    "connection": "mydb",
    "schema": "public"
  }
}
` + "```" + `

**Returns:** A short list of backend-generated query suggestions derived from the actual storage units in the resolved schema.`

const defaultInstructions = `WhoDB MCP Server - Database Management Tools

Available tools:
- whodb_query: Execute SQL queries against a database
- whodb_schemas: List database schemas/namespaces
- whodb_tables: List tables in a schema
- whodb_columns: Describe columns in a table
- whodb_connections: List available database connections
- whodb_confirm: Confirm pending write operations (enabled by default)
- whodb_pending: List pending write confirmations awaiting approval
- whodb_explain: Run EXPLAIN for a SQL query
- whodb_diff: Compare schema metadata between two connections
- whodb_erd: Load graph/relationship metadata for a schema
- whodb_audit: Run data quality checks on one schema or table
- whodb_suggestions: Load backend-generated starter queries

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
2. Get tables with columns: whodb_tables(connection="mydb", include_columns=true)
3. Query data: whodb_query(connection="mydb", query="SELECT * FROM users LIMIT 10")
   → Results include column_types alongside column names
4. Inspect a plan: whodb_explain(connection="mydb", query="SELECT ...")
5. Compare environments: whodb_diff(from_connection="staging", to_connection="prod")
6. Explore relationships: whodb_erd(connection="mydb")
7. Audit one schema: whodb_audit(connection="mydb")
8. Get starter queries: whodb_suggestions(connection="mydb")
9. Write data: whodb_query(connection="mydb", query="INSERT INTO...") -> user confirms -> whodb_confirm(token="...")
   → Tokens are valid for 5 minutes and retryable. Use whodb_pending to recover lost tokens.

Best practices:
- Send ONE query at a time (multi-statement queries are blocked for security)
- Always use LIMIT for exploratory queries
- Check schema structure before writing queries
- Prefer specific column selection over SELECT *
- For writes, explain to the user what will happen before proposing the query
- Use parameterized queries when incorporating user input: whodb_query(query="SELECT * FROM users WHERE id = $1", parameters=[42])
  Placeholders: PostgreSQL uses $1, $2; MySQL/SQLite/DuckDB/ClickHouse use ?
`

const platformInstructions = `WhoDB MCP Server - Hosted Platform Tools

This server is running in hosted platform mode. It exposes only whodb_platform_* tools
backed by the current hosted WhoDB login and selected organization/project.

Read tools return stable metadata such as count, request_id, scope, warnings, and
truncated where applicable. Most read tools accept an optional fields array to
project top-level output fields, for example ["id", "name"]. Prefer fields on
read calls: request only the fields needed for the current answer, then make a
second read with additional fields only if more detail is required. Heavy fields
such as function files, file text, tabular rows, or source content should be
requested only when needed.

Available tools:
- whodb_platform_status: Show hosted login and selected workspace
- whodb_platform_orgs: List hosted organizations visible to the user
- whodb_platform_projects: List hosted projects for an organization
- whodb_platform_sources: List hosted sources in the selected project
- whodb_platform_source_types: List source types available for creation
- whodb_platform_source_fields: List connection fields for one source type
- whodb_platform_source_objects: Browse hosted source objects
- whodb_platform_source_columns: Inspect hosted source object columns
- whodb_platform_source_rows: Preview hosted source object rows
- whodb_platform_source_constraints: Inspect editable source field constraints
- whodb_platform_source_content: Read hosted source content when supported
- whodb_platform_source_config: Inspect redacted hosted source config
- whodb_platform_source_test: Test saved or draft hosted source connections
- whodb_platform_secrets: List secret metadata without secret values
- whodb_platform_ai_providers: List AI provider metadata without API keys
- whodb_platform_ai_provider_models: List models for one AI provider
- whodb_platform_ontologies / whodb_platform_ontology: List or inspect ontologies
- whodb_platform_ontology_fast_lookups: List saved fast lookups for an ontology
- whodb_platform_ontology_fast_lookup_suggestions: List suggested fast lookups
- whodb_platform_ontology_rows: Preview ontology rows
- whodb_platform_ontology_follow_link: Follow an ontology link from a row
- whodb_platform_datasets / whodb_platform_dataset: List or inspect datasets
- whodb_platform_dataset_rows: Preview dataset rows
- whodb_platform_lineage / whodb_platform_lineage_neighbors / whodb_platform_project_lineage: Inspect lineage
- whodb_platform_transforms / whodb_platform_transform_runs: List transforms and runs
- whodb_platform_functions / whodb_platform_function: List or inspect ontology functions
- whodb_platform_files / whodb_platform_file_preview / whodb_platform_file_search / whodb_platform_tabular_files: Browse project files
- whodb_platform_storage_usage: Inspect project storage usage
- whodb_platform_source_create: Prepare hosted source creation for confirmation
- whodb_platform_source_update: Prepare hosted source updates for confirmation
- whodb_platform_source_delete: Prepare hosted source deletion for confirmation
- whodb_platform_create: Create hosted resources such as secrets, AI providers, ontology, datasets, transforms, folders, functions, and source objects
- whodb_platform_update: Update hosted resources such as secrets, AI providers, ontology, datasets, transforms, functions, and source objects
- whodb_platform_delete: Delete hosted resources such as secrets, AI providers, ontology, datasets, transforms, files, folders, functions, and source objects
- whodb_platform_action: Run hosted actions such as transform/run, file upload/rename/move/promote_to_dataset, folder rename/move, and function deploy/redeploy
- whodb_platform_pending: List pending hosted platform confirmations
- whodb_platform_confirm: Confirm pending hosted platform writes

Setup:
1. Run whodb-cli login
2. Run whodb-cli use --org <org> --project <project>
3. Start this server with whodb-cli mcp serve --platform

Hosted create, update, delete, and action tools follow the same permission mode
as local MCP writes. In default confirm-writes mode, they return confirmation
tokens and do not execute until whodb_platform_confirm is called. The assistant
must explain confirmation_preview and ask the user before confirming destructive
or mutating operations. In --read-only or --safe-mode, hosted platform write tools
are not exposed. In --allow-write, writes execute immediately without confirmation.
`
