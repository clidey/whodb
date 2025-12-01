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

package testutil

import (
	"os"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/database"
)

func SetupTestEnvironment(t *testing.T) (string, func()) {
	t.Helper()

	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)

	cleanup := func() {
		os.Setenv("HOME", origHome)
	}

	return tempDir, cleanup
}

func CreateTestSQLiteConnection(t *testing.T, tempDir string) *config.Connection {
	t.Helper()

	dbPath := tempDir + "/test.db"
	return &config.Connection{
		Name:     "test-sqlite",
		Type:     "Sqlite",
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
