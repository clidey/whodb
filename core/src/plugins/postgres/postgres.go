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

package postgres

import (
	"database/sql"
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	"gorm.io/gorm"
)

var (
	supportedOperators = map[string]string{
		"=": "=", ">=": ">=", ">": ">", "<=": "<=", "<": "<", "<>": "<>",
		"!=": "!=", "BETWEEN": "BETWEEN", "NOT BETWEEN": "NOT BETWEEN",
		"LIKE": "LIKE", "NOT LIKE": "NOT LIKE", "ILIKE": "ILIKE", "NOT ILIKE": "NOT ILIKE",
		"IN": "IN", "NOT IN": "NOT IN",
		"IS NULL": "IS NULL", "IS NOT NULL": "IS NOT NULL", "AND": "AND", "OR": "OR", "NOT": "NOT",
	}
)

type PostgresPlugin struct {
	gorm_plugin.GormPlugin
}

func (p *PostgresPlugin) GetSupportedOperators() map[string]string {
	return supportedOperators
}

func (p *PostgresPlugin) FormTableName(schema string, storageUnit string) string {
	// Keep raw concatenation; actual SQL builders will quote via GORM Dialector
	if schema == "" {
		return storageUnit
	}
	return schema + "." + storageUnit
}

func (p *PostgresPlugin) GetAllSchemasQuery() string {
	return "SELECT schema_name AS schemaname FROM information_schema.schemata"
}

func (p *PostgresPlugin) GetTableInfoQuery() string {
	return `
		SELECT
			t.table_name,
			t.table_type,
			pg_size_pretty(pg_total_relation_size(quote_ident(t.table_schema) || '.' || quote_ident(t.table_name))) AS total_size,
			pg_size_pretty(pg_relation_size(quote_ident(t.table_schema) || '.' || quote_ident(t.table_name))) AS data_size
		FROM
			information_schema.tables t
		WHERE
			t.table_schema = ?;
	`

	// AND t.table_type = 'BASE TABLE' this removes the view tables
}

func (p *PostgresPlugin) GetStorageUnitExistsQuery() string {
	return `SELECT to_regclass($1 || '.' || $2) IS NOT NULL`
}

func (p *PostgresPlugin) GetPlaceholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

func (p *PostgresPlugin) GetTableNameAndAttributes(rows *sql.Rows) (string, []engine.Record) {
	var tableName, tableType, totalSize, dataSize string
	if err := rows.Scan(&tableName, &tableType, &totalSize, &dataSize); err != nil {
		log.Logger.WithError(err).Error("Failed to scan table info row data")
		return "", nil
	}

	attributes := []engine.Record{
		{Key: "Type", Value: tableType},
		{Key: "Total Size", Value: totalSize},
		{Key: "Data Size", Value: dataSize},
	}

	return tableName, attributes
}

func (p *PostgresPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]string, error) {
		var databases []struct {
			Datname string `gorm:"column:datname"`
		}
		if err := db.Table("pg_database").
			Select("datname").
			Where("datistemplate = ?", false).
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

func (p *PostgresPlugin) executeRawSQL(config *engine.PluginConfig, query string, params ...interface{}) (*engine.GetRowsResult, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.GetRowsResult, error) {
		rows, err := db.Raw(query, params...).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return p.ConvertRawToRows(rows)
	})
}

func (p *PostgresPlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return p.executeRawSQL(config, query)
}

func (p *PostgresPlugin) GetForeignKeyRelationships(config *engine.PluginConfig, schema string, storageUnit string) (map[string]*engine.ForeignKeyRelationship, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (map[string]*engine.ForeignKeyRelationship, error) {
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

		rows, err := db.Raw(query, schema, storageUnit).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		relationships := make(map[string]*engine.ForeignKeyRelationship)
		for rows.Next() {
			var columnName, referencedTable, referencedColumn string
			if err := rows.Scan(&columnName, &referencedTable, &referencedColumn); err != nil {
				log.Logger.WithError(err).Error("Failed to scan foreign key relationship")
				continue
			}
			relationships[columnName] = &engine.ForeignKeyRelationship{
				ColumnName:       columnName,
				ReferencedTable:  referencedTable,
				ReferencedColumn: referencedColumn,
			}
		}

		return relationships, nil
	})
}

// NormalizeType converts PostgreSQL type aliases to their canonical form.
func (p *PostgresPlugin) NormalizeType(typeName string) string {
	return NormalizeType(typeName)
}

func NewPostgresPlugin() *engine.Plugin {
	plugin := &PostgresPlugin{}
	plugin.Type = engine.DatabaseType_Postgres
	plugin.PluginFunctions = plugin
	plugin.GormPluginFunctions = plugin
	return &plugin.Plugin
}
