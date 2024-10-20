package clickhouse

import (
	"context"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
)

func (p *ClickHousePlugin) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, fields map[string]string) (bool, error) {
	conn, err := DB(config)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	var columns []string
	for field, fieldType := range fields {
		columns = append(columns, fmt.Sprintf("%s %s", field, fieldType))
	}

	query := fmt.Sprintf("CREATE TABLE %s.%s (%s) ENGINE = MergeTree() ORDER BY tuple()",
		schema, storageUnit, strings.Join(columns, ", "))

	err = conn.Exec(context.Background(), query)
	return err == nil, err
}

func (p *ClickHousePlugin) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (bool, error) {
	conn, err := DB(config)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	var columns []string
	var placeholders []string
	var args []interface{}

	for _, value := range values {
		columns = append(columns, value.Key)
		placeholders = append(placeholders, "?")
		args = append(args, value.Value)
	}

	query := fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES (%s)",
		schema, storageUnit, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	err = conn.Exec(context.Background(), query, args...)
	return err == nil, err
}
