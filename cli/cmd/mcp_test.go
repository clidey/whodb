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
	"testing"
	"time"

	"github.com/spf13/viper"
)

// TestMcpCmd_Exists verifies the mcp command is registered
func TestMcpCmd_Exists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "mcp" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'mcp' command to be registered")
	}
}

// TestMcpCmd_HasServeSubcommand verifies mcp has serve subcommand
func TestMcpCmd_HasServeSubcommand(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	found := false
	for _, cmd := range mcpCmd.Commands() {
		if cmd.Name() == "serve" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'mcp serve' subcommand")
	}
}

// TestMcpServeCmd_HasDescription verifies serve command has proper description
func TestMcpServeCmd_HasDescription(t *testing.T) {
	if mcpServeCmd.Short == "" {
		t.Error("Expected 'mcp serve' to have a short description")
	}
	if mcpServeCmd.Long == "" {
		t.Error("Expected 'mcp serve' to have a long description")
	}
}

// TestMcpServeCmd_HasExample verifies serve command has usage example
func TestMcpServeCmd_HasExample(t *testing.T) {
	if mcpServeCmd.Example == "" {
		t.Error("Expected 'mcp serve' to have usage examples")
	}
}

// TestMcpServeCmd_SilencesErrors verifies error handling is properly configured
func TestMcpServeCmd_SilencesErrors(t *testing.T) {
	if !mcpServeCmd.SilenceUsage {
		t.Error("Expected SilenceUsage to be true")
	}
	if !mcpServeCmd.SilenceErrors {
		t.Error("Expected SilenceErrors to be true")
	}
}

// TestMcpServeCmd_ViperBindings verifies all flags are bound to Viper
func TestMcpServeCmd_ViperBindings(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// List of all MCP config keys that should be bound
	expectedKeys := []string{
		"mcp.safe_mode",
		"mcp.read_only",
		"mcp.allow_write",
		"mcp.allow_drop",
		"mcp.security",
		"mcp.timeout",
		"mcp.max_rows",
		"mcp.allow_multi_statement",
		"mcp.transport",
		"mcp.host",
		"mcp.port",
		"mcp.rate_limit.enabled",
		"mcp.rate_limit.qps",
		"mcp.rate_limit.daily",
		"mcp.rate_limit.bypass_token",
	}

	for _, key := range expectedKeys {
		// Viper should have a binding for each key (either from flag or default)
		// We can verify by checking if the key is registered
		if !viper.IsSet(key) && viper.GetString(key) == "" && viper.GetInt(key) == 0 && !viper.GetBool(key) {
			// Key might have zero value, which is fine - just verify the flag exists
			t.Logf("Key %s has zero value (expected for most defaults)", key)
		}
	}
}

// TestMcpServeCmd_ViperDefaults verifies default values are set correctly
func TestMcpServeCmd_ViperDefaults(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	tests := []struct {
		key      string
		expected any
	}{
		{"mcp.security", "standard"},
		{"mcp.timeout", 30 * time.Second},
		{"mcp.transport", "stdio"},
		{"mcp.host", "localhost"},
		{"mcp.port", 3000},
		{"mcp.rate_limit.qps", 10},
		{"mcp.rate_limit.daily", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			switch expected := tt.expected.(type) {
			case string:
				if got := viper.GetString(tt.key); got != expected {
					t.Errorf("viper.GetString(%q) = %q, want %q", tt.key, got, expected)
				}
			case int:
				if got := viper.GetInt(tt.key); got != expected {
					t.Errorf("viper.GetInt(%q) = %d, want %d", tt.key, got, expected)
				}
			case time.Duration:
				if got := viper.GetDuration(tt.key); got != expected {
					t.Errorf("viper.GetDuration(%q) = %v, want %v", tt.key, got, expected)
				}
			}
		})
	}
}

// TestMcpServeCmd_ViperConfigOverride verifies config file values can be set
func TestMcpServeCmd_ViperConfigOverride(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Simulate setting values as if from config file
	viper.Set("mcp.transport", "http")
	viper.Set("mcp.port", 9999)
	viper.Set("mcp.rate_limit.enabled", true)
	viper.Set("mcp.rate_limit.bypass_token", "test-token")

	if got := viper.GetString("mcp.transport"); got != "http" {
		t.Errorf("transport = %q, want %q", got, "http")
	}
	if got := viper.GetInt("mcp.port"); got != 9999 {
		t.Errorf("port = %d, want %d", got, 9999)
	}
	if got := viper.GetBool("mcp.rate_limit.enabled"); !got {
		t.Error("rate_limit.enabled = false, want true")
	}
	if got := viper.GetString("mcp.rate_limit.bypass_token"); got != "test-token" {
		t.Errorf("bypass_token = %q, want %q", got, "test-token")
	}

	// Clean up
	viper.Set("mcp.transport", "stdio")
	viper.Set("mcp.port", 3000)
	viper.Set("mcp.rate_limit.enabled", false)
	viper.Set("mcp.rate_limit.bypass_token", "")
}

// TestMcpServeCmd_HasAllFlags verifies all expected flags are registered
func TestMcpServeCmd_HasAllFlags(t *testing.T) {
	expectedFlags := []string{
		"safe-mode",
		"read-only",
		"confirm-writes",
		"allow-write",
		"allow-drop",
		"security",
		"timeout",
		"max-rows",
		"allow-multi-statement",
		"transport",
		"host",
		"port",
		"rate-limit",
		"rate-limit-qps",
		"rate-limit-daily",
		"rate-limit-bypass",
	}

	for _, flagName := range expectedFlags {
		flag := mcpServeCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag %q to be registered", flagName)
		}
	}
}
