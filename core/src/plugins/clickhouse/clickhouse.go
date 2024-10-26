package clickhouse

import (
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
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
	return conn.Ping(context.Background()) == nil
}

func (p *ClickHousePlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	conn, err := DB(config)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	rows, err := conn.Query(context.Background(), "SHOW DATABASES")
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

	rows, err := conn.Query(context.Background(), query)
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

func getTableSchema(conn driver.Conn, schema string, tableName string) ([]engine.Record, error) {
	query := fmt.Sprintf(`
		SELECT 
			name,
			type
		FROM system.columns
		WHERE database = '%s' AND table = '%s'
		ORDER BY position
	`, schema, tableName)

	rows, err := conn.Query(context.Background(), query)
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

func (p *ClickHousePlugin) Chat(config *engine.PluginConfig, schema string, model string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
	// Implement chat functionality similar to MySQL implementation
	// You may need to adapt this based on ClickHouse specifics
	return nil, fmt.Errorf("chat functionality not implemented for ClickHouse")
}

func NewClickHousePlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_ClickHouse,
		PluginFunctions: &ClickHousePlugin{},
	}
}
