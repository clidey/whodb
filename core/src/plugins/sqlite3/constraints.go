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

package sqlite3

import (
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	"gorm.io/gorm"
)

// GetColumnConstraints retrieves column constraints for SQLite tables
func (p *Sqlite3Plugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error) {
	constraints := make(map[string]map[string]any)

	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		// Use SQLite-specific SQL builder.
		builder, ok := p.CreateSQLBuilder(db).(*SQLiteSQLBuilder)
		if !ok {
			return false, fmt.Errorf("failed to create SQLite SQL builder")
		}

		// Get table schema including nullability
		// SQLite PRAGMA commands don't support placeholders
		tableInfoQuery, err := builder.PragmaQuery("table_info", storageUnit)
		if err != nil {
			return false, err
		}

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
			var dfltValue any
			var pk int

			if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
				continue
			}

			if constraints[name] == nil {
				constraints[name] = map[string]any{}
			}
			constraints[name]["nullable"] = notNull == 0

			// Primary key columns
			if pk == 1 {
				constraints[name]["primary"] = true
				constraints[name]["unique"] = true
			}
		}

		// Get unique indexes
		indexListQuery, err := builder.PragmaQuery("index_list", storageUnit)
		if err != nil {
			// This is not a critical error? table might not have indexes.
			return true, nil
		}

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
				indexInfoQuery, err := builder.PragmaQuery("index_info", name)
				if err != nil {
					return false, err
				}
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
						constraints[columnName] = map[string]any{}
					}
					constraints[columnName]["unique"] = true
				}
			}
		}

		// Get CHECK constraints from sqlite_master
		// SQLite stores CHECK constraints in the CREATE TABLE statement
		checkQuery := `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?`
		var createSQL string
		err = db.Raw(checkQuery, storageUnit).Row().Scan(&createSQL)
		if err == nil && createSQL != "" {
			// Parse CHECK constraints from the CREATE TABLE statement
			p.parseCheckConstraints(createSQL, constraints)
		}

		return true, nil
	})

	if err != nil {
		return constraints, err
	}

	return constraints, nil
}

// parseCheckConstraints extracts CHECK constraints from SQLite's CREATE TABLE statement
func (p *Sqlite3Plugin) parseCheckConstraints(createSQL string, constraints map[string]map[string]any) {
	// SQLite stores CHECK constraints in the CREATE TABLE statement like:
	// CREATE TABLE products (
	//   price DECIMAL(10,2) NOT NULL CHECK (price >= 0),
	//   stock_quantity INT CHECK (stock_quantity >= 0),
	//   status TEXT CHECK (status IN ('active', 'inactive'))
	// )

	upper := strings.ToUpper(createSQL)
	searchStart := 0

	for {
		checkIdx := strings.Index(upper[searchStart:], "CHECK")
		if checkIdx == -1 {
			break
		}
		checkIdx += searchStart

		parenStart := strings.Index(createSQL[checkIdx:], "(")
		if parenStart == -1 {
			searchStart = checkIdx + 5
			continue
		}
		parenStart += checkIdx

		depth := 1
		parenEnd := -1
		for i := parenStart + 1; i < len(createSQL); i++ {
			switch createSQL[i] {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					parenEnd = i
				}
			}
			if parenEnd != -1 {
				break
			}
		}

		if parenEnd == -1 {
			searchStart = checkIdx + 5
			continue
		}

		checkClause := createSQL[parenStart+1 : parenEnd]
		p.parseSingleCheckConstraint(checkClause, constraints)

		searchStart = parenEnd + 1
	}
}

// parseSingleCheckConstraint parses a single CHECK constraint clause
func (p *Sqlite3Plugin) parseSingleCheckConstraint(checkClause string, constraints map[string]map[string]any) {
	columnName := gorm_plugin.ExtractColumnNameFromClause(checkClause)
	if columnName == "" {
		log.WithField("checkClause", checkClause).Debug("Could not extract column name from CHECK clause")
		return
	}

	// SQLite column names are case-insensitive. To ensure we merge constraints
	// with those from PRAGMA table_info (which may use a different case),
	// do a case-insensitive lookup first.
	var existingKey string
	lowerName := strings.ToLower(columnName)
	for key := range constraints {
		if strings.ToLower(key) == lowerName {
			existingKey = key
			break
		}
	}
	if existingKey != "" {
		columnName = existingKey // Use the existing key's case
	}

	colConstraints := gorm_plugin.EnsureConstraintEntry(constraints, columnName)

	minMax := gorm_plugin.ParseMinMaxConstraints(checkClause)
	gorm_plugin.ApplyMinMaxToConstraints(colConstraints, minMax)

	if values := gorm_plugin.ParseINClauseValues(checkClause); len(values) > 0 {
		colConstraints["check_values"] = values
		log.WithFields(map[string]any{
			"column":      columnName,
			"checkValues": values,
		}).Debug("Parsed CHECK IN constraint")
	}
}
