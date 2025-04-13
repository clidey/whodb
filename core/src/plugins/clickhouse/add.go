/*
 * Copyright 2025 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package clickhouse

import (
	"fmt"
	"github.com/clidey/whodb/core/src/engine"
	"strings"
)

func (p *ClickHousePlugin) GetCreateTableQuery(schema string, storageUnit string, columns []engine.Record) string {
	var columnDefs []string
	var primaryKeys []string

	for _, column := range columns {
		parts := []string{column.Key, column.Value}

		if nullable, ok := column.Extra["nullable"]; ok && nullable == "false" {
			parts = append(parts, "NOT NULL")
		}

		columnDefs = append(columnDefs, strings.Join(parts, " "))

		if primary, ok := column.Extra["primary"]; ok && primary == "true" {
			primaryKeys = append(primaryKeys, fmt.Sprintf("%s", column.Key))
		}
	}

	// Determine ORDER BY clause
	orderByClause := ""
	if len(primaryKeys) > 0 {
		orderByClause = strings.Join(primaryKeys, ", ")
	} else if len(columnDefs) > 0 {
		firstColParts := strings.SplitN(columnDefs[0], " ", 2)
		if len(firstColParts) > 0 {
			orderByClause = firstColParts[0]
		}
	}

	return fmt.Sprintf(`
		CREATE TABLE %s.%s 
		(%s) 
		ENGINE = MergeTree()
		ORDER BY (%s)`, schema, storageUnit, strings.Join(columnDefs, ", "), orderByClause)
}
