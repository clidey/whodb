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
	rowCountRow := db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM '%s'", tableName)).Row()
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
	return plugins.WithConnection[*engine.GetRowsResult](config, p.DB, func(db *gorm.DB) (*engine.GetRowsResult, error) {
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
