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

package duckdb

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	mapset "github.com/deckarep/golang-set/v2"
	"gorm.io/gorm"
)

var (
	supportedColumnDataTypes = mapset.NewSet(
		"BOOLEAN", "TINYINT", "SMALLINT", "INTEGER", "BIGINT", "UTINYINT", "USMALLINT", "UINTEGER", "UBIGINT", "HUGEINT",
		"REAL", "DOUBLE", "DECIMAL", "NUMERIC",
		"VARCHAR", "CHAR", "TEXT", "STRING", "BPCHAR",
		"BYTEA", "BLOB", "VARBINARY", "BINARY",
		"DATE", "TIME", "TIMESTAMP", "TIMESTAMPTZ", "INTERVAL",
		"UUID", "JSON",
		"ARRAY", "LIST", "STRUCT", "MAP", "UNION",
		"BIT", "BITSTRING",
	)
)

type DuckDBPlugin struct {
	gorm_plugin.GormPlugin
}

func (p *DuckDBPlugin) GetSupportedColumnDataTypes() mapset.Set[string] {
	return supportedColumnDataTypes
}

func (p *DuckDBPlugin) GetAllSchemasQuery() string {
	return ""
}

func (p *DuckDBPlugin) FormTableName(schema string, storageUnit string) string {
	return storageUnit
}

func (p *DuckDBPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	directory := getDefaultDirectory()
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	databases := []string{}
	for _, e := range entries {
		if !e.IsDir() {
			fileName := e.Name()
			// Accept .duckdb, .ddb, and .db files as requested
			if strings.HasSuffix(fileName, ".duckdb") || strings.HasSuffix(fileName, ".ddb") || strings.HasSuffix(fileName, ".db") {
				databases = append(databases, fileName)
			}
		}
	}

	return databases, nil
}

func (p *DuckDBPlugin) GetAllSchemas(config *engine.PluginConfig) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (p *DuckDBPlugin) GetTableInfoQuery() string {
	return `
		SELECT
			table_name,
			table_type
		FROM
			information_schema.tables
		WHERE
			table_schema = 'main'
			AND table_type IN ('BASE TABLE', 'LOCAL TEMPORARY')
	`
}

func (p *DuckDBPlugin) GetTableNameAndAttributes(rows *sql.Rows, db *gorm.DB) (string, []engine.Record) {
	var tableName, tableType string
	if err := rows.Scan(&tableName, &tableType); err != nil {
		log.Printf("Error scanning table info: %v", err)
		return "", nil
	}

	var rowCount int64
	rowCountRow := db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Row()
	err := rowCountRow.Scan(&rowCount)
	if err != nil {
		log.Printf("Error getting row count for table %s: %v", tableName, err)
		return "", nil
	}

	attributes := []engine.Record{
		{Key: "Type", Value: tableType},
		{Key: "Count", Value: fmt.Sprintf("%d", rowCount)},
	}

	return tableName, attributes
}

func (p *DuckDBPlugin) GetSchemaTableQuery() string {
	return `
		SELECT
			table_name AS TABLE_NAME,
			column_name AS COLUMN_NAME,
			data_type AS DATA_TYPE
		FROM
			information_schema.columns
		WHERE
			table_schema = 'main'
		ORDER BY
			table_name, ordinal_position
	`
}

func (p *DuckDBPlugin) executeRawSQL(config *engine.PluginConfig, query string, params ...interface{}) (*engine.GetRowsResult, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.GetRowsResult, error) {
		rows, err := db.Raw(query, params...).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return p.ConvertRawToRows(rows)
	})
}

func (p *DuckDBPlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return p.executeRawSQL(config, query)
}

func NewDuckDBPlugin() *engine.Plugin {
	plugin := &DuckDBPlugin{}
	plugin.Type = engine.DatabaseType_DuckDB
	plugin.PluginFunctions = plugin
	plugin.GormPluginFunctions = plugin
	return &plugin.Plugin
}