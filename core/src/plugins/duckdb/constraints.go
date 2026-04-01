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
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	"gorm.io/gorm"
)

// GetColumnConstraints retrieves column constraints for DuckDB tables.
func (p *DuckDBPlugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error) {
	constraints := make(map[string]map[string]any)

	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		// Get nullability
		rows, err := db.Raw(`
			SELECT column_name, is_nullable
			FROM information_schema.columns
			WHERE table_schema = ? AND table_name = ?
		`, schema, storageUnit).Rows()
		if err != nil {
			return false, err
		}
		defer rows.Close()

		for rows.Next() {
			var columnName, isNullable string
			if err := rows.Scan(&columnName, &isNullable); err != nil {
				continue
			}
			gorm_plugin.EnsureConstraintEntry(constraints, columnName)["nullable"] = isNullable == "YES"
		}

		// Get primary keys and unique constraints via duckdb_constraints()
		// (information_schema.table_constraints may not be populated in all DuckDB versions)
		constraintRows, err := db.Raw(`
			SELECT unnest(dc.constraint_column_names) AS column_name, dc.constraint_type
			FROM duckdb_constraints() dc
			WHERE dc.constraint_type IN ('PRIMARY KEY', 'UNIQUE')
				AND dc.schema_name = ? AND dc.table_name = ?
		`, schema, storageUnit).Rows()
		if err == nil {
			defer constraintRows.Close()
			for constraintRows.Next() {
				var columnName, constraintType string
				if err := constraintRows.Scan(&columnName, &constraintType); err != nil {
					continue
				}
				switch constraintType {
				case "PRIMARY KEY":
					gorm_plugin.EnsureConstraintEntry(constraints, columnName)["primary"] = true
				case "UNIQUE":
					gorm_plugin.EnsureConstraintEntry(constraints, columnName)["unique"] = true
				}
			}
		}

		return true, nil
	})

	if err != nil {
		return constraints, err
	}

	return constraints, nil
}
