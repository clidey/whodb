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

package common

import (
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

const CSVDelimiter = '|'

// EscapeCSVValue properly escapes a value for CSV export
func EscapeCSVValue(value string) string {
	// CSV standard: if value contains delimiter, newline, or quote, wrap in quotes
	// and escape quotes by doubling them
	if strings.ContainsAny(value, string(CSVDelimiter)+"\n\r\"") {
		return fmt.Sprintf("%q", value)
	}
	return value
}

// EscapeFormula escapes values that could be interpreted as formulas in spreadsheet applications
func EscapeFormula(value string) string {
	if len(value) == 0 {
		return value
	}
	
	// Check if the first character is a formula indicator
	firstChar := value[0]
	if firstChar == '=' || firstChar == '+' || firstChar == '-' || firstChar == '@' || firstChar == '\t' || firstChar == '\r' {
		// Prefix with single quote to prevent formula execution
		return "'" + value
	}
	
	return value
}

// FormatCSVHeader creates a header with column name and type
func FormatCSVHeader(columnName, dataType string) string {
	return fmt.Sprintf("%s:%s", columnName, dataType)
}

// CreateCSVWriter creates a CSV writer with our standard configuration
func CreateCSVWriter(w io.Writer) *csv.Writer {
	writer := csv.NewWriter(w)
	writer.Comma = CSVDelimiter
	return writer
}

// WriteSelectedRowsToCSV writes pre-selected rows to CSV format with formula protection
func WriteSelectedRowsToCSV(writer io.Writer, selectedRows []map[string]any, delimiter rune) error {
	csvWriter := csv.NewWriter(writer)
	csvWriter.Comma = delimiter

	if len(selectedRows) == 0 {
		return nil
	}

	// Extract headers from the first row
	var headers []string
	for key := range selectedRows[0] {
		headers = append(headers, key)
	}

	// Write headers
	if err := csvWriter.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}

	// Write data rows
	for _, row := range selectedRows {
		record := make([]string, len(headers))
		for i, header := range headers {
			value := row[header]
			record[i] = EscapeFormula(formatValue(value))
		}
		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	csvWriter.Flush()
	return csvWriter.Error()
}

// formatValue converts a value to string representation
func formatValue(value any) string {
	if value == nil {
		return ""
	}
	
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	default:
		return fmt.Sprintf("%v", value)
	}
}

