package clickhouse

import (
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
)

func (p *ClickHousePlugin) UpdateStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	conn, err := DB(config)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	// Get the primary key columns
	primaryKeys, err := getPrimaryKeyColumns(conn, schema, storageUnit)
	if err != nil {
		return false, err
	}

	// Prepare the UPDATE query
	var updateClauses []string
	var whereClauses []string
	var args []interface{}

	for column, value := range values {
		if isPrimaryKey(column, primaryKeys) {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", column))
		} else {
			updateClauses = append(updateClauses, fmt.Sprintf("%s = ?", column))
		}
		args = append(args, value)
	}

	if len(updateClauses) == 0 {
		return false, fmt.Errorf("no columns to update")
	}

	// Construct the final query
	query := fmt.Sprintf("ALTER TABLE %s.%s UPDATE %s WHERE %s",
		schema,
		storageUnit,
		strings.Join(updateClauses, ", "),
		strings.Join(whereClauses, " AND "),
	)

	// Execute the query
	err = conn.Exec(context.Background(), query, args...)
	return err == nil, err
}

func getPrimaryKeyColumns(conn driver.Conn, schema, table string) ([]string, error) {
	query := `
		SELECT name
		FROM system.columns
		WHERE database = ? AND table = ? AND is_in_primary_key = 1
	`

	rows, err := conn.Query(context.Background(), query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var primaryKeys []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return nil, err
		}
		primaryKeys = append(primaryKeys, column)
	}

	return primaryKeys, nil
}

func isPrimaryKey(column string, primaryKeys []string) bool {
	for _, pk := range primaryKeys {
		if column == pk {
			return true
		}
	}
	return false
}
