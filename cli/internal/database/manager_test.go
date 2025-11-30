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
	"os"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
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
