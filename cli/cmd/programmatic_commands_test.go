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
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// setupTestEnv creates an isolated test environment
func setupTestEnv(t *testing.T) func() {
	t.Helper()
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	return func() {
		os.Setenv("HOME", origHome)
	}
}

// TestSchemasCmd_Exists verifies the schemas command is registered
func TestSchemasCmd_Exists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "schemas" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'schemas' command to be registered")
	}
}

// TestSchemasCmd_Flags verifies the schemas command has expected flags
func TestSchemasCmd_Flags(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	flags := []string{"connection", "format", "quiet"}
	for _, flag := range flags {
		if schemasCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected '--%s' flag on schemas command", flag)
		}
	}
}

// TestTablesCmd_Exists verifies the tables command is registered
func TestTablesCmd_Exists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "tables" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'tables' command to be registered")
	}
}

// TestTablesCmd_Flags verifies the tables command has expected flags
func TestTablesCmd_Flags(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	flags := []string{"connection", "schema", "format", "quiet"}
	for _, flag := range flags {
		if tablesCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected '--%s' flag on tables command", flag)
		}
	}
}

// TestColumnsCmd_Exists verifies the columns command is registered
func TestColumnsCmd_Exists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "columns" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'columns' command to be registered")
	}
}

// TestColumnsCmd_Flags verifies the columns command has expected flags
func TestColumnsCmd_Flags(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	flags := []string{"connection", "schema", "table", "format", "quiet"}
	for _, flag := range flags {
		if columnsCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected '--%s' flag on columns command", flag)
		}
	}
}

// TestColumnsCmd_RequiresTable verifies --table is required
func TestColumnsCmd_RequiresTable(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Reset flag values
	columnsTable = ""
	columnsConnection = ""

	err := columnsCmd.RunE(columnsCmd, []string{})
	if err == nil {
		t.Error("Expected error when --table is not provided")
	}
	if !strings.Contains(err.Error(), "--table") {
		t.Errorf("Expected error message to mention --table, got: %v", err)
	}
}

// TestConnectionsCmd_Exists verifies the connections command is registered
func TestConnectionsCmd_Exists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "connections" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'connections' command to be registered")
	}
}

// TestConnectionsCmd_HasSubcommands verifies connections has expected subcommands
func TestConnectionsCmd_HasSubcommands(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	subcommands := map[string]bool{
		"list":   false,
		"add":    false,
		"remove": false,
		"test":   false,
	}

	for _, cmd := range connectionsCmd.Commands() {
		// Use Name() instead of Use since Use may include args like "remove [name]"
		if _, ok := subcommands[cmd.Name()]; ok {
			subcommands[cmd.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("Expected 'connections %s' subcommand", name)
		}
	}
}

// TestConnectionsListCmd_EmptyJSON verifies list returns empty array when no connections
func TestConnectionsListCmd_EmptyJSON(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Set format to JSON
	connectionsFormat = "json"
	connectionsQuiet = true

	// Capture stdout since the empty array is printed via fmt.Println
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := connectionsListCmd.RunE(connectionsListCmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := strings.TrimSpace(buf.String())

	if output != "[]" {
		t.Errorf("Expected empty JSON array, got: '%s'", output)
	}
}

// TestConnectionsAddCmd_RequiresName verifies --name is required
func TestConnectionsAddCmd_RequiresName(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Reset flags
	connAddName = ""
	connAddType = "Postgres"
	connAddDatabase = "test"

	err := connectionsAddCmd.RunE(connectionsAddCmd, []string{})
	if err == nil {
		t.Error("Expected error when --name is not provided")
	}
	if !strings.Contains(err.Error(), "--name") {
		t.Errorf("Expected error message to mention --name, got: %v", err)
	}
}

// TestConnectionsAddCmd_RequiresType verifies --type is required
func TestConnectionsAddCmd_RequiresType(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Reset flags
	connAddName = "test"
	connAddType = ""
	connAddDatabase = "test"

	err := connectionsAddCmd.RunE(connectionsAddCmd, []string{})
	if err == nil {
		t.Error("Expected error when --type is not provided")
	}
	if !strings.Contains(err.Error(), "--type") {
		t.Errorf("Expected error message to mention --type, got: %v", err)
	}
}

// TestConnectionsAddCmd_RequiresDatabase verifies --database is required
func TestConnectionsAddCmd_RequiresDatabase(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Reset flags
	connAddName = "test"
	connAddType = "Postgres"
	connAddHost = "localhost"
	connAddDatabase = ""

	err := connectionsAddCmd.RunE(connectionsAddCmd, []string{})
	if err == nil {
		t.Error("Expected error when --database is not provided")
	}
	if !strings.Contains(err.Error(), "--database") {
		t.Errorf("Expected error message to mention --database, got: %v", err)
	}
}

// TestConnectionsAddAndList verifies add then list workflow
func TestConnectionsAddAndList(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Add a connection
	connAddName = "test-conn"
	connAddType = "Postgres"
	connAddHost = "localhost"
	connAddPort = 5432
	connAddUser = "testuser"
	connAddPassword = "testpass"
	connAddDatabase = "testdb"
	connAddSchema = "public"
	connectionsQuiet = true

	err := connectionsAddCmd.RunE(connectionsAddCmd, []string{})
	if err != nil {
		t.Fatalf("Failed to add connection: %v", err)
	}

	// List connections
	connectionsFormat = "json"
	var buf bytes.Buffer
	connectionsListCmd.SetOut(&buf)

	err = connectionsListCmd.RunE(connectionsListCmd, []string{})
	if err != nil {
		t.Fatalf("Failed to list connections: %v", err)
	}

	// Parse JSON output
	var conns []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &conns); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if len(conns) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(conns))
	}

	if conns[0]["name"] != "test-conn" {
		t.Errorf("Expected connection name 'test-conn', got %v", conns[0]["name"])
	}

	// Verify password is not included in output
	if _, hasPassword := conns[0]["password"]; hasPassword {
		t.Error("Password should not be included in list output")
	}
}

// TestConnectionsRemoveCmd_RequiresArg verifies remove requires a connection name
func TestConnectionsRemoveCmd_RequiresArg(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	err := connectionsRemoveCmd.Args(connectionsRemoveCmd, []string{})
	if err == nil {
		t.Error("Expected error when connection name is not provided")
	}
}

// TestConnectionsRemoveCmd_NotFound verifies remove fails for non-existent connection
func TestConnectionsRemoveCmd_NotFound(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	err := connectionsRemoveCmd.RunE(connectionsRemoveCmd, []string{"nonexistent"})
	if err == nil {
		t.Error("Expected error for non-existent connection")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' in error, got: %v", err)
	}
}

// TestExportCmd_Exists verifies the export command is registered
func TestExportCmd_Exists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "export" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'export' command to be registered")
	}
}

// TestExportCmd_Flags verifies the export command has expected flags
func TestExportCmd_Flags(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	flags := []string{"connection", "schema", "table", "query", "format", "output", "delimiter", "quiet"}
	for _, flag := range flags {
		if exportCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected '--%s' flag on export command", flag)
		}
	}
}

// TestExportCmd_RequiresTableOrQuery verifies either --table or --query is required
func TestExportCmd_RequiresTableOrQuery(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Reset flags
	exportTable = ""
	exportQuery = ""
	exportOutput = "test.csv"

	err := exportCmd.RunE(exportCmd, []string{})
	if err == nil {
		t.Error("Expected error when neither --table nor --query is provided")
	}
	if !strings.Contains(err.Error(), "--table") && !strings.Contains(err.Error(), "--query") {
		t.Errorf("Expected error message to mention --table or --query, got: %v", err)
	}
}

// TestExportCmd_CannotUseBothTableAndQuery verifies mutual exclusivity
func TestExportCmd_CannotUseBothTableAndQuery(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Set both
	exportTable = "users"
	exportQuery = "SELECT * FROM users"
	exportOutput = "test.csv"

	err := exportCmd.RunE(exportCmd, []string{})
	if err == nil {
		t.Error("Expected error when both --table and --query are provided")
	}
	if !strings.Contains(err.Error(), "cannot use both") {
		t.Errorf("Expected 'cannot use both' in error, got: %v", err)
	}
}

// TestExportCmd_RequiresOutput verifies --output is required
func TestExportCmd_RequiresOutput(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Reset flags
	exportTable = "users"
	exportQuery = ""
	exportOutput = ""

	err := exportCmd.RunE(exportCmd, []string{})
	if err == nil {
		t.Error("Expected error when --output is not provided")
	}
	if !strings.Contains(err.Error(), "--output") {
		t.Errorf("Expected error message to mention --output, got: %v", err)
	}
}

// TestHistoryCmd_Exists verifies the history command is registered
func TestHistoryCmd_Exists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "history" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'history' command to be registered")
	}
}

// TestHistoryCmd_HasSubcommands verifies history has expected subcommands
func TestHistoryCmd_HasSubcommands(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	subcommands := map[string]bool{
		"list":   false,
		"search": false,
		"clear":  false,
	}

	for _, cmd := range historyCmd.Commands() {
		// Use Name() instead of Use since Use may include args like "search [pattern]"
		if _, ok := subcommands[cmd.Name()]; ok {
			subcommands[cmd.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("Expected 'history %s' subcommand", name)
		}
	}
}

// TestHistoryListCmd_EmptyJSON verifies list returns empty array when no history
func TestHistoryListCmd_EmptyJSON(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	// Set format to JSON
	historyFormat = "json"
	historyQuiet = true

	// Capture stdout since the empty array is printed via fmt.Println
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := historyListCmd.RunE(historyListCmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := strings.TrimSpace(buf.String())

	if output != "[]" {
		t.Errorf("Expected empty JSON array, got: %s", output)
	}
}

// TestHistorySearchCmd_RequiresPattern verifies search requires a pattern argument
func TestHistorySearchCmd_RequiresPattern(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	err := historySearchCmd.Args(historySearchCmd, []string{})
	if err == nil {
		t.Error("Expected error when pattern is not provided")
	}
}

// TestHistoryClearCmd_NoError verifies clear works even with empty history
func TestHistoryClearCmd_NoError(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	historyQuiet = true

	err := historyClearCmd.RunE(historyClearCmd, []string{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

// TestAllNewCommandsRegistered verifies all new commands are registered on rootCmd
func TestAllNewCommandsRegistered(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	expectedCommands := []string{
		"schemas",
		"tables",
		"columns",
		"connections",
		"export",
		"history",
	}

	registeredCommands := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		registeredCommands[cmd.Use] = true
	}

	for _, expected := range expectedCommands {
		if !registeredCommands[expected] {
			t.Errorf("Expected '%s' command to be registered", expected)
		}
	}
}
