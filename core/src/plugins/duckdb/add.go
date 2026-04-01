/*
 * Copyright 2026 Clidey, Inc.
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

package duckdb

import (
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	"gorm.io/gorm"
)

// GetCreateTableQuery builds a CREATE TABLE statement for DuckDB.
// For integer PK columns, it creates a sequence and uses DEFAULT nextval().
func (p *DuckDBPlugin) GetCreateTableQuery(db *gorm.DB, schema string, storageUnit string, columns []engine.Record) string {
	builder := gorm_plugin.NewSQLBuilder(db, p)

	// Determine if we need a sequence for an integer PK
	var seqSQL string
	columnDefs := gorm_plugin.RecordsToColumnDefs(columns, func(def gorm_plugin.ColumnDef, column engine.Record) gorm_plugin.ColumnDef {
		lowerType := strings.ToLower(column.Value)
		if strings.Contains(lowerType, "int") || strings.Contains(lowerType, "integer") {
			seqName := fmt.Sprintf("%s_%s_seq", storageUnit, column.Key)
			seqSQL = fmt.Sprintf("CREATE SEQUENCE IF NOT EXISTS %s; ", builder.QuoteIdentifier(seqName))
			def.Extra = fmt.Sprintf("PRIMARY KEY DEFAULT nextval('%s')", strings.ReplaceAll(seqName, "'", "''"))
		} else {
			def.Primary = true
		}
		return def
	})

	return seqSQL + builder.CreateTableQuery(schema, storageUnit, columnDefs)
}
