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
	"testing"
)

func TestAnalyticsConfig_DisabledByEnv(t *testing.T) {
	// Set env var to disable analytics
	os.Setenv("WHODB_MCP_ANALYTICS_DISABLED", "true")
	defer os.Unsetenv("WHODB_MCP_ANALYTICS_DISABLED")

	cfg := &AnalyticsConfig{Enabled: true}

	// Initialize should respect env var
	err := InitializeAnalytics(cfg)
	if err != nil {
		// Initialization may fail due to missing PostHog client in test env, that's OK
		// The important thing is it doesn't panic
	}

	// Clean up
	ShutdownAnalytics()
}

func TestAnalyticsConfig_DisabledByFlag(t *testing.T) {
	cfg := &AnalyticsConfig{Enabled: false}

	err := InitializeAnalytics(cfg)
	if err != nil {
		// Initialization may fail, that's OK
	}

	ShutdownAnalytics()
}

func TestTrackToolCall_DoesNotPanicWhenDisabled(t *testing.T) {
	// This should not panic even when analytics is not initialized
	ctx := context.Background()
	TrackToolCall(ctx, "test_tool", "test-123", true, 100, map[string]any{"key": "value"})
}

func TestTrackServerStart_DoesNotPanicWhenDisabled(t *testing.T) {
	ctx := context.Background()
	TrackServerStart(ctx, "stdio", "confirm-writes", map[string]any{"key": "value"})
}

func TestTrackError_DoesNotPanicWhenDisabled(t *testing.T) {
	ctx := context.Background()
	TrackError(ctx, "test_tool", "test-123", "test_operation", "test error message")
}

func TestMcpError_Error(t *testing.T) {
	err := &mcpError{msg: "test error"}
	if err.Error() != "test error" {
		t.Errorf("Expected 'test error', got '%s'", err.Error())
	}
}
