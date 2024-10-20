package clickhouse

import (
	"context"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
)

func (p *ClickHousePlugin) DeleteRow(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
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

	// Prepare the DELETE query
	var whereClauses []string
	var args []interface{}

	for column, value := range values {
		if isPrimaryKey(column, primaryKeys) {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", column))
			args = append(args, value)
		}
	}

	if len(whereClauses) == 0 {
		return false, fmt.Errorf("no primary key columns specified for deletion")
	}

	// Construct the final query
	query := fmt.Sprintf(`
		ALTER TABLE %s.%s
		DELETE WHERE %s
	`,
		schema,
		storageUnit,
		strings.Join(whereClauses, " AND "),
	)

	// Execute the query
	err = conn.Exec(context.Background(), query, args...)
	return err == nil, err
}
