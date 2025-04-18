// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clickhouse

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/clidey/whodb/core/src/engine"
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
)

type ClickHousePlugin struct {
	gorm_plugin.GormPlugin
}

func (p *ClickHousePlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	return plugins.WithConnection[[]string](config, p.DB, func(db *gorm.DB) ([]string, error) {
		var databases []struct {
			Datname string `gorm:"column:datname"`
		}
		if err := db.Raw("SELECT name AS datname FROM system.databases").Scan(&databases).Error; err != nil {
			return nil, err
		}
		databaseNames := []string{}
		for _, database := range databases {
			databaseNames = append(databaseNames, database.Datname)
		}
		return databaseNames, nil
	})
}

func (p *ClickHousePlugin) FormTableName(schema string, storageUnit string) string {
	return fmt.Sprintf("%s.%s", schema, storageUnit)
}

func (p *ClickHousePlugin) GetSupportedColumnDataTypes() mapset.Set[string] {
	return supportedColumnDataTypes
}

func (p *ClickHousePlugin) GetAllSchemas(config *engine.PluginConfig) ([]string, error) {
	// technically a table is considered a schema in clickhouse
	return nil, errors.ErrUnsupported
}

func (p *ClickHousePlugin) GetTableInfoQuery() string {
	return `
		SELECT 
			name,
			engine,
			total_rows,
			formatReadableSize(total_bytes) as total_size
		FROM system.tables 
		WHERE database = ?
	`
}

func (p *ClickHousePlugin) GetTableNameAndAttributes(rows *sql.Rows, db *gorm.DB) (string, []engine.Record) {
	var tableName, tableType string
	var totalRows *uint64
	var totalSize *string

	if err := rows.Scan(&tableName, &tableType, &totalRows, &totalSize); err != nil {
		log.Fatal(err)
	}

	rowCount := "0"
	if totalRows != nil {
		rowCount = strconv.FormatUint(*totalRows, 10)
	}

	size := "unknown"
	if totalSize != nil {
		size = *totalSize
	}

	attributes := []engine.Record{
		{Key: "Type", Value: tableType},
		{Key: "Total Size", Value: size},
		{Key: "Count", Value: rowCount},
	}

	return tableName, attributes
}

func (p *ClickHousePlugin) GetSchemaTableQuery() string {
	return `
		SELECT 
		    table AS TABLE_NAME,
			name AS COLUMN_NAME,
			type AS DATA_TYPE
		FROM system.columns
		WHERE database = ?
		ORDER BY TABLE_NAME, position
	`
}

func (p *ClickHousePlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return p.executeRawSQL(config, query)
}

func (p *ClickHousePlugin) executeRawSQL(config *engine.PluginConfig, query string, params ...interface{}) (*engine.GetRowsResult, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.GetRowsResult, error) {
		rows, err := db.Raw(query, params...).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return p.ConvertRawToRows(rows)
	})
}

func NewClickHousePlugin() *engine.Plugin {
	clickhousePlugin := &ClickHousePlugin{}
	clickhousePlugin.Type = engine.DatabaseType_ClickHouse
	clickhousePlugin.PluginFunctions = clickhousePlugin
	clickhousePlugin.GormPluginFunctions = clickhousePlugin
	return &clickhousePlugin.Plugin
}
