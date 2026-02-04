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
	"os"

	"github.com/clidey/whodb/cli/pkg/analytics"
)

// AnalyticsConfig holds MCP analytics configuration.
type AnalyticsConfig struct {
	// Enabled controls whether analytics are active. Default: true
	Enabled bool
	// AppVersion is the CLI version for tracking.
	AppVersion string
}

// InitializeAnalytics sets up PostHog analytics for the MCP server.
// Analytics are enabled by default and can be disabled via:
// - WHODB_MCP_ANALYTICS_DISABLED=true environment variable
// - --no-analytics flag
func InitializeAnalytics(cfg *AnalyticsConfig) error {
	if cfg == nil {
		cfg = &AnalyticsConfig{Enabled: true}
	}

	// Check environment variable override (MCP-specific)
	if os.Getenv("WHODB_MCP_ANALYTICS_DISABLED") == "true" {
		cfg.Enabled = false
	}

	// Initialize the shared analytics package
	if err := analytics.Initialize(cfg.AppVersion); err != nil {
		return err
	}

	// Apply MCP-specific enabled state
	if !cfg.Enabled {
		analytics.SetEnabled(false)
	}

	return nil
}

// ShutdownAnalytics flushes pending events and closes the analytics client.
func ShutdownAnalytics() {
	analytics.Shutdown()
}

// IsAnalyticsEnabled returns whether analytics are currently active.
func IsAnalyticsEnabled() bool {
	return analytics.IsEnabled()
}

// TrackToolCall captures an MCP tool invocation event.
func TrackToolCall(ctx context.Context, toolName, requestID string, success bool, durationMs int64, props map[string]any) {
	analytics.TrackMCPToolCall(ctx, toolName, requestID, success, durationMs, props)
}

// TrackServerStart captures an MCP server start event.
func TrackServerStart(ctx context.Context, transport string, securityMode string, props map[string]any) {
	analytics.TrackMCPServerStart(ctx, transport, securityMode, props)
}

// TrackError captures an MCP error event.
func TrackError(ctx context.Context, toolName, requestID, operation string, errMsg string) {
	analytics.TrackError(ctx, operation, &mcpError{msg: errMsg}, map[string]any{
		"tool_name":  toolName,
		"request_id": requestID,
	})
}

// mcpError is a simple error wrapper for analytics.
type mcpError struct {
	msg string
}

func (e *mcpError) Error() string {
	return e.msg
}
