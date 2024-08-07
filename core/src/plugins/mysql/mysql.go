package mysql

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/gorm"
)

type MySQLPlugin struct{}

func (p *MySQLPlugin) IsAvailable(config *engine.PluginConfig) bool {
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

func (p *MySQLPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	db, err := DB(config)
	if err != nil {
		return nil, err
	}
	sqlDb, err := db.DB()
	if err != nil {
		return nil, err
	}
	defer sqlDb.Close()

	var databases []struct {
		DatabaseName string `gorm:"column:databasename"`
	}
	if err := db.Raw("SHOW DATABASES").Scan(&databases).Error; err != nil {
		return nil, err
	}

	databaseNames := []string{}
	for _, database := range databases {
		databaseNames = append(databaseNames, database.DatabaseName)
	}

	return databaseNames, nil
}

func (p *MySQLPlugin) GetSchema(config *engine.PluginConfig) ([]string, error) {
	db, err := DB(config)
	if err != nil {
		return nil, err
	}
	sqlDb, err := db.DB()
	if err != nil {
		return nil, err
	}
	defer sqlDb.Close()
	var schemas []struct {
		SchemaName string `gorm:"column:schemaname"`
	}
	if err := db.Raw("SELECT SCHEMA_NAME AS schemaname FROM INFORMATION_SCHEMA.SCHEMATA").Scan(&schemas).Error; err != nil {
		return nil, err
	}
	schemaNames := []string{}
	for _, schema := range schemas {
		schemaNames = append(schemaNames, schema.SchemaName)
	}
	return schemaNames, nil
}

func (p *MySQLPlugin) GetStorageUnits(config *engine.PluginConfig, schema string) ([]engine.StorageUnit, error) {
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
	rows, err := db.Raw(fmt.Sprintf(`
		SELECT
			TABLE_NAME,
			TABLE_TYPE,
			IFNULL(ROUND(((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024), 2), 0) AS total_size,
			IFNULL(ROUND((DATA_LENGTH / 1024 / 1024), 2), 0) AS data_size,
			IFNULL(TABLE_ROWS, 0) AS row_count
		FROM
			INFORMATION_SCHEMA.TABLES
		WHERE
			TABLE_SCHEMA = '%v'
	`, schema)).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allTablesWithColumns, err := getTableSchema(db, schema)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var tableName, tableType string
		var totalSize, dataSize float64
		var rowCount int64
		if err := rows.Scan(&tableName, &tableType, &totalSize, &dataSize, &rowCount); err != nil {
			log.Fatal(err)
		}

		attributes := []engine.Record{
			{Key: "Table Type", Value: tableType},
			{Key: "Total Size", Value: fmt.Sprintf("%.2f MB", totalSize)},
			{Key: "Data Size", Value: fmt.Sprintf("%.2f MB", dataSize)},
			{Key: "Count", Value: fmt.Sprintf("%d", rowCount)},
		}

		attributes = append(attributes, allTablesWithColumns[tableName]...)

		storageUnits = append(storageUnits, engine.StorageUnit{
			Name:       tableName,
			Attributes: attributes,
		})
	}
	return storageUnits, nil
}

func getTableSchema(db *gorm.DB, schema string) (map[string][]engine.Record, error) {
	var result []struct {
		TableName  string `gorm:"column:table_name"`
		ColumnName string `gorm:"column:column_name"`
		DataType   string `gorm:"column:data_type"`
	}

	query := fmt.Sprintf(`
		SELECT TABLE_NAME, COLUMN_NAME, DATA_TYPE
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = '%v'
		ORDER BY TABLE_NAME, ORDINAL_POSITION
	`, schema)

	if err := db.Raw(query).Scan(&result).Error; err != nil {
		return nil, err
	}

	tableColumnsMap := make(map[string][]engine.Record)
	for _, row := range result {
		tableColumnsMap[row.TableName] = append(tableColumnsMap[row.TableName], engine.Record{Key: row.ColumnName, Value: row.DataType})
	}

	return tableColumnsMap, nil
}

func (p *MySQLPlugin) GetRows(config *engine.PluginConfig, schema string, storageUnit string, where string, pageSize int, pageOffset int) (*engine.GetRowsResult, error) {
	query := fmt.Sprintf("SELECT * FROM `%v`.`%s`", schema, storageUnit)
	if len(where) > 0 {
		query = fmt.Sprintf("%v WHERE %v", query, where)
	}
	query = fmt.Sprintf("%v LIMIT ? OFFSET ?", query)
	return p.executeRawSQL(config, query, pageSize, pageOffset)
}

func (p *MySQLPlugin) executeRawSQL(config *engine.PluginConfig, query string, params ...interface{}) (*engine.GetRowsResult, error) {
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

func (p *MySQLPlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return p.executeRawSQL(config, query)
}

func NewMySQLPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_MySQL,
		PluginFunctions: &MySQLPlugin{},
	}
}
func NewMyMariaDBPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_MariaDB,
		PluginFunctions: &MySQLPlugin{},
	}
}
