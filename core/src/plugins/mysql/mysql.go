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

package mysql

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	mapset "github.com/deckarep/golang-set/v2"
	"gorm.io/gorm"
)

var (
	supportedColumnDataTypes = mapset.NewSet(
		"TINYINT", "SMALLINT", "MEDIUMINT", "INT", "INTEGER", "BIGINT", "FLOAT", "DOUBLE", "DECIMAL",
		"DATE", "DATETIME", "TIMESTAMP", "TIME", "YEAR",
		"CHAR", "VARCHAR(255)", "BINARY", "VARBINARY", "TINYBLOB", "BLOB", "MEDIUMBLOB", "LONGBLOB",
		"TINYTEXT", "TEXT", "MEDIUMTEXT", "LONGTEXT",
		"ENUM", "SET", "JSON", "BOOLEAN", "VARCHAR(100)", "VARCHAR(1000)",
	)

	supportedOperators = map[string]string{
		"=": "=", ">=": ">=", ">": ">", "<=": "<=", "<": "<", "<>": "<>",
		"!=": "!=", "!>": "!>", "!<": "!<", "BETWEEN": "BETWEEN", "NOT BETWEEN": "NOT BETWEEN",
		"LIKE": "LIKE", "NOT LIKE": "NOT LIKE", "IN": "IN", "NOT IN": "NOT IN",
		"IS NULL": "IS NULL", "IS NOT NULL": "IS NOT NULL", "AND": "AND", "OR": "OR", "NOT": "NOT",
	}
)

type MySQLPlugin struct {
	gorm_plugin.GormPlugin
}

func (p *MySQLPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	return nil, errors.ErrUnsupported
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

func (p *MySQLPlugin) GetSupportedColumnDataTypes() mapset.Set[string] {
	return supportedColumnDataTypes
}

func (p *MySQLPlugin) GetSupportedOperators() map[string]string {
	return supportedOperators
}

func (p *MySQLPlugin) GetSchemaTableQuery() string {
	return `SELECT TABLE_NAME, COLUMN_NAME, DATA_TYPE
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = ?
			ORDER BY TABLE_NAME, ORDINAL_POSITION`
}

func (p *MySQLPlugin) GetTableInfoQuery() string {
	return `
		SELECT
			TABLE_NAME,
			TABLE_TYPE,
			IFNULL(ROUND(((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024), 2), 0) AS total_size,
			IFNULL(ROUND((DATA_LENGTH / 1024 / 1024), 2), 0) AS data_size,
			IFNULL(TABLE_ROWS, 0) AS row_count
		FROM
			INFORMATION_SCHEMA.TABLES
		WHERE
			TABLE_SCHEMA = ?`
}

func (p *MySQLPlugin) GetPlaceholder(index int) string {
	return "?"
}

func (p *MySQLPlugin) GetTableNameAndAttributes(rows *sql.Rows, db *gorm.DB) (string, []engine.Record) {
	var tableName, tableType string
	var totalSize, dataSize float64
	var rowCount int64
	if err := rows.Scan(&tableName, &tableType, &totalSize, &dataSize, &rowCount); err != nil {
		log.Logger.WithError(err).Error("Failed to scan MySQL table information")
		return "", []engine.Record{}
	}

	// If row count is 0 or suspiciously low, do a select count which shouldn't be too expensive
	// MySQL's TABLE_ROWS is just an estimate that can be very inaccurate
	if rowCount < 100 {
		var actualCount int64
		countQuery := db.Table(tableName).Select("COUNT(*)")
		if err := countQuery.Scan(&actualCount).Error; err == nil {
			rowCount = actualCount
		}
	}

	attributes := []engine.Record{
		{Key: "Type", Value: tableType},
		{Key: "Total Size", Value: fmt.Sprintf("%.2f MB", totalSize)},
		{Key: "Data Size", Value: fmt.Sprintf("%.2f MB", dataSize)},
		{Key: "Count", Value: fmt.Sprintf("%d", rowCount)},
	}
	return tableName, attributes
}

func (p *MySQLPlugin) executeRawSQL(config *engine.PluginConfig, query string, params ...interface{}) (*engine.GetRowsResult, error) {
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
