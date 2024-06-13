package postgres

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins/common"
)

type PostgresPlugin struct{}

func (p *PostgresPlugin) GetStorageUnits(config *engine.PluginConfig) ([]engine.StorageUnit, error) {
	db, err := DB(config)
	if err != nil {
		return nil, err
	}
	storageUnits := []engine.StorageUnit{}
	rows, err := db.Raw(`
		SELECT
			table_name,
			table_type,
			table_schema,
			pg_size_pretty(pg_total_relation_size('"' || table_schema || '"."' || table_name || '"')) AS total_size,
			pg_size_pretty(pg_relation_size('"' || table_schema || '"."' || table_name || '"')) AS data_size,
			COALESCE((SELECT reltuples::bigint FROM pg_class WHERE oid = ('"' || table_schema || '"."' || table_name || '"')::regclass), 0) AS row_count
		FROM
			information_schema.tables
		WHERE
			table_schema = 'public'
	`).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, tableType, tableSchema, totalSize, dataSize string
		var rowCount int64
		if err := rows.Scan(&tableName, &tableType, &tableSchema, &totalSize, &dataSize, &rowCount); err != nil {
			return nil, err
		}

		attributes := map[string]string{
			"Table Type":   tableType,
			"Table Schema": tableSchema,
			"Total Size":   totalSize,
			"Data Size":    dataSize,
			"Row Count":    fmt.Sprintf("%d", rowCount),
		}

		storageUnits = append(storageUnits, engine.StorageUnit{
			Name:       tableName,
			Attributes: attributes,
		})
	}
	return storageUnits, nil
}

func (p *PostgresPlugin) GetRows(config *engine.PluginConfig, storageUnit string) (*engine.GetRowsResult, error) {
	if !common.IsValidSQLTableName(storageUnit) {
		return nil, errors.New("invalid table name")
	}

	db, err := DB(config)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?", storageUnit)
	rows, err := db.Raw(query, 10, 1).Rows()
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

func (p *PostgresPlugin) GetColumns(config *engine.PluginConfig, storageUnit string, row string) (map[string][]string, error) {
	return nil, nil
}

func (p *PostgresPlugin) GetConstraints(config *engine.PluginConfig) map[string]string {
	return nil
}

func (p *PostgresPlugin) RawExecute(config *engine.PluginConfig, sql string) error {
	return nil
}

func NewPostgresPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_Postgres,
		PluginFunctions: &PostgresPlugin{},
	}
}
