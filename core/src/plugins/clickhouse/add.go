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
	"errors"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	"gorm.io/gorm"
)

// ErrUpsertNotSupported is returned when upsert mode is attempted on ClickHouse.
var ErrUpsertNotSupported = errors.New("import.error.upsert_not_supported")

func (p *ClickHousePlugin) GetCreateTableQuery(db *gorm.DB, schema string, storageUnit string, columns []engine.Record) string {
	builder := gorm_plugin.NewSQLBuilder(db, p)

	// ClickHouse doesn't decorate primary columns in the column definition;
	// instead it uses them for the ORDER BY clause.
	var primaryKeys []string
	columnDefs := gorm_plugin.RecordsToColumnDefs(columns, func(def gorm_plugin.ColumnDef, column engine.Record) gorm_plugin.ColumnDef {
		primaryKeys = append(primaryKeys, column.Key)
		// ClickHouse primary keys still respect nullable constraints
		if nullable, ok := column.Extra["nullable"]; ok && nullable == "false" {
			def.NotNull = true
		}
		return def
	})

	// Determine ORDER BY clause
	orderByClause := ""
	if len(primaryKeys) > 0 {
		quotedKeys := make([]string, len(primaryKeys))
		for i, key := range primaryKeys {
			quotedKeys[i] = builder.QuoteIdentifier(key)
		}
		orderByClause = strings.Join(quotedKeys, ", ")
	} else if len(columns) > 0 {
		orderByClause = builder.QuoteIdentifier(columns[0].Key)
	}

	suffix := fmt.Sprintf("ENGINE = MergeTree() ORDER BY (%s)", orderByClause)
	return builder.CreateTableQueryWithSuffix(schema, storageUnit, columnDefs, suffix)
}

// BulkAddRows rejects upsert mode (ClickHouse has no ON CONFLICT support) and
// delegates all other modes to the base GormPlugin implementation.
func (p *ClickHousePlugin) BulkAddRows(config *engine.PluginConfig, schema string, storageUnit string, rows [][]engine.Record) (bool, error) {
	if len(config.UpsertPKColumns) > 0 {
		return false, ErrUpsertNotSupported
	}
	return p.GormPlugin.BulkAddRows(config, schema, storageUnit, rows)
}
