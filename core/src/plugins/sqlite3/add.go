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
		columnDef := fmt.Sprintf("%s %s", column.Key, column.Value)

		if primary, ok := column.Extra["primary"]; ok && primary == "true" {
			lowerType := strings.ToLower(column.Value)
			if strings.Contains(lowerType, "int") {
				// Convert to INTEGER type for proper primary key behavior in SQLite
				// SQLite's "INTEGER PRIMARY KEY" automatically creates an auto-incrementing column
				// without needing the explicit AUTOINCREMENT keyword
				if !strings.Contains(lowerType, "integer") {
					columnDef = fmt.Sprintf("\"%s\" INTEGER", column.Key)
				}
				columnDef = fmt.Sprintf("%s PRIMARY KEY", columnDef)
			} else {
				columnDef = fmt.Sprintf("%s PRIMARY KEY", columnDef)
			}
		} else {
			if nullable, ok := column.Extra["nullable"]; ok && nullable == "false" {
				columnDef = fmt.Sprintf("%s NOT NULL", columnDef)
			}
		}

		columnDefs = append(columnDefs, columnDef)
	}

	columnDefsStr := strings.Join(columnDefs, ", ")

	createTableQuery := "CREATE TABLE %s (%s)"
	return fmt.Sprintf(createTableQuery, storageUnit, columnDefsStr)
}
