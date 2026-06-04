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

package sqlite3

import (
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/engine"
)

func newSQLiteRuntimeTestFixture(t *testing.T, statements ...string) (*Sqlite3Plugin, *engine.PluginConfig, *gorm.DB) {
	t.Helper()

	t.Setenv("WHODB_CLI", "true")
	dbPath := filepath.Join(t.TempDir(), "runtime.sqlite")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB handle: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("failed to execute setup statement %q: %v", statement, err)
		}
	}

	plugin := NewSqlite3Plugin().PluginFunctions.(*Sqlite3Plugin)
	config := engine.NewPluginConfig(&engine.Credentials{
		Type:     string(engine.DatabaseType_Sqlite3),
		Database: dbPath,
	})
	return plugin, config, db
}

func findSQLiteColumn(columns []engine.Column, name string) *engine.Column {
	for i := range columns {
		if columns[i].Name == name {
			return &columns[i]
		}
	}
	return nil
}

func TestSQLiteColumnMetadataAndGeneratedColumns(t *testing.T) {
	plugin, config, db := newSQLiteRuntimeTestFixture(t,
		`CREATE TABLE parents (id INTEGER PRIMARY KEY, name TEXT);`,
		`CREATE TABLE orders (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER REFERENCES parents(id),
			qty INTEGER,
			total INTEGER GENERATED ALWAYS AS (qty * 2) STORED
		) STRICT;`,
	)

	if !plugin.IsTableStrict(db, "orders") {
		t.Fatal("expected STRICT tables to be detected")
	}
	if plugin.IsTableStrict(db, "parents") {
		t.Fatal("did not expect non-STRICT table to be marked strict")
	}

	columns, err := plugin.GetColumnsForTable(config, "", "orders")
	if err != nil {
		t.Fatalf("expected column lookup to succeed, got %v", err)
	}
	if err := plugin.MarkGeneratedColumns(config, "", "orders", columns); err != nil {
		t.Fatalf("expected generated column marking to succeed, got %v", err)
	}

	idCol := findSQLiteColumn(columns, "id")
	if idCol == nil || !idCol.IsPrimary || !idCol.IsAutoIncrement {
		t.Fatalf("expected INTEGER PRIMARY KEY to be marked auto increment, got %#v", idCol)
	}

	parentCol := findSQLiteColumn(columns, "parent_id")
	if parentCol == nil || !parentCol.IsForeignKey || parentCol.ReferencedTable == nil || *parentCol.ReferencedTable != "parents" {
		t.Fatalf("expected foreign key metadata to be populated, got %#v", parentCol)
	}

	generatedOnly := []engine.Column{{Name: "total", Type: "INTEGER"}}
	if err := plugin.MarkGeneratedColumns(config, "", "orders", generatedOnly); err != nil {
		t.Fatalf("expected generated-only column marking to succeed, got %v", err)
	}
	totalCol := findSQLiteColumn(generatedOnly, "total")
	if totalCol == nil || !totalCol.IsComputed {
		t.Fatalf("expected generated column to be marked computed, got %#v", totalCol)
	}

	relationships, err := plugin.GetForeignKeyRelationships(config, "", "orders")
	if err != nil {
		t.Fatalf("expected foreign key relationship lookup to succeed, got %v", err)
	}
	if rel, ok := relationships["parent_id"]; !ok || rel.ReferencedTable != "parents" || rel.ReferencedColumn != "id" {
		t.Fatalf("expected parent_id relationship to be returned, got %#v", relationships)
	}
}

func TestSQLiteGetColumnConstraintsParsesChecksAndUniqueIndexes(t *testing.T) {
	plugin, config, _ := newSQLiteRuntimeTestFixture(t,
		`CREATE TABLE products (
			id INTEGER PRIMARY KEY,
			sku TEXT UNIQUE,
			price REAL NOT NULL CHECK(price >= 0),
			status TEXT CHECK(status IN ('active', 'archived'))
		);`,
	)

	constraints, err := plugin.GetColumnConstraints(config, "", "products")
	if err != nil {
		t.Fatalf("expected constraint lookup to succeed, got %v", err)
	}

	if constraints["id"]["primary"] != true || constraints["id"]["unique"] != true {
		t.Fatalf("expected primary key constraints for id, got %#v", constraints["id"])
	}
	if constraints["sku"]["unique"] != true {
		t.Fatalf("expected UNIQUE index to be mapped for sku, got %#v", constraints["sku"])
	}
	if constraints["price"]["nullable"] != false {
		t.Fatalf("expected NOT NULL to be mapped for price, got %#v", constraints["price"])
	}
	if min, ok := constraints["price"]["check_min"].(float64); !ok || min != 0 {
		t.Fatalf("expected price check_min=0, got %#v", constraints["price"])
	}
	if values, ok := constraints["status"]["check_values"].([]string); !ok || len(values) != 2 || values[0] != "active" || values[1] != "archived" {
		t.Fatalf("expected enum values for status, got %#v", constraints["status"])
	}
}

func TestSQLiteRawExecutePreservesDateTimeAndBlobValues(t *testing.T) {
	plugin, config, _ := newSQLiteRuntimeTestFixture(t,
		`CREATE TABLE events (
			id INTEGER PRIMARY KEY,
			created_at DATETIME,
			payload BLOB,
			name TEXT
		);`,
		`INSERT INTO events (created_at, payload, name) VALUES ('not-a-date', X'CAFE', 'alice');`,
	)

	result, err := plugin.RawExecute(config, "SELECT created_at, payload, name FROM events;")
	if err != nil {
		t.Fatalf("expected raw execution to succeed, got %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected one row, got %#v", result)
	}
	if result.Rows[0][0] != "not-a-date" {
		t.Fatalf("expected datetime text to be preserved, got %#v", result.Rows[0])
	}
	if result.Rows[0][1] != "0xcafe" {
		t.Fatalf("expected blob to be hex encoded, got %#v", result.Rows[0])
	}
	if len(result.Columns) != 3 || result.Columns[0].Type != "DATETIME" || result.Columns[1].Type != "BLOB" {
		t.Fatalf("expected original sqlite column types to be restored, got %#v", result.Columns)
	}
}
