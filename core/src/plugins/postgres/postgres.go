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
	"database/sql"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	sourcecatalogspecs "github.com/clidey/whodb/core/src/sourcecatalog/specs"
	"gorm.io/gorm"
)

var (
	supportedOperators = sourcecatalogspecs.PostgreSQLSupportedOperators
)

type PostgresPlugin struct {
	gorm_plugin.GormPlugin
}

func (p *PostgresPlugin) GetSupportedOperators() map[string]string {
	return supportedOperators
}

// GetLastInsertID returns the most recently auto-generated ID using PostgreSQL's lastval().
func (p *PostgresPlugin) GetLastInsertID(db *gorm.DB) (int64, error) {
	var id int64
	if err := db.Raw("SELECT lastval()").Scan(&id).Error; err != nil {
		if strings.Contains(err.Error(), "lastval is not yet defined") {
			return 0, nil
		}
		return 0, err
	}
	return id, nil
}

func (p *PostgresPlugin) GetAllSchemasQuery() string {
	return "SELECT schema_name AS schemaname FROM information_schema.schemata WHERE has_schema_privilege(schema_name, 'USAGE')"
}

func (p *PostgresPlugin) GetTableInfoQuery() string {
	// Guard pg_*_relation_size with has_table_privilege so a single restricted
	// relation does not error the whole listing. NULL sizes propagate up and
	// are emitted as absent attributes rather than "0".
	return `
		SELECT
			t.table_name,
			t.table_type,
			CASE WHEN c.oid IS NOT NULL AND has_table_privilege(c.oid, 'SELECT')
				THEN pg_total_relation_size(c.oid)::bigint
				ELSE NULL END AS total_size,
			CASE WHEN c.oid IS NOT NULL AND has_table_privilege(c.oid, 'SELECT')
				THEN pg_relation_size(c.oid)::bigint
				ELSE NULL END AS data_size
		FROM
			information_schema.tables t
		LEFT JOIN pg_namespace n ON n.nspname = t.table_schema
		LEFT JOIN pg_class c ON c.relname = t.table_name AND c.relnamespace = n.oid
		WHERE
			t.table_schema = ?;
	`
}

func (p *PostgresPlugin) GetStorageUnitExistsQuery() string {
	return `SELECT to_regclass($1 || '.' || $2) IS NOT NULL`
}

func (p *PostgresPlugin) GetPlaceholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

func (p *PostgresPlugin) GetTableNameAndAttributes(rows *sql.Rows) (string, []engine.Record) {
	var tableName, tableType string
	var totalSize, dataSize sql.NullInt64
	if err := rows.Scan(&tableName, &tableType, &totalSize, &dataSize); err != nil {
		log.WithError(err).Error("Failed to scan table info row data")
		return "", nil
	}

	attributes := []engine.Record{
		{Key: "Type", Value: tableType},
	}
	if totalSize.Valid {
		attributes = append(attributes, engine.Record{Key: "Total Size", Value: fmt.Sprintf("%d", totalSize.Int64)})
	}
	if dataSize.Valid {
		attributes = append(attributes, engine.Record{Key: "Data Size", Value: fmt.Sprintf("%d", dataSize.Int64)})
	}

	return tableName, attributes
}

func (p *PostgresPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]string, error) {
		var databases []struct {
			Datname string `gorm:"column:datname"`
		}
		if err := db.Raw("SELECT datname FROM pg_database WHERE datistemplate = false AND datallowconn AND has_database_privilege(datname, 'CONNECT')").
			Scan(&databases).Error; err != nil {
			return nil, err
		}
		databaseNames := []string{}
		for _, database := range databases {
			databaseNames = append(databaseNames, database.Datname)
		}
		return databaseNames, nil
	})
}

func (p *PostgresPlugin) RawExecute(config *engine.PluginConfig, query string, params ...any) (*engine.GetRowsResult, error) {
	return p.ExecuteRawSQL(config, func(cfg *engine.PluginConfig) (*gorm.DB, error) {
		return p.openDB(cfg, true)
	}, query, params...)
}

func (p *PostgresPlugin) GetForeignKeyRelationships(config *engine.PluginConfig, schema string, storageUnit string) (map[string]*engine.ForeignKeyRelationship, error) {
	query := `
		SELECT
			kcu.column_name,
			ccu.table_name AS referenced_table,
			ccu.column_name AS referenced_column
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = ?
			AND tc.table_name = ?
	`
	return p.QueryForeignKeyRelationships(config, query, schema, storageUnit)
}

// NormalizeType converts PostgreSQL type aliases to their canonical form.
func (p *PostgresPlugin) NormalizeType(typeName string) string {
	return common.NormalizeTypeWithMap(typeName, sourcecatalogspecs.PostgresAliasMap)
}

// MarkGeneratedColumns detects PostgreSQL generated columns (GENERATED ALWAYS AS)
// and marks them as IsComputed.
func (p *PostgresPlugin) MarkGeneratedColumns(config *engine.PluginConfig, schema string, storageUnit string, columns []engine.Column) error {
	computed, err := p.QueryComputedColumns(config, `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ? AND is_generated = 'ALWAYS'
	`, schema, storageUnit)
	if err != nil {
		return err
	}

	for i := range columns {
		if computed[columns[i].Name] {
			columns[i].IsComputed = true
		}
	}
	return nil
}

// IsArrayType returns true for PostgreSQL array types which use an underscore prefix
// (e.g., _int4 for int[], _text for text[]).
func (p *PostgresPlugin) IsArrayType(columnType string) bool {
	return strings.HasPrefix(columnType, "_")
}

func init() {
	engine.RegisterPlugin(NewPostgresPlugin())
}

func NewPostgresPlugin() *engine.Plugin {
	plugin := &PostgresPlugin{}
	plugin.Type = engine.DatabaseType_Postgres
	plugin.PluginFunctions = plugin
	plugin.GormPluginFunctions = plugin
	return &plugin.Plugin
}
