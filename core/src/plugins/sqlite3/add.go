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

package sqlite3

import (
	"fmt"
	"github.com/clidey/whodb/core/src/engine"
	"strings"
)

func (p *Sqlite3Plugin) GetCreateTableQuery(schema string, storageUnit string, columns []engine.Record) string {
	var columnDefs []string

	for _, column := range columns {
		parts := []string{column.Key}

		// Handle primary key with INTEGER type for auto-increment
		if primary, ok := column.Extra["primary"]; ok && primary == "true" {
			if strings.Contains(strings.ToLower(column.Value), "int") {
				parts = append(parts, "INTEGER PRIMARY KEY")
			} else {
				parts = append(parts, column.Value, "PRIMARY KEY")
			}
		} else {
			parts = append(parts, column.Value)

			// Add NOT NULL constraint if specified
			if nullable, ok := column.Extra["nullable"]; ok && nullable == "false" {
				parts = append(parts, "NOT NULL")
			}
		}

		columnDefs = append(columnDefs, strings.Join(parts, " "))
	}

	return fmt.Sprintf("CREATE TABLE %s (%s)", storageUnit, strings.Join(columnDefs, ", "))
}
