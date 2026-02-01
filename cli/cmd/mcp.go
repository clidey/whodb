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
	"github.com/spf13/viper"
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

// rate limiting flags (HTTP transport only)
var (
	mcpRateLimitEnabled bool
	mcpRateLimitQPS     int
	mcpRateLimitDaily   int
	mcpRateLimitBypass  string
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
	Long: `Start WhoDB as an MCP server.

TRANSPORT:
  --transport stdio  (default) Communicate via stdin/stdout for CLI integration
  --transport http   Run as HTTP service for cloud/shared deployments

  HTTP mode options:
    --host HOST      Bind address (default: localhost)
    --port PORT      Listen port (default: 3000)

  HTTP mode exposes:
    /mcp      - MCP endpoint (streaming HTTP)
    /health   - Health check endpoint (returns {"status":"ok"} + rate limit stats)

RATE LIMITING (HTTP transport only):
  Protect your server from abuse with IP-based rate limiting.

    --rate-limit           Enable rate limiting (disabled by default)
    --rate-limit-qps N     Max requests per second per IP (default: 10)
    --rate-limit-daily N   Max requests per day per IP (default: 1000, 0=unlimited)
    --rate-limit-bypass T  Token for trusted clients to bypass limits

  Rate-limited responses include:
    - HTTP 429 Too Many Requests
    - Retry-After header with seconds to wait
    - X-RateLimit-Limit and X-RateLimit-Remaining headers

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

Connection Resolution:
  Tools accept a 'connection' parameter that references either:
  1. Environment profiles, for example:
     - WHODB_POSTGRES='[{"alias":"prod","host":"localhost","user":"user","password":"pass","database":"db","port":"5432"}]'
     - WHODB_MYSQL_1='{"alias":"staging","host":"localhost","user":"user","password":"pass","database":"db","port":"3306"}'
  2. Saved connection from 'whodb-cli connections add'

  Saved connections take precedence when names collide.

CONFIGURATION FILE:
  All MCP options can be set in ~/.whodb-cli/config.yaml under the 'mcp' key.
  CLI flags override config file values. Example config:

    mcp:
      transport: http
      host: 0.0.0.0
      port: 8080
      security: strict
      timeout: 60s
      max_rows: 1000
      read_only: false
      allow_write: false
      allow_drop: false
      allow_multi_statement: false
      rate_limit:
        enabled: true
        qps: 10
        daily: 1000
        bypass_token: my-secret-token`,
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

  # HTTP with rate limiting (recommended for public endpoints)
  whodb-cli mcp serve --transport=http --rate-limit --rate-limit-qps=5 --rate-limit-daily=500

  # HTTP with rate limit bypass for trusted clients
  whodb-cli mcp serve --transport=http --rate-limit --rate-limit-bypass=my-secret-token
  # Clients bypass limits by including: X-RateLimit-Bypass: my-secret-token

  # Use config file for defaults, override specific options via CLI
  # ~/.whodb-cli/config.yaml:
  #   mcp:
  #     transport: http
  #     port: 8080
  #     rate_limit:
  #       enabled: true
  #       bypass_token: my-secret-token
  whodb-cli mcp serve --port=9000  # Overrides port from config

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

		// Read all settings from Viper (merges config file + env vars + CLI flags)
		safeMode := viper.GetBool("mcp.safe_mode")
		allowWrite := viper.GetBool("mcp.allow_write")
		isReadOnly := viper.GetBool("mcp.read_only")
		securityLevel := viper.GetString("mcp.security")
		timeout := viper.GetDuration("mcp.timeout")
		maxRows := viper.GetInt("mcp.max_rows")
		allowMultiStatement := viper.GetBool("mcp.allow_multi_statement")
		allowDrop := viper.GetBool("mcp.allow_drop")

		transport := viper.GetString("mcp.transport")
		host := viper.GetString("mcp.host")
		port := viper.GetInt("mcp.port")

		rateLimitEnabled := viper.GetBool("mcp.rate_limit.enabled")
		rateLimitQPS := viper.GetInt("mcp.rate_limit.qps")
		rateLimitDaily := viper.GetInt("mcp.rate_limit.daily")
		rateLimitBypass := viper.GetString("mcp.rate_limit.bypass_token")

		// Determine mode based on settings
		// Default: confirm-writes (human-in-the-loop)
		// Priority: --safe-mode > --allow-write > --read-only > default (confirm-writes)
		readOnly := false
		confirmWrites := true // Default: confirm-writes enabled

		if safeMode {
			// Safe mode: read-only + strict security
			readOnly = true
			confirmWrites = false
			securityLevel = "strict"
		} else if allowWrite {
			confirmWrites = false // No confirmation needed
		} else if isReadOnly {
			readOnly = true
			confirmWrites = false // Read-only, no writes to confirm
		}

		// Build server options
		opts := &whodbmcp.ServerOptions{
			ReadOnly:            readOnly,
			ConfirmWrites:       confirmWrites,
			SecurityLevel:       whodbmcp.SecurityLevel(securityLevel),
			QueryTimeout:        timeout,
			MaxRows:             maxRows,
			AllowMultiStatement: allowMultiStatement,
			AllowDrop:           allowDrop,
		}

		server := whodbmcp.NewServer(opts)

		// Run with selected transport
		switch whodbmcp.TransportType(transport) {
		case whodbmcp.TransportHTTP:
			return whodbmcp.RunHTTP(ctx, server, &whodbmcp.HTTPOptions{
				Host: host,
				Port: port,
				RateLimit: whodbmcp.RateLimitOptions{
					Enabled:     rateLimitEnabled,
					QPS:         rateLimitQPS,
					Daily:       rateLimitDaily,
					BypassToken: rateLimitBypass,
				},
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

	// Rate limiting flags (HTTP transport only)
	mcpServeCmd.Flags().BoolVar(&mcpRateLimitEnabled, "rate-limit", false,
		"Enable IP-based rate limiting (HTTP transport only)")
	mcpServeCmd.Flags().IntVar(&mcpRateLimitQPS, "rate-limit-qps", 10,
		"Max requests per second per IP (default: 10)")
	mcpServeCmd.Flags().IntVar(&mcpRateLimitDaily, "rate-limit-daily", 1000,
		"Max requests per day per IP (default: 1000, 0=unlimited)")
	mcpServeCmd.Flags().StringVar(&mcpRateLimitBypass, "rate-limit-bypass", "",
		"Token for trusted clients to bypass rate limits (via X-RateLimit-Bypass header)")

	// Bind all flags to Viper for config file support
	// Config file uses nested structure: mcp.transport, mcp.rate_limit.enabled, etc.
	viper.BindPFlag("mcp.safe_mode", mcpServeCmd.Flags().Lookup("safe-mode"))
	viper.BindPFlag("mcp.read_only", mcpServeCmd.Flags().Lookup("read-only"))
	viper.BindPFlag("mcp.allow_write", mcpServeCmd.Flags().Lookup("allow-write"))
	viper.BindPFlag("mcp.allow_drop", mcpServeCmd.Flags().Lookup("allow-drop"))
	viper.BindPFlag("mcp.security", mcpServeCmd.Flags().Lookup("security"))
	viper.BindPFlag("mcp.timeout", mcpServeCmd.Flags().Lookup("timeout"))
	viper.BindPFlag("mcp.max_rows", mcpServeCmd.Flags().Lookup("max-rows"))
	viper.BindPFlag("mcp.allow_multi_statement", mcpServeCmd.Flags().Lookup("allow-multi-statement"))
	viper.BindPFlag("mcp.transport", mcpServeCmd.Flags().Lookup("transport"))
	viper.BindPFlag("mcp.host", mcpServeCmd.Flags().Lookup("host"))
	viper.BindPFlag("mcp.port", mcpServeCmd.Flags().Lookup("port"))
	viper.BindPFlag("mcp.rate_limit.enabled", mcpServeCmd.Flags().Lookup("rate-limit"))
	viper.BindPFlag("mcp.rate_limit.qps", mcpServeCmd.Flags().Lookup("rate-limit-qps"))
	viper.BindPFlag("mcp.rate_limit.daily", mcpServeCmd.Flags().Lookup("rate-limit-daily"))
	viper.BindPFlag("mcp.rate_limit.bypass_token", mcpServeCmd.Flags().Lookup("rate-limit-bypass"))

	// Set defaults for Viper (same as flag defaults)
	viper.SetDefault("mcp.security", "standard")
	viper.SetDefault("mcp.timeout", 30*time.Second)
	viper.SetDefault("mcp.transport", "stdio")
	viper.SetDefault("mcp.host", "localhost")
	viper.SetDefault("mcp.port", 3000)
	viper.SetDefault("mcp.rate_limit.qps", 10)
	viper.SetDefault("mcp.rate_limit.daily", 1000)

	// Mark flags as mutually exclusive
	mcpServeCmd.MarkFlagsMutuallyExclusive("read-only", "allow-write")
}
