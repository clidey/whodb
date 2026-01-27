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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
)

var (
	testHomeOnce sync.Once
	testHome     string
)

func setupTestEnv(t *testing.T) {
	t.Helper()
	testHomeOnce.Do(func() {
		dir, err := os.MkdirTemp("", "whodb-cli-test-home-")
		if err != nil {
			t.Fatalf("Failed to create test home: %v", err)
		}
		testHome = dir
	})
	if err := os.Setenv("HOME", testHome); err != nil {
		t.Fatalf("Failed to set HOME: %v", err)
	}
	if err := os.Setenv("USERPROFILE", testHome); err != nil {
		t.Fatalf("Failed to set USERPROFILE: %v", err)
	}
	if err := os.Setenv("XDG_DATA_HOME", testHome); err != nil {
		t.Fatalf("Failed to set XDG_DATA_HOME: %v", err)
	}
	if err := os.Setenv("APPDATA", testHome); err != nil {
		t.Fatalf("Failed to set APPDATA: %v", err)
	}

	cleanupConfigFiles(t)

	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		key := parts[0]
		if strings.HasPrefix(key, "WHODB_") && !strings.HasPrefix(key, "WHODB_CLI_") {
			t.Setenv(key, "")
		}
	}
}

func cleanupConfigFiles(t *testing.T) {
	t.Helper()

	configPath, err := config.GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath failed: %v", err)
	}
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Remove config file failed: %v", err)
	}

	configDir, err := config.GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir failed: %v", err)
	}
	historyPath := filepath.Join(configDir, "history.json")
	if err := os.Remove(historyPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Remove history file failed: %v", err)
	}
}

func TestResolveConnection_FromEnvProfile(t *testing.T) {
	setupTestEnv(t)

	t.Setenv("WHODB_POSTGRES", `[{"alias":"prod","host":"localhost","user":"alice","password":"secret","database":"app","port":"5432"}]`)

	conn, err := ResolveConnection("prod")
	if err != nil {
		t.Fatalf("ResolveConnection() error = %v", err)
	}

	if conn.Type != "Postgres" {
		t.Errorf("Type = %v, want Postgres", conn.Type)
	}
	if conn.Port != 5432 {
		t.Errorf("Port = %v, want 5432", conn.Port)
	}
	if conn.Database != "app" {
		t.Errorf("Database = %v, want app", conn.Database)
	}
	if !conn.IsProfile {
		t.Error("Expected env connection to be marked as profile")
	}
}

func TestResolveConnection_FromEnvProfileWithGeneratedName(t *testing.T) {
	setupTestEnv(t)

	t.Setenv("WHODB_MYSQL_1", `{"host":"db.local","user":"bob","password":"pw","database":"northwind","port":"3307"}`)

	conn, err := ResolveConnection("mysql-1")
	if err != nil {
		t.Fatalf("ResolveConnection() error = %v", err)
	}

	if conn.Type != "MySQL" {
		t.Errorf("Type = %v, want MySQL", conn.Type)
	}
	if conn.Port != 3307 {
		t.Errorf("Port = %v, want 3307", conn.Port)
	}
}

func TestResolveConnection_SavedOverridesEnv(t *testing.T) {
	setupTestEnv(t)

	cfg := config.DefaultConfig()
	cfg.AddConnection(config.Connection{
		Name:     "prod",
		Type:     "Postgres",
		Host:     "saved-host",
		Port:     5432,
		Username: "saved-user",
		Database: "saved-db",
	})
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	t.Setenv("WHODB_POSTGRES", `[{"alias":"prod","host":"env-host","user":"env-user","password":"env-pass","database":"env-db"}]`)

	conn, err := ResolveConnection("prod")
	if err != nil {
		t.Fatalf("ResolveConnection() error = %v", err)
	}
	if conn.Host != "saved-host" {
		t.Errorf("Host = %v, want saved-host", conn.Host)
	}
}

func TestListAvailableConnections_IncludesSavedAndEnv(t *testing.T) {
	setupTestEnv(t)

	cfg := config.DefaultConfig()
	cfg.AddConnection(config.Connection{
		Name:     "local",
		Type:     "Postgres",
		Host:     "localhost",
		Port:     5432,
		Username: "user",
		Database: "db",
	})
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	t.Setenv("WHODB_POSTGRES", `[{"alias":"prod","host":"env-host","user":"env-user","password":"env-pass","database":"env-db"}]`)

	conns, err := ListAvailableConnections()
	if err != nil {
		t.Fatalf("ListAvailableConnections() error = %v", err)
	}

	connMap := make(map[string]bool)
	for _, c := range conns {
		connMap[c] = true
	}

	if !connMap["local"] {
		t.Error("Expected saved connection 'local'")
	}
	if !connMap["prod"] {
		t.Error("Expected env connection 'prod'")
	}
}

func TestResolveConnectionOrDefault_SingleConnection(t *testing.T) {
	setupTestEnv(t)

	t.Setenv("WHODB_POSTGRES", `[{"alias":"only","host":"localhost","user":"alice","password":"secret","database":"app"}]`)

	conn, err := ResolveConnectionOrDefault("")
	if err != nil {
		t.Fatalf("ResolveConnectionOrDefault() error = %v", err)
	}
	if conn.Name != "only" {
		t.Errorf("Name = %v, want only", conn.Name)
	}
}

func TestResolveConnectionOrDefault_MultipleConnections(t *testing.T) {
	setupTestEnv(t)

	t.Setenv("WHODB_POSTGRES", `[{"alias":"db1","host":"localhost","user":"a","password":"p","database":"db1"},{"alias":"db2","host":"localhost","user":"b","password":"p","database":"db2"}]`)

	_, err := ResolveConnectionOrDefault("")
	if err == nil {
		t.Fatal("Expected error for multiple connections without name")
	}
	if !strings.Contains(err.Error(), "multiple connections") {
		t.Fatalf("Expected 'multiple connections' error, got: %v", err)
	}
}

func TestResolveConnectionOrDefault_NoConnections(t *testing.T) {
	setupTestEnv(t)

	_, err := ResolveConnectionOrDefault("")
	if err == nil {
		t.Fatal("Expected error for no connections")
	}
	if !strings.Contains(err.Error(), "no database connections") {
		t.Fatalf("Expected 'no database connections' error, got: %v", err)
	}
}
