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

package mysql

import (
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins/gorm"
	"gorm.io/gorm"
)

func (p *MySQLPlugin) GetCreateTableQuery(db *gorm.DB, schema string, storageUnit string, columns []engine.Record) string {
	builder := p.GormPluginFunctions.CreateSQLBuilder(db)

	// Convert engine.Record to ColumnDef
	columnDefs := make([]gorm_plugin.ColumnDef, len(columns))
	for i, column := range columns {
		def := gorm_plugin.ColumnDef{
			Name: column.Key,
			Type: column.Value,
		}

		if primary, ok := column.Extra["primary"]; ok && primary == "true" {
			lowerType := strings.ToLower(column.Value)
			if strings.Contains(lowerType, "int") {
				def.Primary = true
				def.Extra = "AUTO_INCREMENT"
			} else {
				def.Primary = true
			}
		} else {
			if nullable, ok := column.Extra["nullable"]; ok && nullable == "false" {
				def.NotNull = true
			}
		}

		columnDefs[i] = def
	}

	// MySQL/MariaDB syntax uses database.table rather than schema.table
	return builder.CreateTableQuery(schema, storageUnit, columnDefs)
}
