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
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/clidey/whodb/cli/internal/agentmanifest"
	"github.com/clidey/whodb/cli/internal/config"
	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/internal/doctor"
	"github.com/clidey/whodb/cli/internal/history"
	"github.com/clidey/whodb/cli/internal/runbooks"
	"github.com/clidey/whodb/cli/internal/schemadiff"
	"github.com/clidey/whodb/cli/internal/skillinstaller"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/providers"
	"github.com/spf13/cobra"
)

type automationEnvelopeResult[T any] struct {
	Command string `json:"command"`
	Success bool   `json:"success"`
	Data    T      `json:"data"`
}

func setCommandBuffers(t *testing.T, cmd *cobra.Command) (*bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	t.Cleanup(func() {
		cmd.SetOut(os.Stdout)
		cmd.SetErr(os.Stderr)
	})

	return &out, &errOut
}

func decodeJSONEnvelope[T any](t *testing.T, buf *bytes.Buffer) automationEnvelopeResult[T] {
	t.Helper()

	var envelope automationEnvelopeResult[T]
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("Failed to parse JSON envelope: %v", err)
	}

	return envelope
}

func createSQLiteTestDatabase(t *testing.T, filename string, queries ...string) string {
	t.Helper()

	if err := os.Setenv("WHODB_CLI", "true"); err != nil {
		t.Fatalf("Failed to set WHODB_CLI: %v", err)
	}

	dbPath := filepath.Join(t.TempDir(), filename)
	file, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Failed to close SQLite file: %v", err)
	}

	mgr, err := dbmgr.NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	conn := &config.Connection{
		Name:     "test-sqlite",
		Type:     "Sqlite3",
		Host:     dbPath,
		Database: dbPath,
	}
	if err := mgr.Connect(conn); err != nil {
		t.Skipf("SQLite not available: %v", err)
	}

	for _, query := range queries {
		if _, err := mgr.ExecuteQuery(query); err != nil {
			_ = mgr.Disconnect()
			t.Fatalf("Failed to seed SQLite database: %v", err)
		}
	}

	if err := mgr.Disconnect(); err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}

	return dbPath
}

func saveTestConnection(t *testing.T, conn config.Connection) {
	t.Helper()

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	cfg.AddConnection(conn)
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
}

type commandTestProvider struct {
	id          string
	name        string
	connections []providers.DiscoveredConnection
}

func (p *commandTestProvider) Type() providers.ProviderType { return providers.ProviderTypeAWS }

func (p *commandTestProvider) ID() string { return p.id }

func (p *commandTestProvider) Name() string { return p.name }

func (p *commandTestProvider) DiscoverConnections(ctx context.Context) ([]providers.DiscoveredConnection, error) {
	return append([]providers.DiscoveredConnection(nil), p.connections...), nil
}

func (p *commandTestProvider) TestConnection(ctx context.Context) error { return nil }

func (p *commandTestProvider) RefreshConnection(ctx context.Context, connectionID string) (bool, error) {
	return false, nil
}

func (p *commandTestProvider) Close(ctx context.Context) error { return nil }

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

func TestConnectCmd_HelpIncludesAlphaWarning(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	if !strings.Contains(connectCmd.Long, "ALPHA WARNING:") {
		t.Fatalf("expected connect help to include alpha warning, got %q", connectCmd.Long)
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
		t.Fatal("Expected error when --table is not provided")
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

func TestConnectionsListCmd_EnvProfileJSON(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	os.Setenv("WHODB_POSTGRES", `[{"alias":"prod","host":"localhost","user":"user","password":"pass","database":"db","port":"5432"}]`)

	connectionsFormat = "json"
	connectionsQuiet = true

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

	var conns []map[string]any
	if err := json.Unmarshal([]byte(output), &conns); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if len(conns) != 1 {
		t.Fatalf("Expected 1 connection, got %d", len(conns))
	}

	conn := conns[0]
	if conn["name"] != "prod" {
		t.Errorf("Expected name 'prod', got %v", conn["name"])
	}
	if conn["type"] != "Postgres" {
		t.Errorf("Expected type 'Postgres', got %v", conn["type"])
	}
	if conn["host"] != "localhost" {
		t.Errorf("Expected host 'localhost', got %v", conn["host"])
	}
	if conn["database"] != "db" {
		t.Errorf("Expected database 'db', got %v", conn["database"])
	}
	if conn["source"] != "env" {
		t.Errorf("Expected source 'env', got %v", conn["source"])
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
		t.Fatal("Expected error when --name is not provided")
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
		t.Fatal("Expected error when --type is not provided")
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
		t.Fatal("Expected error when --database is not provided")
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
	connAddPassword = ""
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

func TestConnectionsAddCmd_JSONEnvelope(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	connectionsFormat = "json"
	connectionsQuiet = false
	connAddName = "json-conn"
	connAddType = "Postgres"
	connAddHost = "localhost"
	connAddPort = 5432
	connAddUser = "testuser"
	connAddPassword = ""
	connAddDatabase = "testdb"
	connAddSchema = "public"

	outBuf, errBuf := setCommandBuffers(t, connectionsAddCmd)

	err := connectionsAddCmd.RunE(connectionsAddCmd, []string{})
	if err != nil {
		t.Fatalf("Failed to add connection: %v", err)
	}

	envelope := decodeJSONEnvelope[safeConnectionOutput](t, outBuf)
	if envelope.Command != "connections.add" {
		t.Errorf("Expected command connections.add, got %q", envelope.Command)
	}
	if !envelope.Success {
		t.Error("Expected success to be true")
	}
	if envelope.Data.Name != "json-conn" {
		t.Errorf("Expected name json-conn, got %q", envelope.Data.Name)
	}
	if envelope.Data.Source != "config" {
		t.Errorf("Expected source config, got %q", envelope.Data.Source)
	}
	if errBuf.Len() != 0 {
		t.Errorf("Expected no stderr output, got %q", errBuf.String())
	}
}

func TestConnectionsAddCmd_SSLAdvanced(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	defer func() {
		connAddName = ""
		connAddType = ""
		connAddHost = ""
		connAddPort = 0
		connAddUser = ""
		connAddPassword = ""
		connAddDatabase = ""
		connAddSchema = ""
		connAddSSLMode = ""
		connAddSSLCA = ""
		connAddSSLCert = ""
		connAddSSLKey = ""
		connAddSSLServerName = ""
		connectionsFormat = "auto"
		connectionsQuiet = false
	}()

	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	certPath := filepath.Join(dir, "client-cert.pem")
	keyPath := filepath.Join(dir, "client-key.pem")
	if err := os.WriteFile(caPath, []byte("ca-content"), 0o600); err != nil {
		t.Fatalf("Write CA file failed: %v", err)
	}
	if err := os.WriteFile(certPath, []byte("cert-content"), 0o600); err != nil {
		t.Fatalf("Write client cert file failed: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("key-content"), 0o600); err != nil {
		t.Fatalf("Write client key file failed: %v", err)
	}

	connectionsFormat = "json"
	connectionsQuiet = false
	connAddName = "ssl-db"
	connAddType = "Postgres"
	connAddHost = "localhost"
	connAddPort = 5432
	connAddUser = "alice"
	connAddPassword = ""
	connAddDatabase = "app"
	connAddSchema = "public"
	connAddSSLMode = "verify-identity"
	connAddSSLCA = caPath
	connAddSSLCert = certPath
	connAddSSLKey = keyPath
	connAddSSLServerName = "db.internal"

	outBuf, errBuf := setCommandBuffers(t, connectionsAddCmd)

	if err := connectionsAddCmd.RunE(connectionsAddCmd, []string{}); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	conn, err := cfg.GetConnection("ssl-db")
	if err != nil {
		t.Fatalf("expected saved connection: %v", err)
	}

	if conn.Advanced["SSL Mode"] != "verify-identity" {
		t.Fatalf("expected SSL mode verify-identity, got %#v", conn.Advanced)
	}
	if conn.Advanced["SSL CA Content"] != "ca-content" {
		t.Fatalf("expected CA content, got %#v", conn.Advanced)
	}
	if conn.Advanced["SSL Client Cert Content"] != "cert-content" {
		t.Fatalf("expected client cert content, got %#v", conn.Advanced)
	}
	if conn.Advanced["SSL Client Key Content"] != "key-content" {
		t.Fatalf("expected client key content, got %#v", conn.Advanced)
	}
	if conn.Advanced["SSL Server Name"] != "db.internal" {
		t.Fatalf("expected server name, got %#v", conn.Advanced)
	}

	envelope := decodeJSONEnvelope[safeConnectionOutput](t, outBuf)
	if envelope.Command != "connections.add" {
		t.Errorf("Expected command connections.add, got %q", envelope.Command)
	}
	if errBuf.Len() != 0 {
		t.Errorf("Expected no stderr output, got %q", errBuf.String())
	}
}

func TestConnectionsAddCmd_FromDiscovered(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	defer func() {
		connAddName = ""
		connAddType = ""
		connAddHost = ""
		connAddPort = 0
		connAddUser = ""
		connAddPassword = ""
		connAddDatabase = ""
		connAddSchema = ""
		connAddFromDiscovered = ""
		connAddSSLMode = ""
		connAddSSLCA = ""
		connAddSSLCert = ""
		connAddSSLKey = ""
		connAddSSLServerName = ""
		connectionsFormat = "auto"
		connectionsQuiet = false
		_ = providers.GetDefaultRegistry().Close(context.Background())
	}()

	originalAWS := env.IsAWSProviderEnabled
	originalAzure := env.IsAzureProviderEnabled
	originalGCP := env.IsGCPProviderEnabled
	t.Cleanup(func() {
		env.IsAWSProviderEnabled = originalAWS
		env.IsAzureProviderEnabled = originalAzure
		env.IsGCPProviderEnabled = originalGCP
	})
	env.IsAWSProviderEnabled = true
	env.IsAzureProviderEnabled = false
	env.IsGCPProviderEnabled = false

	if err := providers.GetDefaultRegistry().Register(&commandTestProvider{
		id:   "aws-prod",
		name: "AWS Prod",
		connections: []providers.DiscoveredConnection{{
			ID:           "aws-prod/prod-db",
			ProviderType: providers.ProviderTypeAWS,
			ProviderID:   "aws-prod",
			Name:         "prod-db",
			DatabaseType: engine.DatabaseType_Postgres,
			Status:       providers.ConnectionStatusAvailable,
			Metadata: map[string]string{
				"endpoint": "prod-db.example.com",
				"port":     "5432",
			},
		}},
	}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	connectionsFormat = "json"
	connAddName = ""
	connAddType = ""
	connAddUser = "alice"
	connAddPassword = ""
	connAddDatabase = "app"
	connAddFromDiscovered = "aws-prod/prod-db"

	outBuf, errBuf := setCommandBuffers(t, connectionsAddCmd)

	if err := connectionsAddCmd.RunE(connectionsAddCmd, []string{}); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	conn, err := cfg.GetConnection("prod-db")
	if err != nil {
		t.Fatalf("expected saved connection: %v", err)
	}

	if conn.Host != "prod-db.example.com" || conn.Port != 5432 {
		t.Fatalf("expected discovered host/port, got %#v", conn)
	}
	if conn.Advanced["SSL Mode"] != "require" {
		t.Fatalf("expected discovered SSL prefill, got %#v", conn.Advanced)
	}

	envelope := decodeJSONEnvelope[safeConnectionOutput](t, outBuf)
	if envelope.Data.Name != "prod-db" {
		t.Fatalf("expected discovered name to be used, got %#v", envelope.Data)
	}
	if errBuf.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", errBuf.String())
	}
}

func TestConnectionsRemoveCmd_JSONEnvelope(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	saveTestConnection(t, config.Connection{
		Name:     "remove-me",
		Type:     "Postgres",
		Host:     "localhost",
		Port:     5432,
		Username: "testuser",
		Database: "testdb",
	})

	connectionsFormat = "json"
	connectionsQuiet = false

	outBuf, errBuf := setCommandBuffers(t, connectionsRemoveCmd)

	err := connectionsRemoveCmd.RunE(connectionsRemoveCmd, []string{"remove-me"})
	if err != nil {
		t.Fatalf("Failed to remove connection: %v", err)
	}

	envelope := decodeJSONEnvelope[struct {
		Name string `json:"name"`
	}](t, outBuf)
	if envelope.Command != "connections.remove" {
		t.Errorf("Expected command connections.remove, got %q", envelope.Command)
	}
	if envelope.Data.Name != "remove-me" {
		t.Errorf("Expected removed name remove-me, got %q", envelope.Data.Name)
	}
	if errBuf.Len() != 0 {
		t.Errorf("Expected no stderr output, got %q", errBuf.String())
	}
}

func TestConnectionsTestCmd_JSONEnvelope(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	dbPath := createSQLiteTestDatabase(t,
		"connections-test.db",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
	)
	saveTestConnection(t, config.Connection{
		Name:     "sqlite-json",
		Type:     "Sqlite3",
		Host:     dbPath,
		Database: dbPath,
	})

	connectionsFormat = "json"
	connectionsQuiet = false

	outBuf, errBuf := setCommandBuffers(t, connectionsTestCmd)

	err := connectionsTestCmd.RunE(connectionsTestCmd, []string{"sqlite-json"})
	if err != nil {
		t.Fatalf("Failed to test connection: %v", err)
	}

	envelope := decodeJSONEnvelope[connectionTestOutput](t, outBuf)
	if envelope.Command != "connections.test" {
		t.Errorf("Expected command connections.test, got %q", envelope.Command)
	}
	if envelope.Data.Connection.Name != "sqlite-json" {
		t.Errorf("Expected connection name sqlite-json, got %q", envelope.Data.Connection.Name)
	}
	if envelope.Data.Connection.Type != "Sqlite3" {
		t.Errorf("Expected type Sqlite3, got %q", envelope.Data.Connection.Type)
	}
	if errBuf.Len() != 0 {
		t.Errorf("Expected no stderr output, got %q", errBuf.String())
	}
}

// TestConnectionsRemoveCmd_RequiresArg verifies remove requires a connection name
func TestConnectionsRemoveCmd_RequiresArg(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	err := connectionsRemoveCmd.Args(connectionsRemoveCmd, []string{})
	if err == nil {
		t.Fatal("Expected error when connection name is not provided")
	}
}

// TestConnectionsRemoveCmd_NotFound verifies remove fails for non-existent connection
func TestConnectionsRemoveCmd_NotFound(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	err := connectionsRemoveCmd.RunE(connectionsRemoveCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("Expected error for non-existent connection")
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
		t.Fatal("Expected error when neither --table nor --query is provided")
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
		t.Fatal("Expected error when both --table and --query are provided")
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
		t.Fatal("Expected error when --output is not provided")
	}
	if !strings.Contains(err.Error(), "--output") {
		t.Errorf("Expected error message to mention --output, got: %v", err)
	}
}

func TestDiffCmd_Exists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "diff" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'diff' command to be registered")
	}
}

func TestDiffCmd_Flags(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	flags := []string{"from", "to", "schema", "from-schema", "to-schema", "format", "quiet"}
	for _, flag := range flags {
		if diffCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected '--%s' flag on diff command", flag)
		}
	}
}

func TestDiffCmd_JSONEnvelope(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	fromDB := createSQLiteTestDatabase(t,
		"diff-from.db",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)",
		"CREATE TABLE legacy (id INTEGER PRIMARY KEY, note TEXT)",
	)
	toDB := createSQLiteTestDatabase(t,
		"diff-to.db",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT NOT NULL, status TEXT)",
		"CREATE TABLE audit_log (id INTEGER PRIMARY KEY, action TEXT)",
	)

	saveTestConnection(t, config.Connection{
		Name:     "diff-from",
		Type:     "Sqlite3",
		Host:     fromDB,
		Database: fromDB,
	})
	saveTestConnection(t, config.Connection{
		Name:     "diff-to",
		Type:     "Sqlite3",
		Host:     toDB,
		Database: toDB,
	})

	diffFromConnection = "diff-from"
	diffToConnection = "diff-to"
	diffSchema = ""
	diffFromSchema = ""
	diffToSchema = ""
	diffFormat = "json"
	diffQuiet = false

	outBuf, errBuf := setCommandBuffers(t, diffCmd)

	err := diffCmd.RunE(diffCmd, []string{})
	if err != nil {
		t.Fatalf("Diff command failed: %v", err)
	}

	envelope := decodeJSONEnvelope[schemadiff.Result](t, outBuf)
	if envelope.Command != "diff" {
		t.Errorf("Expected command diff, got %q", envelope.Command)
	}
	if !envelope.Data.Summary.HasDifferences {
		t.Fatal("Expected schema differences")
	}
	if envelope.Data.Summary.AddedStorageUnits != 1 {
		t.Errorf("Expected 1 added storage unit, got %d", envelope.Data.Summary.AddedStorageUnits)
	}
	if envelope.Data.Summary.RemovedStorageUnits != 1 {
		t.Errorf("Expected 1 removed storage unit, got %d", envelope.Data.Summary.RemovedStorageUnits)
	}
	if envelope.Data.Summary.ChangedStorageUnits != 1 {
		t.Errorf("Expected 1 changed storage unit, got %d", envelope.Data.Summary.ChangedStorageUnits)
	}
	if envelope.Data.Summary.AddedColumns != 3 {
		t.Errorf("Expected 3 added columns, got %d", envelope.Data.Summary.AddedColumns)
	}
	if envelope.Data.Summary.RemovedColumns != 2 {
		t.Errorf("Expected 2 removed columns, got %d", envelope.Data.Summary.RemovedColumns)
	}
	if envelope.Data.Summary.ChangedColumns != 1 {
		t.Errorf("Expected 1 changed column, got %d", envelope.Data.Summary.ChangedColumns)
	}
	if envelope.Data.Summary.AddedRelationships != 0 {
		t.Errorf("Expected 0 added relationships, got %d", envelope.Data.Summary.AddedRelationships)
	}
	if envelope.Data.Summary.RemovedRelationships != 0 {
		t.Errorf("Expected 0 removed relationships, got %d", envelope.Data.Summary.RemovedRelationships)
	}
	if envelope.Data.Summary.ChangedRelationships != 0 {
		t.Errorf("Expected 0 changed relationships, got %d", envelope.Data.Summary.ChangedRelationships)
	}
	if len(envelope.Data.StorageUnits) != 3 {
		t.Fatalf("Expected 3 storage unit diffs, got %d", len(envelope.Data.StorageUnits))
	}
	if errBuf.Len() != 0 {
		t.Errorf("Expected no stderr output, got %q", errBuf.String())
	}
}

func TestSuggestionsCmd_Exists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "suggestions" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected suggestions command to be registered")
	}
}

func TestSuggestionsCmd_JSON(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	defer func() {
		suggestionsConnection = ""
		suggestionsSchema = ""
		suggestionsFormat = "table"
		suggestionsQuiet = false
	}()

	dbPath := createSQLiteTestDatabase(t,
		"suggestions-test.db",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
		"CREATE TABLE orders (id INTEGER PRIMARY KEY, user_id INTEGER)",
		"CREATE TABLE payments (id INTEGER PRIMARY KEY, amount REAL)",
	)
	saveTestConnection(t, config.Connection{
		Name:     "suggestions-db",
		Type:     "Sqlite3",
		Host:     dbPath,
		Database: dbPath,
	})

	suggestionsConnection = "suggestions-db"
	suggestionsSchema = ""
	suggestionsFormat = "json"
	suggestionsQuiet = false

	outBuf, errBuf := setCommandBuffers(t, suggestionsCmd)

	if err := suggestionsCmd.RunE(suggestionsCmd, []string{}); err != nil {
		t.Fatalf("Suggestions command failed: %v", err)
	}

	var suggestions []struct {
		Description string `json:"description"`
		Category    string `json:"category"`
	}
	if err := json.Unmarshal(outBuf.Bytes(), &suggestions); err != nil {
		t.Fatalf("Failed to parse suggestions JSON: %v", err)
	}

	if len(suggestions) != 3 {
		t.Fatalf("Expected 3 suggestions, got %#v", suggestions)
	}
	if !strings.Contains(suggestions[0].Description, "users") {
		t.Fatalf("Expected first suggestion to mention users, got %#v", suggestions[0])
	}
	if errBuf.Len() != 0 {
		t.Errorf("Expected no stderr output, got %q", errBuf.String())
	}
}

func TestQueryCmd_NDJSON(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	defer func() {
		queryConnection = ""
		queryFormat = "auto"
		queryQuiet = false
	}()

	dbPath := createSQLiteTestDatabase(t,
		"query-ndjson-test.db",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
		"INSERT INTO users (id, name) VALUES (1, 'Alice')",
		"INSERT INTO users (id, name) VALUES (2, 'Bob')",
	)
	saveTestConnection(t, config.Connection{
		Name:     "query-ndjson-db",
		Type:     "Sqlite3",
		Host:     dbPath,
		Database: dbPath,
	})

	queryConnection = "query-ndjson-db"
	queryFormat = "ndjson"
	queryQuiet = false

	outBuf, errBuf := setCommandBuffers(t, queryCmd)

	if err := queryCmd.RunE(queryCmd, []string{"SELECT id, name FROM users ORDER BY id"}); err != nil {
		t.Fatalf("Query command failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(outBuf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 NDJSON lines, got %d: %q", len(lines), outBuf.String())
	}

	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("Failed to parse first NDJSON line: %v", err)
	}
	if first["name"] != "Alice" {
		t.Fatalf("Expected first row Alice, got %#v", first)
	}
	if errBuf.Len() != 0 {
		t.Errorf("Expected no stderr output, got %q", errBuf.String())
	}
}

func TestQueryCmd_AutoFormatSuppressesInformationalOutputWhenPiped(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	defer func() {
		queryConnection = ""
		queryFormat = "auto"
		queryQuiet = false
	}()

	dbPath := createSQLiteTestDatabase(t,
		"query-auto-format-test.db",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
		"INSERT INTO users (id, name) VALUES (1, 'Alice')",
	)
	saveTestConnection(t, config.Connection{
		Name:     "query-auto-format-db",
		Type:     "Sqlite3",
		Host:     dbPath,
		Database: dbPath,
	})

	queryConnection = "query-auto-format-db"
	queryFormat = "auto"
	queryQuiet = false

	outBuf, errBuf := setCommandBuffers(t, queryCmd)

	if err := queryCmd.RunE(queryCmd, []string{"SELECT id, name FROM users"}); err != nil {
		t.Fatalf("Query command failed: %v", err)
	}

	if !strings.Contains(outBuf.String(), "id\tname") {
		t.Fatalf("Expected plain tabular output, got %q", outBuf.String())
	}
	if errBuf.Len() != 0 {
		t.Errorf("Expected no stderr output, got %q", errBuf.String())
	}
}

func TestQueryCmd_StreamNDJSON(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	defer func() {
		queryConnection = ""
		queryFormat = "auto"
		queryQuiet = false
		queryStream = false
	}()

	dbPath := createSQLiteTestDatabase(t,
		"query-stream-test.db",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
		"INSERT INTO users (id, name) VALUES (1, 'Alice')",
		"INSERT INTO users (id, name) VALUES (2, 'Bob')",
	)
	saveTestConnection(t, config.Connection{
		Name:     "query-stream-db",
		Type:     "Sqlite3",
		Host:     dbPath,
		Database: dbPath,
	})

	queryConnection = "query-stream-db"
	queryFormat = "ndjson"
	queryQuiet = false
	queryStream = true

	outBuf, errBuf := setCommandBuffers(t, queryCmd)

	if err := queryCmd.RunE(queryCmd, []string{"SELECT id, name FROM users ORDER BY id"}); err != nil {
		t.Fatalf("Query command failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(outBuf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 NDJSON lines, got %d: %q", len(lines), outBuf.String())
	}
	if errBuf.Len() != 0 {
		t.Errorf("Expected no stderr output, got %q", errBuf.String())
	}
}

func TestExportCmd_StreamTableCSV(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	defer func() {
		exportConnection = ""
		exportSchema = ""
		exportTable = ""
		exportQuery = ""
		exportFormat = ""
		exportOutput = ""
		exportDelimiter = ","
		exportQuiet = false
		exportStream = false
	}()

	dbPath := createSQLiteTestDatabase(t,
		"export-stream-table.db",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
		"INSERT INTO users (id, name) VALUES (1, 'Alice')",
		"INSERT INTO users (id, name) VALUES (2, 'Bob')",
	)
	saveTestConnection(t, config.Connection{
		Name:     "export-stream-table-db",
		Type:     "Sqlite3",
		Host:     dbPath,
		Database: dbPath,
	})

	exportConnection = "export-stream-table-db"
	exportTable = "users"
	exportFormat = "csv"
	exportOutput = filepath.Join(t.TempDir(), "users.csv")
	exportDelimiter = ","
	exportQuiet = true
	exportStream = true

	outBuf, errBuf := setCommandBuffers(t, exportCmd)

	if err := exportCmd.RunE(exportCmd, []string{}); err != nil {
		t.Fatalf("Export command failed: %v", err)
	}

	content, err := os.ReadFile(exportOutput)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(content), "id,name") || !strings.Contains(string(content), "Alice") {
		t.Fatalf("Unexpected streamed CSV content: %q", string(content))
	}
	if outBuf.Len() != 0 || errBuf.Len() != 0 {
		t.Fatalf("Expected quiet export output, got stdout=%q stderr=%q", outBuf.String(), errBuf.String())
	}
}

func TestExportCmd_StreamQueryCSV(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	defer func() {
		exportConnection = ""
		exportSchema = ""
		exportTable = ""
		exportQuery = ""
		exportFormat = ""
		exportOutput = ""
		exportDelimiter = ","
		exportQuiet = false
		exportStream = false
	}()

	dbPath := createSQLiteTestDatabase(t,
		"export-stream-query.db",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
		"INSERT INTO users (id, name) VALUES (1, 'Alice')",
		"INSERT INTO users (id, name) VALUES (2, 'Bob')",
	)
	saveTestConnection(t, config.Connection{
		Name:     "export-stream-query-db",
		Type:     "Sqlite3",
		Host:     dbPath,
		Database: dbPath,
	})

	exportConnection = "export-stream-query-db"
	exportQuery = "SELECT id, name FROM users ORDER BY id"
	exportFormat = "csv"
	exportOutput = filepath.Join(t.TempDir(), "users-query.csv")
	exportDelimiter = ","
	exportQuiet = true
	exportStream = true

	outBuf, errBuf := setCommandBuffers(t, exportCmd)

	if err := exportCmd.RunE(exportCmd, []string{}); err != nil {
		t.Fatalf("Export command failed: %v", err)
	}

	content, err := os.ReadFile(exportOutput)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(content), "id,name") || !strings.Contains(string(content), "Bob") {
		t.Fatalf("Unexpected streamed query CSV content: %q", string(content))
	}
	if outBuf.Len() != 0 || errBuf.Len() != 0 {
		t.Fatalf("Expected quiet export output, got stdout=%q stderr=%q", outBuf.String(), errBuf.String())
	}
}

// TestMockDataCmd_Exists verifies the mock-data command is registered.
func TestMockDataCmd_Exists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "mock-data" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'mock-data' command to be registered")
	}
}

// TestMockDataCmd_Flags verifies the mock-data command has expected flags.
func TestMockDataCmd_Flags(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	flags := []string{"connection", "schema", "table", "rows", "overwrite", "analyze", "yes", "fk-density-ratio", "format", "quiet"}
	for _, flag := range flags {
		if mockDataCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected '--%s' flag on mock-data command", flag)
		}
	}
}

// TestMockDataCmd_RequiresConnection verifies --connection is required.
func TestMockDataCmd_RequiresConnection(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	oldConnection, oldTable, oldRows := mockDataConnection, mockDataTable, mockDataRows
	t.Cleanup(func() {
		mockDataConnection, mockDataTable, mockDataRows = oldConnection, oldTable, oldRows
	})

	mockDataConnection = ""
	mockDataTable = "users"
	mockDataRows = 10

	err := mockDataCmd.RunE(mockDataCmd, []string{})
	if err == nil {
		t.Fatal("Expected error when --connection is not provided")
	}
	if !strings.Contains(err.Error(), "--connection") {
		t.Errorf("Expected error message to mention --connection, got: %v", err)
	}
}

// TestMockDataCmd_RequiresRows verifies --rows must be positive.
func TestMockDataCmd_RequiresRows(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	oldConnection, oldTable, oldRows := mockDataConnection, mockDataTable, mockDataRows
	t.Cleanup(func() {
		mockDataConnection, mockDataTable, mockDataRows = oldConnection, oldTable, oldRows
	})

	mockDataConnection = "dev"
	mockDataTable = "users"
	mockDataRows = 0

	err := mockDataCmd.RunE(mockDataCmd, []string{})
	if err == nil {
		t.Fatal("Expected error when --rows is not positive")
	}
	if !strings.Contains(err.Error(), "--rows") {
		t.Errorf("Expected error message to mention --rows, got: %v", err)
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
		"load":   false,
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
		t.Fatal("Expected error when pattern is not provided")
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

func TestHistoryLoadCmd_PlainJSONAndMissingID(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	mgr, err := history.NewManager()
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	if err := mgr.Add("SELECT * FROM users", true, "sqlite"); err != nil {
		t.Fatalf("Failed to add history entry: %v", err)
	}
	entry := mgr.GetAll()[0]

	historyFormat = "plain"
	outBuf, errBuf := setCommandBuffers(t, historyLoadCmd)
	if err := historyLoadCmd.RunE(historyLoadCmd, []string{entry.ID}); err != nil {
		t.Fatalf("history load failed: %v", err)
	}
	if strings.TrimSpace(outBuf.String()) != entry.Query {
		t.Fatalf("unexpected loaded query %q", outBuf.String())
	}
	if errBuf.Len() != 0 {
		t.Fatalf("expected no stderr from plain load, got %q", errBuf.String())
	}

	historyFormat = "json"
	jsonOut, jsonErr := setCommandBuffers(t, historyLoadCmd)
	if err := historyLoadCmd.RunE(historyLoadCmd, []string{entry.ID}); err != nil {
		t.Fatalf("history load json failed: %v", err)
	}
	var loaded history.Entry
	if err := json.Unmarshal(jsonOut.Bytes(), &loaded); err != nil {
		t.Fatalf("failed to decode loaded history entry: %v", err)
	}
	if loaded.ID != entry.ID || loaded.Query != entry.Query || loaded.Database != entry.Database || !loaded.Success {
		t.Fatalf("unexpected loaded history entry: %+v", loaded)
	}
	if jsonErr.Len() != 0 {
		t.Fatalf("expected no stderr from json load, got %q", jsonErr.String())
	}

	if err := historyLoadCmd.RunE(historyLoadCmd, []string{"missing-id"}); err == nil || !strings.Contains(err.Error(), "entry \"missing-id\" not found") {
		t.Fatalf("expected missing-id error, got %v", err)
	}
}

func TestHistoryLoadCmd_NDJSONMatchesJSONShape(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	mgr, err := history.NewManager()
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	if err := mgr.Add("SELECT * FROM users", true, "postgres"); err != nil {
		t.Fatalf("Failed to add history entry: %v", err)
	}
	entry := mgr.GetAll()[0]

	historyFormat = "ndjson"
	outBuf, errBuf := setCommandBuffers(t, historyLoadCmd)
	if err := historyLoadCmd.RunE(historyLoadCmd, []string{entry.ID}); err != nil {
		t.Fatalf("history load ndjson failed: %v", err)
	}
	if errBuf.Len() != 0 {
		t.Fatalf("expected no stderr from ndjson load, got %q", errBuf.String())
	}

	var loaded history.Entry
	if err := json.Unmarshal(outBuf.Bytes(), &loaded); err != nil {
		t.Fatalf("failed to decode ndjson history entry: %v", err)
	}
	if loaded.ID != entry.ID || loaded.Query != entry.Query || loaded.Database != entry.Database || loaded.Success != entry.Success {
		t.Fatalf("unexpected ndjson-loaded history entry: %+v", loaded)
	}
	if !loaded.Timestamp.Equal(entry.Timestamp) {
		t.Fatalf("expected ndjson timestamp %v to match entry timestamp %v", loaded.Timestamp, entry.Timestamp)
	}
}

func TestHistoryClearCmd_JSONEnvelope(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	mgr, err := history.NewManager()
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	if err := mgr.Add("SELECT 1", true, "sqlite"); err != nil {
		t.Fatalf("Failed to add history entry: %v", err)
	}
	if err := mgr.Add("SELECT 2", true, "sqlite"); err != nil {
		t.Fatalf("Failed to add history entry: %v", err)
	}

	historyFormat = "json"
	historyQuiet = false

	outBuf, errBuf := setCommandBuffers(t, historyClearCmd)

	err = historyClearCmd.RunE(historyClearCmd, []string{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	envelope := decodeJSONEnvelope[struct {
		RemovedCount int `json:"removedCount"`
	}](t, outBuf)
	if envelope.Command != "history.clear" {
		t.Errorf("Expected command history.clear, got %q", envelope.Command)
	}
	if envelope.Data.RemovedCount != 2 {
		t.Errorf("Expected removedCount 2, got %d", envelope.Data.RemovedCount)
	}
	if errBuf.Len() != 0 {
		t.Errorf("Expected no stderr output, got %q", errBuf.String())
	}
}

func TestAuditCmd_JSONEnvelope(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	dbPath := createSQLiteTestDatabase(t,
		"audit-test.db",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)",
		"INSERT INTO users (id, name, email) VALUES (1, 'Alice', 'alice@example.com')",
		"INSERT INTO users (id, name, email) VALUES (2, 'Bob', NULL)",
	)

	auditConnection = ""
	auditSchema = ""
	auditTable = "users"
	auditFormat = "json"
	auditNullWarning = 0
	auditNullError = 0
	auditQuiet = false
	auditType = "sqlite3"
	auditDatabase = dbPath
	auditHost = ""
	auditPort = 0
	auditUser = ""

	outBuf, errBuf := setCommandBuffers(t, auditCmd)

	err := auditCmd.RunE(auditCmd, []string{})
	if err != nil {
		t.Fatalf("Audit command failed: %v", err)
	}

	envelope := decodeJSONEnvelope[auditCommandOutput](t, outBuf)
	if envelope.Command != "audit" {
		t.Errorf("Expected command audit, got %q", envelope.Command)
	}
	if envelope.Data.Summary.TablesScanned != 1 {
		t.Errorf("Expected 1 table scanned, got %d", envelope.Data.Summary.TablesScanned)
	}
	if len(envelope.Data.Results) != 1 {
		t.Fatalf("Expected 1 audit result, got %d", len(envelope.Data.Results))
	}
	if envelope.Data.Results[0].TableName != "users" {
		t.Errorf("Expected users table audit, got %q", envelope.Data.Results[0].TableName)
	}
	if errBuf.Len() != 0 {
		t.Errorf("Expected no stderr output, got %q", errBuf.String())
	}
}

func TestMockDataCmd_Analyze_JSONEnvelope(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	dbPath := createSQLiteTestDatabase(t,
		"mock-data-test.db",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
	)
	saveTestConnection(t, config.Connection{
		Name:     "mock-data-json",
		Type:     "Sqlite3",
		Host:     dbPath,
		Database: dbPath,
	})

	mockDataConnection = "mock-data-json"
	mockDataSchema = ""
	mockDataTable = "users"
	mockDataRows = 3
	mockDataFormat = "json"
	mockDataQuiet = false
	mockDataOverwrite = false
	mockDataAnalyzeOnly = true
	mockDataConfirm = false
	mockDataFKDensityRatio = 0

	outBuf, errBuf := setCommandBuffers(t, mockDataCmd)

	err := mockDataCmd.RunE(mockDataCmd, []string{})
	if err != nil {
		t.Fatalf("Mock data command failed: %v", err)
	}

	envelope := decodeJSONEnvelope[mockDataCommandOutput](t, outBuf)
	if envelope.Command != "mock-data.analyze" {
		t.Errorf("Expected command mock-data.analyze, got %q", envelope.Command)
	}
	if envelope.Data.StorageUnit != "users" {
		t.Errorf("Expected storage unit users, got %q", envelope.Data.StorageUnit)
	}
	if envelope.Data.RowCount != 3 {
		t.Errorf("Expected row count 3, got %d", envelope.Data.RowCount)
	}
	if envelope.Data.Analysis.TotalRows <= 0 {
		t.Errorf("Expected analysis totalRows > 0, got %d", envelope.Data.Analysis.TotalRows)
	}
	if errBuf.Len() != 0 {
		t.Errorf("Expected no stderr output, got %q", errBuf.String())
	}
}

func TestExplainCmd_Exists(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "explain [SQL]" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'explain' command to be registered")
	}
}

func TestExplainCmd_JSON(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	dbPath := createSQLiteTestDatabase(t,
		"explain-test.db",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
		"INSERT INTO users (name) VALUES ('alice')",
	)
	saveTestConnection(t, config.Connection{
		Name:     "explain-json",
		Type:     "Sqlite3",
		Host:     dbPath,
		Database: dbPath,
	})

	explainConnection = "explain-json"
	explainFormat = "json"
	explainQuiet = false

	outBuf, errBuf := setCommandBuffers(t, explainCmd)

	if err := explainCmd.RunE(explainCmd, []string{"SELECT * FROM users"}); err != nil {
		t.Fatalf("Explain command failed: %v", err)
	}

	var rows []map[string]any
	if err := json.Unmarshal(outBuf.Bytes(), &rows); err != nil {
		t.Fatalf("Failed to decode explain JSON: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("Expected explain output rows")
	}
	if errBuf.Len() != 0 {
		t.Errorf("Expected no stderr output, got %q", errBuf.String())
	}
}

func TestBookmarksCmd_SaveLoadDelete(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	bookmarkFormat = "json"
	bookmarkQuiet = false

	saveOut, saveErr := setCommandBuffers(t, bookmarksSaveCmd)
	if err := bookmarksSaveCmd.RunE(bookmarksSaveCmd, []string{"users", "SELECT * FROM users"}); err != nil {
		t.Fatalf("bookmarks save failed: %v", err)
	}
	saveEnvelope := decodeJSONEnvelope[config.SavedQuery](t, saveOut)
	if saveEnvelope.Command != "bookmarks.save" {
		t.Fatalf("expected bookmarks.save command, got %q", saveEnvelope.Command)
	}
	if saveEnvelope.Data.Name != "users" {
		t.Fatalf("expected bookmark name users, got %q", saveEnvelope.Data.Name)
	}
	if saveErr.Len() != 0 {
		t.Fatalf("expected no stderr from save, got %q", saveErr.String())
	}

	bookmarkFormat = "plain"
	loadOut, _ := setCommandBuffers(t, bookmarksLoadCmd)
	if err := bookmarksLoadCmd.RunE(bookmarksLoadCmd, []string{"users"}); err != nil {
		t.Fatalf("bookmarks load failed: %v", err)
	}
	if strings.TrimSpace(loadOut.String()) != "SELECT * FROM users" {
		t.Fatalf("unexpected loaded bookmark query %q", loadOut.String())
	}

	bookmarkFormat = "json"
	deleteOut, deleteErr := setCommandBuffers(t, bookmarksDeleteCmd)
	if err := bookmarksDeleteCmd.RunE(bookmarksDeleteCmd, []string{"users"}); err != nil {
		t.Fatalf("bookmarks delete failed: %v", err)
	}
	deleteEnvelope := decodeJSONEnvelope[struct {
		Name string `json:"name"`
	}](t, deleteOut)
	if deleteEnvelope.Command != "bookmarks.delete" {
		t.Fatalf("expected bookmarks.delete command, got %q", deleteEnvelope.Command)
	}
	if deleteEnvelope.Data.Name != "users" {
		t.Fatalf("expected deleted bookmark users, got %q", deleteEnvelope.Data.Name)
	}
	if deleteErr.Len() != 0 {
		t.Fatalf("expected no stderr from delete, got %q", deleteErr.String())
	}
}

func TestBookmarksLoadCmd_NDJSONMatchesJSONShape(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	bookmarkFormat = "json"
	bookmarkQuiet = false

	if err := bookmarksSaveCmd.RunE(bookmarksSaveCmd, []string{"users", "SELECT * FROM users"}); err != nil {
		t.Fatalf("bookmarks save failed: %v", err)
	}

	bookmarkFormat = "ndjson"
	outBuf, errBuf := setCommandBuffers(t, bookmarksLoadCmd)
	if err := bookmarksLoadCmd.RunE(bookmarksLoadCmd, []string{"users"}); err != nil {
		t.Fatalf("bookmarks load ndjson failed: %v", err)
	}
	if errBuf.Len() != 0 {
		t.Fatalf("expected no stderr from ndjson load, got %q", errBuf.String())
	}

	var loaded config.SavedQuery
	if err := json.Unmarshal(outBuf.Bytes(), &loaded); err != nil {
		t.Fatalf("failed to decode ndjson bookmark: %v", err)
	}
	if loaded.Name != "users" || loaded.Query != "SELECT * FROM users" {
		t.Fatalf("unexpected ndjson-loaded bookmark: %+v", loaded)
	}
}

func TestProfilesCmd_SaveShowDelete(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	saveTestConnection(t, config.Connection{
		Name:     "profile-conn",
		Type:     "Sqlite3",
		Host:     "/tmp/profile.db",
		Database: "/tmp/profile.db",
	})

	profilesFormat = "json"
	profilesQuiet = false
	profilesSaveConn = "profile-conn"
	profilesSaveTheme = "Nord"
	profilesSavePageSize = 50
	profilesSaveTimeout = 15

	saveOut, saveErr := setCommandBuffers(t, profilesSaveCmd)
	if err := profilesSaveCmd.RunE(profilesSaveCmd, []string{"prod"}); err != nil {
		t.Fatalf("profiles save failed: %v", err)
	}
	saveEnvelope := decodeJSONEnvelope[config.Profile](t, saveOut)
	if saveEnvelope.Command != "profiles.save" {
		t.Fatalf("expected profiles.save command, got %q", saveEnvelope.Command)
	}
	if saveEnvelope.Data.Connection != "profile-conn" {
		t.Fatalf("expected profile connection profile-conn, got %q", saveEnvelope.Data.Connection)
	}
	if saveErr.Len() != 0 {
		t.Fatalf("expected no stderr from profiles save, got %q", saveErr.String())
	}

	showOut, _ := setCommandBuffers(t, profilesShowCmd)
	if err := profilesShowCmd.RunE(profilesShowCmd, []string{"prod"}); err != nil {
		t.Fatalf("profiles show failed: %v", err)
	}
	var shown config.Profile
	if err := json.Unmarshal(showOut.Bytes(), &shown); err != nil {
		t.Fatalf("failed to decode profile JSON: %v", err)
	}
	if shown.Name != "prod" {
		t.Fatalf("expected shown profile prod, got %q", shown.Name)
	}

	deleteOut, deleteErr := setCommandBuffers(t, profilesDeleteCmd)
	if err := profilesDeleteCmd.RunE(profilesDeleteCmd, []string{"prod"}); err != nil {
		t.Fatalf("profiles delete failed: %v", err)
	}
	deleteEnvelope := decodeJSONEnvelope[struct {
		Name string `json:"name"`
	}](t, deleteOut)
	if deleteEnvelope.Command != "profiles.delete" {
		t.Fatalf("expected profiles.delete command, got %q", deleteEnvelope.Command)
	}
	if deleteEnvelope.Data.Name != "prod" {
		t.Fatalf("expected deleted profile prod, got %q", deleteEnvelope.Data.Name)
	}
	if deleteErr.Len() != 0 {
		t.Fatalf("expected no stderr from profiles delete, got %q", deleteErr.String())
	}
}

func TestERDCmd_JSONEnvelope(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	dbPath := createSQLiteTestDatabase(t,
		"erd-test.db",
		"PRAGMA foreign_keys = ON",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
		"CREATE TABLE orders (id INTEGER PRIMARY KEY, user_id INTEGER, FOREIGN KEY(user_id) REFERENCES users(id))",
	)
	saveTestConnection(t, config.Connection{
		Name:     "erd-json",
		Type:     "Sqlite3",
		Host:     dbPath,
		Database: dbPath,
	})

	erdConnection = "erd-json"
	erdSchema = ""
	erdFormat = "json"
	erdQuiet = false

	outBuf, errBuf := setCommandBuffers(t, erdCmd)
	if err := erdCmd.RunE(erdCmd, []string{}); err != nil {
		t.Fatalf("ERD command failed: %v", err)
	}

	envelope := decodeJSONEnvelope[erdCommandOutput](t, outBuf)
	if envelope.Command != "erd" {
		t.Fatalf("expected erd command envelope, got %q", envelope.Command)
	}
	if len(envelope.Data.StorageUnits) == 0 {
		t.Fatal("expected ERD output to contain storage units")
	}
	if errBuf.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", errBuf.String())
	}
}

func TestAgentSchemaCmd_JSON(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	agentSchemaFormat = "json"

	outBuf, errBuf := setCommandBuffers(t, agentSchemaCmd)
	if err := agentSchemaCmd.RunE(agentSchemaCmd, []string{}); err != nil {
		t.Fatalf("agent schema command failed: %v", err)
	}

	var manifest agentmanifest.Manifest
	if err := json.Unmarshal(outBuf.Bytes(), &manifest); err != nil {
		t.Fatalf("failed to decode agent schema output: %v", err)
	}
	if manifest.Name != "whodb" {
		t.Fatalf("expected manifest name whodb, got %q", manifest.Name)
	}
	if len(manifest.SourceTypes) == 0 {
		t.Fatal("expected source types in manifest")
	}
	if len(manifest.MCPTools) == 0 {
		t.Fatal("expected MCP tools in manifest")
	}
	if manifest.PlatformMCP.EnabledByFlag != "--platform" {
		t.Fatalf("expected platform MCP flag --platform, got %q", manifest.PlatformMCP.EnabledByFlag)
	}
	if !manifest.PlatformMCP.RequiresLogin || !manifest.PlatformMCP.RequiresWorkspace {
		t.Fatalf("expected platform MCP to require login and workspace, got %#v", manifest.PlatformMCP)
	}
	if len(manifest.PlatformMCP.Prompts) == 0 {
		t.Fatal("expected platform MCP prompts in manifest")
	}
	if len(manifest.PlatformMCP.Resources) == 0 {
		t.Fatal("expected platform MCP resources in manifest")
	}
	if errBuf.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", errBuf.String())
	}
}

func TestDoctorCmd_JSONEnvelope(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	dbPath := createSQLiteTestDatabase(t,
		"doctor-test.db",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)",
	)
	saveTestConnection(t, config.Connection{
		Name:     "doctor-sqlite",
		Type:     "Sqlite3",
		Host:     dbPath,
		Database: dbPath,
	})

	doctorConnection = "doctor-sqlite"
	doctorSchema = ""
	doctorFormat = "json"
	doctorQuiet = false

	outBuf, errBuf := setCommandBuffers(t, doctorCmd)
	if err := doctorCmd.RunE(doctorCmd, []string{}); err != nil {
		t.Fatalf("doctor command failed: %v", err)
	}

	envelope := decodeJSONEnvelope[doctor.Report](t, outBuf)
	if envelope.Command != "doctor" {
		t.Fatalf("expected doctor command envelope, got %q", envelope.Command)
	}
	if envelope.Data.Connection.Name != "doctor-sqlite" {
		t.Fatalf("expected doctor connection name, got %q", envelope.Data.Connection.Name)
	}
	if len(envelope.Data.Checks) == 0 {
		t.Fatal("expected doctor checks")
	}
	if errBuf.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", errBuf.String())
	}
}

func TestRunbooksListAndDryRun_JSON(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	runbooksFormat = "json"
	runbooksQuiet = false

	listOut, listErr := setCommandBuffers(t, runbooksListCmd)
	if err := runbooksListCmd.RunE(runbooksListCmd, []string{}); err != nil {
		t.Fatalf("runbooks list failed: %v", err)
	}
	var definitions []runbooks.Definition
	if err := json.Unmarshal(listOut.Bytes(), &definitions); err != nil {
		t.Fatalf("failed to decode runbooks list: %v", err)
	}
	if len(definitions) == 0 {
		t.Fatal("expected built-in runbooks")
	}
	if listErr.Len() != 0 {
		t.Fatalf("expected no stderr from runbooks list, got %q", listErr.String())
	}

	runbooksDryRun = true
	runbooksConnection = ""
	runbooksSchema = ""
	runbooksFrom = ""
	runbooksTo = ""
	runbooksFromSchema = ""
	runbooksToSchema = ""

	runOut, runErr := setCommandBuffers(t, runbooksRunCmd)
	if err := runbooksRunCmd.RunE(runbooksRunCmd, []string{"schema-audit"}); err != nil {
		t.Fatalf("runbooks dry-run failed: %v", err)
	}
	envelope := decodeJSONEnvelope[runbooks.Result](t, runOut)
	if envelope.Command != "runbooks.run" {
		t.Fatalf("expected runbooks.run command envelope, got %q", envelope.Command)
	}
	if !envelope.Data.DryRun {
		t.Fatal("expected dry-run result")
	}
	if len(envelope.Data.Steps) == 0 {
		t.Fatal("expected planned runbook steps")
	}
	if runErr.Len() != 0 {
		t.Fatalf("expected no stderr from runbooks dry-run, got %q", runErr.String())
	}
}

func TestSkillsListAndInstall_JSON(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	skillsFormat = "json"
	skillsQuiet = false

	listOut, listErr := setCommandBuffers(t, skillsListCmd)
	if err := skillsListCmd.RunE(skillsListCmd, []string{}); err != nil {
		t.Fatalf("skills list failed: %v", err)
	}
	var items []skillinstaller.Item
	if err := json.Unmarshal(listOut.Bytes(), &items); err != nil {
		t.Fatalf("failed to decode skills list: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected bundled skills")
	}
	if listErr.Len() != 0 {
		t.Fatalf("expected no stderr from skills list, got %q", listErr.String())
	}

	targetDir := filepath.Join(t.TempDir(), "skills")
	skillsTarget = ""
	skillsTargetDir = targetDir
	skillsAgentsDir = ""
	skillsIncludeAgents = false
	skillsForce = false

	installOut, installErr := setCommandBuffers(t, skillsInstallCmd)
	if err := skillsInstallCmd.RunE(skillsInstallCmd, []string{"whodb"}); err != nil {
		t.Fatalf("skills install failed: %v", err)
	}
	envelope := decodeJSONEnvelope[skillinstaller.InstallResult](t, installOut)
	if envelope.Command != "skills.install" {
		t.Fatalf("expected skills.install command envelope, got %q", envelope.Command)
	}
	if len(envelope.Data.Skills) != 1 {
		t.Fatalf("expected one installed skill, got %d", len(envelope.Data.Skills))
	}
	if _, err := os.Stat(filepath.Join(targetDir, "whodb", "SKILL.md")); err != nil {
		t.Fatalf("expected installed skill file: %v", err)
	}
	if installErr.Len() != 0 {
		t.Fatalf("expected no stderr from skills install, got %q", installErr.String())
	}
}

func TestSkillsInstall_DryRunJSONDoesNotWrite(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	defer func() {
		skillsDryRun = false
	}()

	path := filepath.Join(testHome, ".cursor", "mcp.json")
	_ = os.Remove(path)
	_ = os.Remove(path + ".whodb.bak")

	skillsFormat = "json"
	skillsQuiet = false
	skillsTarget = "cursor"
	skillsTargetDir = ""
	skillsAgentsDir = ""
	skillsIncludeAgents = false
	skillsForce = false
	skillsDryRun = true

	installOut, installErr := setCommandBuffers(t, skillsInstallCmd)
	if err := skillsInstallCmd.RunE(skillsInstallCmd, []string{}); err != nil {
		t.Fatalf("skills dry-run failed: %v", err)
	}
	envelope := decodeJSONEnvelope[skillinstaller.InstallResult](t, installOut)
	if !envelope.Data.DryRun {
		t.Fatal("expected dry-run result")
	}
	if len(envelope.Data.Configs) != 1 {
		t.Fatalf("expected one planned config, got %#v", envelope.Data.Configs)
	}
	if envelope.Data.Configs[0].Action != "create" {
		t.Fatalf("expected create action, got %#v", envelope.Data.Configs[0])
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected dry-run to leave config absent, got %v", err)
	}
	if _, err := os.Stat(path + ".whodb.bak"); !os.IsNotExist(err) {
		t.Fatalf("expected dry-run to leave backup absent, got %v", err)
	}
	if installErr.Len() != 0 {
		t.Fatalf("expected no stderr from skills dry-run, got %q", installErr.String())
	}
}

func TestSkillsInstall_CodexTargetUsesAgentsSkills(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	skillsTarget = "codex"
	skillsTargetDir = ""
	skillsAgentsDir = ""
	skillsIncludeAgents = false
	skillsForce = false

	result, err := skillinstaller.Install(skillinstaller.InstallOptions{
		Name:   "whodb",
		Target: skillsTarget,
	})
	if err != nil {
		t.Fatalf("skills install failed: %v", err)
	}
	expectedPath := filepath.Join(testHome, ".codex", "skills", "whodb", "SKILL.md")
	if len(result.Skills) != 1 || result.Skills[0].Path != expectedPath {
		t.Fatalf("expected Codex skill at %q, got %#v", expectedPath, result.Skills)
	}
}

func TestSkillsInstall_CodexTargetRequiresAgentsDirForAgents(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	_, err := skillinstaller.Install(skillinstaller.InstallOptions{
		Target:        "codex",
		IncludeAgents: true,
	})
	if err == nil {
		t.Fatal("expected codex target with agents to require an explicit agents directory")
	}
	if !strings.Contains(err.Error(), "--include-agents requires --target claude-code or --agents-dir") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSkillsInstall_CodexTargetUsesExplicitAgentsDir(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	agentsDir := filepath.Join(t.TempDir(), "agents")
	result, err := skillinstaller.Install(skillinstaller.InstallOptions{
		Name:          "whodb",
		Target:        "codex",
		AgentsDir:     agentsDir,
		IncludeAgents: true,
		Force:         true,
	})
	if err != nil {
		t.Fatalf("skills install failed: %v", err)
	}
	if len(result.Skills) != 1 {
		t.Fatalf("expected one installed skill, got %#v", result.Skills)
	}
	if len(result.Agents) == 0 {
		t.Fatal("expected bundled agents to install into explicit agents dir")
	}
	if _, err := os.Stat(filepath.Join(agentsDir, "database-analyst.md")); err != nil {
		t.Fatalf("expected installed agent file: %v", err)
	}
}

func TestSkillsInstall_AssistantIntegrationTargets(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir failed: %v", err)
	}

	targets := []struct {
		name      string
		checkPath string
	}{
		{name: "cursor", checkPath: filepath.Join(testHome, ".cursor", "mcp.json")},
		{name: "vscode", checkPath: filepath.Join(configDir, "Code", "User", "mcp.json")},
		{name: "github-copilot", checkPath: filepath.Join(testHome, ".copilot", "mcp-config.json")},
		{name: "gemini-cli", checkPath: filepath.Join(testHome, ".gemini", "extensions", "whodb", "gemini-extension.json")},
		{name: "windsurf", checkPath: filepath.Join(testHome, ".codeium", "mcp_config.json")},
		{name: "opencode", checkPath: filepath.Join(testHome, ".config", "opencode", "opencode.json")},
		{name: "cline", checkPath: filepath.Join(testHome, ".cline", "data", "settings", "cline_mcp_settings.json")},
		{name: "zed", checkPath: expectedZedSettingsPath(t)},
		{name: "continue", checkPath: filepath.Join(testHome, ".continue", "config.yaml")},
		{name: "aider", checkPath: filepath.Join(testHome, ".aider.conf.yml")},
	}

	for _, target := range targets {
		t.Run(target.name, func(t *testing.T) {
			result, err := skillinstaller.Install(skillinstaller.InstallOptions{
				Target: target.name,
				Force:  true,
			})
			if err != nil {
				t.Fatalf("install failed: %v", err)
			}
			if len(result.Configs)+len(result.Rules)+len(result.Extensions) == 0 {
				t.Fatalf("expected installed integration files, got %#v", result)
			}
			data, err := os.ReadFile(target.checkPath)
			if err != nil {
				t.Fatalf("expected %s to exist: %v", target.checkPath, err)
			}
			content := string(data)
			if target.name == "aider" {
				if !strings.Contains(content, "whodb-conventions.md") {
					t.Fatalf("expected aider config to read WhoDB conventions, got %q", content)
				}
				return
			}
			if !strings.Contains(content, "whodb") || !strings.Contains(content, "@clidey/whodb-cli") {
				t.Fatalf("expected WhoDB MCP config in %s, got %q", target.checkPath, content)
			}
		})
	}

	if _, err := os.Stat(filepath.Join(testHome, "Documents", "Cline", "Rules", "whodb.md")); err != nil {
		t.Fatalf("expected Cline rule file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(testHome, ".aider", "whodb-conventions.md")); err != nil {
		t.Fatalf("expected aider conventions file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(testHome, ".gemini", "extensions", "whodb", "GEMINI.md")); err != nil {
		t.Fatalf("expected Gemini context file: %v", err)
	}
}

func expectedZedSettingsPath(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "linux" && os.Getenv("XDG_CONFIG_HOME") != "" {
		return filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "zed", "settings.json")
	}
	return filepath.Join(testHome, ".config", "zed", "settings.json")
}

func TestSkillsInstall_AssistantIntegrationPreservesExistingJSONConfig(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	path := filepath.Join(testHome, ".cursor", "mcp.json")
	_ = os.Remove(path + ".whodb.bak")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	existing := `{"mcpServers":{"other":{"command":"node","args":["server.js"]}}}`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	result, err := skillinstaller.Install(skillinstaller.InstallOptions{Target: "cursor"})
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if len(result.Configs) != 1 {
		t.Fatalf("expected one config result, got %#v", result.Configs)
	}

	var config map[string]map[string]any
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("JSON parse failed: %v", err)
	}
	servers := config["mcpServers"]
	if servers["other"] == nil {
		t.Fatalf("expected existing server to be preserved, got %#v", servers)
	}
	if servers["whodb"] == nil {
		t.Fatalf("expected whodb server to be added, got %#v", servers)
	}
	backup, err := os.ReadFile(path + ".whodb.bak")
	if err != nil {
		t.Fatalf("expected existing config backup: %v", err)
	}
	if string(backup) != existing {
		t.Fatalf("expected backup to match original config, got %q", string(backup))
	}
}

func TestSkillsInstall_DryRunDoesNotWriteSkill(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	targetDir := filepath.Join(t.TempDir(), "skills")
	result, err := skillinstaller.Install(skillinstaller.InstallOptions{
		Name:      "whodb",
		TargetDir: targetDir,
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}
	if !result.DryRun {
		t.Fatal("expected dry-run result")
	}
	if len(result.Skills) != 1 {
		t.Fatalf("expected one planned skill, got %#v", result.Skills)
	}
	if result.Skills[0].Action != "create" {
		t.Fatalf("expected create action, got %#v", result.Skills[0])
	}
	if _, err := os.Stat(filepath.Join(targetDir, "whodb", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("expected dry-run to leave skill absent, got %v", err)
	}
}

func TestSkillsInstall_DryRunReportsBackupAndDoesNotWriteConfig(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	path := filepath.Join(testHome, ".cursor", "mcp.json")
	_ = os.Remove(path + ".whodb.bak")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	existing := `{"mcpServers":{"other":{"command":"node","args":["server.js"]}}}`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	result, err := skillinstaller.Install(skillinstaller.InstallOptions{
		Target: "cursor",
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}
	if !result.DryRun {
		t.Fatal("expected dry-run result")
	}
	if len(result.Configs) != 1 {
		t.Fatalf("expected one planned config, got %#v", result.Configs)
	}
	planned := result.Configs[0]
	if planned.Action != "update" || planned.BackupPath != path+".whodb.bak" {
		t.Fatalf("expected update with backup path, got %#v", planned)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != existing {
		t.Fatalf("expected dry-run to leave config unchanged, got %q", string(data))
	}
	if _, err := os.Stat(path + ".whodb.bak"); !os.IsNotExist(err) {
		t.Fatalf("expected dry-run to leave backup absent, got %v", err)
	}
}

func TestSkillsInstall_AssistantIntegrationDoesNotBackupNewJSONConfig(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	path := filepath.Join(testHome, ".cursor", "mcp.json")
	_ = os.Remove(path)
	_ = os.Remove(path + ".whodb.bak")
	if _, err := skillinstaller.Install(skillinstaller.InstallOptions{Target: "cursor"}); err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file: %v", err)
	}
	if _, err := os.Stat(path + ".whodb.bak"); !os.IsNotExist(err) {
		t.Fatalf("expected no backup for new config, got %v", err)
	}
}

func TestSkillsInstall_AssistantIntegrationBacksUpExistingYAMLConfig(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	path := filepath.Join(testHome, ".continue", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	existing := "name: Existing\nversion: 1.0.0\nschema: v1\nrules:\n  - keep this rule\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if _, err := skillinstaller.Install(skillinstaller.InstallOptions{Target: "continue"}); err != nil {
		t.Fatalf("install failed: %v", err)
	}
	backup, err := os.ReadFile(path + ".whodb.bak")
	if err != nil {
		t.Fatalf("expected existing YAML config backup: %v", err)
	}
	if string(backup) != existing {
		t.Fatalf("expected backup to match original YAML config, got %q", string(backup))
	}
}

func TestSkillsInstall_AssistantIntegrationPreservesExistingJSONCConfig(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	path := filepath.Join(testHome, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	existing := `{
  // Existing servers can be hand-edited JSONC.
  "mcpServers": {
    "other": {
      "command": "node",
      "args": [
        "server.js",
      ],
      "url": "https://example.test//keep",
    },
    /* trailing comma below */
  },
}
`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	result, err := skillinstaller.Install(skillinstaller.InstallOptions{Target: "cursor"})
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if len(result.Configs) != 1 {
		t.Fatalf("expected one config result, got %#v", result.Configs)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	var config map[string]map[string]map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("rewritten config should be strict JSON: %v", err)
	}
	servers := config["mcpServers"]
	if servers["other"] == nil {
		t.Fatalf("expected existing server to be preserved, got %#v", servers)
	}
	if servers["other"]["url"] != "https://example.test//keep" {
		t.Fatalf("expected string content to be preserved, got %#v", servers["other"])
	}
	if servers["whodb"] == nil {
		t.Fatalf("expected whodb server to be added, got %#v", servers)
	}
	if strings.Contains(string(data), "// Existing") || strings.Contains(string(data), "/* trailing") {
		t.Fatalf("expected rewritten config to be normalized JSON, got %q", string(data))
	}
}

// TestAllNewCommandsRegistered verifies all new commands are registered on rootCmd
func TestAllNewCommandsRegistered(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	expectedCommands := []string{
		"agent",
		"doctor",
		"runbooks",
		"skills",
		"schemas",
		"tables",
		"columns",
		"connections",
		"diff",
		"erd",
		"explain",
		"export",
		"history",
		"bookmarks",
		"profiles",
	}

	registeredCommands := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		registeredCommands[cmd.Name()] = true
	}

	for _, expected := range expectedCommands {
		if !registeredCommands[expected] {
			t.Errorf("Expected '%s' command to be registered", expected)
		}
	}
}
