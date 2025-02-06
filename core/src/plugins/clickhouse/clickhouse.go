package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/clidey/whodb/core/src/engine"
)

type ClickHousePlugin struct{}

func (p *ClickHousePlugin) IsAvailable(config *engine.PluginConfig) bool {
	conn, err := DB(config)
	if err != nil {
		return false
	}
	defer conn.Close()
	return conn.PingContext(context.Background()) == nil
}

func (p *ClickHousePlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	conn, err := DB(config)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	rows, err := conn.QueryContext(context.Background(), "SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			return nil, err
		}
		databases = append(databases, dbName)
	}

	return databases, nil
}

func (p *ClickHousePlugin) GetSchema(config *engine.PluginConfig) ([]string, error) {
	return []string{config.Credentials.Database}, nil
}

func (p *ClickHousePlugin) GetStorageUnits(config *engine.PluginConfig, schema string) ([]engine.StorageUnit, error) {
	conn, err := DB(config)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	query := fmt.Sprintf(`
		SELECT 
			name,
			engine,
			total_rows,
			formatReadableSize(total_bytes) as total_size
		FROM system.tables 
		WHERE database = '%s'
	`, schema)

	rows, err := conn.QueryContext(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var storageUnits []engine.StorageUnit
	for rows.Next() {
		var name, tableType string
		var totalRows uint64
		var totalSize string
		if err := rows.Scan(&name, &tableType, &totalRows, &totalSize); err != nil {
			return nil, err
		}

		attributes := []engine.Record{
			{Key: "Table Type", Value: tableType},
			{Key: "Total Size", Value: totalSize},
			{Key: "Count", Value: strconv.FormatUint(totalRows, 10)},
		}

		columns, err := getTableSchema(conn, schema, name)
		if err != nil {
			return nil, err
		}
		attributes = append(attributes, columns...)

		storageUnits = append(storageUnits, engine.StorageUnit{
			Name:       name,
			Attributes: attributes,
		})
	}

	return storageUnits, nil
}

func getAllTableSchema(conn *sql.DB, schema string) (map[string][]engine.Record, error) {
	query := fmt.Sprintf(`
		SELECT 
			table,
			name,
			type
		FROM system.columns
		WHERE database = '%s'
		ORDER BY table, position
	`, schema)

	rows, err := conn.QueryContext(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tableColumnsMap := make(map[string][]engine.Record)
	for rows.Next() {
		var tableName, columnName, dataType string
		if err := rows.Scan(&tableName, &columnName, &dataType); err != nil {
			return nil, err
		}
		tableColumnsMap[tableName] = append(tableColumnsMap[tableName], engine.Record{Key: columnName, Value: dataType})
	}

	return tableColumnsMap, nil
}

func getTableSchema(conn *sql.DB, schema string, tableName string) ([]engine.Record, error) {
	query := fmt.Sprintf(`
		SELECT 
			name,
			type
		FROM system.columns
		WHERE database = '%s' AND table = '%s'
		ORDER BY position
	`, schema, tableName)

	rows, err := conn.QueryContext(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []engine.Record
	for rows.Next() {
		var name, dataType string
		if err := rows.Scan(&name, &dataType); err != nil {
			return nil, err
		}
		result = append(result, engine.Record{Key: name, Value: dataType})
	}

	return result, nil
}

func NewClickHousePlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_ClickHouse,
		PluginFunctions: &ClickHousePlugin{},
	}
}
