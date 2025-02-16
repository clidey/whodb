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
