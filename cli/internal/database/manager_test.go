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

package database

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/clidey/whodb/cli/internal/bootstrap"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/types"
)

func TestNewManager(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.config == nil {
		t.Fatal("Manager config is nil")
	}
}

func TestGetCurrentConnection_NotConnected(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	conn := mgr.GetCurrentConnection()
	if conn != nil {
		t.Error("Expected nil connection when not connected")
	}
}

func TestGetSSLStatusSummary_NotConnected(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	_, err = mgr.GetSSLStatusSummary()
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestFormatSSLStatusSummary(t *testing.T) {
	tests := []struct {
		name   string
		status *engine.SSLStatus
		want   string
	}{
		{
			name:   "nil",
			status: nil,
			want:   "",
		},
		{
			name: "enabled without mode",
			status: &engine.SSLStatus{
				IsEnabled: true,
			},
			want: "SSL/TLS: enabled",
		},
		{
			name: "enabled with mode",
			status: &engine.SSLStatus{
				IsEnabled: true,
				Mode:      "verify-full",
			},
			want: "SSL/TLS: enabled (verify-full)",
		},
		{
			name: "disabled without mode",
			status: &engine.SSLStatus{
				IsEnabled: false,
			},
			want: "SSL/TLS: disabled",
		},
		{
			name: "disabled with mode",
			status: &engine.SSLStatus{
				IsEnabled: false,
				Mode:      "required",
			},
			want: "SSL/TLS: required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatSSLStatusSummary(tt.status); got != tt.want {
				t.Fatalf("formatSSLStatusSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetEnvConnectionsIncludesCatalogAlias(t *testing.T) {
	setupTestEnv(t)

	envCreds := []types.DatabaseCredentials{{
		Hostname: "ferret-host",
		Username: "ferret-user",
		Database: "ferret-db",
		Port:     "27017",
	}}
	envValue, err := json.Marshal(envCreds)
	if err != nil {
		t.Fatalf("failed to marshal env credentials: %v", err)
	}
	t.Setenv("WHODB_FERRETDB", string(envValue))

	mgr := &Manager{}
	for _, conn := range mgr.getEnvConnections() {
		if conn.Type == string(engine.DatabaseType_FerretDB) &&
			conn.Host == "ferret-host" &&
			conn.Port == 27017 &&
			conn.IsProfile {
			return
		}
	}

	t.Fatal("expected FerretDB env connection to be discovered from the shared catalog")
}

func TestDisconnect(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	err = mgr.Disconnect()
	if err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}

	conn := mgr.GetCurrentConnection()
	if conn != nil {
		t.Error("Expected nil connection after disconnect")
	}
}

func TestListConnections(t *testing.T) {
	setupTestEnv(t)

	cfg := config.DefaultConfig()
	cfg.AddConnection(config.Connection{
		Name: "test1",
		Type: "postgres",
		Host: "localhost",
	})
	cfg.AddConnection(config.Connection{
		Name: "test2",
		Type: "mysql",
		Host: "localhost",
	})
	cfg.Save()

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	conns := mgr.ListConnections()
	if len(conns) != 2 {
		t.Fatalf("Expected 2 connections, got %d", len(conns))
	}
}

func TestGetConnection(t *testing.T) {
	setupTestEnv(t)

	cfg := config.DefaultConfig()
	cfg.AddConnection(config.Connection{
		Name:     "test-db",
		Type:     "postgres",
		Host:     "localhost",
		Port:     5432,
		Username: "testuser",
		Database: "testdb",
	})
	cfg.Save()

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	conn, err := mgr.GetConnection("test-db")
	if err != nil {
		t.Fatalf("GetConnection failed: %v", err)
	}

	if conn.Name != "test-db" {
		t.Errorf("Expected connection name 'test-db', got '%s'", conn.Name)
	}

	if conn.Type != "postgres" {
		t.Errorf("Expected connection type 'postgres', got '%s'", conn.Type)
	}
}

func TestGetConnection_NotFound(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	_, err = mgr.GetConnection("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent connection")
	}
}

func TestGetSchemas_NotConnected(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	_, err = mgr.GetSchemas()
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestGetStorageUnits_NotConnected(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	_, err = mgr.GetStorageUnits("public")
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestExecuteQuery_NotConnected(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	_, err = mgr.ExecuteQuery("SELECT 1")
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestGetRows_NotConnected(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	_, err = mgr.GetRows("public", "users", nil, 50, 0)
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestGetColumns_NotConnected(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	_, err = mgr.GetColumns("public", "users")
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestExportToCSV_NotConnected(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	err = mgr.ExportToCSV("public", "users", "test.csv", ",")
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestExportToExcel_NotConnected(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	err = mgr.ExportToExcel("public", "users", "test.xlsx")
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestExportResultsToCSV_NilResult(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	err = mgr.ExportResultsToCSV(nil, "test.csv", ",")
	if err == nil {
		t.Error("Expected error when result is nil")
	}
}

func TestExportResultsToExcel_NilResult(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	err = mgr.ExportResultsToExcel(nil, "test.xlsx")
	if err == nil {
		t.Error("Expected error when result is nil")
	}
}

func TestGetAIProviders(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	providers := mgr.GetAIProviders()
	if providers == nil {
		t.Error("GetAIProviders returned nil")
	}
}

func TestBuildCredentials(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	conn := &Connection{
		Type:     "postgres",
		Host:     "localhost",
		Port:     5432,
		Username: "testuser",
		Password: "testpass",
		Database: "testdb",
	}

	creds := mgr.buildCredentials(conn)

	if creds.Type != "postgres" {
		t.Errorf("Expected type 'postgres', got '%s'", creds.Type)
	}

	if creds.Hostname != "localhost" {
		t.Errorf("Expected hostname 'localhost', got '%s'", creds.Hostname)
	}

	if creds.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", creds.Username)
	}

	if creds.Password != "testpass" {
		t.Errorf("Expected password 'testpass', got '%s'", creds.Password)
	}

	if creds.Database != "testdb" {
		t.Errorf("Expected database 'testdb', got '%s'", creds.Database)
	}

	hasPort := false
	for _, record := range creds.Advanced {
		if record.Key == "Port" && record.Value == "5432" {
			hasPort = true
			break
		}
	}
	if !hasPort {
		t.Error("Expected Port in Advanced credentials")
	}
}

// Tests for context-aware methods

func TestExecuteQueryWithContext_NotConnected(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	_, err = mgr.ExecuteQueryWithContext(ctx, "SELECT 1")
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestExecuteQueryWithContext_Cancelled(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create an already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = mgr.ExecuteQueryWithContext(ctx, "SELECT 1")
	// Should return error (either cancelled or not connected)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
}

func TestGetRowsWithContext_NotConnected(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	_, err = mgr.GetRowsWithContext(ctx, "public", "users", nil, 50, 0)
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestGetSchemasWithContext_NotConnected(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	_, err = mgr.GetSchemasWithContext(ctx)
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestGetStorageUnitsWithContext_NotConnected(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	_, err = mgr.GetStorageUnitsWithContext(ctx, "public")
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestGetAIModelsWithContext_NotConnected(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	_, err = mgr.GetAIModelsWithContext(ctx, "", "ollama", "")
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestSendAIChatWithContext_NotConnected(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	_, err = mgr.SendAIChatWithContext(ctx, "", "ollama", "", "public", "llama2", "", "test query")
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestGetConfig(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	cfg := mgr.GetConfig()
	if cfg == nil {
		t.Error("GetConfig returned nil")
	}

	// Verify it returns the same config
	if cfg != mgr.config {
		t.Error("GetConfig did not return the manager's config")
	}
}

func TestContextTimeout(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create a context that times out immediately
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Give it a moment to actually timeout
	time.Sleep(10 * time.Millisecond)

	_, err = mgr.ExecuteQueryWithContext(ctx, "SELECT 1")
	// Should return error (either deadline exceeded or not connected)
	if err == nil {
		t.Error("Expected error with timed out context")
	}
}

type contextCaptureKey string

type contextCapturePlugin struct {
	engine.BasePlugin
	rawContext          context.Context
	rowsContext         context.Context
	schemasContext      context.Context
	storageUnitsContext context.Context
}

func (p *contextCapturePlugin) RawExecute(config *engine.PluginConfig, query string, params ...any) (*engine.GetRowsResult, error) {
	p.rawContext = config.Context
	return &engine.GetRowsResult{}, nil
}

func (p *contextCapturePlugin) GetRows(config *engine.PluginConfig, req *engine.GetRowsRequest) (*engine.GetRowsResult, error) {
	p.rowsContext = config.Context
	return &engine.GetRowsResult{}, nil
}

func (p *contextCapturePlugin) GetAllSchemas(config *engine.PluginConfig) ([]string, error) {
	p.schemasContext = config.Context
	return []string{"public"}, nil
}

func (p *contextCapturePlugin) GetStorageUnits(config *engine.PluginConfig, schema string) ([]engine.StorageUnit, error) {
	p.storageUnitsContext = config.Context
	return []engine.StorageUnit{{Name: "users"}}, nil
}

func (p *contextCapturePlugin) StorageUnitExists(config *engine.PluginConfig, schema string, storageUnit string) (bool, error) {
	return true, nil
}

func newContextCaptureManager(plugin engine.PluginFunctions) *Manager {
	bootstrap.Ensure()

	eng := &engine.Engine{
		Plugins: []*engine.Plugin{{
			Type:            engine.DatabaseType_Postgres,
			PluginFunctions: plugin,
		}},
	}
	src.MainEngine = eng

	return &Manager{
		currentConnection: &Connection{Type: string(engine.DatabaseType_Postgres), Database: "app"},
		config:            config.DefaultConfig(),
		cache:             NewMetadataCache(DefaultCacheTTL),
	}
}

func TestExecuteQueryWithContext_PropagatesPluginContext(t *testing.T) {
	setupTestEnv(t)

	plugin := &contextCapturePlugin{}
	mgr := newContextCaptureManager(plugin)
	ctx := context.WithValue(context.Background(), contextCaptureKey("query"), "raw")

	if _, err := mgr.ExecuteQueryWithContext(ctx, "SELECT 1"); err != nil {
		t.Fatalf("ExecuteQueryWithContext failed: %v", err)
	}

	if plugin.rawContext == nil {
		t.Fatal("expected plugin context to be set")
	}
	if got := plugin.rawContext.Value(contextCaptureKey("query")); got != "raw" {
		t.Fatalf("expected propagated context value %q, got %v", "raw", got)
	}
}

func TestGetRowsWithContext_PropagatesPluginContext(t *testing.T) {
	setupTestEnv(t)

	plugin := &contextCapturePlugin{}
	mgr := newContextCaptureManager(plugin)
	ctx := context.WithValue(context.Background(), contextCaptureKey("rows"), "rows")

	if _, err := mgr.GetRowsWithContext(ctx, "public", "users", nil, 50, 0); err != nil {
		t.Fatalf("GetRowsWithContext failed: %v", err)
	}

	if plugin.rowsContext == nil {
		t.Fatal("expected plugin context to be set")
	}
	if got := plugin.rowsContext.Value(contextCaptureKey("rows")); got != "rows" {
		t.Fatalf("expected propagated context value %q, got %v", "rows", got)
	}
}

func TestGetSchemasWithContext_PropagatesPluginContext(t *testing.T) {
	setupTestEnv(t)

	plugin := &contextCapturePlugin{}
	mgr := newContextCaptureManager(plugin)
	ctx := context.WithValue(context.Background(), contextCaptureKey("schemas"), "schemas")

	if _, err := mgr.GetSchemasWithContext(ctx); err != nil {
		t.Fatalf("GetSchemasWithContext failed: %v", err)
	}

	if plugin.schemasContext == nil {
		t.Fatal("expected plugin context to be set")
	}
	if got := plugin.schemasContext.Value(contextCaptureKey("schemas")); got != "schemas" {
		t.Fatalf("expected propagated context value %q, got %v", "schemas", got)
	}
}

func TestGetStorageUnitsWithContext_PropagatesPluginContext(t *testing.T) {
	setupTestEnv(t)

	plugin := &contextCapturePlugin{}
	mgr := newContextCaptureManager(plugin)
	ctx := context.WithValue(context.Background(), contextCaptureKey("storage_units"), "storage_units")

	if _, err := mgr.GetStorageUnitsWithContext(ctx, "public"); err != nil {
		t.Fatalf("GetStorageUnitsWithContext failed: %v", err)
	}

	if plugin.storageUnitsContext == nil {
		t.Fatal("expected plugin context to be set")
	}
	if got := plugin.storageUnitsContext.Value(contextCaptureKey("storage_units")); got != "storage_units" {
		t.Fatalf("expected propagated context value %q, got %v", "storage_units", got)
	}
}

type metadataBatchPlugin struct {
	engine.BasePlugin
	mu                   sync.Mutex
	columnsByStorageUnit map[string][]engine.Column
	constraintsByUnit    map[string]map[string]map[string]any
	columnCalls          []string
	constraintCalls      []string
}

func (p *metadataBatchPlugin) GetColumnsForTable(config *engine.PluginConfig, schema string, storageUnit string) ([]engine.Column, error) {
	p.mu.Lock()
	p.columnCalls = append(p.columnCalls, storageUnit)
	p.mu.Unlock()
	return p.columnsByStorageUnit[storageUnit], nil
}

func (p *metadataBatchPlugin) StorageUnitExists(config *engine.PluginConfig, schema string, storageUnit string) (bool, error) {
	return true, nil
}

func (p *metadataBatchPlugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error) {
	p.mu.Lock()
	p.constraintCalls = append(p.constraintCalls, storageUnit)
	p.mu.Unlock()
	return p.constraintsByUnit[storageUnit], nil
}

func TestGetColumnsForStorageUnits(t *testing.T) {
	plugin := &metadataBatchPlugin{
		columnsByStorageUnit: map[string][]engine.Column{
			"users":  {{Name: "id", Type: "integer"}},
			"orders": {{Name: "order_id", Type: "integer"}},
		},
	}
	mgr := newContextCaptureManager(plugin)

	columnsByUnit, err := mgr.GetColumnsForStorageUnits("public", []string{"users", "orders"})
	if err != nil {
		t.Fatalf("GetColumnsForStorageUnits failed: %v", err)
	}

	if len(columnsByUnit["users"]) != 1 || columnsByUnit["users"][0].Name != "id" {
		t.Fatalf("unexpected users columns: %#v", columnsByUnit["users"])
	}
	if len(columnsByUnit["orders"]) != 1 || columnsByUnit["orders"][0].Name != "order_id" {
		t.Fatalf("unexpected orders columns: %#v", columnsByUnit["orders"])
	}

	if len(plugin.columnCalls) != 2 {
		t.Fatalf("expected 2 column calls, got %d", len(plugin.columnCalls))
	}
	if _, ok := mgr.GetCache().GetColumns("public", "users"); !ok {
		t.Fatal("expected users columns to be cached")
	}
	if _, ok := mgr.GetCache().GetColumns("public", "orders"); !ok {
		t.Fatal("expected orders columns to be cached")
	}
}

func TestGetColumnConstraintsForStorageUnits(t *testing.T) {
	plugin := &metadataBatchPlugin{
		constraintsByUnit: map[string]map[string]map[string]any{
			"users": {
				"id": {"unique": true},
			},
			"orders": {
				"order_id": {"default": "nextval"},
			},
		},
	}
	mgr := newContextCaptureManager(plugin)

	constraintsByUnit, err := mgr.GetColumnConstraintsForStorageUnits("public", []string{"users", "orders"})
	if err != nil {
		t.Fatalf("GetColumnConstraintsForStorageUnits failed: %v", err)
	}

	if got := constraintsByUnit["users"]["id"]["unique"]; got != true {
		t.Fatalf("unexpected users constraint value: %#v", constraintsByUnit["users"])
	}
	if got := constraintsByUnit["orders"]["order_id"]["default"]; got != "nextval" {
		t.Fatalf("unexpected orders constraint value: %#v", constraintsByUnit["orders"])
	}
	if len(plugin.constraintCalls) != 2 {
		t.Fatalf("expected 2 constraint calls, got %d", len(plugin.constraintCalls))
	}
}

// Tests for MetadataCache

func TestMetadataCache_NewCache(t *testing.T) {
	cache := NewMetadataCache(5 * time.Minute)
	if cache == nil {
		t.Fatal("NewMetadataCache returned nil")
	}
	if cache.ttl != 5*time.Minute {
		t.Errorf("Expected TTL of 5 minutes, got %v", cache.ttl)
	}
}

func TestMetadataCache_Schemas(t *testing.T) {
	cache := NewMetadataCache(5 * time.Minute)

	// Initially, cache should miss
	schemas, ok := cache.GetSchemas()
	if ok {
		t.Error("Expected cache miss for schemas initially")
	}
	if schemas != nil {
		t.Error("Expected nil schemas on cache miss")
	}

	// Set schemas
	testSchemas := []string{"public", "private", "test"}
	cache.SetSchemas(testSchemas)

	// Now should hit
	schemas, ok = cache.GetSchemas()
	if !ok {
		t.Error("Expected cache hit after SetSchemas")
	}
	if len(schemas) != 3 {
		t.Errorf("Expected 3 schemas, got %d", len(schemas))
	}
	if schemas[0] != "public" {
		t.Errorf("Expected first schema 'public', got '%s'", schemas[0])
	}
}

func TestMetadataCache_Tables(t *testing.T) {
	cache := NewMetadataCache(5 * time.Minute)

	// Initially, cache should miss
	tables, ok := cache.GetTables("public")
	if ok {
		t.Error("Expected cache miss for tables initially")
	}
	if tables != nil {
		t.Error("Expected nil tables on cache miss")
	}

	// Set tables for a schema
	testTables := []engine.StorageUnit{
		{Name: "users"},
		{Name: "orders"},
	}
	cache.SetTables("public", testTables)

	// Now should hit for "public"
	tables, ok = cache.GetTables("public")
	if !ok {
		t.Error("Expected cache hit after SetTables")
	}
	if len(tables) != 2 {
		t.Errorf("Expected 2 tables, got %d", len(tables))
	}

	// Should still miss for different schema
	_, ok = cache.GetTables("private")
	if ok {
		t.Error("Expected cache miss for different schema")
	}
}

func TestMetadataCache_Columns(t *testing.T) {
	cache := NewMetadataCache(5 * time.Minute)

	// Initially, cache should miss
	columns, ok := cache.GetColumns("public", "users")
	if ok {
		t.Error("Expected cache miss for columns initially")
	}
	if columns != nil {
		t.Error("Expected nil columns on cache miss")
	}

	// Set columns for a table
	testColumns := []engine.Column{
		{Name: "id", Type: "integer"},
		{Name: "name", Type: "varchar"},
	}
	cache.SetColumns("public", "users", testColumns)

	// Now should hit
	columns, ok = cache.GetColumns("public", "users")
	if !ok {
		t.Error("Expected cache hit after SetColumns")
	}
	if len(columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(columns))
	}

	// Should miss for different table
	_, ok = cache.GetColumns("public", "orders")
	if ok {
		t.Error("Expected cache miss for different table")
	}
}

func TestMetadataCache_Clear(t *testing.T) {
	cache := NewMetadataCache(5 * time.Minute)

	// Populate cache
	cache.SetSchemas([]string{"public"})
	cache.SetTables("public", []engine.StorageUnit{{Name: "users"}})
	cache.SetColumns("public", "users", []engine.Column{{Name: "id", Type: "integer"}})

	// Verify populated
	if _, ok := cache.GetSchemas(); !ok {
		t.Error("Expected schemas to be cached")
	}
	if _, ok := cache.GetTables("public"); !ok {
		t.Error("Expected tables to be cached")
	}
	if _, ok := cache.GetColumns("public", "users"); !ok {
		t.Error("Expected columns to be cached")
	}

	// Clear cache
	cache.Clear()

	// Verify cleared
	if _, ok := cache.GetSchemas(); ok {
		t.Error("Expected schemas cache miss after Clear")
	}
	if _, ok := cache.GetTables("public"); ok {
		t.Error("Expected tables cache miss after Clear")
	}
	if _, ok := cache.GetColumns("public", "users"); ok {
		t.Error("Expected columns cache miss after Clear")
	}
}

func TestMetadataCache_TTLExpiration(t *testing.T) {
	// Use a very short TTL for testing
	cache := NewMetadataCache(10 * time.Millisecond)

	cache.SetSchemas([]string{"public"})

	// Should hit immediately
	if _, ok := cache.GetSchemas(); !ok {
		t.Error("Expected cache hit immediately after set")
	}

	// Wait for TTL to expire
	time.Sleep(20 * time.Millisecond)

	// Should miss after TTL
	if _, ok := cache.GetSchemas(); ok {
		t.Error("Expected cache miss after TTL expiration")
	}
}

func TestManager_CacheInitialized(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr.GetCache() == nil {
		t.Error("Expected cache to be initialized")
	}
}

func TestManager_InvalidateCache(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Manually populate cache
	mgr.GetCache().SetSchemas([]string{"test"})

	// Verify populated
	if _, ok := mgr.GetCache().GetSchemas(); !ok {
		t.Error("Expected cache to be populated")
	}

	// Invalidate
	mgr.InvalidateCache()

	// Verify cleared
	if _, ok := mgr.GetCache().GetSchemas(); ok {
		t.Error("Expected cache to be cleared after InvalidateCache")
	}
}

func TestManager_DisconnectClearsCache(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Manually populate cache
	mgr.GetCache().SetSchemas([]string{"test"})

	// Disconnect should clear cache
	mgr.Disconnect()

	// Verify cleared
	if _, ok := mgr.GetCache().GetSchemas(); ok {
		t.Error("Expected cache to be cleared after Disconnect")
	}
}
