// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
