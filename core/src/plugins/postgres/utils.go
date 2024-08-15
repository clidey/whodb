package postgres

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getPrimaryKeyColumns(db *gorm.DB, schema string, tableName string) ([]string, error) {
	var primaryKeys []string
	query := `
		SELECT a.attname
		FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		JOIN pg_class c ON c.oid = i.indrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ? AND c.relname = ? AND i.indisprimary;
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
	case "integer", "smallint", "bigint":
		return strconv.ParseInt(value, 10, 64)
	case "numeric", "real", "double precision":
		return strconv.ParseFloat(value, 64)
	case "boolean":
		return strconv.ParseBool(value)
	case "uuid":
		_, err := uuid.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("invalid UUID format: %v", err)
		}
		return value, nil
	case "date":
		_, err := time.Parse("2006-01-02", value)
		if err != nil {
			return nil, fmt.Errorf("invalid date format: %v", err)
		}
		return value, nil
	case "timestamp", "timestamp with time zone", "timestamp without time zone":
		_, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp format: %v", err)
		}
		return value, nil
	default:
		return value, nil
	}
}
