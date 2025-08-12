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
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

// GetColumnConstraints retrieves column constraints for PostgreSQL tables
func (p *PostgresPlugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]interface{}, error) {
	constraints := make(map[string]map[string]interface{})
	
	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		// Get nullability using prepared statement
		nullabilityQuery := `
			SELECT column_name, is_nullable 
			FROM information_schema.columns 
			WHERE table_schema = ? AND table_name = ?`
		
		rows, err := db.Raw(nullabilityQuery, schema, storageUnit).Rows()
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
				constraints[columnName] = map[string]interface{}{}
			}
			constraints[columnName]["nullable"] = strings.EqualFold(isNullable, "YES")
		}
		
		// Get unique single-column indexes using prepared statement
		uniqueQuery := `
			SELECT a.attname AS column_name 
			FROM pg_index i 
			JOIN pg_class c ON c.oid = i.indrelid 
			JOIN pg_namespace n ON n.oid = c.relnamespace 
			JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(i.indkey) 
			WHERE c.relname = ? 
			AND n.nspname = ? 
			AND i.indisunique = true 
			AND i.indnkeyatts = 1`
		
		uniqueRows, err := db.Raw(uniqueQuery, storageUnit, schema).Rows()
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
				constraints[columnName] = map[string]interface{}{}
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