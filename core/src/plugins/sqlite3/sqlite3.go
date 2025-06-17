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

package sqlite3

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	mapset "github.com/deckarep/golang-set/v2"

	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/gorm"
)

var (
	supportedColumnDataTypes = mapset.NewSet(
		"NULL", "INTEGER", "REAL", "TEXT", "BLOB",
		"NUMERIC", "BOOLEAN", "DATE", "DATETIME",
	)
)

type Sqlite3Plugin struct {
	gorm_plugin.GormPlugin
}

func (p *Sqlite3Plugin) GetSupportedColumnDataTypes() mapset.Set[string] {
	return supportedColumnDataTypes
}

func (p *Sqlite3Plugin) GetAllSchemasQuery() string {
	return ""
}

func (p *Sqlite3Plugin) FormTableName(schema string, storageUnit string) string {
	return storageUnit
}

func (p *Sqlite3Plugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	directory := getDefaultDirectory()
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	databases := []string{}
	for _, e := range entries {
		databases = append(databases, e.Name())
	}

	return databases, nil
}

func (p *Sqlite3Plugin) GetAllSchemas(config *engine.PluginConfig) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (p *Sqlite3Plugin) GetTableInfoQuery() string {
	return `
		SELECT
			name AS table_name,
			type AS table_type
		FROM
			sqlite_master
		WHERE
			type='table' AND name NOT LIKE 'sqlite_%'
	`
}

func (p *Sqlite3Plugin) GetTableNameAndAttributes(rows *sql.Rows, db *gorm.DB) (string, []engine.Record) {
	var tableName, tableType string
	if err := rows.Scan(&tableName, &tableType); err != nil {
		log.Fatal(err)
	}

	var rowCount int64
	escapedTableName := p.EscapeIdentifier(tableName)
	rowCountRow := db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s", escapedTableName)).Row()
	err := rowCountRow.Scan(&rowCount)
	if err != nil {
		return "", nil
	}

	attributes := []engine.Record{
		{Key: "Type", Value: tableType},
		{Key: "Count", Value: fmt.Sprintf("%d", rowCount)},
	}

	return tableName, attributes
}

func (p *Sqlite3Plugin) GetSchemaTableQuery() string {
	return `
		SELECT m.name AS TABLE_NAME,
			   p.name AS COLUMN_NAME,
			   p.type AS DATA_TYPE
		FROM sqlite_master m,
			 pragma_table_info(m.name) p
		WHERE m.type = 'table'
		  AND m.name NOT LIKE 'sqlite_%';
	`
}

func (p *Sqlite3Plugin) executeRawSQL(config *engine.PluginConfig, query string, params ...interface{}) (*engine.GetRowsResult, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.GetRowsResult, error) {
		rows, err := db.Raw(query, params...).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return p.ConvertRawToRows(rows)
	})
}

func (p *Sqlite3Plugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return p.executeRawSQL(config, query)
}

func NewSqlite3Plugin() *engine.Plugin {
	plugin := &Sqlite3Plugin{}
	plugin.Type = engine.DatabaseType_Sqlite3
	plugin.PluginFunctions = plugin
	plugin.GormPluginFunctions = plugin
	return &plugin.Plugin
}
