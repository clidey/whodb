package clickhouse

import (
	"context"
	"fmt"
	"github.com/clidey/whodb/core/src/common"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
)

func (p *ClickHousePlugin) DeleteRow(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	readOnly := common.GetRecordValueOrDefault(config.Credentials.Advanced, readOnlyKey, "disable")
	if readOnly != "disable" {
		return false, fmt.Errorf("readonly mode don't allow DeleteRow")
	}
	conn, err := DB(config)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	// Get column types and primary keys
	columnTypes, err := getColumnTypes(conn, schema, storageUnit)
	if err != nil {
		return false, err
	}

	primaryKeys, err := getPrimaryKeyColumns(conn, schema, storageUnit)
	if err != nil {
		return false, err
	}

	if len(primaryKeys) == 0 {
		return false, fmt.Errorf("no primary keys found for table %s", storageUnit)
	}

	// Build WHERE clause using primary keys
	var whereClauses []string
	var args []interface{}

	// Ensure all primary keys are provided and build WHERE clause
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

		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", pk))
		args = append(args, convertedValue)
	}

	if len(whereClauses) == 0 {
		return false, fmt.Errorf("no primary key columns specified for deletion")
	}

	// Construct the DELETE query
	query := fmt.Sprintf(`
		ALTER TABLE %s.%s
		DELETE WHERE %s`,
		schema,
		storageUnit,
		strings.Join(whereClauses, " AND "))

	// Execute the query
	_, err = conn.ExecContext(context.Background(), query, args...)
	if err != nil {
		return false, fmt.Errorf("delete failed: %w (query: %s, args: %+v)", err, query, args)
	}

	return true, nil
}
