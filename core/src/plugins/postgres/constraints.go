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

package postgres

import (
	"regexp"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	"gorm.io/gorm"
)

// Pre-compiled regex for PostgreSQL type casts like ::text, ::character varying
var pgTypeCastPattern = regexp.MustCompile(`::\w+(\s+\w+)?`)

// GetColumnConstraints retrieves column constraints for PostgreSQL tables
func (p *PostgresPlugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error) {
	constraints := make(map[string]map[string]any)

	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		// Get primary keys using Postgres system catalogs
		fullTableName := schema + "." + storageUnit
		primaryRows, err := db.Raw(`
			SELECT a.attname AS column_name
			FROM pg_index i
			JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
			WHERE i.indrelid = $1::regclass AND i.indisprimary
		`, fullTableName).Rows()
		if err == nil {
			defer primaryRows.Close()
			for primaryRows.Next() {
				var columnName string
				if err := primaryRows.Scan(&columnName); err != nil {
					continue
				}
				gorm_plugin.EnsureConstraintEntry(constraints, columnName)["primary"] = true
			}
		}

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

			gorm_plugin.EnsureConstraintEntry(constraints, columnName)["nullable"] = strings.EqualFold(isNullable, "YES")
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

			gorm_plugin.EnsureConstraintEntry(constraints, columnName)["unique"] = true
		}

		// Get CHECK constraints using GORM's query builder
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

		// Get ENUM type values for columns that use native PostgreSQL ENUMs
		enumRows, err := db.Raw(`
			SELECT c.column_name, e.enumlabel
			FROM information_schema.columns c
			JOIN pg_type t ON t.typname = c.udt_name
			JOIN pg_enum e ON e.enumtypid = t.oid
			WHERE c.table_schema = $1 AND c.table_name = $2
			ORDER BY c.column_name, e.enumsortorder
		`, schema, storageUnit).Rows()
		if err == nil {
			defer enumRows.Close()

			// Group enum values by column name
			enumValues := make(map[string][]string)
			for enumRows.Next() {
				var columnName, enumLabel string
				if err := enumRows.Scan(&columnName, &enumLabel); err != nil {
					continue
				}
				enumValues[columnName] = append(enumValues[columnName], enumLabel)
			}

			// Add enum values to constraints
			for columnName, values := range enumValues {
				gorm_plugin.EnsureConstraintEntry(constraints, columnName)["check_values"] = values
			}
		}
		// Ignore error if query fails (table may not have any ENUM columns)

		return true, nil
	})

	if err != nil {
		return constraints, err
	}

	return constraints, nil
}

// parseCheckConstraint parses PostgreSQL CHECK constraint clauses to extract column constraints
func (p *PostgresPlugin) parseCheckConstraint(checkClause string, constraints map[string]map[string]any) {
	// PostgreSQL formats CHECK constraints like:
	// - CHECK ((price >= (0)::numeric))
	// - CHECK ((stock_quantity >= 0))
	// - CHECK ((age >= 18) AND (age <= 120))
	// - CHECK ((status)::text = ANY (ARRAY['active'::text, 'inactive'::text]))
	// - CHECK (status IN ('pending', 'completed', 'canceled'))

	log.Logger.WithField("checkClause", checkClause).Debug("Parsing CHECK constraint")

	// Remove CHECK keyword and outer parentheses
	clause := strings.TrimPrefix(checkClause, "CHECK ")
	clause = strings.Trim(clause, "()")

	columnName := gorm_plugin.ExtractColumnNameFromClause(clause)
	if columnName == "" {
		return
	}

	if !common.ValidateColumnName(columnName) {
		return
	}

	colConstraints := gorm_plugin.EnsureConstraintEntry(constraints, columnName)

	minMax := gorm_plugin.ParseMinMaxConstraints(clause)
	gorm_plugin.ApplyMinMaxToConstraints(colConstraints, minMax)

	// Try to extract values from ARRAY[...] syntax
	// This handles any PostgreSQL format: ANY(ARRAY[...]), ANY((ARRAY[...])::text[]), etc.
	if values := extractArrayValues(clause); len(values) > 0 {
		colConstraints["check_values"] = values
		log.Logger.WithField("column", columnName).WithField("values", values).Debug("Extracted check_values via ARRAY pattern")
		return
	}

	// Fallback to IN clause parsing
	if values := gorm_plugin.ParseINClauseValues(clause); len(values) > 0 {
		colConstraints["check_values"] = values
		log.Logger.WithField("column", columnName).WithField("values", values).Debug("Extracted check_values via IN clause pattern")
	} else {
		log.Logger.WithField("column", columnName).WithField("clause", clause).Debug("No check_values extracted from clause")
	}
}

// extractArrayValues extracts values from PostgreSQL ARRAY[...] syntax.
// it just finds ARRAY[ and extracts values up to the matching ].
func extractArrayValues(clause string) []string {
	upper := strings.ToUpper(clause)
	arrayIdx := strings.Index(upper, "ARRAY[")
	if arrayIdx == -1 {
		return nil
	}

	// Find the start of values (after "ARRAY[")
	startIdx := arrayIdx + 6

	// Find matching closing bracket, accounting for nested brackets
	depth := 1
	endIdx := -1
	for i := startIdx; i < len(clause); i++ {
		switch clause[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				endIdx = i
			}
		}
		if endIdx != -1 {
			break
		}
	}

	if endIdx == -1 || endIdx <= startIdx {
		return nil
	}

	content := clause[startIdx:endIdx]

	var values []string
	parts := strings.Split(content, ",")
	for _, part := range parts {
		cleaned := strings.TrimSpace(part)
		// Remove PostgreSQL type casts like ::text, ::character varying
		cleaned = pgTypeCastPattern.ReplaceAllString(cleaned, "")
		cleaned = strings.Trim(cleaned, "'\"")
		if cleaned != "" {
			values = append(values, cleaned)
		}
	}

	return values
}
