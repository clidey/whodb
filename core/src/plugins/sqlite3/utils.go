package sqlite3

import (
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

func getTableInfo(db *gorm.DB, tableName string) ([]string, map[string]string, error) {
	var primaryKeys []string
	columnTypes := make(map[string]string)
	pragmaQuery := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := db.Raw(pragmaQuery, tableName).Rows()
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			name      string
			type_     string
			notnull   int
			dfltValue interface{}
			pk        int
		)
		if err := rows.Scan(&cid, &name, &type_, &notnull, &dfltValue, &pk); err != nil {
			return nil, nil, err
		}
		columnTypes[name] = type_
		if pk == 1 {
			primaryKeys = append(primaryKeys, name)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	if len(primaryKeys) == 0 {
		return nil, nil, fmt.Errorf("no primary key found for table %s", tableName)
	}

	return primaryKeys, columnTypes, nil
}

func convertStringValue(value, columnType string) (interface{}, error) {
	switch columnType {
	case "INTEGER":
		return strconv.ParseInt(value, 10, 64)
	case "REAL":
		return strconv.ParseFloat(value, 64)
	case "BOOLEAN":
		return strconv.ParseBool(value)
	case "DATE":
		_, err := time.Parse("2006-01-02", value)
		if err != nil {
			return nil, fmt.Errorf("invalid date format: %v", err)
		}
		return value, nil
	case "DATETIME":
		_, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, fmt.Errorf("invalid datetime format: %v", err)
		}
		return value, nil
	default:
		return value, nil
	}
}
