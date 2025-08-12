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
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

// GetColumnConstraints retrieves column constraints for MySQL/MariaDB tables
func (p *MySQLPlugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error) {
	constraints := make(map[string]map[string]any)

	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		// Get nullability using prepared statement
		nullabilityQuery := `
			SELECT COLUMN_NAME, IS_NULLABLE 
			FROM information_schema.columns 
			WHERE table_schema = DATABASE() AND table_name = ?`

		rows, err := db.Raw(nullabilityQuery, storageUnit).Rows()
		if err != nil {
			return false, err
		}
		defer rows.Close()

		for rows.Next() {
			var columnName, isNullable string
			if err := rows.Scan(&columnName, &isNullable); err != nil {
				continue
			}

			if constraints[columnName] == nil {
				constraints[columnName] = map[string]any{}
			}
			constraints[columnName]["nullable"] = strings.EqualFold(isNullable, "YES")
		}

		// Get unique single-column indexes using prepared statement
		uniqueQuery := `
			SELECT COLUMN_NAME 
			FROM information_schema.statistics 
			WHERE table_schema = DATABASE() 
			AND table_name = ? 
			AND NON_UNIQUE = 0 
			GROUP BY COLUMN_NAME, INDEX_NAME 
			HAVING COUNT(*) = 1`

		uniqueRows, err := db.Raw(uniqueQuery, storageUnit).Rows()
		if err != nil {
			return false, err
		}
		defer uniqueRows.Close()

		for uniqueRows.Next() {
			var columnName string
			if err := uniqueRows.Scan(&columnName); err != nil {
				continue
			}

			if constraints[columnName] == nil {
				constraints[columnName] = map[string]any{}
			}
			constraints[columnName]["unique"] = true
		}

		return true, nil
	})

	if err != nil {
		return constraints, err
	}

	return constraints, nil
}
