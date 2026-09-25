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
	"github.com/clidey/whodb/core/src/importer"
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

func TestSQLiteReadOnlyRawExecuteRejectsWrites(t *testing.T) {
	plugin, config, _ := newSQLiteRuntimeTestFixture(t,
		"CREATE TABLE read_only_guard (id INTEGER PRIMARY KEY)",
		"INSERT INTO read_only_guard VALUES (1)",
	)
	config.ReadOnly = true

	queries := []string{
		"INSERT INTO read_only_guard VALUES (2)",
		"WITH x AS (SELECT 1) DELETE FROM read_only_guard WHERE id=1",
	}
	for _, query := range queries {
		if _, err := plugin.RawExecute(config, query); err == nil {
			t.Errorf("expected SQLite read-only execution to reject %q", query)
		}
	}
	config.MultiStatement = true
	if _, err := plugin.RawExecute(config, "SELECT 1; DELETE FROM read_only_guard WHERE id=1"); err == nil {
		t.Error("expected SQLite read-only execution to reject a multi-statement write")
	}
	if _, err := plugin.RawExecute(config, "PRAGMA query_only=OFF; DELETE FROM read_only_guard WHERE id=1; COMMIT"); err == nil {
		t.Error("expected SQLite read-only execution to reject a script that disables query_only")
	}
	if _, err := plugin.RawExecute(config, "SELECT 1; SELECT 2"); err != nil {
		t.Fatalf("expected SQLite read-only execution to allow a read-only script: %v", err)
	}
	config.MultiStatement = false
	rows, err := plugin.RawExecute(config, "SELECT COUNT(*) FROM read_only_guard")
	if err != nil {
		t.Fatalf("expected SQLite read-only execution to allow SELECT: %v", err)
	}
	if len(rows.Rows) != 1 || rows.Rows[0][0] != "1" {
		t.Fatalf("SQLite guard table was modified: %#v", rows.Rows)
	}

	config.ReadOnly = false
	if _, err := plugin.RawExecute(config, "INSERT INTO read_only_guard VALUES (2)"); err != nil {
		t.Fatalf("expected SQLite connection to return to read-write mode: %v", err)
	}
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

func TestClearTableDataTreatsCraftedExistingNameAsOneIdentifier(t *testing.T) {
	craftedTable := `products"; DELETE FROM secrets;--`
	plugin, config, db := newSQLiteRuntimeTestFixture(t,
		`CREATE TABLE "products""; DELETE FROM secrets;--" (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE secrets (id INTEGER PRIMARY KEY, token TEXT)`,
		`INSERT INTO "products""; DELETE FROM secrets;--" (id, name) VALUES (1, 'widget')`,
		`INSERT INTO secrets (id, token) VALUES (1, 'super-secret')`,
	)

	ok, err := plugin.ClearTableData(config, "", craftedTable)
	if err != nil || !ok {
		t.Fatalf("expected crafted table to be cleared safely, got ok=%v err=%v", ok, err)
	}

	var targetRows int64
	if err := db.Raw(`SELECT COUNT(*) FROM "products""; DELETE FROM secrets;--"`).Scan(&targetRows).Error; err != nil {
		t.Fatal(err)
	}
	if targetRows != 0 {
		t.Fatalf("expected crafted table to be empty, got %d rows", targetRows)
	}

	var secretRows int64
	if err := db.Raw(`SELECT COUNT(*) FROM secrets`).Scan(&secretRows).Error; err != nil {
		t.Fatal(err)
	}
	if secretRows != 1 {
		t.Fatalf("expected secrets table to remain untouched, got %d rows", secretRows)
	}
}

func TestOverwriteImportTreatsCraftedExistingNameAsOneIdentifier(t *testing.T) {
	craftedTable := `products"; DELETE FROM secrets;--`
	plugin, config, db := newSQLiteRuntimeTestFixture(t,
		`CREATE TABLE "products""; DELETE FROM secrets;--" (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE secrets (id INTEGER PRIMARY KEY, token TEXT)`,
		`INSERT INTO "products""; DELETE FROM secrets;--" (id, name) VALUES (1, 'old')`,
		`INSERT INTO secrets (id, token) VALUES (1, 'super-secret')`,
	)

	result, err := importer.Execute(plugin, config, &importer.ExecuteRequest{
		StorageUnit: craftedTable,
		Mode:        importer.ModeOverwrite,
		Parsed: &importer.ParsedFile{
			Columns: []string{"id", "name"},
			Rows:    [][]string{{"2", "new"}},
		},
		Mapping: []importer.ColumnMapping{
			{SourceColumn: "id", TargetColumn: new("id")},
			{SourceColumn: "name", TargetColumn: new("name")},
		},
		TargetColumns: []engine.Column{
			{Name: "id", Type: "INTEGER", IsPrimary: true},
			{Name: "name", Type: "TEXT"},
		},
	})
	if err != nil {
		t.Fatalf("overwrite import failed: %v", err)
	}
	if result.RowsImported != 1 {
		t.Fatalf("expected one imported row, got %d", result.RowsImported)
	}

	var names []string
	if err := db.Raw(`SELECT name FROM "products""; DELETE FROM secrets;--" ORDER BY id`).Scan(&names).Error; err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "new" {
		t.Fatalf("expected only the imported row, got %#v", names)
	}

	var secretRows int64
	if err := db.Raw(`SELECT COUNT(*) FROM secrets`).Scan(&secretRows).Error; err != nil {
		t.Fatal(err)
	}
	if secretRows != 1 {
		t.Fatalf("expected secrets table to remain untouched, got %d rows", secretRows)
	}
}
