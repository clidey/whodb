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
	"os"
	"testing"
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

// setupTestEnv creates an isolated test environment (if not already defined)
func setupMcpTestEnv(t *testing.T) func() {
	t.Helper()
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	return func() {
		os.Setenv("HOME", origHome)
	}
}
