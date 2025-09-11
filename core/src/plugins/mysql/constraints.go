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
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

// GetColumnConstraints retrieves column constraints for MySQL/MariaDB tables
func (p *MySQLPlugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error) {
	constraints := make(map[string]map[string]any)

	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		// Get nullability using GORM's query builder
		rows, err := db.Table("information_schema.columns").
			Select("COLUMN_NAME, IS_NULLABLE").
			Where("table_schema = DATABASE() AND table_name = ?", storageUnit).
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
				constraints[columnName] = map[string]any{}
			}
			constraints[columnName]["nullable"] = strings.EqualFold(isNullable, "YES")
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

			if constraints[columnName] == nil {
				constraints[columnName] = map[string]any{}
			}
			constraints[columnName]["unique"] = true
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

	// Remove backticks and outer parentheses for easier parsing
	clause := strings.ReplaceAll(checkClause, "`", "")
	clause = strings.Trim(clause, "()")
	clauseLower := strings.ToLower(clause)

	// Extract column name - it's usually the first word before an operator
	tokens := strings.Fields(clause)
	if len(tokens) < 3 {
		return
	}

	columnName := tokens[0]

	// Validate column name to prevent SQL injection
	if !common.ValidateColumnName(columnName) {
		return
	}

	// Check for >= or > constraints
	if strings.Contains(clause, ">=") {
		idx := strings.Index(clause, ">=")
		if idx > 0 && idx+2 < len(clause) {
			valueStr := strings.TrimSpace(clause[idx+2:])
			// Extract the number part
			valueStr = extractNumber(valueStr)
			if val, err := strconv.ParseFloat(valueStr, 64); err == nil {
				if constraints[columnName] == nil {
					constraints[columnName] = map[string]any{}
				}
				constraints[columnName]["check_min"] = val
			}
		}
	} else if strings.Contains(clause, ">") && !strings.Contains(clause, ">=") {
		idx := strings.Index(clause, ">")
		if idx > 0 && idx+1 < len(clause) {
			valueStr := strings.TrimSpace(clause[idx+1:])
			valueStr = extractNumber(valueStr)
			if val, err := strconv.ParseFloat(valueStr, 64); err == nil {
				if constraints[columnName] == nil {
					constraints[columnName] = map[string]any{}
				}
				constraints[columnName]["check_min"] = val + 1
			}
		}
	}

	// Check for <= or < constraints
	if strings.Contains(clause, "<=") {
		idx := strings.Index(clause, "<=")
		if idx > 0 && idx+2 < len(clause) {
			valueStr := strings.TrimSpace(clause[idx+2:])
			valueStr = extractNumber(valueStr)
			if val, err := strconv.ParseFloat(valueStr, 64); err == nil {
				if constraints[columnName] == nil {
					constraints[columnName] = map[string]any{}
				}
				constraints[columnName]["check_max"] = val
			}
		}
	} else if strings.Contains(clause, "<") && !strings.Contains(clause, "<=") {
		idx := strings.Index(clause, "<")
		if idx > 0 && idx+1 < len(clause) {
			valueStr := strings.TrimSpace(clause[idx+1:])
			valueStr = extractNumber(valueStr)
			if val, err := strconv.ParseFloat(valueStr, 64); err == nil {
				if constraints[columnName] == nil {
					constraints[columnName] = map[string]any{}
				}
				constraints[columnName]["check_max"] = val - 1
			}
		}
	}

	// Check for BETWEEN constraints
	if strings.Contains(clauseLower, " between ") {
		betweenIdx := strings.Index(clauseLower, " between ")
		if betweenIdx > 0 {
			afterBetween := clause[betweenIdx+9:] // length of " between "
			andIdx := strings.Index(strings.ToLower(afterBetween), " and ")
			if andIdx > 0 {
				minStr := strings.TrimSpace(afterBetween[:andIdx])
				maxStr := strings.TrimSpace(afterBetween[andIdx+5:]) // length of " and "

				minStr = extractNumber(minStr)
				maxStr = extractNumber(maxStr)

				if constraints[columnName] == nil {
					constraints[columnName] = map[string]any{}
				}

				if minVal, err := strconv.ParseFloat(minStr, 64); err == nil {
					constraints[columnName]["check_min"] = minVal
				}
				if maxVal, err := strconv.ParseFloat(maxStr, 64); err == nil {
					constraints[columnName]["check_max"] = maxVal
				}
			}
		}
	}

	// Check for IN constraints
	if strings.Contains(clauseLower, " in ") || strings.Contains(clauseLower, " in(") {
		inIdx := strings.Index(clauseLower, " in")
		if inIdx > 0 {
			afterIn := clause[inIdx+3:]
			// Find the opening parenthesis
			parenIdx := strings.Index(afterIn, "(")
			if parenIdx >= 0 {
				afterParen := afterIn[parenIdx+1:]
				// Find the closing parenthesis
				closeIdx := strings.Index(afterParen, ")")
				if closeIdx > 0 {
					valuesStr := afterParen[:closeIdx]
					values := parseInValues(valuesStr)
					if len(values) > 0 {
						if constraints[columnName] == nil {
							constraints[columnName] = map[string]any{}
						}
						constraints[columnName]["check_values"] = values
					}
				}
			}
		}
	}
}

// extractNumber extracts the numeric part from a string
func extractNumber(s string) string {
	s = strings.TrimSpace(s)
	// Remove parentheses if present
	s = strings.Trim(s, "()")

	// Find where the number ends
	endIdx := 0
	for i, ch := range s {
		if ch == '-' && i == 0 {
			continue // Allow negative sign at start
		}
		if ch >= '0' && ch <= '9' || ch == '.' {
			endIdx = i + 1
		} else {
			break
		}
	}

	if endIdx > 0 {
		return s[:endIdx]
	}
	return s
}

// parseInValues parses the values from an IN clause with SQL injection protection
func parseInValues(valuesStr string) []string {
	var values []string
	parts := strings.Split(valuesStr, ",")

	for _, part := range parts {
		cleaned := strings.TrimSpace(part)

		// Remove charset prefixes like _utf8mb4
		if idx := strings.Index(cleaned, "'"); idx > 0 {
			cleaned = cleaned[idx:]
		}

		// Remove quotes
		cleaned = strings.Trim(cleaned, "'\"")

		if cleaned != "" {
			// Validate the value to prevent SQL injection
			if sanitized, ok := common.SanitizeConstraintValue(cleaned); ok {
				values = append(values, sanitized)
			}
			// Skip malicious values silently
		}
	}

	return values
}
