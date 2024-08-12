package postgres

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/gorm"
)

type PostgresPlugin struct{}

func (p *PostgresPlugin) IsAvailable(config *engine.PluginConfig) bool {
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

func (p *PostgresPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
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
		Datname string `gorm:"column:datname"`
	}
	if err := db.Raw("SELECT datname AS datname FROM pg_database WHERE datistemplate = false").Scan(&databases).Error; err != nil {
		return nil, err
	}
	databaseNames := []string{}
	for _, database := range databases {
		databaseNames = append(databaseNames, database.Datname)
	}
	return databaseNames, nil
}

func (p *PostgresPlugin) GetSchema(config *engine.PluginConfig) ([]string, error) {
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
	if err := db.Raw("SELECT schema_name AS schemaname FROM information_schema.schemata").Scan(&schemas).Error; err != nil {
		return nil, err
	}
	schemaNames := []string{}
	for _, schema := range schemas {
		schemaNames = append(schemaNames, schema.SchemaName)
	}
	return schemaNames, nil
}

func (p *PostgresPlugin) GetStorageUnits(config *engine.PluginConfig, schema string) ([]engine.StorageUnit, error) {
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
			t.table_name,
			t.table_type,
			pg_size_pretty(pg_total_relation_size('"' || t.table_schema || '"."' || t.table_name || '"')) AS total_size,
			pg_size_pretty(pg_relation_size('"' || t.table_schema || '"."' || t.table_name || '"')) AS data_size,
			COALESCE((SELECT reltuples::bigint FROM pg_class WHERE oid = ('"' || t.table_schema || '"."' || t.table_name || '"')::regclass), 0) AS row_count
		FROM
			information_schema.tables t
		JOIN
			pg_class c ON t.table_name = c.relname AND t.table_schema = c.relnamespace::regnamespace::text
		WHERE
			t.table_schema = '%v'
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
		var tableName, tableType, totalSize, dataSize string
		var rowCount int64
		if err := rows.Scan(&tableName, &tableType, &totalSize, &dataSize, &rowCount); err != nil {
			log.Fatal(err)
		}

		rowCountRecordValue := "unknown"
		if rowCount >= 0 {
			rowCountRecordValue = fmt.Sprintf("%d", rowCount)
		}

		attributes := []engine.Record{
			{Key: "Table Type", Value: tableType},
			{Key: "Total Size", Value: totalSize},
			{Key: "Data Size", Value: dataSize},
			{Key: "Count", Value: rowCountRecordValue},
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
		SELECT table_name, column_name, data_type
		FROM information_schema.columns
		WHERE table_schema = '%v'
		ORDER BY table_name, ordinal_position
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

func (p *PostgresPlugin) GetRows(config *engine.PluginConfig, schema string, storageUnit string, where string, pageSize int, pageOffset int) (*engine.GetRowsResult, error) {
	query := fmt.Sprintf("SELECT * FROM \"%v\".\"%s\"", schema, storageUnit)
	if len(where) > 0 {
		query = fmt.Sprintf("%v WHERE %v", query, where)
	}
	query = fmt.Sprintf("%v LIMIT ? OFFSET ?", query)
	return p.executeRawSQL(config, query, pageSize, pageOffset)
}

func (p *PostgresPlugin) executeRawSQL(config *engine.PluginConfig, query string, params ...interface{}) (*engine.GetRowsResult, error) {
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

func (p *PostgresPlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return p.executeRawSQL(config, query)
}

func NewPostgresPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_Postgres,
		PluginFunctions: &PostgresPlugin{},
	}
}
