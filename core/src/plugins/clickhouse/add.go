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

	// Extract engine settings from advanced configuration
	var engineSettings struct {
		engine      string
		orderBy     string
		partitionBy string
		settings    map[string]string
	}

	engineSettings.engine = "MergeTree" // default engine
	engineSettings.orderBy = "tuple()"  // default order
	engineSettings.settings = make(map[string]string)

	for _, record := range config.Credentials.Advanced {
		switch record.Key {
		case "Engine":
			engineSettings.engine = record.Value
		case "OrderBy":
			engineSettings.orderBy = record.Value
		case "PartitionBy":
			engineSettings.partitionBy = record.Value
		default:
			if strings.HasPrefix(record.Key, "Setting_") {
				key := strings.TrimPrefix(record.Key, "Setting_")
				engineSettings.settings[key] = record.Value
			}
		}
	}

	// Prepare columns
	var columns []string
	for field, fieldType := range fields {
		columns = append(columns, fmt.Sprintf("%s %s", field, fieldType))
	}

	// Build the CREATE TABLE query
	query := fmt.Sprintf("CREATE TABLE %s.%s (\n\t%s\n) ENGINE = %s",
		schema, storageUnit, strings.Join(columns, ",\n\t"), engineSettings.engine)

	// Add ORDER BY clause
	if engineSettings.orderBy != "" {
		query += fmt.Sprintf("\nORDER BY %s", engineSettings.orderBy)
	}

	// Add PARTITION BY clause if specified
	if engineSettings.partitionBy != "" {
		query += fmt.Sprintf("\nPARTITION BY %s", engineSettings.partitionBy)
	}

	// Add engine settings if any
	if len(engineSettings.settings) > 0 {
		var settingsClauses []string
		for key, value := range engineSettings.settings {
			settingsClauses = append(settingsClauses, fmt.Sprintf("%s=%s", key, value))
		}
		query += fmt.Sprintf("\nSETTINGS %s", strings.Join(settingsClauses, ", "))
	}

	_, err = conn.ExecContext(context.Background(), query)
	if err != nil {
		return false, fmt.Errorf("failed to create table: %w (query: %s)", err, query)
	}

	return true, nil
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

	_, err = conn.ExecContext(context.Background(), query, args...)
	return err == nil, err
}
