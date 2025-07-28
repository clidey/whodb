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

package postgres

import (
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
)

func (p *PostgresPlugin) GetCreateTableQuery(schema string, storageUnit string, columns []engine.Record) string {
	var columnDefs []string
	for _, column := range columns {
		columnDef := fmt.Sprintf("%s %s", column.Key, column.Value)

		if primary, ok := column.Extra["primary"]; ok && primary == "true" {
			lowerType := strings.ToLower(column.Value)
			if strings.Contains(lowerType, "int") || strings.Contains(lowerType, "integer") {
				columnDef = fmt.Sprintf("%s %s", columnDef, "PRIMARY KEY GENERATED ALWAYS AS IDENTITY")
			} else {
				columnDef = fmt.Sprintf("%s %s", columnDef, "PRIMARY KEY")
			}
		} else {
			if nullable, ok := column.Extra["nullable"]; ok && nullable == "false" {
				columnDef = fmt.Sprintf("%s %s", columnDef, "NOT NULL")
			}
		}

		columnDefs = append(columnDefs, columnDef)
	}

	columnDefsStr := strings.Join(columnDefs, ", ")

	createTableQuery := "CREATE TABLE %s.%s (%s)"
	return fmt.Sprintf(createTableQuery, schema, storageUnit, columnDefsStr)
}
