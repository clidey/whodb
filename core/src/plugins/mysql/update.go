package mysql

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/gorm"
)

func (p *MySQLPlugin) UpdateStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	db, err := DB(config)
	if err != nil {
		return false, err
	}

	sqlDb, err := db.DB()
	if err != nil {
		return false, err
	}
	defer sqlDb.Close()

	pkColumns, err := getPrimaryKeyColumns(db, schema, storageUnit)
	if err != nil {
		return false, err
	}

	columnTypes, err := getColumnTypes(db, schema, storageUnit)
	if err != nil {
		return false, err
	}

	conditions := make(map[string]interface{})
	convertedValues := make(map[string]interface{})
	for column, strValue := range values {
		columnType, exists := columnTypes[column]
		if !exists {
			return false, fmt.Errorf("column '%s' does not exist in table %s", column, storageUnit)
		}

		convertedValue, err := convertStringValue(strValue, columnType)
		if err != nil {
			return false, fmt.Errorf("failed to convert value for column '%s': %v", column, err)
		}

		if common.ContainsString(pkColumns, column) {
			conditions[column] = convertedValue
		} else {
			convertedValues[column] = convertedValue
		}
	}

	tableName := fmt.Sprintf("%s.%s", schema, storageUnit)
	dbConditions := db.Table(tableName)
	for key, value := range conditions {
		dbConditions = dbConditions.Where(fmt.Sprintf("%s = ?", key), value)
	}

	result := dbConditions.Table(tableName).Updates(convertedValues)
	if result.Error != nil {
		return false, result.Error
	}

	if result.RowsAffected == 0 {
		return false, errors.New("no rows were updated")
	}

	return true, nil
}

func getPrimaryKeyColumns(db *gorm.DB, schema string, tableName string) ([]string, error) {
	var primaryKeys []string
	query := `
		SELECT k.column_name
		FROM information_schema.table_constraints t
		JOIN information_schema.key_column_usage k
		USING (constraint_name, table_schema, table_name)
		WHERE t.constraint_type = 'PRIMARY KEY'
		AND t.table_schema = ?
		AND t.table_name = ?;
	`
	rows, err := db.Raw(query, schema, tableName).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var pkColumn string
		if err := rows.Scan(&pkColumn); err != nil {
			return nil, err
		}
		primaryKeys = append(primaryKeys, pkColumn)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(primaryKeys) == 0 {
		return nil, fmt.Errorf("no primary key found for table %s", tableName)
	}

	return primaryKeys, nil
}

func (p *MySQLPlugin) DeleteStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	return false, errors.New("not implemented")
}

func getColumnTypes(db *gorm.DB, schema, tableName string) (map[string]string, error) {
	columnTypes := make(map[string]string)
	query := `
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?;
	`
	rows, err := db.Raw(query, schema, tableName).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var columnName, dataType string
		if err := rows.Scan(&columnName, &dataType); err != nil {
			return nil, err
		}
		columnTypes[columnName] = dataType
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columnTypes, nil
}

func convertStringValue(value, columnType string) (interface{}, error) {
	switch columnType {
	case "int", "bigint", "smallint", "tinyint", "mediumint":
		return strconv.Atoi(value)
	case "boolean", "bit":
		return strconv.ParseBool(value)
	case "float", "double", "decimal":
		return strconv.ParseFloat(value, 64)
	default:
		return value, nil
	}
}
