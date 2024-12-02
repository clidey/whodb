package clickhouse

import (
	"context"
	"fmt"
	"github.com/clidey/whodb/core/src/engine"
	"strings"
	"time"
)

func (p *ClickHousePlugin) UpdateStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	conn, err := DB(config)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	columnTypes, err := getColumnTypes(conn, schema, storageUnit)
	if err != nil {
		return false, err
	}

	primaryKeys, err := getPrimaryKeyColumns(conn, schema, storageUnit)
	if err != nil {
		return false, err
	}

	var whereArgs []interface{}
	var whereClauses []string

	// Build WHERE clause using only primary keys
	for _, pk := range primaryKeys {
		value, exists := values[pk]
		if !exists {
			return false, fmt.Errorf("primary key %s value not provided", pk)
		}

		colType, exists := columnTypes[pk]
		if !exists {
			return false, fmt.Errorf("column %s does not exist", pk)
		}

		convertedValue, err := convertStringValue(value, colType)
		if err != nil {
			return false, fmt.Errorf("error converting value for primary key %s: %w", pk, err)
		}

		// Handle datetime formatting if needed
		if t, ok := convertedValue.(time.Time); ok {
			convertedValue = t
		}

		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", pk))
		whereArgs = append(whereArgs, convertedValue)
	}

	var setArgs []interface{}
	var setClauses []string

	for column, value := range values {
		if isPrimaryKey(column, primaryKeys) {
			continue
		}

		colType, exists := columnTypes[column]
		if !exists {
			return false, fmt.Errorf("column %s does not exist", column)
		}

		convertedValue, err := convertStringValue(value, colType)
		if err != nil {
			return false, fmt.Errorf("error converting value for column %s: %w", column, err)
		}

		// Handle datetime formatting if needed
		if t, ok := convertedValue.(time.Time); ok {
			convertedValue = t
		}

		setClauses = append(setClauses, fmt.Sprintf("%s = ?", column))
		setArgs = append(setArgs, convertedValue)
	}

	if len(setClauses) == 0 {
		return false, fmt.Errorf("no columns to update")
	}

	if len(whereClauses) == 0 {
		return false, fmt.Errorf("no primary key values provided")
	}

	args := append(setArgs, whereArgs...)

	query := fmt.Sprintf(`
		ALTER TABLE %s.%s
		UPDATE %s
		WHERE %s`,
		schema, storageUnit,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "))

	err = conn.Exec(context.Background(), query, args...)
	if err != nil {
		return false, fmt.Errorf("update failed: %w", err)
	}

	return true, nil
}
