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
	"regexp"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

// GetColumnConstraints retrieves column constraints for PostgreSQL tables
func (p *PostgresPlugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]interface{}, error) {
	constraints := make(map[string]map[string]interface{})

	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		// Get nullability using GORM's query builder
		rows, err := db.Table("information_schema.columns").
			Select("column_name, is_nullable").
			Where("table_schema = ? AND table_name = ?", schema, storageUnit).
			Rows()
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

		// Get unique single-column indexes using GORM's query builder
		uniqueRows, err := db.Table("pg_index i").
			Select("a.attname AS column_name").
			Joins("JOIN pg_class c ON c.oid = i.indrelid").
			Joins("JOIN pg_namespace n ON n.oid = c.relnamespace").
			Joins("JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(i.indkey)").
			Where("c.relname = ? AND n.nspname = ? AND i.indisunique = true AND i.indnkeyatts = 1", storageUnit, schema).
			Rows()
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

		// Get CHECK constraints using GORM's query builder
		fullTableName := schema + "." + storageUnit
		checkRows, err := db.Table("pg_constraint").
			Select("conname AS constraint_name, pg_get_constraintdef(oid) AS check_clause").
			Where("contype = 'c' AND conrelid = ?::regclass", fullTableName).
			Rows()
		if err == nil {
			defer checkRows.Close()

			for checkRows.Next() {
				var constraintName, checkClause string
				if err := checkRows.Scan(&constraintName, &checkClause); err != nil {
					continue
				}

				// Parse the CHECK clause to extract column and condition
				p.parseCheckConstraint(checkClause, constraints)
			}
		}
		// Ignore error if query fails

		return true, nil
	})

	if err != nil {
		return constraints, err
	}

	return constraints, nil
}

// parseCheckConstraint parses PostgreSQL CHECK constraint clauses to extract column constraints
func (p *PostgresPlugin) parseCheckConstraint(checkClause string, constraints map[string]map[string]interface{}) {
	// PostgreSQL formats CHECK constraints like:
	// - CHECK ((price >= (0)::numeric))
	// - CHECK ((stock_quantity >= 0))
	// - CHECK ((age >= 18) AND (age <= 120))
	// - CHECK ((status)::text = ANY (ARRAY['active'::text, 'inactive'::text]))

	// Remove CHECK keyword and outer parentheses
	clause := strings.TrimPrefix(checkClause, "CHECK ")
	clause = strings.Trim(clause, "()")

	// Pattern for >= or > constraints
	minPattern := regexp.MustCompile(`\(?(\w+)\)?\s*>=?\s*\(?([\-]?\d+(?:\.\d+)?)\)?`)
	if matches := minPattern.FindStringSubmatch(clause); len(matches) > 2 {
		columnName := matches[1]
		// Validate column name to prevent SQL injection
		if !common.ValidateColumnName(columnName) {
			return
		}
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
	maxPattern := regexp.MustCompile(`\(?(\w+)\)?\s*<=?\s*\(?([\-]?\d+(?:\.\d+)?)\)?`)
	if matches := maxPattern.FindStringSubmatch(clause); len(matches) > 2 {
		columnName := matches[1]
		// Validate column name to prevent SQL injection
		if !common.ValidateColumnName(columnName) {
			return
		}
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

	// Pattern for ANY (ARRAY[...]) constraints (PostgreSQL's way of doing IN)
	anyArrayPattern := regexp.MustCompile(`\(?(\w+)\)?.*?ANY\s*\(ARRAY\[(.*?)\]`)
	if matches := anyArrayPattern.FindStringSubmatch(clause); len(matches) > 2 {
		columnName := matches[1]
		// Validate column name to prevent SQL injection
		if !common.ValidateColumnName(columnName) {
			return
		}
		if constraints[columnName] == nil {
			constraints[columnName] = map[string]interface{}{}
		}
		// Extract values from ARRAY
		valuesStr := matches[2]
		var values []string
		// Split by comma and clean up
		parts := strings.Split(valuesStr, ",")
		for _, part := range parts {
			cleaned := strings.TrimSpace(part)
			// Remove ::text or other type casts
			cleaned = regexp.MustCompile(`::\w+`).ReplaceAllString(cleaned, "")
			cleaned = strings.Trim(cleaned, "'\"")
			if cleaned != "" {
				if sanitized, ok := common.SanitizeConstraintValue(cleaned); ok {
					values = append(values, sanitized)
				}
			}
		}
		if len(values) > 0 {
			constraints[columnName]["check_values"] = values
		}
	}
}
