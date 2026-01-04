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

package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/core/src/engine"
)

func TestNewManager(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.engine == nil {
		t.Fatal("Manager engine is nil")
	}

	if mgr.config == nil {
		t.Fatal("Manager config is nil")
	}
}

func TestGetCurrentConnection_NotConnected(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	conn := mgr.GetCurrentConnection()
	if conn != nil {
		t.Error("Expected nil connection when not connected")
	}
}

func TestDisconnect(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tables, ok = cache.GetTables("private")
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
	columns, ok = cache.GetColumns("public", "orders")
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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr.GetCache() == nil {
		t.Error("Expected cache to be initialized")
	}
}

func TestManager_InvalidateCache(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
