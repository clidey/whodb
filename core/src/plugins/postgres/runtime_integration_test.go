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

package postgres

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/importer"
	"github.com/clidey/whodb/core/src/query"
)

func postgresIntegrationPlugin(t *testing.T) *PostgresPlugin {
	t.Helper()

	plugin, ok := NewPostgresPlugin().PluginFunctions.(*PostgresPlugin)
	if !ok {
		t.Fatalf("unexpected postgres plugin type %T", NewPostgresPlugin().PluginFunctions)
	}
	return plugin
}

func postgresIntegrationConfig() *engine.PluginConfig {
	return engine.NewPluginConfig(&engine.Credentials{
		Type:     string(engine.DatabaseType_Postgres),
		Hostname: "localhost",
		Username: "user",
		Password: "jio53$*(@nfe)",
		Database: "test_db",
		Advanced: []engine.Record{{Key: "Port", Value: "5432"}},
	})
}

func waitForPostgresOrders(t *testing.T, plugin engine.PluginFunctions, config *engine.PluginConfig) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		if !plugin.IsAvailable(context.Background(), config) {
			time.Sleep(2 * time.Second)
			continue
		}

		exists, err := plugin.StorageUnitExists(config, "test_schema", "orders")
		if err == nil && exists {
			rows, rowsErr := plugin.GetRows(config, &engine.GetRowsRequest{
				Schema:      "test_schema",
				StorageUnit: "orders",
				Sort:        []*query.SortCondition{{Column: "id", Direction: query.SortDirectionAsc}},
				PageSize:    1,
			})
			if rowsErr == nil && len(rows.Rows) > 0 {
				return
			}
		}

		time.Sleep(2 * time.Second)
	}

	t.Fatal("timed out waiting for seeded postgres data")
}

func waitForPostgresFamilyConnection(t *testing.T, plugin engine.PluginFunctions, config *engine.PluginConfig) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		if plugin.IsAvailable(context.Background(), config) {
			if _, err := plugin.RawExecute(config, "SELECT 1"); err == nil {
				return
			}
		}
		time.Sleep(2 * time.Second)
	}

	t.Fatalf("timed out waiting for %s connection", config.Credentials.Type)
}

func findPostgresColumn(t *testing.T, columns []engine.Column, name string) engine.Column {
	t.Helper()

	for _, column := range columns {
		if column.Name == name {
			return column
		}
	}

	t.Fatalf("column %q not found in %#v", name, columns)
	return engine.Column{}
}

func TestPostgresSeededRuntimePaths(t *testing.T) {
	plugin := postgresIntegrationPlugin(t)
	config := postgresIntegrationConfig()
	waitForPostgresOrders(t, plugin, config)

	databases, err := plugin.GetDatabases(config)
	if err != nil {
		t.Fatalf("GetDatabases failed: %v", err)
	}
	if !slices.Contains(databases, "test_db") {
		t.Fatalf("expected databases %#v to contain test_db", databases)
	}

	rawRows, err := plugin.RawExecute(config, "SELECT status FROM test_schema.orders ORDER BY id LIMIT 1")
	if err != nil {
		t.Fatalf("RawExecute failed: %v", err)
	}
	if len(rawRows.Rows) != 1 {
		t.Fatalf("expected one postgres row, got %#v", rawRows.Rows)
	}

	relationships, err := plugin.GetForeignKeyRelationships(config, "test_schema", "orders")
	if err != nil {
		t.Fatalf("GetForeignKeyRelationships failed: %v", err)
	}
	relationship, ok := relationships["user_id"]
	if !ok {
		t.Fatalf("expected user_id foreign key in %#v", relationships)
	}
	if relationship.ReferencedTable != "users" || relationship.ReferencedColumn != "id" {
		t.Fatalf("unexpected postgres foreign key relationship %#v", relationship)
	}

	sslStatus, err := plugin.GetSSLStatus(config)
	if err != nil {
		t.Fatalf("GetSSLStatus failed: %v", err)
	}
	if sslStatus.IsEnabled || sslStatus.Mode != "disabled" {
		t.Fatalf("expected postgres SSL to be disabled, got %#v", sslStatus)
	}

	table := fmt.Sprintf("intg_pg_ms_%d", time.Now().UnixNano())
	_, _ = plugin.RawExecute(config, "DROP TABLE IF EXISTS test_schema."+table)
	defer plugin.RawExecute(config, "DROP TABLE IF EXISTS test_schema."+table)

	multiStatementConfig := *config
	multiStatementConfig.MultiStatement = true

	_, err = plugin.RawExecute(&multiStatementConfig, fmt.Sprintf(`
DROP TABLE IF EXISTS test_schema.%[1]s;
CREATE TABLE test_schema.%[1]s (
	id SERIAL PRIMARY KEY,
	name TEXT NOT NULL
);
INSERT INTO test_schema.%[1]s (name) VALUES ('alpha'), ('beta');
`, table))
	if err != nil {
		t.Fatalf("multi-statement RawExecute failed: %v", err)
	}

	exists, err := plugin.StorageUnitExists(config, "test_schema", table)
	if err != nil || !exists {
		t.Fatalf("expected postgres table %q to exist, exists=%t err=%v", table, exists, err)
	}

	insertedRows, err := plugin.RawExecute(config, fmt.Sprintf("SELECT name FROM test_schema.%s ORDER BY id", table))
	if err != nil {
		t.Fatalf("failed to read multi-statement postgres table: %v", err)
	}
	if len(insertedRows.Rows) != 2 {
		t.Fatalf("expected two postgres rows after multi-statement RawExecute, got %#v", insertedRows.Rows)
	}
}

func TestPostgresReadOnlyRawExecuteRejectsAdvisoryBypasses(t *testing.T) {
	plugin := postgresIntegrationPlugin(t)
	config := postgresIntegrationConfig()
	waitForPostgresOrders(t, plugin, config)

	table := fmt.Sprintf("read_only_guard_%d", time.Now().UnixNano())
	if _, err := plugin.RawExecute(config, "CREATE TABLE "+table+" (id integer primary key)"); err != nil {
		t.Fatalf("failed to create PostgreSQL guard table: %v", err)
	}
	t.Cleanup(func() { _, _ = plugin.RawExecute(config, "DROP TABLE IF EXISTS "+table) })
	if _, err := plugin.RawExecute(config, "INSERT INTO "+table+" VALUES (1), (2), (3)"); err != nil {
		t.Fatalf("failed to seed PostgreSQL guard table: %v", err)
	}

	config.ReadOnly = true
	queries := []string{
		"WITH x AS(DELETE FROM " + table + " WHERE id=2 RETURNING *) SELECT * FROM x",
		"EXPLAIN (ANALYZE,BUFFERS) DELETE FROM " + table + " WHERE id=3",
		`EXPLAIN ("analyze" true) DELETE FROM ` + table + " WHERE id=1",
	}
	for _, query := range queries {
		if _, err := plugin.RawExecute(config, query); err == nil {
			t.Errorf("expected PostgreSQL read-only execution to reject %q", query)
		}
	}
	config.MultiStatement = true
	if _, err := plugin.RawExecute(config, "SELECT 1; DELETE FROM "+table+" WHERE id=1"); err == nil {
		t.Error("expected PostgreSQL read-only execution to reject a multi-statement write")
	}
	config.MultiStatement = false

	config.ReadOnly = false
	rows, err := plugin.RawExecute(config, "SELECT COUNT(*) FROM "+table)
	if err != nil {
		t.Fatalf("failed to verify PostgreSQL guard table: %v", err)
	}
	if len(rows.Rows) != 1 || rows.Rows[0][0] != "3" {
		t.Fatalf("PostgreSQL guard table was modified: %#v", rows.Rows)
	}
}

func TestPostgresFamilyReadOnlyRawExecuteRejectsWrites(t *testing.T) {
	tests := []struct {
		name   string
		plugin engine.PluginFunctions
		config *engine.PluginConfig
	}{
		{name: "Postgres", plugin: NewPostgresPlugin().PluginFunctions, config: postgresIntegrationConfig()},
		{name: "CockroachDB", plugin: NewCockroachDBPlugin().PluginFunctions, config: postgresFamilyIntegrationConfig(engine.DatabaseType_CockroachDB, "26257")},
		{name: "YugabyteDB", plugin: NewYugabyteDBPlugin().PluginFunctions, config: postgresFamilyIntegrationConfig(engine.DatabaseType_YugabyteDB, "5434")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			waitForPostgresFamilyConnection(t, test.plugin, test.config)
			table := fmt.Sprintf("test_schema.read_only_family_guard_%d", time.Now().UnixNano())
			if _, err := test.plugin.RawExecute(test.config, "CREATE TABLE "+table+" (id integer primary key)"); err != nil {
				t.Fatalf("failed to create %s guard table: %v", test.name, err)
			}
			t.Cleanup(func() { _, _ = test.plugin.RawExecute(test.config, "DROP TABLE IF EXISTS "+table) })
			if _, err := test.plugin.RawExecute(test.config, "INSERT INTO "+table+" VALUES (1)"); err != nil {
				t.Fatalf("failed to seed %s guard table: %v", test.name, err)
			}

			test.config.ReadOnly = true
			if _, err := test.plugin.RawExecute(test.config, "INSERT INTO "+table+" VALUES (2)"); err == nil {
				t.Errorf("expected %s read-only execution to reject INSERT", test.name)
			}
			test.config.ReadOnly = false
			rows, err := test.plugin.RawExecute(test.config, "SELECT COUNT(*) FROM "+table)
			if err != nil {
				t.Fatalf("failed to verify %s guard table: %v", test.name, err)
			}
			if len(rows.Rows) != 1 || rows.Rows[0][0] != "1" {
				t.Fatalf("%s guard table was modified: %#v", test.name, rows.Rows)
			}
		})
	}
}

func TestPostgresGeneratedColumnsAndLastInsertID(t *testing.T) {
	plugin := postgresIntegrationPlugin(t)
	config := postgresIntegrationConfig()
	waitForPostgresOrders(t, plugin, config)

	table := fmt.Sprintf("intg_pg_gen_%d", time.Now().UnixNano())
	_, _ = plugin.RawExecute(config, "DROP TABLE IF EXISTS test_schema."+table)
	defer plugin.RawExecute(config, "DROP TABLE IF EXISTS test_schema."+table)

	_, err := plugin.RawExecute(config, fmt.Sprintf(`
CREATE TABLE test_schema.%[1]s (
	id SERIAL PRIMARY KEY,
	subtotal INT NOT NULL,
	tax INT NOT NULL,
	total INT GENERATED ALWAYS AS (subtotal + tax) STORED
)
`, table))
	if err != nil {
		t.Fatalf("failed to create postgres generated-column table: %v", err)
	}

	columns, err := plugin.GetColumnsForTable(config, "test_schema", table)
	if err != nil {
		t.Fatalf("GetColumnsForTable failed: %v", err)
	}
	if err := plugin.MarkGeneratedColumns(config, "test_schema", table, columns); err != nil {
		t.Fatalf("MarkGeneratedColumns failed: %v", err)
	}
	if !findPostgresColumn(t, columns, "total").IsComputed {
		t.Fatalf("expected total column to be marked as computed, got %#v", columns)
	}

	db, err := plugin.openDB(config, false)
	if err != nil {
		t.Fatalf("openDB failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB failed: %v", err)
	}
	defer sqlDB.Close()

	var insertedID int64
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(fmt.Sprintf("INSERT INTO test_schema.%s (subtotal, tax) VALUES (?, ?)", table), 10, 3).Error; err != nil {
			return err
		}

		var lastInsertErr error
		insertedID, lastInsertErr = plugin.GetLastInsertID(tx)
		return lastInsertErr
	})
	if err != nil {
		t.Fatalf("failed to insert postgres row and read lastval(): %v", err)
	}
	if insertedID <= 0 {
		t.Fatalf("expected postgres last insert id > 0, got %d", insertedID)
	}

	totals, err := plugin.RawExecute(config, fmt.Sprintf("SELECT total FROM test_schema.%s WHERE id = %d", table, insertedID))
	if err != nil {
		t.Fatalf("failed to query postgres generated total: %v", err)
	}
	if len(totals.Rows) != 1 || totals.Rows[0][0] != "13" {
		t.Fatalf("expected generated postgres total 13, got %#v", totals.Rows)
	}
}

func TestPostgresOverwriteImportAcceptsQuotedTableName(t *testing.T) {
	tests := []struct {
		name   string
		plugin engine.PluginFunctions
		config *engine.PluginConfig
	}{
		{name: "Postgres", plugin: NewPostgresPlugin().PluginFunctions, config: postgresIntegrationConfig()},
		{name: "CockroachDB", plugin: NewCockroachDBPlugin().PluginFunctions, config: postgresFamilyIntegrationConfig(engine.DatabaseType_CockroachDB, "26257")},
		{name: "YugabyteDB", plugin: NewYugabyteDBPlugin().PluginFunctions, config: postgresFamilyIntegrationConfig(engine.DatabaseType_YugabyteDB, "5434")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testPostgresOverwriteImportAcceptsQuotedTableName(t, test.plugin, test.config)
		})
	}
}

func postgresFamilyIntegrationConfig(databaseType engine.DatabaseType, port string) *engine.PluginConfig {
	return engine.NewPluginConfig(&engine.Credentials{
		Type:     string(databaseType),
		Hostname: "localhost",
		Username: "user",
		Password: "password",
		Database: "test_db",
		Advanced: []engine.Record{{Key: "Port", Value: port}},
	})
}

func testPostgresOverwriteImportAcceptsQuotedTableName(t *testing.T, plugin engine.PluginFunctions, config *engine.PluginConfig) {
	t.Helper()
	waitForPostgresFamilyConnection(t, plugin, config)

	guardTable := fmt.Sprintf("guard_%d", time.Now().UnixNano()%1_000_000)
	table := `identifier"; DELETE FROM test_schema.` + guardTable + `;--`
	quotedTable := `"` + strings.ReplaceAll(table, `"`, `""`) + `"`
	_, _ = plugin.RawExecute(config, `DROP TABLE IF EXISTS test_schema.`+quotedTable)
	_, _ = plugin.RawExecute(config, `DROP TABLE IF EXISTS test_schema.`+guardTable)
	t.Cleanup(func() {
		_, _ = plugin.RawExecute(config, `DROP TABLE IF EXISTS test_schema.`+quotedTable)
		_, _ = plugin.RawExecute(config, `DROP TABLE IF EXISTS test_schema.`+guardTable)
	})

	setup := []string{
		`CREATE TABLE test_schema.` + guardTable + ` (id INTEGER PRIMARY KEY)`,
		`INSERT INTO test_schema.` + guardTable + ` VALUES (1)`,
		`CREATE TABLE test_schema.` + quotedTable + ` (id INTEGER PRIMARY KEY, name VARCHAR(50))`,
		`INSERT INTO test_schema.` + quotedTable + ` VALUES (1, 'old')`,
	}
	for _, statement := range setup {
		if _, err := plugin.RawExecute(config, statement); err != nil {
			t.Fatalf("failed to prepare quoted postgres table with %q: %v", statement, err)
		}
	}

	columns, err := plugin.GetColumnsForTable(config, "test_schema", table)
	if err != nil {
		t.Fatalf("failed to inspect quoted %s table: %v", config.Credentials.Type, err)
	}
	nameColumn := findPostgresColumn(t, columns, "name")
	if nameColumn.Type != "CHARACTER VARYING(50)" || nameColumn.Length == nil || *nameColumn.Length != 50 {
		t.Fatalf("quoted %s metadata lost declared type information: %#v", config.Credentials.Type, columns)
	}

	result, err := importer.Execute(plugin, config, &importer.ExecuteRequest{
		Schema:      "test_schema",
		StorageUnit: table,
		Mode:        importer.ModeOverwrite,
		Parsed:      &importer.ParsedFile{Columns: []string{"id", "name"}, Rows: [][]string{{"2", "new"}}},
		Mapping: []importer.ColumnMapping{
			{SourceColumn: "id", TargetColumn: new("id")},
			{SourceColumn: "name", TargetColumn: new("name")},
		},
		TargetColumns: []engine.Column{{Name: "id", Type: "INTEGER", IsPrimary: true}, {Name: "name", Type: "TEXT"}},
	})
	if err != nil {
		t.Fatalf("overwrite import through quoted %s table failed: %v", config.Credentials.Type, err)
	}
	if result.RowsImported != 1 {
		t.Fatalf("expected one imported postgres row, got %d", result.RowsImported)
	}

	rows, err := plugin.RawExecute(config, `SELECT name FROM test_schema.`+quotedTable)
	if err != nil || len(rows.Rows) != 1 || rows.Rows[0][0] != "new" {
		t.Fatalf("unexpected postgres target rows: rows=%#v err=%v", rows, err)
	}
	guard, err := plugin.RawExecute(config, `SELECT COUNT(*) FROM test_schema.`+guardTable)
	if err != nil || len(guard.Rows) != 1 || guard.Rows[0][0] != "1" {
		t.Fatalf("postgres guard table was modified: rows=%#v err=%v", guard, err)
	}
}

func TestQuestDBReadPathsAcceptQuotedTableName(t *testing.T) {
	plugin := NewQuestDBPlugin().PluginFunctions
	config := engine.NewPluginConfig(&engine.Credentials{
		Type:     string(engine.DatabaseType_QuestDB),
		Hostname: "localhost",
		Username: "user",
		Password: "password",
		Database: "qdb",
		Advanced: []engine.Record{{Key: "Port", Value: "8812"}},
	})

	deadline := time.Now().Add(2 * time.Minute)
	ready := false
	for time.Now().Before(deadline) {
		if plugin.IsAvailable(context.Background(), config) {
			exists, err := plugin.StorageUnitExists(config, "", "users")
			if err == nil && exists {
				ready = true
				break
			}
		}
		time.Sleep(2 * time.Second)
	}
	if !ready {
		t.Fatal("timed out waiting for seeded QuestDB data")
	}

	guardTable := fmt.Sprintf("guard_%d", time.Now().UnixNano()%1_000_000)
	hostileTable := `identifier"; DROP TABLE ` + guardTable + `;--`
	exists, err := plugin.StorageUnitExists(config, "", hostileTable)
	if err != nil {
		t.Fatalf("QuestDB hostile identifier existence check failed: %v", err)
	}
	if exists {
		t.Fatalf("unexpected pre-existing QuestDB table %q", hostileTable)
	}

	table := fmt.Sprintf("identifier safe %d", time.Now().UnixNano()%1_000_000)
	quotedTable := `"` + strings.ReplaceAll(table, `"`, `""`) + `"`
	_, _ = plugin.RawExecute(config, `DROP TABLE IF EXISTS `+guardTable)
	t.Cleanup(func() {
		_, _ = plugin.RawExecute(config, `DROP TABLE IF EXISTS `+quotedTable)
		_, _ = plugin.RawExecute(config, `DROP TABLE IF EXISTS `+guardTable)
	})

	setup := []string{
		`CREATE TABLE ` + guardTable + ` (id INT)`,
		`INSERT INTO ` + guardTable + ` VALUES (1)`,
		`CREATE TABLE ` + quotedTable + ` (id INT, name STRING)`,
		`INSERT INTO ` + quotedTable + ` VALUES (1, 'safe')`,
	}
	for _, statement := range setup {
		if _, err := plugin.RawExecute(config, statement); err != nil {
			t.Fatalf("failed to prepare quoted QuestDB table with %q: %v", statement, err)
		}
	}

	columns, err := plugin.GetColumnsForTable(config, "", table)
	if err != nil {
		t.Fatalf("failed to inspect quoted QuestDB table: %v", err)
	}
	if len(columns) != 2 || columns[0].Name != "id" || columns[1].Name != "name" {
		t.Fatalf("unexpected quoted QuestDB table metadata: %#v", columns)
	}

	rows, err := plugin.GetRows(config, &engine.GetRowsRequest{StorageUnit: table, PageSize: 10})
	if err != nil || len(rows.Rows) != 1 {
		t.Fatalf("failed to read quoted QuestDB table: rows=%#v err=%v", rows, err)
	}
	guard, err := plugin.RawExecute(config, `SELECT COUNT(*) FROM `+guardTable)
	if err != nil || len(guard.Rows) != 1 || guard.Rows[0][0] != "1" {
		t.Fatalf("QuestDB guard table was modified: rows=%#v err=%v", guard, err)
	}
}
