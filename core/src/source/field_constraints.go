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

package source

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// NormalizeFieldConstraints converts legacy plugin constraint maps into the
// source-level field constraint contract.
func NormalizeFieldConstraints(constraints map[string]map[string]any) []FieldConstraints {
	names := make([]string, 0, len(constraints))
	for name := range constraints {
		names = append(names, name)
	}
	sort.Strings(names)

	fields := make([]FieldConstraints, 0, len(names))
	for _, name := range names {
		fields = append(fields, NormalizeFieldConstraint(name, constraints[name]))
	}
	return fields
}

// MergeFieldConstraintsWithColumns enriches normalized constraints with column
// metadata from the source object.
func MergeFieldConstraintsWithColumns(fields []FieldConstraints, columns []Column) []FieldConstraints {
	if len(columns) == 0 {
		return fields
	}

	fieldIndexes := make(map[string]int, len(fields))
	for i, field := range fields {
		fieldIndexes[strings.ToLower(field.Name)] = i
	}

	merged := make([]FieldConstraints, 0, len(columns)+len(fields))
	usedFields := make(map[int]struct{}, len(fields))
	for _, column := range columns {
		index, ok := fieldIndexes[strings.ToLower(column.Name)]
		if ok {
			field := enrichFieldConstraintWithColumn(fields[index], column)
			merged = append(merged, field)
			usedFields[index] = struct{}{}
			continue
		}
		merged = append(merged, enrichFieldConstraintWithColumn(FieldConstraints{Name: column.Name}, column))
	}

	for i, field := range fields {
		if _, used := usedFields[i]; !used {
			merged = append(merged, field)
		}
	}
	return merged
}

func enrichFieldConstraintWithColumn(field FieldConstraints, column Column) FieldConstraints {
	if field.Type == "" {
		field.Type = column.Type
	}
	if field.MetadataFidelity == "" {
		field.MetadataFidelity = column.MetadataFidelity
	}
	if column.IsPrimary {
		field.Primary = true
	}
	if column.IsAutoIncrement {
		field.Identity = true
	}
	if field.Length == nil {
		field.Length = column.Length
	}
	if field.Precision == nil {
		field.Precision = column.Precision
	}
	if field.Scale == nil {
		field.Scale = column.Scale
	}
	if field.ForeignKey == nil && column.IsForeignKey && column.ReferencedTable != nil && column.ReferencedColumn != nil {
		field.ForeignKey = &ForeignKeyDefinition{Table: *column.ReferencedTable, Column: *column.ReferencedColumn}
	}
	return field
}

// NormalizeFieldConstraint converts one legacy plugin constraint map into the
// source-level field constraint contract.
func NormalizeFieldConstraint(name string, constraints map[string]any) FieldConstraints {
	normalized := normalizeConstraintKeys(constraints)
	field := FieldConstraints{Name: name}

	if value, ok := stringConstraint(normalized, "type"); ok {
		field.Type = value
	}
	if value, ok := boolConstraint(normalized, "nullable"); ok {
		field.Nullable = &value
	}
	field.Primary = boolConstraintAny(normalized, "primary", "is_primary")
	field.Unique = boolConstraintAny(normalized, "unique")
	field.Identity = boolConstraintAny(normalized, "identity", "auto_increment", "autoincrement", "is_identity")
	if value, ok := stringConstraintAny(normalized, "default", "default_value"); ok {
		field.DefaultValue = &value
	}
	if values, ok := stringListConstraintAny(normalized, "allowed_values", "check_values"); ok {
		field.AllowedValues = values
	}
	if value, ok := floatConstraintAny(normalized, "check_min", "min", "minimum"); ok {
		field.CheckMin = &value
	}
	if value, ok := floatConstraintAny(normalized, "check_max", "max", "maximum"); ok {
		field.CheckMax = &value
	}
	if value, ok := intConstraintAny(normalized, "length"); ok {
		field.Length = &value
	}
	if value, ok := intConstraintAny(normalized, "precision"); ok {
		field.Precision = &value
	}
	if value, ok := intConstraintAny(normalized, "scale"); ok {
		field.Scale = &value
	}
	if table, ok := stringConstraintAny(normalized, "referenced_table", "foreign_table", "foreign_key_table"); ok {
		if column, ok := stringConstraintAny(normalized, "referenced_column", "foreign_column", "foreign_key_column"); ok {
			field.ForeignKey = &ForeignKeyDefinition{Table: table, Column: column}
		}
	}

	return field
}

func normalizeConstraintKeys(constraints map[string]any) map[string]any {
	normalized := make(map[string]any, len(constraints))
	for key, value := range constraints {
		normalized[strings.ToLower(strings.TrimSpace(key))] = value
	}
	return normalized
}

func boolConstraintAny(constraints map[string]any, keys ...string) bool {
	value, ok := boolConstraint(constraints, keys...)
	return ok && value
}

func boolConstraint(constraints map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		value, ok := constraints[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return typed, true
		case string:
			parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
			if err == nil {
				return parsed, true
			}
		}
	}
	return false, false
}

func stringConstraintAny(constraints map[string]any, keys ...string) (string, bool) {
	return stringConstraint(constraints, keys...)
}

func stringConstraint(constraints map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		value, ok := constraints[key]
		if !ok || value == nil {
			continue
		}
		if str, ok := value.(string); ok {
			return str, true
		}
		return fmt.Sprint(value), true
	}
	return "", false
}

func stringListConstraintAny(constraints map[string]any, keys ...string) ([]string, bool) {
	for _, key := range keys {
		value, ok := constraints[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case []string:
			return append([]string(nil), typed...), true
		case []any:
			values := make([]string, 0, len(typed))
			for _, item := range typed {
				values = append(values, fmt.Sprint(item))
			}
			return values, true
		case string:
			parts := strings.Split(typed, CreationListSeparator())
			values := make([]string, 0, len(parts))
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					values = append(values, trimmed)
				}
			}
			return values, true
		}
	}
	return nil, false
}

func floatConstraintAny(constraints map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		value, ok := constraints[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return typed, true
		case float32:
			return float64(typed), true
		case int:
			return float64(typed), true
		case int32:
			return float64(typed), true
		case int64:
			return float64(typed), true
		case string:
			parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func intConstraintAny(constraints map[string]any, keys ...string) (int, bool) {
	for _, key := range keys {
		value, ok := constraints[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case int:
			return typed, true
		case int32:
			return int(typed), true
		case int64:
			return int(typed), true
		case float64:
			return int(typed), true
		case string:
			parsed, err := strconv.Atoi(strings.TrimSpace(typed))
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}
