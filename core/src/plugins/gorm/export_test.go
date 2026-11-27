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

package gorm_plugin

import (
	"path/filepath"
	"reflect"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/query"
)

type exportTestPlugin struct {
	GormPlugin
	db *gorm.DB
}

func newExportTestPlugin(t *testing.T) *exportTestPlugin {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "export.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite test database: %v", err)
	}
	for _, statement := range []string{
		`CREATE TABLE orders (id INTEGER PRIMARY KEY, amount NUMERIC, customer_name TEXT, receipt BLOB)`,
		`INSERT INTO orders VALUES (1, -1200.5, '=1+1', x'deadbeef')`,
		`INSERT INTO orders VALUES (2, 3, NULL, NULL)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("failed to seed orders table: %v", err)
		}
	}

	plugin := &exportTestPlugin{db: db}
	plugin.Type = engine.DatabaseType_Postgres
	plugin.PluginFunctions = plugin
	plugin.GormPluginFunctions = plugin
	return plugin
}

func (p *exportTestPlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
	return p.db, nil
}

type exportedRow struct {
	values []string
	text   []bool
}

func collectExport(rows *[]exportedRow) engine.ExportWriter {
	return func(row []string, text []bool) error {
		*rows = append(*rows, exportedRow{values: append([]string(nil), row...), text: text})
		return nil
	}
}

func TestExportDataMatchesGridRows(t *testing.T) {
	plugin := newExportTestPlugin(t)
	config := engine.NewPluginConfig(&engine.Credentials{Type: string(engine.DatabaseType_Postgres)})

	grid, err := plugin.GetRows(config, &engine.GetRowsRequest{StorageUnit: "orders", Sort: []*query.SortCondition{}, PageSize: 10})
	if err != nil {
		t.Fatalf("GetRows returned error: %v", err)
	}

	var exported []exportedRow
	if err := plugin.ExportData(config, "", "orders", collectExport(&exported), nil); err != nil {
		t.Fatalf("ExportData returned error: %v", err)
	}

	if len(exported) != len(grid.Rows)+1 {
		t.Fatalf("expected a header and %d rows, got %#v", len(grid.Rows), exported)
	}
	for i, column := range grid.Columns {
		if exported[0].values[i] != column.Name {
			t.Fatalf("expected header %q at %d, got %#v", column.Name, i, exported[0].values)
		}
	}
	for i, row := range grid.Rows {
		if !reflect.DeepEqual(exported[i+1].values, row) {
			t.Fatalf("row %d: export %#v differs from grid %#v", i+1, exported[i+1].values, row)
		}
	}
	if exported[1].values[1] != "-1200.5" || exported[1].values[3] != "0xdeadbeef" {
		t.Fatalf("unexpected exported values: %#v", exported[1].values)
	}
	// Numbers are typed values; text and undeclared types are free text.
	if want := []bool{false, false, true, true}; !reflect.DeepEqual(exported[1].text, want) {
		t.Fatalf("expected free-text flags %v, got %v", want, exported[1].text)
	}
}

func TestExportDataWritesSelectionInGridOrder(t *testing.T) {
	plugin := newExportTestPlugin(t)
	selection := &engine.ExportSelection{
		Columns: []engine.Column{{Name: "total", Type: "NUMERIC(12,2)"}, {Name: "customer", Type: "TEXT"}, {Name: "age", Type: "INT4"}},
		Rows: []map[string]any{
			{"customer": "=cmd", "age": "-3", "total": "-9.50"},
			{"customer": "bob", "total": "1.00"},
		},
	}

	var exported []exportedRow
	if err := plugin.ExportData(nil, "", "", collectExport(&exported), selection); err != nil {
		t.Fatalf("ExportData returned error: %v", err)
	}

	want := [][]string{{"total", "customer", "age"}, {"-9.50", "=cmd", "-3"}, {"1.00", "bob", ""}}
	for i, row := range want {
		if !reflect.DeepEqual(exported[i].values, row) {
			t.Fatalf("row %d: expected %#v, got %#v", i, row, exported[i].values)
		}
	}
	if wantText := []bool{false, true, false}; !reflect.DeepEqual(exported[1].text, wantText) {
		t.Fatalf("expected free-text flags %v, got %v", wantText, exported[1].text)
	}
}
