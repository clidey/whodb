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
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

// GetColumnConstraints retrieves column constraints for SQLite tables
func (p *Sqlite3Plugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]interface{}, error) {
	constraints := make(map[string]map[string]interface{})
	
	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		// SQLite PRAGMA commands don't support placeholders, but we escape the identifier
		escapedTable := p.EscapeIdentifier(storageUnit)
		
		// Get table schema including nullability
		// Using the escaped table name to prevent SQL injection
		tableInfoQuery := fmt.Sprintf(`PRAGMA table_info(%s)`, escapedTable)
		
		rows, err := db.Raw(tableInfoQuery).Rows()
		if err != nil {
			return false, err
		}
		defer rows.Close()
		
		for rows.Next() {
			var cid int
			var name string
			var dataType string
			var notNull int
			var dfltValue interface{}
			var pk int
			
			if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
				continue
			}
			
			if constraints[name] == nil {
				constraints[name] = map[string]interface{}{}
			}
			constraints[name]["nullable"] = notNull == 0
			
			// Primary key columns are unique
			if pk == 1 {
				constraints[name]["unique"] = true
			}
		}
		
		// Get unique indexes
		// Using escaped table name
		indexListQuery := fmt.Sprintf(`PRAGMA index_list(%s)`, escapedTable)
		
		indexRows, err := db.Raw(indexListQuery).Rows()
		if err != nil {
			// Some tables might not have indexes, that's ok
			return true, nil
		}
		defer indexRows.Close()
		
		for indexRows.Next() {
			var seq int
			var name string
			var unique int
			var origin string
			var partial int
			
			if err := indexRows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
				continue
			}
			
			// Only process unique indexes
			if unique == 1 {
				// Get columns in this index
				indexInfoQuery := fmt.Sprintf(`PRAGMA index_info(%s)`, p.EscapeIdentifier(name))
				infoRows, err := db.Raw(indexInfoQuery).Rows()
				if err != nil {
					continue
				}
				
				var columnCount int
				var columnName string
				for infoRows.Next() {
					var seqno int
					var cid int
					var colName string
					if err := infoRows.Scan(&seqno, &cid, &colName); err != nil {
						continue
					}
					columnCount++
					columnName = colName
				}
				infoRows.Close()
				
				// Only mark as unique if it's a single-column index
				if columnCount == 1 && columnName != "" {
					if constraints[columnName] == nil {
						constraints[columnName] = map[string]interface{}{}
					}
					constraints[columnName]["unique"] = true
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
