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
)

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
	Long: `Start WhoDB as an MCP server using stdio transport.

The server communicates over stdin/stdout using JSON-RPC, making it
compatible with Claude Desktop, Cursor, and other MCP clients.

SECURITY:
  Write operations require user confirmation by default. This keeps you in
  control while allowing full database functionality.

  Permission Modes (controls whether writes are allowed):
    (default)        - Confirm-writes: All writes require user confirmation
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

Connection Resolution:
  Tools accept a 'connection' parameter that references either:
  1. Environment profiles, for example:
     - WHODB_POSTGRES='[{"alias":"prod","host":"localhost","user":"user","password":"pass","database":"db","port":"5432"}]'
     - WHODB_MYSQL_1='{"alias":"staging","host":"localhost","user":"user","password":"pass","database":"db","port":"3306"}'
  2. Saved connection from 'whodb-cli connections add'

  Saved connections take precedence when names collide.`,
	Example: `  # Start MCP server (confirm-writes by default - you approve each write)
  whodb-cli mcp serve

  # Read-only mode (no writes at all)
  whodb-cli mcp serve --read-only

  # Allow full write access without confirmation (use with caution)
  whodb-cli mcp serve --allow-write

  # Strict security mode (blocks dangerous functions like pg_read_file)
  whodb-cli mcp serve --security=strict

  # Custom timeout and row limit
  whodb-cli mcp serve --timeout=60s --max-rows=500

  # Claude Desktop / Claude Code configuration:
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

		// Determine mode based on flags
		// Default: confirm-writes (human-in-the-loop)
		// Priority: --allow-write > --read-only > default (confirm-writes)
		readOnly := false
		confirmWrites := true // Default: confirm-writes enabled

		if mcpAllowWrite {
			confirmWrites = false // No confirmation needed
		} else if mcpReadOnly {
			readOnly = true
			confirmWrites = false // Read-only, no writes to confirm
		}

		// Build server options from flags
		opts := &whodbmcp.ServerOptions{
			ReadOnly:            readOnly,
			ConfirmWrites:       confirmWrites,
			SecurityLevel:       whodbmcp.SecurityLevel(mcpSecurity),
			QueryTimeout:        mcpTimeout,
			MaxRows:             mcpMaxRows,
			AllowMultiStatement: mcpAllowMultiStatement,
			AllowDrop:           mcpAllowDrop,
		}

		// Create and run the MCP server
		server := whodbmcp.NewServer(opts)
		return whodbmcp.Run(ctx, server)
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpServeCmd)

	// Security flags
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

	// Mark flags as mutually exclusive
	mcpServeCmd.MarkFlagsMutuallyExclusive("read-only", "allow-write")
}
