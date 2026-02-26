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
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/clidey/whodb/cli/pkg/version"
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
	if version.Version == "" {
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
		if !strings.Contains(defaultInstructions, tool) {
			t.Errorf("defaultInstructions should mention %s", tool)
		}
	}
}

// Security options tests

func TestNewServer_SecurityDefaults(t *testing.T) {
	// Test that NewServer(nil) creates a server with safe defaults (confirm-writes mode)
	server := NewServer(nil)
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}

	// NewServer(nil) should enable confirm-writes by default for safe operation
	// The server is created successfully - we can't inspect internal state directly,
	// but the test confirms NewServer handles nil opts without panicking
}

func TestNewServer_WithSecurityOptions(t *testing.T) {
	server := NewServer(&ServerOptions{
		ReadOnly:      false,
		ConfirmWrites: true,
		SecurityLevel: "strict",
		QueryTimeout:  60 * 1e9, // 60 seconds
		MaxRows:       500,
	})
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
}

func TestBuildQueryDescription(t *testing.T) {
	// Test read-only mode description
	readOnlyOpts := &SecurityOptions{ReadOnly: true}
	desc := buildQueryDescription(readOnlyOpts)
	if !strings.Contains(desc, "READ-ONLY") {
		t.Error("Read-only description should contain 'READ-ONLY'")
	}

	// Test confirm-writes mode description
	confirmOpts := &SecurityOptions{ReadOnly: false, ConfirmWrites: true}
	desc = buildQueryDescription(confirmOpts)
	if !strings.Contains(desc, "confirmation") {
		t.Error("Confirm-writes description should mention 'confirmation'")
	}

	// Test allow-write mode description
	writeOpts := &SecurityOptions{ReadOnly: false, ConfirmWrites: false}
	desc = buildQueryDescription(writeOpts)
	if strings.Contains(desc, "READ-ONLY") || strings.Contains(desc, "confirmation") {
		t.Error("Allow-write description should not mention read-only or confirmation")
	}
}

// Tool enablement tests

func TestToolEnablement_AllEnabledByDefault(t *testing.T) {
	te := &ToolEnablement{}

	tools := []string{"query", "schemas", "tables", "columns", "connections", "confirm"}
	for _, tool := range tools {
		if !te.isToolEnabled(tool) {
			t.Errorf("Tool %s should be enabled by default", tool)
		}
	}
}

func TestToolEnablement_OnlySpecificEnabled(t *testing.T) {
	te := &ToolEnablement{
		EnabledTools: []string{"query", "schemas"},
	}

	if !te.isToolEnabled("query") {
		t.Error("Tool 'query' should be enabled")
	}
	if !te.isToolEnabled("schemas") {
		t.Error("Tool 'schemas' should be enabled")
	}
	if te.isToolEnabled("tables") {
		t.Error("Tool 'tables' should be disabled when not in EnabledTools list")
	}
	if te.isToolEnabled("connections") {
		t.Error("Tool 'connections' should be disabled when not in EnabledTools list")
	}
}

func TestToolEnablement_DisabledTakesPrecedence(t *testing.T) {
	te := &ToolEnablement{
		EnabledTools:  []string{"query", "schemas", "tables"},
		DisabledTools: []string{"query"},
	}

	if te.isToolEnabled("query") {
		t.Error("Tool 'query' should be disabled (DisabledTools takes precedence)")
	}
	if !te.isToolEnabled("schemas") {
		t.Error("Tool 'schemas' should be enabled")
	}
}

func TestToolEnablement_DisableOnlySpecific(t *testing.T) {
	te := &ToolEnablement{
		DisabledTools: []string{"confirm"},
	}

	if !te.isToolEnabled("query") {
		t.Error("Tool 'query' should be enabled (not in disabled list)")
	}
	if te.isToolEnabled("confirm") {
		t.Error("Tool 'confirm' should be disabled")
	}
}

func TestNewServer_WithToolEnablement(t *testing.T) {
	server := NewServer(&ServerOptions{
		EnabledTools: []string{"schemas", "tables", "columns"},
	})
	if server == nil {
		t.Fatal("NewServer() returned nil with tool enablement")
	}
}

func TestNewServer_WithDisabledTools(t *testing.T) {
	server := NewServer(&ServerOptions{
		DisabledTools: []string{"query", "confirm"},
	})
	if server == nil {
		t.Fatal("NewServer() returned nil with disabled tools")
	}
}

// Helper function tests

func TestBoolPtr(t *testing.T) {
	truePtr := boolPtr(true)
	falsePtr := boolPtr(false)

	if truePtr == nil || *truePtr != true {
		t.Error("boolPtr(true) should return pointer to true")
	}
	if falsePtr == nil || *falsePtr != false {
		t.Error("boolPtr(false) should return pointer to false")
	}
}

// Prompt content tests

func TestBuildQueryHelpContent(t *testing.T) {
	// Test basic content
	content := buildQueryHelpContent("", "")
	if !strings.Contains(content, "LIMIT") {
		t.Error("Query help should mention LIMIT")
	}
	if !strings.Contains(content, "WhoDB") {
		t.Error("Query help should mention WhoDB")
	}

	// Test with database type
	pgContent := buildQueryHelpContent("postgres", "")
	if !strings.Contains(pgContent, "ILIKE") || !strings.Contains(pgContent, "jsonb") {
		t.Error("Postgres-specific help should mention ILIKE and jsonb")
	}

	mysqlContent := buildQueryHelpContent("mysql", "")
	if !strings.Contains(mysqlContent, "BINARY") {
		t.Error("MySQL-specific help should mention BINARY")
	}

	// Test with query type
	selectContent := buildQueryHelpContent("", "select")
	if !strings.Contains(selectContent, "SELECT") {
		t.Error("Select query help should contain SELECT examples")
	}

	joinContent := buildQueryHelpContent("", "join")
	if !strings.Contains(joinContent, "JOIN") {
		t.Error("Join query help should contain JOIN examples")
	}
}

func TestBuildWorkflowHelpContent(t *testing.T) {
	// Test analysis workflow
	analysisContent := buildWorkflowHelpContent("analysis")
	if !strings.Contains(analysisContent, "Data Analysis") {
		t.Error("Analysis workflow should mention 'Data Analysis'")
	}

	// Test debugging workflow
	debugContent := buildWorkflowHelpContent("debugging")
	if !strings.Contains(debugContent, "Debugging") {
		t.Error("Debugging workflow should mention 'Debugging'")
	}

	// Test relationships workflow
	relContent := buildWorkflowHelpContent("relationships")
	if !strings.Contains(relContent, "foreign") {
		t.Error("Relationships workflow should mention foreign keys")
	}

	// Test default workflow
	defaultContent := buildWorkflowHelpContent("unknown")
	if !strings.Contains(defaultContent, "Schema Exploration") {
		t.Error("Default workflow should list available workflows")
	}
}

// Connection allowlist tests

func TestIsConnectionAllowed_NoRestrictions(t *testing.T) {
	secOpts := &SecurityOptions{
		AllowedConnections: nil, // No restrictions
	}

	// Any connection should be allowed
	if !secOpts.isConnectionAllowed("prod") {
		t.Error("With no restrictions, 'prod' should be allowed")
	}
	if !secOpts.isConnectionAllowed("staging") {
		t.Error("With no restrictions, 'staging' should be allowed")
	}
	if !secOpts.isConnectionAllowed("") {
		t.Error("With no restrictions, empty connection should be allowed")
	}
}

func TestIsConnectionAllowed_EmptySlice(t *testing.T) {
	secOpts := &SecurityOptions{
		AllowedConnections: []string{}, // Empty slice = no restrictions
	}

	if !secOpts.isConnectionAllowed("any") {
		t.Error("With empty AllowedConnections, any connection should be allowed")
	}
}

func TestIsConnectionAllowed_SingleConnection(t *testing.T) {
	secOpts := &SecurityOptions{
		AllowedConnections: []string{"prod"},
	}

	if !secOpts.isConnectionAllowed("prod") {
		t.Error("'prod' should be allowed when it's in the list")
	}
	if secOpts.isConnectionAllowed("staging") {
		t.Error("'staging' should NOT be allowed when only 'prod' is in the list")
	}
	if secOpts.isConnectionAllowed("") {
		t.Error("Empty connection should NOT be allowed when restrictions are set")
	}
}

func TestIsConnectionAllowed_MultipleConnections(t *testing.T) {
	secOpts := &SecurityOptions{
		AllowedConnections: []string{"prod", "staging", "dev"},
	}

	if !secOpts.isConnectionAllowed("prod") {
		t.Error("'prod' should be allowed")
	}
	if !secOpts.isConnectionAllowed("staging") {
		t.Error("'staging' should be allowed")
	}
	if !secOpts.isConnectionAllowed("dev") {
		t.Error("'dev' should be allowed")
	}
	if secOpts.isConnectionAllowed("test") {
		t.Error("'test' should NOT be allowed")
	}
}

func TestNewServer_AllowedConnectionsSetsDefault(t *testing.T) {
	// When AllowedConnections is set and DefaultConnection is not,
	// the first allowed connection becomes the default
	server := NewServer(&ServerOptions{
		AllowedConnections: []string{"staging", "prod"},
	})
	if server == nil {
		t.Fatal("NewServer() returned nil with AllowedConnections")
	}
	// We can't directly inspect secOpts, but the server should be created successfully
}

func TestNewServer_ExplicitDefaultOverridesFirst(t *testing.T) {
	// When both DefaultConnection and AllowedConnections are set,
	// DefaultConnection takes precedence
	server := NewServer(&ServerOptions{
		DefaultConnection:  "prod",
		AllowedConnections: []string{"staging", "dev"},
	})
	if server == nil {
		t.Fatal("NewServer() returned nil with explicit DefaultConnection")
	}
}
