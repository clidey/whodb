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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if len(cfg.Connections) != 0 {
		t.Errorf("Expected 0 connections, got %d", len(cfg.Connections))
	}

	if cfg.History.MaxEntries != 1000 {
		t.Errorf("Expected MaxEntries 1000, got %d", cfg.History.MaxEntries)
	}

	if !cfg.History.Persist {
		t.Error("Expected Persist to be true")
	}

	if cfg.Display.Theme != "dark" {
		t.Errorf("Expected theme 'dark', got '%s'", cfg.Display.Theme)
	}

	if cfg.Display.PageSize != 50 {
		t.Errorf("Expected PageSize 50, got %d", cfg.Display.PageSize)
	}
}

func TestAddConnection(t *testing.T) {
	cfg := DefaultConfig()

	conn := Connection{
		Name:     "test-db",
		Type:     "postgres",
		Host:     "localhost",
		Port:     5432,
		Username: "testuser",
		Database: "testdb",
	}

	cfg.AddConnection(conn)

	if len(cfg.Connections) != 1 {
		t.Fatalf("Expected 1 connection, got %d", len(cfg.Connections))
	}

	if cfg.Connections[0].Name != "test-db" {
		t.Errorf("Expected connection name 'test-db', got '%s'", cfg.Connections[0].Name)
	}
}

func TestAddConnection_Update(t *testing.T) {
	cfg := DefaultConfig()

	conn1 := Connection{
		Name:     "test-db",
		Type:     "postgres",
		Host:     "localhost",
		Port:     5432,
		Username: "testuser",
		Database: "testdb",
	}

	cfg.AddConnection(conn1)

	conn2 := Connection{
		Name:     "test-db",
		Type:     "mysql",
		Host:     "localhost",
		Port:     3306,
		Username: "newuser",
		Database: "newdb",
	}

	cfg.AddConnection(conn2)

	if len(cfg.Connections) != 1 {
		t.Fatalf("Expected 1 connection after update, got %d", len(cfg.Connections))
	}

	if cfg.Connections[0].Type != "mysql" {
		t.Errorf("Expected updated type 'mysql', got '%s'", cfg.Connections[0].Type)
	}

	if cfg.Connections[0].Port != 3306 {
		t.Errorf("Expected updated port 3306, got %d", cfg.Connections[0].Port)
	}
}

func TestRemoveConnection(t *testing.T) {
	cfg := DefaultConfig()

	conn1 := Connection{Name: "conn1", Type: "postgres", Host: "localhost"}
	conn2 := Connection{Name: "conn2", Type: "mysql", Host: "localhost"}

	cfg.AddConnection(conn1)
	cfg.AddConnection(conn2)

	if len(cfg.Connections) != 2 {
		t.Fatalf("Expected 2 connections, got %d", len(cfg.Connections))
	}

	removed := cfg.RemoveConnection("conn1")
	if !removed {
		t.Error("Expected RemoveConnection to return true")
	}

	if len(cfg.Connections) != 1 {
		t.Fatalf("Expected 1 connection after removal, got %d", len(cfg.Connections))
	}

	if cfg.Connections[0].Name != "conn2" {
		t.Errorf("Expected remaining connection 'conn2', got '%s'", cfg.Connections[0].Name)
	}
}

func TestRemoveConnection_NotFound(t *testing.T) {
	cfg := DefaultConfig()

	removed := cfg.RemoveConnection("nonexistent")
	if removed {
		t.Error("Expected RemoveConnection to return false for nonexistent connection")
	}
}

func TestGetConnection(t *testing.T) {
	cfg := DefaultConfig()

	conn := Connection{
		Name:     "test-db",
		Type:     "postgres",
		Host:     "localhost",
		Port:     5432,
		Username: "testuser",
		Database: "testdb",
	}

	cfg.AddConnection(conn)

	retrieved, err := cfg.GetConnection("test-db")
	if err != nil {
		t.Fatalf("GetConnection failed: %v", err)
	}

	if retrieved.Name != "test-db" {
		t.Errorf("Expected name 'test-db', got '%s'", retrieved.Name)
	}

	if retrieved.Type != "postgres" {
		t.Errorf("Expected type 'postgres', got '%s'", retrieved.Type)
	}
}

func TestGetConnection_NotFound(t *testing.T) {
	cfg := DefaultConfig()

	_, err := cfg.GetConnection("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent connection")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	cfg := DefaultConfig()
	conn := Connection{
		Name:     "test-db",
		Type:     "postgres",
		Host:     "localhost",
		Port:     5432,
		Username: "testuser",
		Database: "testdb",
	}
	cfg.AddConnection(conn)

	err := cfg.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	configPath := filepath.Join(tempDir, ".whodb-cli", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created at %s", configPath)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if len(loaded.Connections) != 1 {
		t.Fatalf("Expected 1 connection in loaded config, got %d", len(loaded.Connections))
	}

	if loaded.Connections[0].Name != "test-db" {
		t.Errorf("Expected connection name 'test-db', got '%s'", loaded.Connections[0].Name)
	}
}

func TestLoadConfig_CreatesDefault(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig returned nil")
	}

	configPath := filepath.Join(tempDir, ".whodb-cli", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Default config file was not created at %s", configPath)
	}
}

func TestGetConfigDir(t *testing.T) {
	dir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir failed: %v", err)
	}

	if dir == "" {
		t.Error("GetConfigDir returned empty string")
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("Config directory was not created: %s", dir)
	}
}

func TestGetConfigPath(t *testing.T) {
	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath failed: %v", err)
	}

	if path == "" {
		t.Error("GetConfigPath returned empty string")
	}

	if filepath.Ext(path) != ".yaml" {
		t.Errorf("Expected .yaml extension, got '%s'", filepath.Ext(path))
	}
}
