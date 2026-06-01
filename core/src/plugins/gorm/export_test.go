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
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/engine"
)

type exportTestPlugin struct {
	GormPlugin
	db                     *gorm.DB
	columnsRead            atomic.Int32
	columnsUsedTransaction atomic.Bool
	formatCalls            atomic.Int32
}

func newExportTestPlugin(t *testing.T) *exportTestPlugin {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "export.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite test database: %v", err)
	}
	if err := db.Exec(`CREATE TABLE orders (id INTEGER PRIMARY KEY, customer_name TEXT NOT NULL)`).Error; err != nil {
		t.Fatalf("failed to create orders table: %v", err)
	}
	if err := db.Exec(`INSERT INTO orders (customer_name) VALUES ('alice')`).Error; err != nil {
		t.Fatalf("failed to seed orders table: %v", err)
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

func (p *exportTestPlugin) GetColumnsForTable(config *engine.PluginConfig, schema string, storageUnit string) ([]engine.Column, error) {
	p.columnsRead.Add(1)
	if config != nil && config.Transaction != nil {
		p.columnsUsedTransaction.Store(true)
	}
	return []engine.Column{
		{Name: "id", Type: "INTEGER", IsPrimary: true},
		{Name: "customer_name", Type: "TEXT", IsNullable: false},
	}, nil
}

func (p *exportTestPlugin) FormatValue(val any) string {
	p.formatCalls.Add(1)
	if ts, ok := val.(time.Time); ok {
		return "OVERRIDE:" + ts.Format(time.RFC3339Nano)
	}
	return p.GormPlugin.FormatValue(val)
}

func TestExportDataUsesPluginColumnLookup(t *testing.T) {
	plugin := newExportTestPlugin(t)
	config := engine.NewPluginConfig(&engine.Credentials{Type: string(engine.DatabaseType_Postgres)})

	var written [][]string
	err := plugin.ExportData(config, "", "orders", func(row []string) error {
		written = append(written, append([]string(nil), row...))
		return nil
	}, nil)
	if err != nil {
		t.Fatalf("ExportData returned error: %v", err)
	}

	if plugin.columnsRead.Load() != 1 {
		t.Fatalf("expected export to use plugin GetColumnsForTable exactly once, got %d", plugin.columnsRead.Load())
	}
	if !plugin.columnsUsedTransaction.Load() {
		t.Fatal("expected export column lookup to reuse the active connection")
	}
	if len(written) < 2 {
		t.Fatalf("expected headers and at least one row, got %#v", written)
	}
	if written[0][0] != "id" || written[0][1] != "customer_name" {
		t.Fatalf("unexpected export headers: %#v", written[0])
	}
	if written[1][1] != "alice" {
		t.Fatalf("unexpected exported row: %#v", written[1])
	}
}

func TestExportDataUsesPluginFormatValueOverride(t *testing.T) {
	plugin := newExportTestPlugin(t)
	createdAt := time.Date(2026, 4, 20, 10, 11, 12, 123456789, time.UTC)

	var written [][]string
	err := plugin.ExportData(nil, "", "", func(row []string) error {
		written = append(written, append([]string(nil), row...))
		return nil
	}, []map[string]any{{"created_at": createdAt}})
	if err != nil {
		t.Fatalf("ExportData returned error: %v", err)
	}

	if plugin.formatCalls.Load() == 0 {
		t.Fatal("expected export to call the plugin FormatValue override")
	}
	if len(written) != 2 || written[1][0] != "OVERRIDE:"+createdAt.Format(time.RFC3339Nano) {
		t.Fatalf("unexpected formatted export rows: %#v", written)
	}
}
