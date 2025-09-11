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
	"fmt"
)

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
