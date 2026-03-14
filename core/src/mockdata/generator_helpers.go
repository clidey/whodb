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

package mockdata

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
)

// valueToRecord converts a value to an engine.Record.
// If constraintType is provided (non-empty), it overrides the column type for type hints.
// This is important for MongoDB where schema validation types are more authoritative
// than types inferred from documents.
func valueToRecord(col engine.Column, value any, constraintType string) engine.Record {
	typeHint := col.Type
	if constraintType != "" {
		typeHint = constraintType
	}

	extra := map[string]string{
		"Type": typeHint,
	}

	var valueStr string
	if value == nil {
		valueStr = ""
		extra["IsNull"] = "true"
	} else if f, ok := value.(float64); ok {
		// Use fixed-point notation to avoid scientific notation (e.g., "1e+07")
		// which can cause numeric overflow errors in databases like PostgreSQL
		valueStr = strconv.FormatFloat(f, 'f', -1, 64)
	} else {
		valueStr = fmt.Sprintf("%v", value)
	}

	return engine.Record{
		Key:   col.Name,
		Value: valueStr,
		Extra: extra,
	}
}

// detectDatabaseType returns the simplified database type for a column.
func detectDatabaseType(columnType string) string {
	// Unwrap ClickHouse wrapper types (Nullable, LowCardinality) before classification
	columnType = unwrapTypeModifiers(columnType)
	upperType := strings.ToUpper(columnType)

	// Handle PostgreSQL arrays first
	if strings.Contains(upperType, "[]") {
		return "array"
	}

	// Handle timestamp/time types with timezone suffixes before other checks
	// PostgreSQL uses "TIMESTAMP WITH TIME ZONE", "TIME WITH TIME ZONE", etc.
	if strings.HasPrefix(upperType, "TIMESTAMP") {
		return "datetime"
	}
	if strings.HasPrefix(upperType, "TIME") && !strings.HasPrefix(upperType, "TINYINT") {
		return "datetime"
	}

	// Remove size specifiers like VARCHAR(255) -> VARCHAR
	if idx := strings.Index(upperType, "("); idx > 0 {
		upperType = upperType[:idx]
	}
	upperType = strings.TrimSpace(upperType)

	switch {
	case intTypes.Contains(upperType):
		return "int"
	case uintTypes.Contains(upperType):
		return "uint"
	case floatTypes.Contains(upperType):
		return "float"
	case boolTypes.Contains(upperType):
		return "bool"
	case dateTypes.Contains(upperType):
		return "date"
	case dateTimeTypes.Contains(upperType):
		return "datetime"
	case uuidTypes.Contains(upperType):
		return "uuid"
	case jsonTypes.Contains(upperType):
		return "json"
	case textTypes.Contains(upperType):
		return "text"
	default:
		return "text"
	}
}

// getConstraintsForColumn performs case-insensitive lookup for column constraints.
// This is necessary because SQLite (and some other databases) treat column names
// as case-insensitive, but the constraint map keys might not match the column case.
func getConstraintsForColumn(constraints map[string]map[string]any, columnName string) map[string]any {
	// Try exact match first
	if c, ok := constraints[columnName]; ok {
		return c
	}
	// Try case-insensitive match
	lowerName := strings.ToLower(columnName)
	for key, value := range constraints {
		if strings.ToLower(key) == lowerName {
			return value
		}
	}
	return nil
}
