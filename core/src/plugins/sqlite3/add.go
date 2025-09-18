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
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins/gorm"
	"gorm.io/gorm"
)

func (p *Sqlite3Plugin) GetCreateTableQuery(db *gorm.DB, schema string, storageUnit string, columns []engine.Record) string {
	builder := gorm_plugin.NewSQLBuilder(db, p)

	// Convert engine.Record to ColumnDef
	columnDefs := make([]gorm_plugin.ColumnDef, len(columns))
	for i, column := range columns {
		def := gorm_plugin.ColumnDef{
			Name: column.Key,
		}

		// Handle primary key with INTEGER type for auto-increment
		if primary, ok := column.Extra["primary"]; ok && primary == "true" {
			if strings.Contains(strings.ToLower(column.Value), "int") {
				def.Type = "INTEGER"
				def.Primary = true
			} else {
				def.Type = column.Value
				def.Primary = true
			}
		} else {
			def.Type = column.Value

			// Add NOT NULL constraint if specified
			if nullable, ok := column.Extra["nullable"]; ok && nullable == "false" {
				def.NotNull = true
			}
		}

		columnDefs[i] = def
	}

	// SQLite doesn't use schema, only table name
	return builder.CreateTableQuery("", storageUnit, columnDefs)
}
