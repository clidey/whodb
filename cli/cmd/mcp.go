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

package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	whodbmcp "github.com/clidey/whodb/cli/pkg/mcp"
	"github.com/spf13/cobra"
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

Available tools:
  whodb_query       - Execute SQL queries
  whodb_schemas     - List database schemas
  whodb_tables      - List tables in a schema
  whodb_columns     - Describe table columns
  whodb_connections - List available connections

Connection Resolution:
  Tools accept a 'connection' parameter that references either:
  1. Environment variable: WHODB_{NAME}_URI (e.g., "prod" -> WHODB_PROD_URI)
  2. Saved connection from 'whodb-cli connections add'

  Environment variables take precedence over saved connections.`,
	Example: `  # Start MCP server (for Claude Desktop / Cursor)
  whodb-cli mcp serve

  # Claude Desktop configuration (~/.config/claude/claude_desktop_config.json):
  {
    "mcpServers": {
      "whodb": {
        "command": "whodb-cli",
        "args": ["mcp", "serve"],
        "env": {
          "WHODB_PROD_URI": "postgres://user:pass@host:5432/db"
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

		// Create and run the MCP server
		server := whodbmcp.NewServer(nil)
		return whodbmcp.Run(ctx, server)
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpServeCmd)
}
