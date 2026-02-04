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

// Package analytics provides PostHog analytics for the WhoDB CLI.
// Analytics are enabled by default and can be disabled via:
// - WHODB_CLI_ANALYTICS_DISABLED=true environment variable
// - --no-analytics flag (where supported)
//
// Privacy: No query content, database credentials, or personal data is collected.
// Only usage patterns, error rates, and performance metrics are tracked.
package analytics

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/clidey/whodb/core/src/analytics"
)

// PostHog configuration - same project as core for unified analytics
const (
	posthogAPIKey = "phc_hbXcCoPTdxm5ADL8PmLSYTIUvS6oRWFM2JAK8SMbfnH"
	posthogHost   = "https://us.i.posthog.com"
)

var (
	initialized bool
	initMu      sync.Mutex
	cliVersion  string
)

// Initialize sets up PostHog analytics for the CLI.
// Safe to call multiple times - only initializes once.
func Initialize(version string) error {
	initMu.Lock()
	defer initMu.Unlock()

	if initialized {
		return nil
	}

	cliVersion = version

	// Check if disabled via environment variable
	if os.Getenv("WHODB_CLI_ANALYTICS_DISABLED") == "true" {
		return nil
	}

	if err := analytics.Initialize(analytics.Config{
		APIKey:      posthogAPIKey,
		Host:        posthogHost,
		Environment: "cli",
		AppVersion:  version,
	}); err != nil {
		return err
	}

	analytics.SetEnabled(true)
	initialized = true
	return nil
}

// Shutdown flushes pending events and closes the analytics client.
func Shutdown() {
	analytics.Shutdown()
}

// SetEnabled enables or disables analytics at runtime.
func SetEnabled(enabled bool) {
	analytics.SetEnabled(enabled)
}

// IsEnabled returns whether analytics are currently active.
func IsEnabled() bool {
	return analytics.Enabled()
}

// baseProps returns common properties included in all events.
func baseProps() map[string]any {
	return map[string]any{
		"source":      "cli",
		"cli_version": cliVersion,
	}
}

// mergeProps merges additional properties with base props.
func mergeProps(additional map[string]any) map[string]any {
	props := baseProps()
	for k, v := range additional {
		props[k] = v
	}
	return props
}

// ─────────────────────────────────────────────────────────────────────────────
// CLI Startup & Session Events
// ─────────────────────────────────────────────────────────────────────────────

// TrackCLIStartup captures CLI invocation (called once at startup).
func TrackCLIStartup(ctx context.Context, command string) {
	analytics.Capture(ctx, "cli.startup", mergeProps(map[string]any{
		"command":   command,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}))
}

// TrackSessionStart captures interactive TUI session start.
func TrackSessionStart(ctx context.Context, dbType string) {
	analytics.Capture(ctx, "cli.session.start", mergeProps(map[string]any{
		"db_type":   dbType,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}))
}

// TrackSessionEnd captures interactive TUI session end.
func TrackSessionEnd(ctx context.Context, dbType string, durationSec int64, queryCount int) {
	analytics.Capture(ctx, "cli.session.end", mergeProps(map[string]any{
		"db_type":      dbType,
		"duration_sec": durationSec,
		"query_count":  queryCount,
	}))
}

// ─────────────────────────────────────────────────────────────────────────────
// Connection Events
// ─────────────────────────────────────────────────────────────────────────────

// TrackConnectAttempt captures a connection attempt.
func TrackConnectAttempt(ctx context.Context, dbType string) {
	analytics.Capture(ctx, "cli.connect.attempt", mergeProps(map[string]any{
		"db_type": dbType,
	}))
}

// TrackConnectSuccess captures a successful connection.
func TrackConnectSuccess(ctx context.Context, dbType string, durationMs int64) {
	analytics.Capture(ctx, "cli.connect.success", mergeProps(map[string]any{
		"db_type":     dbType,
		"duration_ms": durationMs,
	}))
}

// TrackConnectError captures a connection failure.
func TrackConnectError(ctx context.Context, dbType string, errorType string, durationMs int64) {
	analytics.Capture(ctx, "cli.connect.error", mergeProps(map[string]any{
		"db_type":     dbType,
		"error_type":  errorType,
		"duration_ms": durationMs,
	}))
}

// TrackConnectionAdd captures adding a saved connection.
func TrackConnectionAdd(ctx context.Context, dbType string) {
	analytics.Capture(ctx, "cli.connections.add", mergeProps(map[string]any{
		"db_type": dbType,
	}))
}

// TrackConnectionRemove captures removing a saved connection.
func TrackConnectionRemove(ctx context.Context) {
	analytics.Capture(ctx, "cli.connections.remove", mergeProps(nil))
}

// TrackConnectionTest captures testing a connection.
func TrackConnectionTest(ctx context.Context, dbType string, success bool, durationMs int64) {
	analytics.Capture(ctx, "cli.connections.test", mergeProps(map[string]any{
		"db_type":     dbType,
		"success":     success,
		"duration_ms": durationMs,
	}))
}

// ─────────────────────────────────────────────────────────────────────────────
// Query Events
// ─────────────────────────────────────────────────────────────────────────────

// TrackQueryExecute captures query execution.
func TrackQueryExecute(ctx context.Context, dbType string, statementType string, success bool, durationMs int64, rowCount int, props map[string]any) {
	eventProps := mergeProps(map[string]any{
		"db_type":        dbType,
		"statement_type": statementType,
		"success":        success,
		"duration_ms":    durationMs,
		"row_count":      rowCount,
	})
	for k, v := range props {
		eventProps[k] = v
	}
	analytics.Capture(ctx, "cli.query.execute", eventProps)
}

// TrackQueryError captures query failure.
func TrackQueryError(ctx context.Context, dbType string, errorType string, durationMs int64) {
	analytics.Capture(ctx, "cli.query.error", mergeProps(map[string]any{
		"db_type":     dbType,
		"error_type":  errorType,
		"duration_ms": durationMs,
	}))
}

// ─────────────────────────────────────────────────────────────────────────────
// Schema Exploration Events
// ─────────────────────────────────────────────────────────────────────────────

// TrackSchemasListed captures schema listing.
func TrackSchemasListed(ctx context.Context, dbType string, schemaCount int, durationMs int64) {
	analytics.Capture(ctx, "cli.schemas.list", mergeProps(map[string]any{
		"db_type":      dbType,
		"schema_count": schemaCount,
		"duration_ms":  durationMs,
	}))
}

// TrackTablesListed captures table listing.
func TrackTablesListed(ctx context.Context, dbType string, tableCount int, durationMs int64) {
	analytics.Capture(ctx, "cli.tables.list", mergeProps(map[string]any{
		"db_type":     dbType,
		"table_count": tableCount,
		"duration_ms": durationMs,
	}))
}

// TrackColumnsListed captures column listing.
func TrackColumnsListed(ctx context.Context, dbType string, columnCount int, durationMs int64) {
	analytics.Capture(ctx, "cli.columns.list", mergeProps(map[string]any{
		"db_type":      dbType,
		"column_count": columnCount,
		"duration_ms":  durationMs,
	}))
}

// ─────────────────────────────────────────────────────────────────────────────
// Export Events
// ─────────────────────────────────────────────────────────────────────────────

// TrackExport captures data export.
func TrackExport(ctx context.Context, dbType string, format string, rowCount int, durationMs int64) {
	analytics.Capture(ctx, "cli.export.execute", mergeProps(map[string]any{
		"db_type":     dbType,
		"format":      format,
		"row_count":   rowCount,
		"duration_ms": durationMs,
	}))
}

// ─────────────────────────────────────────────────────────────────────────────
// MCP Events (re-exported for convenience)
// ─────────────────────────────────────────────────────────────────────────────

// TrackMCPServerStart captures MCP server start.
func TrackMCPServerStart(ctx context.Context, transport string, securityMode string, props map[string]any) {
	eventProps := mergeProps(map[string]any{
		"transport":     transport,
		"security_mode": securityMode,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	})
	for k, v := range props {
		eventProps[k] = v
	}
	analytics.Capture(ctx, "mcp.server.start", eventProps)
}

// TrackMCPToolCall captures MCP tool invocation.
func TrackMCPToolCall(ctx context.Context, toolName string, requestID string, success bool, durationMs int64, props map[string]any) {
	eventProps := mergeProps(map[string]any{
		"tool_name":   toolName,
		"request_id":  requestID,
		"success":     success,
		"duration_ms": durationMs,
	})
	for k, v := range props {
		eventProps[k] = v
	}
	analytics.Capture(ctx, "mcp.tool.call", eventProps)
}

// ─────────────────────────────────────────────────────────────────────────────
// Error Tracking
// ─────────────────────────────────────────────────────────────────────────────

// TrackError captures a generic error event.
func TrackError(ctx context.Context, operation string, err error, props map[string]any) {
	analytics.CaptureError(ctx, operation, err, mergeProps(props))
}
