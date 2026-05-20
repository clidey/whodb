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
	"slices"
	"strings"

	"github.com/clidey/whodb/core/src/common/ssl"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	"gorm.io/gorm"
)

// QuestDBPlugin extends PostgresPlugin with QuestDB-specific catalog behavior.
// QuestDB uses the PostgreSQL wire protocol in our product, but its table
// metadata path is schema-less in practice and does not support the PostgreSQL
// relation size functions used by the base plugin.
type QuestDBPlugin struct {
	PostgresPlugin
}

type questDBColumnMetadata struct {
	name       string
	dataType   string
	isNullable bool
}

// GetTableInfoQuery returns a QuestDB-compatible table info query.
func (p *QuestDBPlugin) GetTableInfoQuery() string {
	return `
		SELECT
			t.table_name,
			t.table_type
		FROM
			information_schema.tables t
		WHERE
			($1 = '' OR t.table_schema = $1)
			AND t.table_schema NOT IN ('information_schema', 'pg_catalog');
	`
}

// GetTableNameAndAttributes parses QuestDB table info rows.
func (p *QuestDBPlugin) GetTableNameAndAttributes(rows *sql.Rows) (string, []engine.Record) {
	var tableName, tableType string
	if err := rows.Scan(&tableName, &tableType); err != nil {
		log.WithError(err).Error("Failed to scan QuestDB table info row data")
		return "", nil
	}

	return tableName, []engine.Record{
		{Key: "Type", Value: tableType},
	}
}

// GetStorageUnitExistsQuery returns a QuestDB-compatible table existence check.
func (p *QuestDBPlugin) GetStorageUnitExistsQuery() string {
	return `
		SELECT CASE
			WHEN COUNT(1) > 0 THEN TRUE
			ELSE FALSE
		END
		FROM information_schema.tables
		WHERE ($1 = '' OR table_schema = $1)
			AND table_name = $2
			AND table_schema NOT IN ('information_schema', 'pg_catalog')
	`
}

func (p *QuestDBPlugin) getColumnsQuery() string {
	return `
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE ($1 = '' OR table_schema = $1)
			AND table_name = $2
		ORDER BY ordinal_position
	`
}

func (p *QuestDBPlugin) normalizeQuestDBColumnMetadata(columnName string, dataType string, isNullable string) questDBColumnMetadata {
	return questDBColumnMetadata{
		name:       columnName,
		dataType:   p.NormalizeType(dataType),
		isNullable: strings.EqualFold(isNullable, "yes"),
	}
}

func (p *QuestDBPlugin) readQuestDBColumns(db *gorm.DB, schema string, tableName string) ([]questDBColumnMetadata, error) {
	rows, err := db.Raw(p.getColumnsQuery(), schema, tableName).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make([]questDBColumnMetadata, 0)
	for rows.Next() {
		var columnName string
		var dataType string
		var isNullable string
		if err := rows.Scan(&columnName, &dataType, &isNullable); err != nil {
			return nil, err
		}
		columns = append(columns, p.normalizeQuestDBColumnMetadata(columnName, dataType, isNullable))
	}
	return columns, nil
}

// GetColumnTypes returns QuestDB column types via information_schema.columns.
func (p *QuestDBPlugin) GetColumnTypes(db *gorm.DB, schema, tableName string) (map[string]gorm_plugin.ColumnTypeInfo, error) {
	columns, err := p.readQuestDBColumns(db, schema, tableName)
	if err != nil {
		return nil, err
	}

	columnTypes := make(map[string]gorm_plugin.ColumnTypeInfo, len(columns))
	for _, column := range columns {
		columnTypes[column.name] = gorm_plugin.ColumnTypeInfo{
			Type:       column.dataType,
			IsNullable: column.isNullable,
		}
	}
	return columnTypes, nil
}

// GetColumnsForTable returns QuestDB columns via information_schema.columns.
func (p *QuestDBPlugin) GetColumnsForTable(config *engine.PluginConfig, schema string, storageUnit string) ([]engine.Column, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]engine.Column, error) {
		columnMetadata, err := p.readQuestDBColumns(db, schema, storageUnit)
		if err != nil {
			log.WithError(err).Error(fmt.Sprintf("Failed to get columns for table %s.%s", schema, storageUnit))
			return nil, err
		}

		primaryKeys, err := p.GetPrimaryKeyColumns(db, schema, storageUnit)
		if err != nil {
			log.WithError(err).Warn(fmt.Sprintf("Failed to get primary keys for table %s.%s", schema, storageUnit))
			primaryKeys = []string{}
		}

		columns := make([]engine.Column, 0, len(columnMetadata))
		for _, column := range columnMetadata {
			columns = append(columns, engine.Column{
				Name:       column.name,
				Type:       column.dataType,
				IsNullable: column.isNullable,
				IsPrimary:  slices.Contains(primaryKeys, column.name),
			})
		}

		return columns, nil
	})
}

// GetPrimaryKeyColQuery returns a primary key query that tolerates schema-less
// QuestDB source references.
func (p *QuestDBPlugin) GetPrimaryKeyColQuery() string {
	return `
		SELECT a.attname
		FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		JOIN pg_class c ON c.oid = i.indrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE ($1 = '' OR n.nspname = $1) AND c.relname = $2 AND i.indisprimary;
	`
}

// GetForeignKeyRelationships returns an empty relationship set because the
// QuestDB fixtures and source model treat QuestDB tables as lacking foreign-key
// graph metadata.
func (p *QuestDBPlugin) GetForeignKeyRelationships(_ *engine.PluginConfig, _, _ string) (map[string]*engine.ForeignKeyRelationship, error) {
	return map[string]*engine.ForeignKeyRelationship{}, nil
}

// MarkGeneratedColumns is a no-op for QuestDB.
// The inherited PostgreSQL probe queries information_schema.columns.is_generated,
// which QuestDB does not expose.
func (p *QuestDBPlugin) MarkGeneratedColumns(config *engine.PluginConfig, schema string, storageUnit string, columns []engine.Column) error {
	return nil
}

// GetSSLStatus derives QuestDB SSL status from connection configuration.
// QuestDB speaks the PostgreSQL wire protocol but does not expose pg_stat_ssl,
// so the generic PostgreSQL runtime query fails.
func (p *QuestDBPlugin) GetSSLStatus(config *engine.PluginConfig) (*engine.SSLStatus, error) {
	if cached := plugins.GetCachedSSLStatus(config); cached != nil {
		return cached, nil
	}

	sslConfig := ssl.ParseSSLConfig(engine.DatabaseType_QuestDB, config.Credentials.Advanced, config.Credentials.Hostname, config.Credentials.IsProfile)

	var status *engine.SSLStatus
	if sslConfig == nil || !sslConfig.IsEnabled() {
		status = &engine.SSLStatus{
			IsEnabled: false,
			Mode:      string(ssl.SSLModeDisabled),
		}
	} else {
		status = &engine.SSLStatus{
			IsEnabled: true,
			Mode:      string(sslConfig.Mode),
		}
	}

	plugins.SetCachedSSLStatus(config, status)
	return status, nil
}

// GetCreateTableQuery generates QuestDB-compatible CREATE TABLE DDL.
// QuestDB only supports bare column definitions (name + type). It does not
// enforce PRIMARY KEY, NOT NULL, UNIQUE, CHECK, FK, DEFAULT, or IDENTITY.
func (p *QuestDBPlugin) GetCreateTableQuery(db *gorm.DB, schema string, storageUnit string, columns []engine.Record) string {
	builder := gorm_plugin.NewSQLBuilder(db, p)

	columnDefs := gorm_plugin.RecordsToColumnDefs(columns, func(def gorm_plugin.ColumnDef, _ engine.Record) gorm_plugin.ColumnDef {
		return def
	})

	for i := range columnDefs {
		columnDefs[i].Primary = false
		columnDefs[i].NotNull = false
		columnDefs[i].Nullable = true
		columnDefs[i].Unique = false
		columnDefs[i].Default = nil
		columnDefs[i].CheckValues = nil
		columnDefs[i].CheckMin = nil
		columnDefs[i].CheckMax = nil
		columnDefs[i].ReferencesTable = ""
		columnDefs[i].ReferencesColumn = ""
		columnDefs[i].Extra = ""
	}

	return builder.CreateTableQuery(schema, storageUnit, columnDefs)
}

// NewQuestDBPlugin creates a QuestDB plugin that reuses the PostgreSQL runtime
// while overriding the incompatible catalog and metadata paths.
func NewQuestDBPlugin() *engine.Plugin {
	questDBPlugin := &QuestDBPlugin{}
	questDBPlugin.Type = engine.DatabaseType_QuestDB
	questDBPlugin.PluginFunctions = questDBPlugin
	questDBPlugin.GormPluginFunctions = questDBPlugin
	return &questDBPlugin.Plugin
}

func init() {
	engine.RegisterPlugin(NewQuestDBPlugin())
}
