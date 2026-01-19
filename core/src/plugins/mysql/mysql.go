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

package mysql

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
		"LIKE": "LIKE", "NOT LIKE": "NOT LIKE", "IN": "IN", "NOT IN": "NOT IN",
		"IS NULL": "IS NULL", "IS NOT NULL": "IS NOT NULL", "AND": "AND", "OR": "OR", "NOT": "NOT",
	}
)

type MySQLPlugin struct {
	gorm_plugin.GormPlugin
}

func (p *MySQLPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]string, error) {
		var databases []struct {
			Database string `gorm:"column:Database"`
		}
		if err := db.Raw("SHOW DATABASES").Scan(&databases).Error; err != nil {
			return nil, err
		}
		var databaseNames []string
		for _, database := range databases {
			databaseNames = append(databaseNames, database.Database)
		}
		return databaseNames, nil
	})
}

func (p *MySQLPlugin) GetAllSchemasQuery() string {
	return "SELECT SCHEMA_NAME AS schemaname FROM INFORMATION_SCHEMA.SCHEMATA"
}

func (p *MySQLPlugin) FormTableName(schema string, storageUnit string) string {
	if schema == "" {
		return storageUnit
	}
	return schema + "." + storageUnit
}

func (p *MySQLPlugin) GetSupportedOperators() map[string]string {
	return supportedOperators
}

func (p *MySQLPlugin) GetTableInfoQuery() string {
	return `
		SELECT
			TABLE_NAME,
			TABLE_TYPE,
			IFNULL(ROUND(((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024), 2), 0) AS total_size,
			IFNULL(ROUND((DATA_LENGTH / 1024 / 1024), 2), 0) AS data_size
		FROM
			INFORMATION_SCHEMA.TABLES
		WHERE
			TABLE_SCHEMA = ?`
}

func (p *MySQLPlugin) GetStorageUnitExistsQuery() string {
	return `SELECT EXISTS(SELECT 1 FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?)`
}

func (p *MySQLPlugin) GetPlaceholder(index int) string {
	return "?"
}

func (p *MySQLPlugin) GetTableNameAndAttributes(rows *sql.Rows) (string, []engine.Record) {
	var tableName, tableType string
	var totalSize, dataSize float64
	if err := rows.Scan(&tableName, &tableType, &totalSize, &dataSize); err != nil {
		log.Logger.WithError(err).Error("Failed to scan MySQL table information")
		return "", []engine.Record{}
	}

	attributes := []engine.Record{
		{Key: "Type", Value: tableType},
		{Key: "Total Size", Value: fmt.Sprintf("%.2f MB", totalSize)},
		{Key: "Data Size", Value: fmt.Sprintf("%.2f MB", dataSize)},
	}
	return tableName, attributes
}

func (p *MySQLPlugin) executeRawSQL(config *engine.PluginConfig, query string, params ...any) (*engine.GetRowsResult, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.GetRowsResult, error) {
		rows, err := db.Raw(query, params...).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return p.ConvertRawToRows(rows)
	})
}

func (p *MySQLPlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return p.executeRawSQL(config, query)
}

// CreateSQLBuilder creates a MySQL-specific SQL builder
func (p *MySQLPlugin) CreateSQLBuilder(db *gorm.DB) gorm_plugin.SQLBuilderInterface {
	return NewMySQLSQLBuilder(db, p)
}

func (p *MySQLPlugin) GetForeignKeyRelationships(config *engine.PluginConfig, schema string, storageUnit string) (map[string]*engine.ForeignKeyRelationship, error) {
	query := `
		SELECT
			COLUMN_NAME,
			REFERENCED_TABLE_NAME,
			REFERENCED_COLUMN_NAME
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = ?
			AND TABLE_NAME = ?
			AND REFERENCED_TABLE_NAME IS NOT NULL
	`
	return p.QueryForeignKeyRelationships(config, query, schema, storageUnit)
}

// NormalizeType converts MySQL type aliases to their canonical form.
func (p *MySQLPlugin) NormalizeType(typeName string) string {
	return NormalizeType(typeName)
}

func NewMySQLPlugin() *engine.Plugin {
	mysqlPlugin := &MySQLPlugin{}
	mysqlPlugin.Type = engine.DatabaseType_MySQL
	mysqlPlugin.PluginFunctions = mysqlPlugin
	mysqlPlugin.GormPluginFunctions = mysqlPlugin
	return &mysqlPlugin.Plugin
}

func NewMyMariaDBPlugin() *engine.Plugin {
	mysqlPlugin := &MySQLPlugin{}
	mysqlPlugin.Type = engine.DatabaseType_MariaDB
	mysqlPlugin.PluginFunctions = mysqlPlugin
	mysqlPlugin.GormPluginFunctions = mysqlPlugin
	return &mysqlPlugin.Plugin
}
