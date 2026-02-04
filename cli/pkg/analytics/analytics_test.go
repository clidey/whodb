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

package analytics

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestInitialize_DisabledByEnv(t *testing.T) {
	os.Setenv("WHODB_CLI_ANALYTICS_DISABLED", "true")
	defer os.Unsetenv("WHODB_CLI_ANALYTICS_DISABLED")

	err := Initialize("test-version")
	if err != nil {
		t.Errorf("Initialize should not error when disabled: %v", err)
	}

	// Should not be enabled when disabled via env
	if IsEnabled() {
		t.Error("Analytics should be disabled when WHODB_CLI_ANALYTICS_DISABLED=true")
	}

	Shutdown()
}

func TestInitialize_MultipleCallsSafe(t *testing.T) {
	// Reset state
	initMu.Lock()
	initialized = false
	initMu.Unlock()

	// Multiple calls should not error
	err1 := Initialize("v1")
	err2 := Initialize("v2")

	if err1 != nil {
		t.Errorf("First Initialize should not error: %v", err1)
	}
	if err2 != nil {
		t.Errorf("Second Initialize should not error: %v", err2)
	}

	Shutdown()
}

func TestBaseProps(t *testing.T) {
	cliVersion = "test-1.0.0"
	props := baseProps()

	if props["source"] != "cli" {
		t.Errorf("Expected source='cli', got %v", props["source"])
	}
	if props["cli_version"] != "test-1.0.0" {
		t.Errorf("Expected cli_version='test-1.0.0', got %v", props["cli_version"])
	}
}

func TestMergeProps(t *testing.T) {
	cliVersion = "test-1.0.0"
	additional := map[string]any{
		"custom_key": "custom_value",
		"number":     42,
	}

	merged := mergeProps(additional)

	// Should have base props
	if merged["source"] != "cli" {
		t.Errorf("Expected source='cli', got %v", merged["source"])
	}

	// Should have additional props
	if merged["custom_key"] != "custom_value" {
		t.Errorf("Expected custom_key='custom_value', got %v", merged["custom_key"])
	}
	if merged["number"] != 42 {
		t.Errorf("Expected number=42, got %v", merged["number"])
	}
}

func TestMergeProps_NilInput(t *testing.T) {
	cliVersion = "test-1.0.0"
	merged := mergeProps(nil)

	// Should still have base props
	if merged["source"] != "cli" {
		t.Errorf("Expected source='cli', got %v", merged["source"])
	}
}

// Test that tracking functions don't panic when analytics is not initialized
func TestTrackFunctions_NoPanicWhenUninitialized(t *testing.T) {
	ctx := context.Background()

	// These should not panic even when analytics is not initialized
	TrackCLIStartup(ctx, "test")
	TrackSessionStart(ctx, "postgres")
	TrackSessionEnd(ctx, "postgres", 100, 5)
	TrackConnectAttempt(ctx, "postgres")
	TrackConnectSuccess(ctx, "postgres", 100)
	TrackConnectError(ctx, "postgres", "timeout", 100)
	TrackConnectionAdd(ctx, "postgres")
	TrackConnectionRemove(ctx)
	TrackConnectionTest(ctx, "postgres", true, 100)
	TrackQueryExecute(ctx, "postgres", "SELECT", true, 100, 10, nil)
	TrackQueryError(ctx, "postgres", "syntax_error", 100)
	TrackSchemasListed(ctx, "postgres", 5, 100)
	TrackTablesListed(ctx, "postgres", 10, 100)
	TrackColumnsListed(ctx, "postgres", 8, 100)
	TrackExport(ctx, "postgres", "csv", 100, 500)
	TrackMCPServerStart(ctx, "stdio", "safe", nil)
	TrackMCPToolCall(ctx, "query", "req-123", true, 100, nil)
	TrackError(ctx, "test_op", errors.New("test error"), nil)
}

func TestSetEnabled(t *testing.T) {
	// This tests the SetEnabled function wrapper
	SetEnabled(false)
	if IsEnabled() {
		t.Error("Analytics should be disabled after SetEnabled(false)")
	}

	SetEnabled(true)
	// Note: IsEnabled might still return false if core analytics isn't initialized
	// This is expected behavior - we're just testing the wrapper doesn't panic
}
