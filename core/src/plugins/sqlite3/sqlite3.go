package sqlite3

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins/common"
	"gorm.io/gorm"
)

type Sqlite3Plugin struct{}

func (p *Sqlite3Plugin) IsAvailable(config *engine.PluginConfig) bool {
	db, err := DB(config)
	if err != nil {
		return false
	}
	sqlDb, err := db.DB()
	if err != nil {
		return false
	}
	sqlDb.Close()
	return true
}

func (p *Sqlite3Plugin) GetDatabases() ([]string, error) {
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

func (p *Sqlite3Plugin) GetSchema(config *engine.PluginConfig) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (p *Sqlite3Plugin) GetStorageUnits(config *engine.PluginConfig, schema string) ([]engine.StorageUnit, error) {
	db, err := DB(config)
	if err != nil {
		return nil, err
	}
	sqlDb, err := db.DB()
	if err != nil {
		return nil, err
	}
	defer sqlDb.Close()

	storageUnits := []engine.StorageUnit{}
	rows, err := db.Raw(`
		SELECT
			name AS table_name,
			type AS table_type
		FROM
			sqlite_master
		WHERE
			type='table' AND name NOT LIKE 'sqlite_%'
	`).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allTablesWithColumns, err := getTableSchema(db)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var tableName, tableType string
		if err := rows.Scan(&tableName, &tableType); err != nil {
			log.Fatal(err)
		}

		var rowCount int64
		rowCountRow := db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM '%s'", tableName)).Row()
		rowCountRow.Scan(&rowCount)

		var dataSize int64
		db.Raw(fmt.Sprintf("PRAGMA table_info('%s')", tableName)).Scan(&dataSize)

		attributes := []engine.Record{
			{Key: "Table Type", Value: tableType},
			{Key: "Count", Value: fmt.Sprintf("%d", rowCount)},
			{Key: "Data Size", Value: fmt.Sprintf("%d", dataSize)},
		}

		attributes = append(attributes, allTablesWithColumns[tableName]...)

		storageUnits = append(storageUnits, engine.StorageUnit{
			Name:       tableName,
			Attributes: attributes,
		})
	}
	return storageUnits, nil
}

func getTableSchema(db *gorm.DB) (map[string][]engine.Record, error) {
	var tables []struct {
		TableName string `gorm:"column:table_name"`
	}

	query := `
		SELECT name AS table_name
		FROM sqlite_master
		WHERE type='table'
	`
	if err := db.Raw(query).Scan(&tables).Error; err != nil {
		return nil, err
	}

	tableColumnsMap := make(map[string][]engine.Record)

	for _, table := range tables {
		var columns []struct {
			ColumnName string `gorm:"column:name"`
			DataType   string `gorm:"column:type"`
		}

		pragmaQuery := fmt.Sprintf("PRAGMA table_info(%s)", table.TableName)
		if err := db.Raw(pragmaQuery).Scan(&columns).Error; err != nil {
			return nil, err
		}

		for _, column := range columns {
			tableColumnsMap[table.TableName] = append(tableColumnsMap[table.TableName], engine.Record{Key: column.ColumnName, Value: column.DataType})
		}
	}

	return tableColumnsMap, nil
}

func (p *Sqlite3Plugin) GetRows(config *engine.PluginConfig, schema string, storageUnit string, where string, pageSize int, pageOffset int) (*engine.GetRowsResult, error) {
	if !common.IsValidSQLTableName(storageUnit) {
		return nil, errors.New("invalid table name")
	}

	query := fmt.Sprintf("SELECT * FROM \"%s\"", storageUnit)
	if len(where) > 0 {
		query = fmt.Sprintf("%v WHERE %v", query, where)
	}
	query = fmt.Sprintf("%v LIMIT ? OFFSET ?", query)
	return p.executeRawSQL(config, query, pageSize, pageOffset)
}

func (p *Sqlite3Plugin) executeRawSQL(config *engine.PluginConfig, query string, params ...interface{}) (*engine.GetRowsResult, error) {
	db, err := DB(config)
	if err != nil {
		return nil, err
	}

	sqlDb, err := db.DB()
	if err != nil {
		return nil, err
	}
	defer sqlDb.Close()
	rows, err := db.Raw(query, params...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	result := &engine.GetRowsResult{}
	for _, col := range columns {
		for _, colType := range columnTypes {
			if col == colType.Name() {
				result.Columns = append(result.Columns, engine.Column{Name: col, Type: colType.DatabaseTypeName()})
				break
			}
		}
	}

	for rows.Next() {
		columnPointers := make([]interface{}, len(columns))
		row := make([]string, len(columns))
		for i := range columns {
			columnPointers[i] = new(sql.NullString)
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		for i, colPtr := range columnPointers {
			val := colPtr.(*sql.NullString)
			if val.Valid {
				row[i] = val.String
			} else {
				row[i] = ""
			}
		}

		result.Rows = append(result.Rows, row)
	}

	return result, nil
}

func (p *Sqlite3Plugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return p.executeRawSQL(config, query)
}

func NewSqlite3Plugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_Sqlite3,
		PluginFunctions: &Sqlite3Plugin{},
	}
}
