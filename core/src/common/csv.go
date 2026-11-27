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

// EscapeFormula prefixes a value with a single quote when a spreadsheet would
// otherwise read it as a formula. CSV writers call it for free-text cells only:
// numbers, dates and other typed values keep their leading sign.
func EscapeFormula(value string) string {
	if len(value) == 0 {
		return value
	}

	switch value[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + value
	}
	return value
}

// EscapeFormulaRow returns a CSV row with its free-text cells guarded by
// EscapeFormula. text marks those cells; a nil text slice marks every cell.
func EscapeFormulaRow(row []string, text []bool) []string {
	escaped := make([]string, len(row))
	for i, value := range row {
		if text == nil || text[i] {
			escaped[i] = EscapeFormula(value)
		} else {
			escaped[i] = value
		}
	}
	return escaped
}

// FormatCSVHeader creates a header with the column name.
func FormatCSVHeader(columnName, dataType string) string {
	return columnName
}
