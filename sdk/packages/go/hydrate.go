package whodb

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/clidey/whodb/sdk/packages/go/gen"
)

// Row is one hydrated record: column name → native-typed value.
type Row = map[string]any

// coerceValue coerces one stringly-typed cell into its native type per the
// shared cross-language hydration rules. Non-string values pass through
// unchanged: the IPC transport delivers natively-typed values.
func coerceValue(raw any, columnType string) any {
	if raw == nil {
		return nil
	}
	text, ok := raw.(string)
	if !ok {
		return raw
	}
	kind, ok := gen.HydrationRules[strings.ToLower(columnType)]
	if !ok {
		kind = gen.HydrationDefault
	}
	switch kind {
	case "int":
		var value int64
		if err := json.Unmarshal([]byte(text), &value); err == nil {
			return value
		}
		return text
	case "float":
		var value float64
		if err := json.Unmarshal([]byte(text), &value); err == nil {
			return value
		}
		return text
	case "bool":
		return text == "true" || text == "t" || text == "1"
	case "timestamp", "date":
		if parsed, err := time.Parse(time.RFC3339, text); err == nil {
			return parsed
		}
		if parsed, err := time.Parse("2006-01-02", text); err == nil {
			return parsed
		}
		return text
	case "json":
		var value any
		if err := json.Unmarshal([]byte(text), &value); err == nil {
			return value
		}
		return text
	default:
		return text
	}
}

type wireColumn struct {
	name       string
	columnType string
}

// normalizeResult normalizes the two wire result shapes:
// DatasetQueryResult {columns: [name], rows, total} (no column types — types
// come from ontology property metadata) and CE-derived RowsResult
// {Columns: [{Name, Type}], Rows, TotalCount}.
func normalizeResult(result map[string]any) ([]wireColumn, []any, any) {
	if rawColumns, ok := result["columns"].([]any); ok {
		columns := make([]wireColumn, 0, len(rawColumns))
		for _, column := range rawColumns {
			name, _ := column.(string)
			columns = append(columns, wireColumn{name: name})
		}
		rows, _ := result["rows"].([]any)
		return columns, rows, result["total"]
	}
	rawColumns, _ := result["Columns"].([]any)
	columns := make([]wireColumn, 0, len(rawColumns))
	for _, column := range rawColumns {
		object, _ := column.(map[string]any)
		name, _ := object["Name"].(string)
		columnType, _ := object["Type"].(string)
		columns = append(columns, wireColumn{name: name, columnType: columnType})
	}
	rows, _ := result["Rows"].([]any)
	return columns, rows, result["TotalCount"]
}

// propertyTypesOf builds a property-type map (apiName → dataType) from
// ontology entity metadata.
func propertyTypesOf(entity map[string]any) map[string]string {
	types := map[string]string{}
	properties, _ := entity["properties"].([]any)
	for _, property := range properties {
		object, _ := property.(map[string]any)
		apiName, _ := object["apiName"].(string)
		dataType, _ := object["dataType"].(string)
		if apiName != "" && dataType != "" {
			types[apiName] = dataType
		}
	}
	return types
}

// hydrateRows hydrates a wire result into native-typed rows. Ontology
// property metadata, when supplied, overrides the wire column type.
func hydrateRows(result map[string]any, propertyTypes map[string]string) ([]Row, any) {
	columns, rawRows, total := normalizeResult(result)
	rows := make([]Row, 0, len(rawRows))
	for _, rawRow := range rawRows {
		cells, _ := rawRow.([]any)
		row := Row{}
		for index, column := range columns {
			columnType := column.columnType
			if override, ok := propertyTypes[column.name]; ok {
				columnType = override
			}
			var cell any
			if index < len(cells) {
				cell = cells[index]
			}
			row[column.name] = coerceValue(cell, columnType)
		}
		rows = append(rows, row)
	}
	return rows, total
}
