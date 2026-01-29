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

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	"gorm.io/gorm"
)

// GetColumnConstraints retrieves column constraints for MySQL/MariaDB tables
func (p *MySQLPlugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error) {
	constraints := make(map[string]map[string]any)

	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		// Get primary keys using information_schema
		primaryRows, err := db.Table("information_schema.table_constraints t").
			Select("k.COLUMN_NAME").
			Joins("JOIN information_schema.key_column_usage k ON k.CONSTRAINT_NAME = t.CONSTRAINT_NAME AND k.TABLE_SCHEMA = t.TABLE_SCHEMA AND k.TABLE_NAME = t.TABLE_NAME").
			Where("t.CONSTRAINT_TYPE = 'PRIMARY KEY' AND t.TABLE_SCHEMA = DATABASE() AND t.TABLE_NAME = ?", storageUnit).
			Order("k.ORDINAL_POSITION").
			Rows()
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

		// Get nullability and column type (for ENUM detection) using GORM's query builder
		rows, err := db.Table("information_schema.columns").
			Select("COLUMN_NAME, IS_NULLABLE, COLUMN_TYPE").
			Where("table_schema = DATABASE() AND table_name = ?", storageUnit).
			Rows()
		if err != nil {
			return false, err
		}
		defer rows.Close()

		for rows.Next() {
			var columnName, isNullable, columnType string
			if err := rows.Scan(&columnName, &isNullable, &columnType); err != nil {
				continue
			}

			colConstraints := gorm_plugin.EnsureConstraintEntry(constraints, columnName)
			colConstraints["nullable"] = strings.EqualFold(isNullable, "YES")

			// Parse ENUM types to extract allowed values
			// MySQL ENUM columns have column_type like: enum('active','inactive','pending')
			if strings.HasPrefix(strings.ToLower(columnType), "enum(") {
				values := parseEnumValues(columnType)
				if len(values) > 0 {
					colConstraints["check_values"] = values
				}
			}
		}

		// Get unique single-column indexes using GORM's query builder
		uniqueRows, err := db.Table("information_schema.statistics").
			Select("COLUMN_NAME").
			Where("table_schema = DATABASE() AND table_name = ? AND NON_UNIQUE = 0", storageUnit).
			Group("COLUMN_NAME, INDEX_NAME").
			Having("COUNT(*) = 1").
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

		// Get CHECK constraints (MySQL 8.0.16+)
		// We'll parse simple patterns like >= 0, <= 100, etc.
		// MySQL's CHECK_CONSTRAINTS table does not have TABLE_NAME; need to join with TABLE_CONSTRAINTS to get table name
		checkRows, err := db.Table("information_schema.CHECK_CONSTRAINTS cc").
			Select("cc.CONSTRAINT_NAME, cc.CHECK_CLAUSE").
			Joins("JOIN information_schema.TABLE_CONSTRAINTS tc ON cc.CONSTRAINT_SCHEMA = tc.CONSTRAINT_SCHEMA AND cc.CONSTRAINT_NAME = tc.CONSTRAINT_NAME").
			Where("cc.CONSTRAINT_SCHEMA = DATABASE() AND tc.TABLE_NAME = ?", storageUnit).
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
		// Ignore error if CHECK_CONSTRAINTS table doesn't exist (MySQL < 8.0.16)

		return true, nil
	})

	if err != nil {
		return constraints, err
	}

	return constraints, nil
}

// parseCheckConstraint parses MySQL CHECK constraint clauses to extract column constraints
func (p *MySQLPlugin) parseCheckConstraint(checkClause string, constraints map[string]map[string]any) {
	// MySQL formats CHECK constraints like:
	// - (`price` >= 0)
	// - (`stock_quantity` >= 0)
	// - (`age` between 18 and 120)
	// - (`status` in (_utf8mb4'active',_utf8mb4'inactive'))
	// - json_valid(`col_json`) - MariaDB's implicit JSON constraint

	// Check for JSON_VALID constraint BEFORE trimming parentheses
	// (trimming would remove the closing paren from json_valid())
	lowerClause := strings.ToLower(checkClause)
	if strings.Contains(lowerClause, "json_valid(") {
		columnName := extractJSONValidColumn(checkClause)
		if columnName != "" && common.ValidateColumnName(columnName) {
			colConstraints := gorm_plugin.EnsureConstraintEntry(constraints, columnName)
			colConstraints["is_json"] = true
			return
		}
	}

	// Now trim outer parentheses for other constraint types
	clause := strings.Trim(checkClause, "()")

	columnName := gorm_plugin.ExtractColumnNameFromClause(clause)
	if columnName == "" {
		return
	}

	// Validate column name to prevent SQL injection
	if !common.ValidateColumnName(columnName) {
		return
	}

	colConstraints := gorm_plugin.EnsureConstraintEntry(constraints, columnName)

	minMax := gorm_plugin.ParseMinMaxConstraints(clause)
	gorm_plugin.ApplyMinMaxToConstraints(colConstraints, minMax)

	if values := gorm_plugin.ParseINClauseValues(clause); len(values) > 0 {
		colConstraints["check_values"] = values
	}
}

// extractJSONValidColumn extracts the column name from a json_valid() constraint
// Handles: json_valid(`col_name`), json_valid(col_name), JSON_VALID(`col`)
func extractJSONValidColumn(clause string) string {
	lowerClause := strings.ToLower(clause)
	idx := strings.Index(lowerClause, "json_valid(")
	if idx == -1 {
		return ""
	}

	// Find the opening paren position in the original clause
	start := idx + len("json_valid(")
	if start >= len(clause) {
		return ""
	}

	// Find the closing paren
	end := strings.Index(clause[start:], ")")
	if end == -1 {
		return ""
	}

	colRef := strings.TrimSpace(clause[start : start+end])
	// Remove backticks or quotes
	colRef = strings.Trim(colRef, "`\"'")
	return colRef
}

// parseEnumValues extracts values from MySQL ENUM type definition
// Input format: enum('active','inactive','pending')
func parseEnumValues(columnType string) []string {
	// Find the content between parentheses
	start := strings.Index(columnType, "(")
	end := strings.LastIndex(columnType, ")")
	if start == -1 || end == -1 || end <= start {
		return nil
	}

	content := columnType[start+1 : end]
	var values []string

	// Split by comma, handling quoted strings
	// Values are quoted with single quotes: 'value1','value2'
	parts := strings.Split(content, ",")
	for _, part := range parts {
		cleaned := strings.TrimSpace(part)
		cleaned = strings.Trim(cleaned, "'\"")
		if cleaned != "" {
			if sanitized, ok := common.SanitizeConstraintValue(cleaned); ok {
				values = append(values, sanitized)
			}
		}
	}

	return values
}
