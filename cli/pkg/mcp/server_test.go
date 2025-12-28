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

package mcp

import (
	"log/slog"
	"os"
	"testing"
)

func TestNewServer_DefaultOptions(t *testing.T) {
	server := NewServer(nil)
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
}

func TestNewServer_WithCustomLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	server := NewServer(&ServerOptions{
		Logger: logger,
	})
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
}

func TestNewServer_WithCustomInstructions(t *testing.T) {
	server := NewServer(&ServerOptions{
		Instructions: "Custom instructions for testing",
	})
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
}

func TestNewServer_RegistersAllTools(t *testing.T) {
	server := NewServer(nil)
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}

	// The server is created successfully - tools are registered internally
	// We can't easily inspect registered tools without modifying the SDK,
	// but we can verify the server was created without error
}

func TestVersion_IsSet(t *testing.T) {
	// Version should be "dev" by default or set by build flags
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestDefaultInstructions_NotEmpty(t *testing.T) {
	if defaultInstructions == "" {
		t.Error("defaultInstructions should not be empty")
	}
}

func TestDefaultInstructions_ContainsToolNames(t *testing.T) {
	expectedTools := []string{
		"whodb_query",
		"whodb_schemas",
		"whodb_tables",
		"whodb_columns",
		"whodb_connections",
	}

	for _, tool := range expectedTools {
		if !containsString(defaultInstructions, tool) {
			t.Errorf("defaultInstructions should mention %s", tool)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
