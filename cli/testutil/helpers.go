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

package testutil

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/database"
)

var (
	testHomeOnce sync.Once
	testHome     string
)

func SetupTestEnvironment(t *testing.T) (string, func()) {
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
	// Enable CLI mode so SQLite plugin uses paths directly without /db/ prefix
	if err := os.Setenv("WHODB_CLI", "true"); err != nil {
		t.Fatalf("Failed to set WHODB_CLI: %v", err)
	}
	cleanupConfigFiles(t)

	tempDir := t.TempDir()

	cleanup := func() {
		cleanupConfigFiles(t)
	}

	return tempDir, cleanup
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

func CreateTestSQLiteConnection(t *testing.T, tempDir string) *config.Connection {
	t.Helper()

	dbPath := tempDir + "/test.db"
	return &config.Connection{
		Name:     "test-sqlite",
		Type:     "Sqlite3",
		Host:     dbPath,
		Database: dbPath,
	}
}

func SetupTestDatabase(t *testing.T) (*database.Manager, *config.Connection, func()) {
	t.Helper()

	tempDir, cleanupEnv := SetupTestEnvironment(t)

	mgr, err := database.NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	conn := CreateTestSQLiteConnection(t, tempDir)

	// Create the SQLite database file before connecting
	// The SQLite plugin requires the file to exist in desktop mode
	dbFile, err := os.Create(conn.Database)
	if err != nil {
		t.Fatalf("Failed to create SQLite database file: %v", err)
	}
	dbFile.Close()

	err = mgr.Connect(conn)
	if err != nil {
		t.Skipf("Skipping test - database plugin not available: %v", err)
	}

	cleanup := func() {
		mgr.Disconnect()
		cleanupEnv()
	}

	return mgr, conn, cleanup
}

func CreateTestTable(t *testing.T, mgr *database.Manager, tableName string, columns []string) {
	t.Helper()

	query := "CREATE TABLE IF NOT EXISTS " + tableName + " ("
	for i, col := range columns {
		if i > 0 {
			query += ", "
		}
		query += col
	}
	query += ")"

	_, err := mgr.ExecuteQuery(query)
	if err != nil {
		t.Fatalf("CreateTestTable failed: %v", err)
	}
}

func InsertTestData(t *testing.T, mgr *database.Manager, tableName string, values []string) {
	t.Helper()

	for _, value := range values {
		query := "INSERT INTO " + tableName + " VALUES (" + value + ")"
		_, err := mgr.ExecuteQuery(query)
		if err != nil {
			t.Fatalf("InsertTestData failed: %v", err)
		}
	}
}

func AssertNoError(t *testing.T, err error, message string) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s: %v", message, err)
	}
}

func AssertError(t *testing.T, err error, message string) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s: expected error but got none", message)
	}
}

func AssertEqual(t *testing.T, expected, actual any, message string) {
	t.Helper()

	if expected != actual {
		t.Errorf("%s: expected %v, got %v", message, expected, actual)
	}
}

func AssertNotNil(t *testing.T, value any, message string) {
	t.Helper()

	if value == nil {
		t.Fatalf("%s: expected non-nil value", message)
	}
}
