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

package e2e_test

import (
	"os"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/testutil"

	// Register database plugins for tests
	_ "github.com/clidey/whodb/core/src/plugins"
)

func TestEndToEnd_BasicWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test")
	}

	tempDir, cleanup := testutil.SetupTestEnvironment(t)
	defer cleanup()

	mgr, conn, cleanupDB := testutil.SetupTestDatabase(t)
	defer cleanupDB()

	testutil.CreateTestTable(t, mgr, "users", []string{
		"id INTEGER PRIMARY KEY",
		"name TEXT NOT NULL",
		"email TEXT NOT NULL",
	})

	testutil.InsertTestData(t, mgr, "users", []string{
		"1, 'Alice', 'alice@example.com'",
		"2, 'Bob', 'bob@example.com'",
		"3, 'Charlie', 'charlie@example.com'",
	})

	result, err := mgr.ExecuteQuery("SELECT * FROM users")
	testutil.AssertNoError(t, err, "ExecuteQuery failed")
	testutil.AssertNotNil(t, result, "Result is nil")
	testutil.AssertEqual(t, 3, len(result.Rows), "Row count mismatch")

	result, err = mgr.ExecuteQuery("SELECT * FROM users WHERE name = 'Alice'")
	testutil.AssertNoError(t, err, "ExecuteQuery with WHERE failed")
	testutil.AssertEqual(t, 1, len(result.Rows), "Filtered row count mismatch")

	csvPath := tempDir + "/users_export.csv"
	err = mgr.ExportToCSV("", "users", csvPath, ",")
	testutil.AssertNoError(t, err, "ExportToCSV failed")

	_, err = os.Stat(csvPath)
	testutil.AssertNoError(t, err, "CSV file not created")

	xlsxPath := tempDir + "/users_export.xlsx"
	err = mgr.ExportToExcel("", "users", xlsxPath)
	testutil.AssertNoError(t, err, "ExportToExcel failed")

	_, err = os.Stat(xlsxPath)
	testutil.AssertNoError(t, err, "Excel file not created")

	// SQLite doesn't support schemas like PostgreSQL/MySQL, so this may return
	// an "unsupported operation" error. We just verify the call doesn't panic.
	_, _ = mgr.GetSchemas()

	storageUnits, err := mgr.GetStorageUnits("")
	testutil.AssertNoError(t, err, "GetStorageUnits failed")
	testutil.AssertNotNil(t, storageUnits, "StorageUnits is nil")

	found := false
	for _, su := range storageUnits {
		if su.Name == "users" {
			found = true
			break
		}
	}
	if !found {
		t.Error("users table not found in storage units")
	}

	columns, err := mgr.GetColumns("", "users")
	testutil.AssertNoError(t, err, "GetColumns failed")
	testutil.AssertEqual(t, 3, len(columns), "Column count mismatch")

	rows, err := mgr.GetRows("", "users", nil, 2, 0)
	testutil.AssertNoError(t, err, "GetRows failed")
	testutil.AssertEqual(t, 2, len(rows.Rows), "Paginated row count mismatch")

	rows, err = mgr.GetRows("", "users", nil, 2, 2)
	testutil.AssertNoError(t, err, "GetRows with offset failed")
	testutil.AssertEqual(t, 1, len(rows.Rows), "Paginated row count with offset mismatch")

	testutil.AssertNotNil(t, conn, "Connection is nil")
	testutil.AssertEqual(t, "test-sqlite", conn.Name, "Connection name mismatch")
}

func TestEndToEnd_MultipleConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test")
	}

	tempDir, cleanup := testutil.SetupTestEnvironment(t)
	defer cleanup()

	mgr, _, cleanupDB1 := testutil.SetupTestDatabase(t)
	defer cleanupDB1()

	testutil.CreateTestTable(t, mgr, "db1_table", []string{
		"id INTEGER PRIMARY KEY",
		"data TEXT",
	})

	testutil.InsertTestData(t, mgr, "db1_table", []string{
		"1, 'data1'",
		"2, 'data2'",
	})

	result1, err := mgr.ExecuteQuery("SELECT * FROM db1_table")
	testutil.AssertNoError(t, err, "DB1 query failed")
	testutil.AssertEqual(t, 2, len(result1.Rows), "DB1 row count mismatch")

	err = mgr.Disconnect()
	testutil.AssertNoError(t, err, "Disconnect failed")

	// Create the db2 directory and database file
	db2Dir := tempDir + "/db2"
	if err := os.MkdirAll(db2Dir, 0755); err != nil {
		t.Fatalf("Failed to create db2 directory: %v", err)
	}
	conn2 := testutil.CreateTestSQLiteConnection(t, db2Dir)
	// Create the database file before connecting (required by SQLite plugin)
	db2File, err := os.Create(conn2.Database)
	if err != nil {
		t.Fatalf("Failed to create db2 database file: %v", err)
	}
	db2File.Close()

	err = mgr.Connect(conn2)
	testutil.AssertNoError(t, err, "Connect to DB2 failed")

	testutil.CreateTestTable(t, mgr, "db2_table", []string{
		"id INTEGER PRIMARY KEY",
		"value TEXT",
	})

	testutil.InsertTestData(t, mgr, "db2_table", []string{
		"1, 'value1'",
		"2, 'value2'",
		"3, 'value3'",
	})

	result2, err := mgr.ExecuteQuery("SELECT * FROM db2_table")
	testutil.AssertNoError(t, err, "DB2 query failed")
	testutil.AssertEqual(t, 3, len(result2.Rows), "DB2 row count mismatch")
}

func TestEndToEnd_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test")
	}

	_, cleanup := testutil.SetupTestEnvironment(t)
	defer cleanup()

	mgr, _, cleanupDB := testutil.SetupTestDatabase(t)
	defer cleanupDB()

	_, err := mgr.ExecuteQuery("SELECT * FROM nonexistent_table")
	testutil.AssertError(t, err, "Expected error for nonexistent table")

	_, err = mgr.ExecuteQuery("INVALID SQL SYNTAX")
	testutil.AssertError(t, err, "Expected error for invalid SQL")

	_, err = mgr.GetColumns("", "nonexistent_table")
	testutil.AssertError(t, err, "Expected error for nonexistent table columns")

	err = mgr.ExportToCSV("", "nonexistent_table", "/tmp/test.csv", ",")
	testutil.AssertError(t, err, "Expected error for exporting nonexistent table")
}

func TestEndToEnd_ConfigPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test")
	}

	_, cleanup := testutil.SetupTestEnvironment(t)
	defer cleanup()

	_, conn, cleanupDB := testutil.SetupTestDatabase(t)
	defer cleanupDB()

	cfg, err := config.LoadConfig()
	testutil.AssertNoError(t, err, "LoadConfig failed")

	cfg.AddConnection(*conn)
	err = cfg.Save()
	testutil.AssertNoError(t, err, "Config save failed")

	mgr2, err := database.NewManager()
	testutil.AssertNoError(t, err, "NewManager failed")

	conns := mgr2.ListConnections()
	if len(conns) == 0 {
		t.Error("Expected saved connections to persist")
	}

	found := false
	for _, c := range conns {
		if c.Name == conn.Name {
			found = true
			break
		}
	}
	if !found {
		t.Error("Saved connection not found after reload")
	}

	// Verify config file was created at the proper location
	configPath, err := config.GetConfigPath()
	testutil.AssertNoError(t, err, "GetConfigPath failed")
	_, err = os.Stat(configPath)
	testutil.AssertNoError(t, err, "Config file not created")
}
