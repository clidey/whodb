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

package types

import (
	"fmt"
	"strconv"
	"strings"
)

// ClickHouseArrayParser provides robust parsing for ClickHouse array syntax
type ClickHouseArrayParser struct{}

// ParseArray parses a ClickHouse array string into a Go slice
// Handles nested arrays, quoted strings, and proper escaping
func (p *ClickHouseArrayParser) ParseArray(value string, elementType string) ([]interface{}, error) {
	// Remove outer brackets
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return nil, fmt.Errorf("invalid array format: missing brackets")
	}

	value = value[1 : len(value)-1]
	if value == "" {
		return []interface{}{}, nil
	}

	// Parse elements based on nesting and quotes
	elements, err := p.parseElements(value)
	if err != nil {
		return nil, err
	}

	// Convert elements to appropriate types
	result := make([]interface{}, 0, len(elements))
	for _, elem := range elements {
		converted, err := p.convertElement(elem, elementType)
		if err != nil {
			return nil, fmt.Errorf("converting element %q: %w", elem, err)
		}
		result = append(result, converted)
	}

	return result, nil
}

// parseElements splits array elements handling nested arrays and quoted strings
func (p *ClickHouseArrayParser) parseElements(value string) ([]string, error) {
	var elements []string
	var current strings.Builder

	inQuotes := false
	quoteChar := rune(0)
	bracketDepth := 0
	escaped := false

	for i, ch := range value {
		if escaped {
			current.WriteRune(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			current.WriteRune(ch)
			continue
		}

		// Handle quotes
		if !inQuotes && (ch == '\'' || ch == '"') {
			inQuotes = true
			quoteChar = ch
			current.WriteRune(ch)
			continue
		}

		if inQuotes && ch == quoteChar {
			inQuotes = false
			quoteChar = 0
			current.WriteRune(ch)
			continue
		}

		if inQuotes {
			current.WriteRune(ch)
			continue
		}

		// Handle nested arrays
		if ch == '[' {
			bracketDepth++
			current.WriteRune(ch)
			continue
		}

		if ch == ']' {
			bracketDepth--
			if bracketDepth < 0 {
				return nil, fmt.Errorf("unmatched closing bracket at position %d", i)
			}
			current.WriteRune(ch)
			continue
		}

		// Handle element separator
		if ch == ',' && bracketDepth == 0 {
			elem := strings.TrimSpace(current.String())
			if elem != "" {
				elements = append(elements, elem)
			}
			current.Reset()
			continue
		}

		current.WriteRune(ch)
	}

	// Add the last element
	if current.Len() > 0 {
		elem := strings.TrimSpace(current.String())
		if elem != "" {
			elements = append(elements, elem)
		}
	}

	if bracketDepth != 0 {
		return nil, fmt.Errorf("unmatched brackets")
	}

	if inQuotes {
		return nil, fmt.Errorf("unclosed quote")
	}

	return elements, nil
}

// convertElement converts a string element to the appropriate type
func (p *ClickHouseArrayParser) convertElement(elem string, elementType string) (interface{}, error) {
	elem = strings.TrimSpace(elem)

	// Handle nested arrays
	if strings.HasPrefix(elementType, "Array(") {
		innerType := extractInnerType(elementType)
		if strings.HasPrefix(elem, "[") && strings.HasSuffix(elem, "]") {
			return p.ParseArray(elem, innerType)
		}
	}

	// Handle quoted strings
	if (strings.HasPrefix(elem, "'") && strings.HasSuffix(elem, "'")) ||
		(strings.HasPrefix(elem, "\"") && strings.HasSuffix(elem, "\"")) {
		// Remove quotes and handle escapes
		unquoted := elem[1 : len(elem)-1]
		unquoted = strings.ReplaceAll(unquoted, "\\'", "'")
		unquoted = strings.ReplaceAll(unquoted, "\\\"", "\"")
		unquoted = strings.ReplaceAll(unquoted, "\\\\", "\\")
		return unquoted, nil
	}

	// Handle NULL
	if strings.EqualFold(elem, "NULL") {
		return nil, nil
	}

	// Try to parse as number based on element type
	upperType := strings.ToUpper(elementType)
	if isNumericType(upperType) {
		if strings.Contains(elem, ".") {
			// Float
			if f, err := strconv.ParseFloat(elem, 64); err == nil {
				return f, nil
			}
		} else {
			// Integer
			if i, err := strconv.ParseInt(elem, 10, 64); err == nil {
				return i, nil
			}
		}
	}

	// Handle boolean
	if upperType == "BOOL" || upperType == "BOOLEAN" {
		if b, err := strconv.ParseBool(elem); err == nil {
			return b, nil
		}
	}

	// Default to string
	return elem, nil
}

// FormatArray formats a Go slice into ClickHouse array syntax
func (p *ClickHouseArrayParser) FormatArray(value interface{}) string {
	if value == nil {
		return "[]"
	}

	switch v := value.(type) {
	case []interface{}:
		if len(v) == 0 {
			return "[]"
		}
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = p.formatElement(item)
		}
		return "[" + strings.Join(parts, ",") + "]"

	case []string:
		if len(v) == 0 {
			return "[]"
		}
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = p.quoteString(item)
		}
		return "[" + strings.Join(parts, ",") + "]"

	case []int, []int32, []int64:
		return p.formatNumericArray(v)

	case []float32, []float64:
		return p.formatNumericArray(v)

	default:
		// Try to format as string representation
		return fmt.Sprintf("%v", v)
	}
}

// formatElement formats a single array element
func (p *ClickHouseArrayParser) formatElement(elem interface{}) string {
	if elem == nil {
		return "NULL"
	}

	switch v := elem.(type) {
	case string:
		return p.quoteString(v)
	case []interface{}:
		// Nested array
		return p.FormatArray(v)
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "1"
		}
		return "0"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// quoteString adds quotes and escapes special characters
func (p *ClickHouseArrayParser) quoteString(s string) string {
	// Escape special characters
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return "'" + s + "'"
}

// formatNumericArray formats numeric arrays without quotes
func (p *ClickHouseArrayParser) formatNumericArray(value interface{}) string {
	switch v := value.(type) {
	case []int:
		parts := make([]string, len(v))
		for i, n := range v {
			parts[i] = strconv.Itoa(n)
		}
		return "[" + strings.Join(parts, ",") + "]"
	case []int32:
		parts := make([]string, len(v))
		for i, n := range v {
			parts[i] = strconv.FormatInt(int64(n), 10)
		}
		return "[" + strings.Join(parts, ",") + "]"
	case []int64:
		parts := make([]string, len(v))
		for i, n := range v {
			parts[i] = strconv.FormatInt(n, 10)
		}
		return "[" + strings.Join(parts, ",") + "]"
	case []float32:
		parts := make([]string, len(v))
		for i, n := range v {
			parts[i] = strconv.FormatFloat(float64(n), 'f', -1, 32)
		}
		return "[" + strings.Join(parts, ",") + "]"
	case []float64:
		parts := make([]string, len(v))
		for i, n := range v {
			parts[i] = strconv.FormatFloat(n, 'f', -1, 64)
		}
		return "[" + strings.Join(parts, ",") + "]"
	default:
		return "[]"
	}
}

// extractInnerType extracts the inner type from Array(Type)
func extractInnerType(arrayType string) string {
	if !strings.HasPrefix(arrayType, "Array(") {
		return ""
	}
	inner := strings.TrimPrefix(arrayType, "Array(")
	inner = strings.TrimSuffix(inner, ")")
	return inner
}

// isNumericType checks if a type is numeric
func isNumericType(typeName string) bool {
	numericTypes := []string{
		"INT", "INTEGER", "BIGINT", "SMALLINT", "TINYINT",
		"UINT", "UINT8", "UINT16", "UINT32", "UINT64",
		"FLOAT", "DOUBLE", "DECIMAL",
		"INT8", "INT16", "INT32", "INT64",
		"FLOAT32", "FLOAT64",
	}

	for _, t := range numericTypes {
		if strings.Contains(typeName, t) {
			return true
		}
	}
	return false
}
