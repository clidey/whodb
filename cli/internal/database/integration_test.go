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
	"encoding/csv"
	"os"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/xuri/excelize/v2"
)

func TestSQLiteIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	setupTestEnv(t)
	tempDir := t.TempDir()

	dbPath := tempDir + "/test.db"

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	conn := &config.Connection{
		Name:     "test-sqlite",
		Type:     "Sqlite",
		Host:     dbPath,
		Database: dbPath,
	}

	err = mgr.Connect(conn)
	if err != nil {
		t.Skipf("Skipping test - database plugin not available: %v", err)
	}

	if mgr.GetCurrentConnection() == nil {
		t.Fatal("Expected current connection to be set")
	}

	_, err = mgr.ExecuteQuery("CREATE TABLE IF NOT EXISTS test_users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)")
	if err != nil {
		t.Fatalf("Create table failed: %v", err)
	}

	_, err = mgr.ExecuteQuery("INSERT INTO test_users (name, email) VALUES ('John Doe', 'john@example.com')")
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	result, err := mgr.ExecuteQuery("SELECT * FROM test_users")
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if len(result.Rows) == 0 {
		t.Error("Expected at least one row")
	}

	schemas, err := mgr.GetSchemas()
	if err != nil {
		t.Fatalf("GetSchemas failed: %v", err)
	}

	if len(schemas) == 0 {
		t.Error("Expected at least one schema")
	}

	storageUnits, err := mgr.GetStorageUnits("")
	if err != nil {
		t.Fatalf("GetStorageUnits failed: %v", err)
	}

	found := false
	for _, su := range storageUnits {
		if su.Name == "test_users" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find test_users table")
	}

	columns, err := mgr.GetColumns("", "test_users")
	if err != nil {
		t.Fatalf("GetColumns failed: %v", err)
	}

	if len(columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(columns))
	}

	rows, err := mgr.GetRows("", "test_users", nil, 50, 0)
	if err != nil {
		t.Fatalf("GetRows failed: %v", err)
	}

	if len(rows.Rows) == 0 {
		t.Error("Expected at least one row from GetRows")
	}

	err = mgr.Disconnect()
	if err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}

	if mgr.GetCurrentConnection() != nil {
		t.Error("Expected current connection to be nil after disconnect")
	}
}

func TestSQLiteGetGraphIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	setupTestEnv(t)
	tempDir := t.TempDir()
	dbPath := tempDir + "/graph.db"

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	conn := &config.Connection{
		Name:     "test-sqlite-graph",
		Type:     "Sqlite",
		Host:     dbPath,
		Database: dbPath,
	}

	if err := mgr.Connect(conn); err != nil {
		t.Skipf("Skipping test - database plugin not available: %v", err)
	}

	_, err = mgr.ExecuteQuery("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("Create users table failed: %v", err)
	}

	_, err = mgr.ExecuteQuery("CREATE TABLE orders (id INTEGER PRIMARY KEY, user_id INTEGER REFERENCES users(id), total REAL)")
	if err != nil {
		t.Fatalf("Create orders table failed: %v", err)
	}

	graphUnits, err := mgr.GetGraph("")
	if err != nil {
		t.Fatalf("GetGraph failed: %v", err)
	}

	if len(graphUnits) < 2 {
		t.Fatalf("expected at least two graph units, got %d", len(graphUnits))
	}

	var usersGraphUnit *engine.GraphUnit
	for i := range graphUnits {
		if graphUnits[i].Unit.Name == "users" {
			usersGraphUnit = &graphUnits[i]
			break
		}
	}

	if usersGraphUnit == nil {
		t.Fatal("expected users graph unit")
	}

	var ordersRelation *engine.GraphUnitRelationship
	for i := range usersGraphUnit.Relations {
		if usersGraphUnit.Relations[i].Name == "orders" {
			ordersRelation = &usersGraphUnit.Relations[i]
			break
		}
	}

	if ordersRelation == nil {
		t.Fatal("expected users -> orders relation in graph output")
	}
	if ordersRelation.RelationshipType != "OneToMany" {
		t.Fatalf("expected OneToMany relation, got %q", ordersRelation.RelationshipType)
	}
	if ordersRelation.SourceColumn == nil || *ordersRelation.SourceColumn != "user_id" {
		t.Fatalf("expected source column user_id, got %#v", ordersRelation.SourceColumn)
	}
	if ordersRelation.TargetColumn == nil || *ordersRelation.TargetColumn != "id" {
		t.Fatalf("expected target column id, got %#v", ordersRelation.TargetColumn)
	}
}

func TestExportIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	setupTestEnv(t)
	tempDir := t.TempDir()

	dbPath := tempDir + "/test.db"

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	conn := &config.Connection{
		Name:     "test-sqlite",
		Type:     "Sqlite",
		Host:     dbPath,
		Database: dbPath,
	}

	err = mgr.Connect(conn)
	if err != nil {
		t.Skipf("Skipping test - database plugin not available: %v", err)
	}

	_, err = mgr.ExecuteQuery("CREATE TABLE IF NOT EXISTS export_test (id INTEGER PRIMARY KEY, data TEXT)")
	if err != nil {
		t.Fatalf("Create table failed: %v", err)
	}

	_, err = mgr.ExecuteQuery("INSERT INTO export_test (data) VALUES ('test1'), ('test2'), ('test3')")
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	csvPath := tempDir + "/export.csv"
	err = mgr.ExportToCSV("", "export_test", csvPath, ",")
	if err != nil {
		t.Fatalf("ExportToCSV failed: %v", err)
	}

	csvFile, err := os.Open(csvPath)
	if err != nil {
		t.Fatalf("CSV file was not created: %v", err)
	}
	csvRecords, err := csv.NewReader(csvFile).ReadAll()
	_ = csvFile.Close()
	if err != nil {
		t.Fatalf("Reading exported CSV failed: %v", err)
	}
	if len(csvRecords) != 4 {
		t.Fatalf("CSV export has %d records, want the header and 3 rows", len(csvRecords))
	}

	xlsxPath := tempDir + "/export.xlsx"
	err = mgr.ExportToExcel("", "export_test", xlsxPath)
	if err != nil {
		t.Fatalf("ExportToExcel failed: %v", err)
	}

	xlsxFile, err := excelize.OpenFile(xlsxPath)
	if err != nil {
		t.Fatalf("Excel file was not created: %v", err)
	}
	xlsxRows, err := xlsxFile.GetRows("Sheet1")
	_ = xlsxFile.Close()
	if err != nil {
		t.Fatalf("Reading exported Excel file failed: %v", err)
	}
	if len(xlsxRows) != 4 {
		t.Fatalf("Excel export has %d rows, want the header and 3 rows", len(xlsxRows))
	}

	result, err := mgr.ExecuteQuery("SELECT * FROM export_test")
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}

	resultCSVPath := tempDir + "/result_export.csv"
	err = mgr.ExportResultsToCSV(result, resultCSVPath, ",")
	if err != nil {
		t.Fatalf("ExportResultsToCSV failed: %v", err)
	}

	if _, err := os.Stat(resultCSVPath); os.IsNotExist(err) {
		t.Errorf("Result CSV file was not created: %s", resultCSVPath)
	}

	resultXLSXPath := tempDir + "/result_export.xlsx"
	err = mgr.ExportResultsToExcel(result, resultXLSXPath)
	if err != nil {
		t.Fatalf("ExportResultsToExcel failed: %v", err)
	}

	if _, err := os.Stat(resultXLSXPath); os.IsNotExist(err) {
		t.Errorf("Result Excel file was not created: %s", resultXLSXPath)
	}
}

func TestConnect_InvalidPlugin(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	conn := &config.Connection{
		Name:     "invalid-db",
		Type:     "InvalidDBType",
		Host:     "localhost",
		Database: "testdb",
	}

	err = mgr.Connect(conn)
	if err == nil {
		t.Error("Expected error when connecting with invalid plugin")
	}
}

func TestConnect_UnavailableDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	conn := &config.Connection{
		Name:     "unavailable-postgres",
		Type:     "Postgres",
		Host:     "nonexistent-host-12345.example.com",
		Port:     5432,
		Username: "testuser",
		Password: "testpass",
		Database: "testdb",
	}

	err = mgr.Connect(conn)
	if err == nil {
		t.Error("Expected error when connecting to unavailable database")
	}
}
