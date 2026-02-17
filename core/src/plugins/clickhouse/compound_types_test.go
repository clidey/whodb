//go:build integration

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

package clickhouse

import (
	"context"
	"strings"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	gorm_clickhouse "gorm.io/driver/clickhouse"
	"gorm.io/gorm"
)

// TestGormExprCompoundTypes verifies that gorm.Expr() works for ClickHouse
// Map, Tuple, and Array literals through GORM's Create() path.
// Run with: go test -tags integration -run TestGormExprCompoundTypes ./src/plugins/clickhouse/
func TestGormExprCompoundTypes(t *testing.T) {
	db := connectTestDB(t)

	// Create test table with compound types
	db.Exec("DROP TABLE IF EXISTS test_db.test_compound_types")
	err := db.Exec(`
		CREATE TABLE test_db.test_compound_types (
			id UInt32,
			col_map Map(String, Int32),
			col_tuple Tuple(String, Int32, Float64),
			col_array Array(Int32),
			col_enum Enum8('active' = 1, 'inactive' = 2)
		) ENGINE = MergeTree() ORDER BY id
	`).Error
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}
	defer db.Exec("DROP TABLE IF EXISTS test_db.test_compound_types")

	// Use our converters to go from string literals to typed Go values
	mapVal, err := convertMapLiteral("{'key1': 10, 'key2': 20}", "Map(String, Int32)")
	if err != nil {
		t.Fatalf("Failed to convert map: %v", err)
	}
	tupleVal, err := convertTupleLiteral("('hello', 42, 3.14)", "Tuple(String, Int32, Float64)")
	if err != nil {
		t.Fatalf("Failed to convert tuple: %v", err)
	}

	data := map[string]any{
		"id":        uint32(1),
		"col_map":   mapVal,
		"col_tuple": tupleVal,
		"col_array": []int32{1, 2, 3},
		"col_enum":  "active",
	}

	result := db.Table("test_db.test_compound_types").Create(data)
	if result.Error != nil {
		t.Fatalf("Failed to insert with gorm.Expr: %v", result.Error)
	}

	// Verify the data was inserted correctly
	var count int64
	db.Raw("SELECT count() FROM test_db.test_compound_types WHERE id = 1").Scan(&count)
	if count != 1 {
		t.Fatalf("Expected 1 row, got %d", count)
	}

	// Read back and verify values
	var readMap, readTuple, readArray, readEnum string
	db.Raw("SELECT toString(col_map), toString(col_tuple), toString(col_array), col_enum FROM test_db.test_compound_types WHERE id = 1").
		Row().Scan(&readMap, &readTuple, &readArray, &readEnum)

	t.Logf("Map:   %s", readMap)
	t.Logf("Tuple: %s", readTuple)
	t.Logf("Array: %s", readArray)
	t.Logf("Enum:  %s", readEnum)

	if readEnum != "active" {
		t.Errorf("Expected enum 'active', got %q", readEnum)
	}

	// Test FormatColumnValue produces ClickHouse literal syntax
	plugin := &ClickHousePlugin{}
	var scannedMap, scannedTuple, scannedArray any
	row := db.Raw("SELECT col_map, col_tuple, col_array FROM test_db.test_compound_types WHERE id = 1").Row()
	if err := row.Scan(&scannedMap, &scannedTuple, &scannedArray); err != nil {
		t.Fatalf("Failed to scan compound types: %v", err)
	}

	fmtMap, _ := plugin.FormatColumnValue("Map(String, Int32)", &scannedMap)
	fmtTuple, _ := plugin.FormatColumnValue("Tuple(String, Int32, Float64)", &scannedTuple)
	fmtArray, _ := plugin.FormatColumnValue("Array(Int32)", &scannedArray)

	t.Logf("Map   display: %s", fmtMap)
	t.Logf("Tuple display: %s", fmtTuple)
	t.Logf("Array display: %s", fmtArray)

	// Map should use ClickHouse literal syntax with single quotes
	if !strings.HasPrefix(fmtMap, "{") || !strings.HasSuffix(fmtMap, "}") {
		t.Errorf("Map display should use {}, got %q", fmtMap)
	}
	if !strings.Contains(fmtMap, "'key1'") {
		t.Errorf("Map display should single-quote keys, got %q", fmtMap)
	}

	// Tuple should use ClickHouse tuple syntax
	if !strings.HasPrefix(fmtTuple, "(") || !strings.HasSuffix(fmtTuple, ")") {
		t.Errorf("Tuple display should use (), got %q", fmtTuple)
	}
	if !strings.Contains(fmtTuple, "'hello'") {
		t.Errorf("Tuple display should single-quote strings, got %q", fmtTuple)
	}

	// Array should use bracket syntax
	if !strings.HasPrefix(fmtArray, "[") || !strings.HasSuffix(fmtArray, "]") {
		t.Errorf("Array display should use [], got %q", fmtArray)
	}
}

// connectTestDB connects to the local ClickHouse test instance.
func connectTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{"localhost:9000"},
		Auth: clickhouse.Auth{
			Database: "test_db",
			Username: "user",
			Password: "password",
		},
	})

	if err := conn.PingContext(context.Background()); err != nil {
		t.Skipf("ClickHouse not available: %v", err)
	}

	db, err := gorm.Open(gorm_clickhouse.New(gorm_clickhouse.Config{
		Conn: conn,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open GORM connection: %v", err)
	}

	return db
}
