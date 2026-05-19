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
	"strconv"
	"strings"
)

const creationListSeparator = "\x1f"

// RecordsToObjectDefinition normalizes legacy record inputs into a typed object
// definition.
func RecordsToObjectDefinition(name string, fields []Record) ObjectDefinition {
	columns := make([]ColumnDefinition, 0, len(fields))
	for _, field := range fields {
		extra := NormalizeCreationExtra(field.Extra)
		column := ColumnDefinition{
			Name:        field.Key,
			Type:        field.Value,
			Primary:     parseCreationBool(extra["primary"]),
			Unique:      parseCreationBool(extra["unique"]),
			Identity:    parseCreationBool(extra["identity"]) || parseCreationBool(extra["auto_increment"]),
			CheckValues: splitCreationList(extra["check_values"]),
		}
		if raw, ok := extra["nullable"]; ok {
			nullable := parseCreationBool(raw)
			column.Nullable = &nullable
		}
		if raw, ok := extra["default"]; ok {
			column.DefaultValue = &raw
		}
		if raw, ok := extra["check_min"]; ok {
			if parsed, err := strconv.ParseFloat(raw, 64); err == nil {
				column.CheckMin = &parsed
			}
		}
		if raw, ok := extra["check_max"]; ok {
			if parsed, err := strconv.ParseFloat(raw, 64); err == nil {
				column.CheckMax = &parsed
			}
		}
		if table, ok := extra["references_table"]; ok {
			if columnName, hasColumn := extra["references_column"]; hasColumn {
				column.ForeignKey = &ForeignKeyDefinition{Table: table, Column: columnName}
			}
		}
		columns = append(columns, column)
	}
	return ObjectDefinition{Name: name, Columns: columns}
}

// ObjectDefinitionToRecords converts a typed object definition to canonical
// record inputs for plugin create paths.
func ObjectDefinitionToRecords(definition ObjectDefinition) []Record {
	records := make([]Record, 0, len(definition.Columns))
	for _, column := range definition.Columns {
		records = append(records, ColumnDefinitionToRecord(column))
	}
	return records
}

// ColumnDefinitionToRecord converts one typed column definition to canonical
// record metadata.
func ColumnDefinitionToRecord(column ColumnDefinition) Record {
	extra := map[string]string{
		"primary":  strconv.FormatBool(column.Primary),
		"Primary":  strconv.FormatBool(column.Primary),
		"unique":   strconv.FormatBool(column.Unique),
		"identity": strconv.FormatBool(column.Identity),
	}
	if column.Nullable != nil {
		nullable := strconv.FormatBool(*column.Nullable)
		extra["nullable"] = nullable
		extra["Nullable"] = nullable
	}
	if column.DefaultValue != nil {
		extra["default"] = *column.DefaultValue
	}
	if len(column.CheckValues) > 0 {
		extra["check_values"] = strings.Join(column.CheckValues, creationListSeparator)
	}
	if column.CheckMin != nil {
		extra["check_min"] = strconv.FormatFloat(*column.CheckMin, 'f', -1, 64)
	}
	if column.CheckMax != nil {
		extra["check_max"] = strconv.FormatFloat(*column.CheckMax, 'f', -1, 64)
	}
	if column.ForeignKey != nil {
		extra["references_table"] = column.ForeignKey.Table
		extra["references_column"] = column.ForeignKey.Column
	}
	return Record{Key: column.Name, Value: column.Type, Extra: extra}
}

// NormalizeCreationExtra maps legacy and database-specific constraint names to
// the canonical create-object keys used by plugins.
func NormalizeCreationExtra(extra map[string]string) map[string]string {
	normalized := make(map[string]string, len(extra))
	for key, value := range extra {
		canonical := strings.ToLower(strings.TrimSpace(key))
		canonical = strings.ReplaceAll(canonical, " ", "_")
		switch canonical {
		case "primary", "primary_key", "is_primary":
			canonical = "primary"
		case "nullable", "is_nullable":
			canonical = "nullable"
		case "auto_increment", "autoincrement", "generated", "identity":
			canonical = "identity"
		case "default_value":
			canonical = "default"
		case "check_values", "enum", "values":
			canonical = "check_values"
		case "check_min", "min":
			canonical = "check_min"
		case "check_max", "max":
			canonical = "check_max"
		case "referenced_table", "foreign_table", "references_table":
			canonical = "references_table"
		case "referenced_column", "foreign_column", "references_column":
			canonical = "references_column"
		}
		normalized[canonical] = value
	}
	return normalized
}

// CreationListSeparator returns the separator used when list-valued modifiers
// are carried through legacy record extras.
func CreationListSeparator() string {
	return creationListSeparator
}

func parseCreationBool(value string) bool {
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}

func splitCreationList(value string) []string {
	if value == "" {
		return nil
	}
	separator := creationListSeparator
	if !strings.Contains(value, separator) {
		separator = ","
	}
	parts := strings.Split(value, separator)
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
