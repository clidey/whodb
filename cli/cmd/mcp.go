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
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	whodbmcp "github.com/clidey/whodb/cli/pkg/mcp"
	"github.com/clidey/whodb/cli/pkg/version"
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
	mcpAuthToken string
)

// platform flags
var (
	mcpPlatform        bool
	mcpDatabase        bool
	mcpPlatformHost    string
	mcpPlatformOrg     string
	mcpPlatformProject string
)

// tool enablement flags
var (
	mcpEnabledTools  []string
	mcpDisabledTools []string
)

// connection scoping flags
var mcpConnection string
var mcpAllowedConnections []string

// analytics flags
var mcpNoAnalytics bool
var mcpPlatformPolicy string

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
	Long: `Start WhoDB as an MCP server using saved setup settings.

Run whodb setup once to enable platform tools, database-only tools, or both.
Keep one client entry: command whodb, args ["mcp", "serve"]. Advanced flags
override saved settings for this process. Agents can read the active modules
and restrictions at whodb://mcp/configuration.

TRANSPORT:
  --transport stdio  (default) Communicate via stdin/stdout for CLI integration
  --transport http   Run as HTTP service for cloud/shared deployments

  HTTP mode options:
    --host HOST         Bind address (default: localhost)
    --port PORT         Listen port (default: 3000)
    --auth-token TOKEN  Require this bearer token on /mcp requests
                        (can also set WHODB_MCP_AUTH_TOKEN); strongly
                        recommended when binding to a network interface

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

Platform tools (default):
  whodb_platform_hosts              - Discover saved hosts and account metadata
  whodb_platform_orgs               - Discover organizations on a host
  whodb_platform_projects           - Discover projects in an organization
  whodb_platform_workspace_resolve  - Resolve workspace {host, org, project}
  whodb_platform_sources            - List sources in the requested workspace

All hosted tools accept a workspace target for this call only. One MCP server
can access multiple hosts/projects without changing saved defaults. Read
whodb://platform/schema for the complete platform tool contract.

Database-only MCP tools (enable the database module in setup):
  whodb_query       - Execute SQL queries (security-validated)
  whodb_schemas     - List database schemas
  whodb_tables      - List tables in a schema
  whodb_columns     - Describe table columns
  whodb_connections - List available connections
  whodb_explain     - Run EXPLAIN for a query
  whodb_diff        - Compare schema metadata between two connections
  whodb_erd         - Load graph/relationship metadata
  whodb_audit       - Run data quality audits
  whodb_suggestions - Load backend query suggestions
  whodb_confirm     - Confirm pending writes (only with --confirm-writes)
  whodb_pending     - List pending confirmation tokens

Platform is enabled by default; saved setup can enable either or both modules.
Platform tools use saved hosted logins and per-call workspace targets. Database
tools use connection names. --database is an advanced override for database-only
startup. Both modules use the same permission modes: default
confirm-writes returns confirmation tokens, --read-only and --safe-mode hide
hosted platform write tools, and --allow-write executes hosted platform writes
without confirmation.

PLATFORM SESSION SCOPING:
  --platform-host URL       Select the platform host for this MCP process
  --platform-org ORG        Select an organization id, slug, or name
  --platform-project NAME   Select a project id, slug, or name
  --platform-policy FILE    Restrict hosts/workspaces and configure read-only targets

  Organization and project must be provided together. These process-local
  values do not change the workspace saved by whodb use, so separate terminal
  sessions can safely target different projects. The equivalent environment
  variables are WHODB_PLATFORM_SESSION_HOST, WHODB_PLATFORM_SESSION_ORG, and
  WHODB_PLATFORM_SESSION_PROJECT.

TOOL SELECTION:
  --tools           - Comma-separated tool names to enable (default: all)
                      Platform: full names such as whodb_platform_sources
                      Database-only: query, schemas, tables, columns, connections, etc.
  --disable-tools   - Comma-separated tool names to disable (takes precedence)

ANALYTICS:
  Anonymous usage analytics are enabled by default to help improve WhoDB.
  No query content, database credentials, or personal data is ever collected.
  Only tool usage patterns and error rates are tracked.

  To disable analytics:
    --no-analytics                        Flag to disable analytics
    WHODB_MCP_ANALYTICS_DISABLED=true     Environment variable to disable

CONNECTION SCOPING:
  --default-connection NAME     Set default connection (no access restriction)
  --allowed-connections A,B,C   Restrict access to listed connections only

  With --allowed-connections:
  - whodb_connections only shows allowed connections
  - Queries to other connections are rejected
  - First allowed connection becomes the default (unless --default-connection set)

  With --default-connection only:
  - Sets the default, but all connections remain accessible

Connection Resolution:
  Tools accept a 'connection' parameter that references either:
  1. Environment profiles, for example:
     - WHODB_POSTGRES='[{"alias":"prod","host":"localhost","user":"user","password":"pass","database":"db","port":"5432"}]'
     - WHODB_MYSQL_1='{"alias":"staging","host":"localhost","user":"user","password":"pass","database":"db","port":"3306"}'
  2. Saved connection from 'whodb connections add'

  Saved connections take precedence when names collide.`,
	Example: `  # Start MCP server (confirm-writes by default - you approve each write)
  whodb mcp serve

  # Safe mode for demos/playgrounds (read-only + strict security)
  whodb mcp serve --safe-mode

  # Read-only mode (no writes at all)
  whodb mcp serve --read-only

  # Allow full write access without confirmation (use with caution)
  whodb mcp serve --allow-write

  # Strict security mode (blocks dangerous functions like pg_read_file)
  whodb mcp serve --security=strict

  # Custom timeout and row limit
  whodb mcp serve --timeout=60s --max-rows=500

  # Run as HTTP service
  whodb mcp serve --transport=http --port=3000
  # Endpoint: http://localhost:3000/mcp
  # Health:   http://localhost:3000/health

  # HTTP mode bound to all interfaces (can be used in Docker/Kubernetes)
  whodb mcp serve --transport=http --host=0.0.0.0 --port=8080

  # Enable only specific tools (minimal surface for read-only exploration)
  whodb mcp serve --database --tools=schemas,tables,columns,connections,suggestions

  # Disable query tool (only schema exploration)
  whodb mcp serve --database --disable-tools=query,confirm

  # Restrict to specific connections (first becomes default)
  whodb mcp serve --database --allowed-connections=prod,staging

  # Run hosted WhoDB platform MCP mode only
  whodb mcp serve

  # Run an independently scoped platform MCP session
  whodb mcp serve --platform-host=http://localhost:8080 \
    --platform-org=acme --platform-project=analysis

  # Hosted platform MCP config (stdio):
  {
    "mcpServers": {
      "whodb-platform": {
        "command": "whodb",
        "args": ["mcp", "serve"]
      }
    }
  }

  # Set default connection without restricting access
  whodb mcp serve --database --default-connection=prod

  # Claude Desktop / Claude Code configuration (stdio):
  {
    "mcpServers": {
      "whodb": {
        "command": "whodb",
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

		settings, err := configureMCPMode(cmd)
		if err != nil {
			return err
		}
		if err := configureMCPPlatformScope(settings.DefaultWorkspace); err != nil {
			return err
		}

		// Handle shutdown signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			cancel()
		}()

		// Initialize analytics (enabled by default); failure is non-fatal.
		_ = whodbmcp.InitializeAnalytics(&whodbmcp.AnalyticsConfig{
			Enabled:    !mcpNoAnalytics,
			AppVersion: version.Version,
		})
		defer whodbmcp.ShutdownAnalytics()

		// Determine mode based on flags
		// Default: confirm-writes (human-in-the-loop)
		// Priority: --safe-mode > --allow-write > --read-only > default (confirm-writes)
		readOnly := settings.WriteMode == "read-only"
		confirmWrites := settings.WriteMode == "confirm" // Default: confirm-writes enabled
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

		policy := (*whodbmcp.PlatformPolicy)(settings.PlatformPolicy)
		// Build server options from flags
		opts := &whodbmcp.ServerOptions{
			ReadOnly:            readOnly,
			ConfirmWrites:       confirmWrites,
			AllowWrite:          settings.WriteMode == "allow",
			SecurityLevel:       whodbmcp.SecurityLevel(securityLevel),
			QueryTimeout:        mcpTimeout,
			MaxRows:             mcpMaxRows,
			AllowMultiStatement: mcpAllowMultiStatement,
			AllowDrop:           mcpAllowDrop,
			EnabledTools:        settings.EnabledTools,
			DisabledTools:       settings.DisabledTools,
			DefaultConnection:   settings.DefaultConnection,
			AllowedConnections:  settings.AllowedConnections,
			PlatformEnabled:     mcpPlatform,
			DatabaseEnabled:     settings.HasModule("database"),
			PlatformPolicy:      policy,
		}

		server := whodbmcp.NewServer(opts)

		// Determine security mode name for tracking
		securityModeName := "confirm-writes"
		if mcpSafeMode {
			securityModeName = "safe-mode"
		} else if settings.WriteMode == "read-only" {
			securityModeName = "read-only"
		} else if settings.WriteMode == "allow" {
			securityModeName = "allow-write"
		}

		// Track server start
		whodbmcp.TrackServerStart(ctx, mcpTransport, securityModeName, map[string]any{
			"enabled_tools":  mcpEnabledTools,
			"disabled_tools": mcpDisabledTools,
			"security_level": securityLevel,
			"platform":       mcpPlatform,
		})

		// Run with selected transport
		switch whodbmcp.TransportType(mcpTransport) {
		case whodbmcp.TransportHTTP:
			authToken := mcpAuthToken
			if authToken == "" {
				authToken = os.Getenv("WHODB_MCP_AUTH_TOKEN")
			}
			return whodbmcp.RunHTTP(ctx, server, &whodbmcp.HTTPOptions{
				Host:      mcpHost,
				Port:      mcpPort,
				AuthToken: authToken,
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
	mcpServeCmd.Flags().StringVar(&mcpAuthToken, "auth-token", "",
		"Require this bearer token on HTTP requests (can also set WHODB_MCP_AUTH_TOKEN); strongly recommended when binding to a network interface")

	// Platform flags
	mcpServeCmd.Flags().StringVar(&mcpPlatformPolicy, "platform-policy", "", "JSON file restricting platform hosts/workspaces and read-only targets")
	mcpServeCmd.Flags().BoolVar(&mcpPlatform, "platform", true,
		"Compatibility alias: platform MCP is now the default")
	mcpServeCmd.Flags().BoolVar(&mcpDatabase, "database", false, "connect directly to databases without a WhoDB platform server")
	mcpServeCmd.Flags().StringVar(&mcpPlatformHost, "platform-host", "",
		"Hosted platform URL for this MCP process (or WHODB_PLATFORM_SESSION_HOST)")
	mcpServeCmd.Flags().StringVar(&mcpPlatformOrg, "platform-org", "",
		"Hosted organization id, slug, or name for this MCP process (or WHODB_PLATFORM_SESSION_ORG)")
	mcpServeCmd.Flags().StringVar(&mcpPlatformProject, "platform-project", "",
		"Hosted project id, slug, or name for this MCP process (or WHODB_PLATFORM_SESSION_PROJECT)")

	// Tool enablement flags
	mcpServeCmd.Flags().StringSliceVar(&mcpEnabledTools, "tools", nil,
		"Comma-separated tool names to enable (default: all); platform uses full whodb_platform_* names, database-only uses names such as query, schemas")
	mcpServeCmd.Flags().StringSliceVar(&mcpDisabledTools, "disable-tools", nil,
		"Comma-separated tool names to disable (takes precedence over --tools)")

	// Connection scoping
	mcpServeCmd.Flags().StringVar(&mcpConnection, "default-connection", "",
		"Default connection to use when not specified (does not restrict access)")
	mcpServeCmd.Flags().StringSliceVar(&mcpAllowedConnections, "allowed-connections", nil,
		"Comma-separated list of connections to allow (restricts access, first one becomes default)")

	// Analytics flags
	mcpServeCmd.Flags().BoolVar(&mcpNoAnalytics, "no-analytics", false,
		"Disable anonymous usage analytics (can also set WHODB_MCP_ANALYTICS_DISABLED=true)")

	// Mark flags as mutually exclusive
	mcpServeCmd.MarkFlagsMutuallyExclusive("read-only", "allow-write")

	mcpServeCmd.RegisterFlagCompletionFunc("default-connection", completeConnectionNames)
	mcpServeCmd.RegisterFlagCompletionFunc("allowed-connections", completeConnectionNames)
	mcpServeCmd.RegisterFlagCompletionFunc("transport", completeMCPTransports)
	mcpServeCmd.RegisterFlagCompletionFunc("security", completeMCPSecurityLevels)
}

func configureMCPPlatformScope(defaults ...*config.MCPWorkspace) error {
	if !mcpPlatform {
		if mcpPlatformHost != "" || mcpPlatformOrg != "" || mcpPlatformProject != "" {
			return fmt.Errorf("--platform-host, --platform-org, and --platform-project cannot be used with --database")
		}
		return nil
	}
	values := map[string]string{
		platformapi.SessionHostEnv:    mcpPlatformHost,
		platformapi.SessionOrgEnv:     mcpPlatformOrg,
		platformapi.SessionProjectEnv: mcpPlatformProject,
	}
	hasOverride := false
	for key, value := range values {
		if strings.TrimSpace(value) != "" || strings.TrimSpace(os.Getenv(key)) != "" {
			hasOverride = true
		}
	}
	if !hasOverride && len(defaults) > 0 && defaults[0] != nil {
		target := defaults[0]
		values[platformapi.SessionHostEnv] = target.Host
		values[platformapi.SessionOrgEnv] = target.Org
		values[platformapi.SessionProjectEnv] = target.Project
	}
	for key, value := range values {
		if strings.TrimSpace(value) != "" {
			if err := os.Setenv(key, strings.TrimSpace(value)); err != nil {
				return fmt.Errorf("set hosted platform session scope: %w", err)
			}
		}
	}
	return platformapi.SessionScopeFromEnvironment().Validate()
}

func configureMCPMode(cmd *cobra.Command) (config.MCPSettings, error) {
	settings, err := effectiveMCPSettings(cmd)
	if err != nil {
		return settings, err
	}
	mcpPlatform = settings.HasModule("platform")
	if cmd.Flags().Changed("platform") {
		fmt.Fprintln(cmd.ErrOrStderr(), "--platform is no longer necessary: configure modules once with whodb setup.")
	}
	return settings, nil
}
