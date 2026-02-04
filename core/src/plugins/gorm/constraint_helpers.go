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

package gorm_plugin

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	// Patterns for numeric comparison constraints.
	// The \(? and \)? allow optional parentheses around numbers for MSSQL format like ([col]>=(0))
	gtePattern        = regexp.MustCompile(`>=\s*\(?(-?\d+(?:\.\d+)?)\)?`)
	gtPattern         = regexp.MustCompile(`>[^=]\s*\(?(-?\d+(?:\.\d+)?)\)?|>\s*\(?(-?\d+(?:\.\d+)?)\)?$`)
	ltePattern        = regexp.MustCompile(`<=\s*\(?(-?\d+(?:\.\d+)?)\)?`)
	ltPattern         = regexp.MustCompile(`<[^=]\s*\(?(-?\d+(?:\.\d+)?)\)?|<\s*\(?(-?\d+(?:\.\d+)?)\)?$`)
	betweenPattern    = regexp.MustCompile(`(?i)between\s+\(?(-?\d+(?:\.\d+)?)\)?\s+and\s+\(?(-?\d+(?:\.\d+)?)\)?`)
	typeCastPattern   = regexp.MustCompile(`::\w+(\s+\w+)?(\[\])?`)
	columnNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*`)
	// Pattern for OR-style equality constraints: [col]='value' OR [col]='value2'
	// Matches: [column]='value' or column='value' (with single or double quotes)
	orEqualityPattern = regexp.MustCompile(`(?:\[?\w+\]?)\s*=\s*(?:N)?['"]([^'"]+)['"]`)
)

// EnsureConstraintEntry initializes a constraint map entry for a column if it doesn't exist.
func EnsureConstraintEntry(constraints map[string]map[string]any, columnName string) map[string]any {
	if constraints[columnName] == nil {
		constraints[columnName] = map[string]any{}
	}
	return constraints[columnName]
}

// ParseINClauseValues extracts values from a SQL IN clause.
// Handles formats like:
//   - column IN ('val1', 'val2', 'val3')
//   - [column] IN (N'val1', N'val2')  (MSSQL Unicode)
//   - column IN ("val1", "val2")
//   - Multi-line CHECK constraints with whitespace/newlines
func ParseINClauseValues(clause string) []string {
	// Normalize whitespace (newlines, tabs, multiple spaces) to single spaces
	normalizedClause := strings.Join(strings.Fields(clause), " ")
	clauseLower := strings.ToLower(normalizedClause)

	inIdx := strings.Index(clauseLower, " in ")
	if inIdx == -1 {
		inIdx = strings.Index(clauseLower, " in(")
	}
	if inIdx == -1 {
		return nil
	}

	// Use the normalized clause for parsing
	clause = normalizedClause

	startIdx := strings.Index(clause[inIdx:], "(")
	if startIdx == -1 {
		return nil
	}
	startIdx += inIdx + 1

	depth := 1
	endIdx := -1
	for i := startIdx; i < len(clause); i++ {
		switch clause[i] {
		case '(':
			depth++
		case ')':
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

	return ParseValueList(content)
}

// ParseORClauseValues extracts values from OR equality constraints.
// Handles formats like:
//   - ([status]='Pending' OR [status]='Shipped' OR [status]='Delivered')
//   - (col='a' OR col='b')
//   - ([col]=N'value1' OR [col]=N'value2')  (MSSQL Unicode)
func ParseORClauseValues(clause string) []string {
	// Check if this looks like an OR-style constraint
	clauseLower := strings.ToLower(clause)
	if !strings.Contains(clauseLower, " or ") {
		return nil
	}

	// Find all [col]='value' or col='value' patterns
	matches := orEqualityPattern.FindAllStringSubmatch(clause, -1)
	if len(matches) == 0 {
		return nil
	}

	values := make([]string, 0, len(matches))
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			val := match[1]
			if !seen[val] {
				values = append(values, val)
				seen[val] = true
			}
		}
	}

	if len(values) == 0 {
		return nil
	}
	return values
}

// ParseValueList parses a comma-separated list of quoted values.
// Handles:
//   - Single quotes: 'value'
//   - Double quotes: "value"
//   - MySQL charset prefix: _utf8mb4'value'
//   - PostgreSQL type casts: 'value'::text
func ParseValueList(content string) []string {
	var values []string
	parts := strings.Split(content, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		part = strings.TrimPrefix(part, "N")
		part = strings.TrimPrefix(part, "n")

		if idx := strings.Index(part, "'"); idx > 0 && part[0] == '_' {
			part = part[idx:]
		}

		// Remove PostgreSQL type casts like ::text, ::varchar, ::character varying
		part = typeCastPattern.ReplaceAllString(part, "")

		// Remove quotes and add to values
		if len(part) >= 2 {
			if (part[0] == '\'' && part[len(part)-1] == '\'') ||
				(part[0] == '"' && part[len(part)-1] == '"') {
				values = append(values, part[1:len(part)-1])
			}
		}
	}

	return values
}

// MinMaxResult holds the result of parsing min/max constraints
type MinMaxResult struct {
	Min    float64
	Max    float64
	HasMin bool
	HasMax bool
}

// ParseMinMaxConstraints extracts min/max values from CHECK constraint clauses.
// Handles patterns like:
//   - column >= 0
//   - column > 0
//   - column <= 100
//   - column < 100
//   - column BETWEEN 0 AND 100
//   - ([column]>=(0)), ([column]>(0))
func ParseMinMaxConstraints(clause string) MinMaxResult {
	result := MinMaxResult{}
	clauseLower := strings.ToLower(clause)

	// Pattern for >= value
	if matches := gtePattern.FindStringSubmatch(clause); len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Min = val
			result.HasMin = true
		}
	} else if matches := gtPattern.FindStringSubmatch(clause); len(matches) > 0 {
		// Pattern for > value (exclusive, so add 1 for integers)
		// GT pattern has two capture groups due to alternation; use first non-empty one
		numStr := getFirstNonEmptyMatch(matches[1:])
		if numStr != "" {
			if val, err := strconv.ParseFloat(numStr, 64); err == nil {
				// For integers, > 0 means >= 1
				if val == float64(int64(val)) {
					result.Min = val + 1
				} else {
					result.Min = val
				}
				result.HasMin = true
			}
		}
	}

	// Pattern for <= value
	if matches := ltePattern.FindStringSubmatch(clause); len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Max = val
			result.HasMax = true
		}
	} else if matches := ltPattern.FindStringSubmatch(clause); len(matches) > 0 {
		// Pattern for < value (exclusive, so subtract 1 for integers)
		// LT pattern has two capture groups due to alternation; use first non-empty one
		numStr := getFirstNonEmptyMatch(matches[1:])
		if numStr != "" && !strings.Contains(clause[strings.Index(clauseLower, "<"):], "=") {
			if val, err := strconv.ParseFloat(numStr, 64); err == nil {
				// For integers, < 100 means <= 99
				if val == float64(int64(val)) {
					result.Max = val - 1
				} else {
					result.Max = val
				}
				result.HasMax = true
			}
		}
	}

	// Pattern for BETWEEN min AND max
	if strings.Contains(clauseLower, "between") {
		if matches := betweenPattern.FindStringSubmatch(clause); len(matches) > 2 {
			if minVal, err := strconv.ParseFloat(matches[1], 64); err == nil {
				result.Min = minVal
				result.HasMin = true
			}
			if maxVal, err := strconv.ParseFloat(matches[2], 64); err == nil {
				result.Max = maxVal
				result.HasMax = true
			}
		}
	}

	return result
}

// getFirstNonEmptyMatch returns the first non-empty string from the slice.
// Used for regex patterns with alternation that have multiple capture groups.
func getFirstNonEmptyMatch(matches []string) string {
	for _, m := range matches {
		if m != "" {
			return m
		}
	}
	return ""
}

// ApplyMinMaxToConstraints applies MinMaxResult values to a constraint map
func ApplyMinMaxToConstraints(constraints map[string]any, result MinMaxResult) {
	if result.HasMin {
		constraints["check_min"] = result.Min
	}
	if result.HasMax {
		constraints["check_max"] = result.Max
	}
}

// SanitizeConstraintValue removes type casts and normalizes a constraint value.
// Handles PostgreSQL type casts like ::text, ::integer, etc.
func SanitizeConstraintValue(value string) string {
	// Remove PostgreSQL type casts (uses pre-compiled regex)
	value = typeCastPattern.ReplaceAllString(value, "")
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "'\"")
	return value
}

// ExtractColumnNameFromClause extracts the column name from a CHECK constraint clause.
// Handles various database-specific formats:
//   - PostgreSQL: (column)::text = ... or column >= 0
//   - PostgreSQL with functions: length((password)::text) >= 8
//   - MySQL: `column` IN (...) or column >= 0
//   - SQLite: column IN (...) or column >= 0
func ExtractColumnNameFromClause(clause string) string {
	// Remove backticks (MySQL)
	clause = strings.ReplaceAll(clause, "`", "")
	// Remove square brackets (MSSQL)
	clause = strings.ReplaceAll(clause, "[", "")
	clause = strings.ReplaceAll(clause, "]", "")

	// Remove outer parentheses
	clause = strings.TrimSpace(clause)
	clause = strings.TrimPrefix(clause, "(")

	// Check if this starts with a function call like length(...), upper(...), etc.
	// Pattern: word followed by (
	firstWord := columnNamePattern.FindString(clause)
	if firstWord != "" {
		afterWord := strings.TrimPrefix(clause, firstWord)
		afterWord = strings.TrimSpace(afterWord)
		if strings.HasPrefix(afterWord, "(") {
			// This is a function call - extract column from inside
			col := extractColumnFromFunction(afterWord)
			if col != "" {
				return col
			}
		}
	}

	// Remove type casts like ::text
	if idx := strings.Index(clause, "::"); idx > 0 {
		clause = clause[:idx]
	}

	// Remove any trailing parenthesis
	clause = strings.TrimSuffix(clause, ")")
	clause = strings.TrimSpace(clause)

	// Extract the first word (column name)
	match := columnNamePattern.FindString(clause)
	return match
}

// extractColumnFromFunction extracts a column name from inside a function call.
// Input like "((password)::text)" returns "password"
func extractColumnFromFunction(funcArgs string) string {
	// Remove leading parentheses
	funcArgs = strings.TrimLeft(funcArgs, "(")

	// Remove type casts
	if idx := strings.Index(funcArgs, "::"); idx > 0 {
		funcArgs = funcArgs[:idx]
	}

	// Remove trailing parentheses
	funcArgs = strings.TrimRight(funcArgs, ")")
	funcArgs = strings.TrimSpace(funcArgs)

	// Extract column name
	return columnNamePattern.FindString(funcArgs)
}
