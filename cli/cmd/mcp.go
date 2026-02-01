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
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	whodbmcp "github.com/clidey/whodb/cli/pkg/mcp"
	"github.com/spf13/cobra"
)

// security flags
var (
	mcpReadOnly            bool
	mcpConfirmWrites       bool
	mcpAllowWrite          bool
	mcpAllowDrop           bool
	mcpSecurity            string
	mcpTimeout             time.Duration
	mcpMaxRows             int
	mcpAllowMultiStatement bool
	mcpSafeMode            bool
)

// transport flags
var (
	mcpTransport string
	mcpHost      string
	mcpPort      int
)

// tool enablement flags
var (
	mcpEnabledTools  []string
	mcpDisabledTools []string
)

// analytics flags
var mcpNoAnalytics bool

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Model Context Protocol server",
	Long: `Run WhoDB as an MCP (Model Context Protocol) server.

MCP enables AI assistants like Claude to interact with your databases
through a standardized protocol.`,
}

var mcpServeCmd = &cobra.Command{
	Use:           "serve",
	Short:         "Start the MCP server",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Start WhoDB as an MCP server.

TRANSPORT:
  --transport stdio  (default) Communicate via stdin/stdout for CLI integration
  --transport http   Run as HTTP service for cloud/shared deployments

  HTTP mode options:
    --host HOST      Bind address (default: localhost)
    --port PORT      Listen port (default: 3000)

  HTTP mode exposes:
    /mcp      - MCP endpoint (streaming HTTP)
    /health   - Health check endpoint (returns {"status":"ok"})

SECURITY:
  Write operations require user confirmation by default. This keeps you in
  control while allowing full database functionality.

  Permission Modes (controls whether writes are allowed):
    (default)        - Confirm-writes: All writes require user confirmation
    --safe-mode      - Safe mode: read-only + strict security
    --read-only      - Read-only: SELECT, SHOW, DESCRIBE, EXPLAIN only
    --allow-write    - Full write access without confirmation (use with caution)

  Security Levels (additional validation, does NOT override permission mode):
    strict   - Blocks dangerous functions (pg_read_file, COPY, LOAD_FILE, etc.)
    standard - Basic validation (default)
    minimal  - Only blocks DELETE without WHERE (when writes allowed)

  Note: Permission mode takes priority. --read-only blocks all writes regardless of
  security level. Multi-statement queries are blocked by default (--allow-multi-statement).

Available tools:
  whodb_query       - Execute SQL queries (security-validated)
  whodb_schemas     - List database schemas
  whodb_tables      - List tables in a schema
  whodb_columns     - Describe table columns
  whodb_connections - List available connections
  whodb_confirm     - Confirm pending writes (only with --confirm-writes)

TOOL SELECTION:
  --tools           - Comma-separated list of tools to enable (default: all)
                      Valid: query, schemas, tables, columns, connections, confirm
  --disable-tools   - Comma-separated list of tools to disable (takes precedence)

ANALYTICS:
  Anonymous usage analytics are enabled by default to help improve WhoDB.
  No query content, database credentials, or personal data is ever collected.
  Only tool usage patterns and error rates are tracked.

  To disable analytics:
    --no-analytics                        Flag to disable analytics
    WHODB_MCP_ANALYTICS_DISABLED=true     Environment variable to disable

Connection Resolution:
  Tools accept a 'connection' parameter that references either:
  1. Environment profiles, for example:
     - WHODB_POSTGRES='[{"alias":"prod","host":"localhost","user":"user","password":"pass","database":"db","port":"5432"}]'
     - WHODB_MYSQL_1='{"alias":"staging","host":"localhost","user":"user","password":"pass","database":"db","port":"3306"}'
  2. Saved connection from 'whodb-cli connections add'

  Saved connections take precedence when names collide.`,
	Example: `  # Start MCP server (confirm-writes by default - you approve each write)
  whodb-cli mcp serve

  # Safe mode for demos/playgrounds (read-only + strict security)
  whodb-cli mcp serve --safe-mode

  # Read-only mode (no writes at all)
  whodb-cli mcp serve --read-only

  # Allow full write access without confirmation (use with caution)
  whodb-cli mcp serve --allow-write

  # Strict security mode (blocks dangerous functions like pg_read_file)
  whodb-cli mcp serve --security=strict

  # Custom timeout and row limit
  whodb-cli mcp serve --timeout=60s --max-rows=500

  # Run as HTTP service
  whodb-cli mcp serve --transport=http --port=3000
  # Endpoint: http://localhost:3000/mcp
  # Health:   http://localhost:3000/health

  # HTTP mode bound to all interfaces (can be used in Docker/Kubernetes)
  whodb-cli mcp serve --transport=http --host=0.0.0.0 --port=8080

  # Enable only specific tools (minimal surface for read-only exploration)
  whodb-cli mcp serve --tools=schemas,tables,columns,connections

  # Disable query tool (only schema exploration)
  whodb-cli mcp serve --disable-tools=query,confirm

  # Claude Desktop / Claude Code configuration (stdio):
  {
    "mcpServers": {
      "whodb": {
        "command": "whodb-cli",
        "args": ["mcp", "serve"],
        "env": {
          "WHODB_POSTGRES_1": "{\"alias\":\"prod\",\"host\":\"localhost\",\"user\":\"user\",\"password\":\"pass\",\"database\":\"db\"}"
        }
      }
    }
  }`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create context with signal handling
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle shutdown signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			cancel()
		}()

		// Initialize analytics (enabled by default)
		if err := whodbmcp.InitializeAnalytics(&whodbmcp.AnalyticsConfig{
			Enabled:    !mcpNoAnalytics,
			AppVersion: whodbmcp.Version,
		}); err != nil {
			// Analytics initialization failure is non-fatal
			// Continue without analytics
		}
		defer whodbmcp.ShutdownAnalytics()

		// Determine mode based on flags
		// Default: confirm-writes (human-in-the-loop)
		// Priority: --safe-mode > --allow-write > --read-only > default (confirm-writes)
		readOnly := false
		confirmWrites := true // Default: confirm-writes enabled
		securityLevel := mcpSecurity

		if mcpSafeMode {
			// Safe mode: read-only + strict security
			readOnly = true
			confirmWrites = false
			securityLevel = "strict"
		} else if mcpAllowWrite {
			confirmWrites = false // No confirmation needed
		} else if mcpReadOnly {
			readOnly = true
			confirmWrites = false // Read-only, no writes to confirm
		}

		// Build server options from flags
		opts := &whodbmcp.ServerOptions{
			ReadOnly:            readOnly,
			ConfirmWrites:       confirmWrites,
			SecurityLevel:       whodbmcp.SecurityLevel(securityLevel),
			QueryTimeout:        mcpTimeout,
			MaxRows:             mcpMaxRows,
			AllowMultiStatement: mcpAllowMultiStatement,
			AllowDrop:           mcpAllowDrop,
			EnabledTools:        mcpEnabledTools,
			DisabledTools:       mcpDisabledTools,
		}

		server := whodbmcp.NewServer(opts)

		// Determine security mode name for tracking
		securityModeName := "confirm-writes"
		if mcpSafeMode {
			securityModeName = "safe-mode"
		} else if mcpReadOnly {
			securityModeName = "read-only"
		} else if mcpAllowWrite {
			securityModeName = "allow-write"
		}

		// Track server start
		whodbmcp.TrackServerStart(ctx, mcpTransport, securityModeName, map[string]any{
			"enabled_tools":  mcpEnabledTools,
			"disabled_tools": mcpDisabledTools,
			"security_level": securityLevel,
		})

		// Run with selected transport
		switch whodbmcp.TransportType(mcpTransport) {
		case whodbmcp.TransportHTTP:
			return whodbmcp.RunHTTP(ctx, server, &whodbmcp.HTTPOptions{
				Host: mcpHost,
				Port: mcpPort,
			}, opts.Logger)
		default:
			return whodbmcp.Run(ctx, server)
		}
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpServeCmd)

	// Security flags
	mcpServeCmd.Flags().BoolVar(&mcpSafeMode, "safe-mode", false,
		"Enable safe mode (read-only + strict security) for demos and playgrounds")
	mcpServeCmd.Flags().BoolVar(&mcpReadOnly, "read-only", false,
		"Enable read-only mode (blocks all write operations)")
	mcpServeCmd.Flags().BoolVar(&mcpConfirmWrites, "confirm-writes", false,
		"Enable human-in-the-loop write confirmation (this is the default)")
	mcpServeCmd.Flags().BoolVar(&mcpAllowWrite, "allow-write", false,
		"Allow all write operations without confirmation (use with caution)")
	mcpServeCmd.Flags().BoolVar(&mcpAllowDrop, "allow-drop", false,
		"Allow DROP/TRUNCATE even with --allow-write (requires explicit opt-in)")
	mcpServeCmd.Flags().StringVar(&mcpSecurity, "security", "standard",
		"Security level: strict, standard, or minimal")

	// Query limits
	mcpServeCmd.Flags().DurationVar(&mcpTimeout, "timeout", 30*time.Second,
		"Query timeout duration")
	mcpServeCmd.Flags().IntVar(&mcpMaxRows, "max-rows", 0,
		"Limit rows returned per query (0 = unlimited, default)")
	mcpServeCmd.Flags().BoolVar(&mcpAllowMultiStatement, "allow-multi-statement", false,
		"Allow multiple SQL statements in one query (security risk)")

	// Transport flags
	mcpServeCmd.Flags().StringVar(&mcpTransport, "transport", "stdio",
		"Transport type: stdio (default) or http")
	mcpServeCmd.Flags().StringVar(&mcpHost, "host", "localhost",
		"Host to bind to (only used with --transport=http)")
	mcpServeCmd.Flags().IntVar(&mcpPort, "port", 3000,
		"Port to listen on (only used with --transport=http)")

	// Tool enablement flags
	mcpServeCmd.Flags().StringSliceVar(&mcpEnabledTools, "tools", nil,
		"Comma-separated list of tools to enable (default: all). Valid: query, schemas, tables, columns, connections, confirm")
	mcpServeCmd.Flags().StringSliceVar(&mcpDisabledTools, "disable-tools", nil,
		"Comma-separated list of tools to disable (takes precedence over --tools)")

	// Analytics flags
	mcpServeCmd.Flags().BoolVar(&mcpNoAnalytics, "no-analytics", false,
		"Disable anonymous usage analytics (can also set WHODB_MCP_ANALYTICS_DISABLED=true)")

	// Mark flags as mutually exclusive
	mcpServeCmd.MarkFlagsMutuallyExclusive("read-only", "allow-write")
}
