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

	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/engine"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
)

func (p *MySQLPlugin) GetCreateTableQuery(db *gorm.DB, schema string, storageUnit string, columns []engine.Record) string {
	builder := p.GormPluginFunctions.CreateSQLBuilder(db)

	columnDefs := gorm_plugin.RecordsToColumnDefs(columns, func(def gorm_plugin.ColumnDef, column engine.Record) gorm_plugin.ColumnDef {
		extra := engine.NormalizeCreationExtra(column.Extra)
		def.Primary = extra["primary"] == "true" || extra["identity"] == "true"
		if extra["identity"] == "true" && strings.Contains(strings.ToLower(column.Value), "int") {
			def.Extra = "AUTO_INCREMENT"
		}
		return def
	})

	return builder.CreateTableQuery(schema, storageUnit, columnDefs)
}
