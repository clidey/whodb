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
	"strings"
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

func TestMcpServeCmd_HasPlatformFlag(t *testing.T) {
	if mcpServeCmd.Flags().Lookup("platform") == nil {
		t.Fatal("expected mcp serve to expose --platform")
	}
}

func TestPlatformMCP_ServeRejectsLocalToolSelection(t *testing.T) {
	for _, tt := range []struct {
		name  string
		flag  string
		value string
	}{
		{name: "tools", flag: "tools", value: "schemas"},
		{name: "disable tools", flag: "disable-tools", value: "query"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnv(t)
			defer cleanup()
			resetMCPServeFlagsForTest(t)

			if err := mcpServeCmd.Flags().Set("platform", "true"); err != nil {
				t.Fatalf("set platform flag: %v", err)
			}
			if err := mcpServeCmd.Flags().Set(tt.flag, tt.value); err != nil {
				t.Fatalf("set %s flag: %v", tt.flag, err)
			}

			err := mcpServeCmd.RunE(mcpServeCmd, nil)
			if err == nil {
				t.Fatal("mcp serve --platform with local tool selection returned nil error")
			}
			if !strings.Contains(err.Error(), "--tools and --disable-tools apply only to local MCP mode") {
				t.Fatalf("error = %q, want local MCP mode rejection", err)
			}
		})
	}
}

func resetMCPServeFlagsForTest(t *testing.T) {
	t.Helper()
	flags := mcpServeCmd.Flags()
	for _, name := range []string{"platform", "tools", "disable-tools"} {
		flag := flags.Lookup(name)
		if flag == nil {
			t.Fatalf("missing %s flag", name)
		}
		if err := flag.Value.Set(flag.DefValue); err != nil {
			t.Fatalf("reset %s flag: %v", name, err)
		}
		flag.Changed = false
	}
	mcpPlatform = false
	mcpEnabledTools = nil
	mcpDisabledTools = nil
	t.Cleanup(func() {
		for _, name := range []string{"platform", "tools", "disable-tools"} {
			flag := flags.Lookup(name)
			_ = flag.Value.Set(flag.DefValue)
			flag.Changed = false
		}
		mcpPlatform = false
		mcpEnabledTools = nil
		mcpDisabledTools = nil
	})
}
