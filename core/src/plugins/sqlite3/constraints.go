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
	"regexp"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

// GetColumnConstraints retrieves column constraints for SQLite tables
func (p *Sqlite3Plugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]interface{}, error) {
	constraints := make(map[string]map[string]interface{})

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
						constraints[columnName] = map[string]interface{}{}
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
func (p *Sqlite3Plugin) parseCheckConstraints(createSQL string, constraints map[string]map[string]interface{}) {
	// SQLite stores CHECK constraints in the CREATE TABLE statement like:
	// CREATE TABLE products (
	//   price DECIMAL(10,2) NOT NULL CHECK (price >= 0),
	//   stock_quantity INT CHECK (stock_quantity >= 0),
	//   status TEXT CHECK (status IN ('active', 'inactive'))
	// )

	// Find all CHECK clauses
	checkPattern := regexp.MustCompile(`CHECK\s*\((.*?)\)(?:,|\s|$)`)
	checkMatches := checkPattern.FindAllStringSubmatch(createSQL, -1)

	for _, match := range checkMatches {
		if len(match) > 1 {
			checkClause := match[1]
			p.parseSingleCheckConstraint(checkClause, constraints)
		}
	}
}

// parseSingleCheckConstraint parses a single CHECK constraint clause
func (p *Sqlite3Plugin) parseSingleCheckConstraint(checkClause string, constraints map[string]map[string]interface{}) {
	// Pattern for >= or > constraints
	minPattern := regexp.MustCompile(`(\w+)\s*>=?\s*([\-]?\d+(?:\.\d+)?)`)
	if matches := minPattern.FindStringSubmatch(checkClause); len(matches) > 2 {
		columnName := matches[1]
		if constraints[columnName] == nil {
			constraints[columnName] = map[string]interface{}{}
		}
		if val, err := strconv.ParseFloat(matches[2], 64); err == nil {
			if strings.Contains(matches[0], ">=") {
				constraints[columnName]["check_min"] = val
			} else {
				constraints[columnName]["check_min"] = val + 1
			}
		}
	}

	// Pattern for <= or < constraints
	maxPattern := regexp.MustCompile(`(\w+)\s*<=?\s*([\-]?\d+(?:\.\d+)?)`)
	if matches := maxPattern.FindStringSubmatch(checkClause); len(matches) > 2 {
		columnName := matches[1]
		if constraints[columnName] == nil {
			constraints[columnName] = map[string]interface{}{}
		}
		if val, err := strconv.ParseFloat(matches[2], 64); err == nil {
			if strings.Contains(matches[0], "<=") {
				constraints[columnName]["check_max"] = val
			} else {
				constraints[columnName]["check_max"] = val - 1
			}
		}
	}

	// Pattern for BETWEEN constraints
	betweenPattern := regexp.MustCompile(`(\w+)\s+BETWEEN\s+([\-]?\d+(?:\.\d+)?)\s+AND\s+([\-]?\d+(?:\.\d+)?)`)
	if matches := betweenPattern.FindStringSubmatch(strings.ToUpper(checkClause)); len(matches) > 3 {
		// Get original column name (case-sensitive)
		origPattern := regexp.MustCompile(`(\w+)\s+(?i)between`)
		origMatches := origPattern.FindStringSubmatch(checkClause)
		if len(origMatches) > 1 {
			columnName := origMatches[1]
			if constraints[columnName] == nil {
				constraints[columnName] = map[string]interface{}{}
			}
			if minVal, err := strconv.ParseFloat(matches[2], 64); err == nil {
				constraints[columnName]["check_min"] = minVal
			}
			if maxVal, err := strconv.ParseFloat(matches[3], 64); err == nil {
				constraints[columnName]["check_max"] = maxVal
			}
		}
	}

	// Pattern for IN constraints
	inPattern := regexp.MustCompile(`(\w+)\s+IN\s*\((.*?)\)`)
	if matches := inPattern.FindStringSubmatch(strings.ToUpper(checkClause)); len(matches) > 2 {
		// Get original column name (case-sensitive)
		origPattern := regexp.MustCompile(`(\w+)\s+(?i)in\s*\(`)
		origMatches := origPattern.FindStringSubmatch(checkClause)
		if len(origMatches) > 1 {
			columnName := origMatches[1]
			if constraints[columnName] == nil {
				constraints[columnName] = map[string]interface{}{}
			}
			// Extract values from IN clause
			valuesStr := matches[2]
			values := []string{}
			// Split by comma and clean up
			parts := strings.Split(valuesStr, ",")
			for _, part := range parts {
				cleaned := strings.TrimSpace(part)
				cleaned = strings.Trim(cleaned, "'\"")
				if cleaned != "" {
					values = append(values, cleaned)
				}
			}
			if len(values) > 0 {
				constraints[columnName]["check_values"] = values
			}
		}
	}
}
