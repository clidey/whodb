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

