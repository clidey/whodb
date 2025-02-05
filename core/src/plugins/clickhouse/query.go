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
	"github.com/clidey/whodb/core/src/engine"
)

func (p *ClickHousePlugin) GetRows(config *engine.PluginConfig, schema string, storageUnit string, where string, pageSize int, pageOffset int) (*engine.GetRowsResult, error) {
	query := fmt.Sprintf("SELECT * FROM %s.%s", schema, storageUnit)
	if where != "" {
		query += " WHERE " + where
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, pageOffset)

	return p.executeQuery(config, query)
}

func (p *ClickHousePlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return p.executeQuery(config, query)
}

func (p *ClickHousePlugin) executeQuery(config *engine.PluginConfig, query string, params ...interface{}) (*engine.GetRowsResult, error) {
	conn, err := DB(config)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	rows, err := conn.QueryContext(context.Background(), query, params)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	result := &engine.GetRowsResult{
		Columns: make([]engine.Column, len(columnTypes)),
		Rows:    [][]string{},
	}

	for i, ct := range columnTypes {
		result.Columns[i] = engine.Column{
			Name: ct.Name(),
			Type: ct.DatabaseTypeName(),
		}
	}

	for rows.Next() {
		// Create scan destinations based on column types
		scanDest := make([]sql.NullString, len(columnTypes))
		scanArgs := make([]interface{}, len(columnTypes))
		for i := range scanDest {
			scanArgs[i] = &scanDest[i]
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}

		// Convert to strings
		row := make([]string, len(columnTypes))
		for i := range scanDest {
			if scanDest[i].Valid {
				row[i] = scanDest[i].String
			} else {
				row[i] = ""
			}
		}
		result.Rows = append(result.Rows, row)
	}

	return result, nil
}
