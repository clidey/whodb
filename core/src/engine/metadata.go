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

package engine

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/clidey/whodb/core/src/source"
)

// ValidateColumnType checks if a column type string is valid against
// source-owned type definitions and aliases.
func ValidateColumnType(typeName string, sourceType string, metadata *source.TypeSessionMetadata) error {
	if metadata == nil || len(metadata.TypeDefinitions) == 0 {
		// No metadata available - allow any type.
		return nil
	}

	// Parse the type specification
	baseType, hasParams := parseTypeForValidation(typeName)
	baseTypeUpper := strings.ToUpper(baseType)

	// Check if it's an alias and get the canonical type
	if canonical, ok := metadata.AliasMap[baseTypeUpper]; ok {
		baseTypeUpper = strings.ToUpper(canonical)
	}

	// Find the type definition
	var typeDef *TypeDefinition
	for i := range metadata.TypeDefinitions {
		if strings.ToUpper(metadata.TypeDefinitions[i].ID) == baseTypeUpper {
			typeDef = &metadata.TypeDefinitions[i]
			break
		}
	}

	if typeDef == nil {
		return &UnsupportedTypeError{TypeName: typeName, DatabaseType: sourceType}
	}

	// The full type string (including any parameter section) is concatenated
	// verbatim into CREATE TABLE DDL, so a parameter section must be a single
	// balanced parenthesized group of safe characters. This rejects statement
	// terminators, comment markers, and parenthesis breakouts while still
	// allowing legitimate params such as varchar(255), Decimal(10, 2),
	// DateTime64(3, 'UTC'), and Enum8('active' = 1, 'inactive' = 2).
	if hasParams {
		if err := validateTypeParams(typeName); err != nil {
			return &UnsupportedTypeError{TypeName: typeName, DatabaseType: sourceType}
		}
	}

	return nil
}

// validateTypeParams verifies that a parameterized type name has exactly one
// balanced trailing parameter group containing only characters that cannot break
// out of the column-type position in DDL. Enum labels are single-quoted and may
// contain any character except a single quote.
func validateTypeParams(typeName string) error {
	trimmed := strings.TrimSpace(typeName)
	open := strings.Index(trimmed, "(")
	closeIdx := strings.LastIndex(trimmed, ")")
	if open == -1 || closeIdx != len(trimmed)-1 || closeIdx <= open {
		return errors.New("malformed type parameters")
	}

	params := trimmed[open+1 : closeIdx]
	inQuote := false
	for _, r := range params {
		switch {
		case r == '\'':
			inQuote = !inQuote
		case inQuote:
			// Inside a quoted enum label; the surrounding quotes keep it safe.
		case unicode.IsLetter(r) || unicode.IsDigit(r):
		case r == ',' || r == ' ' || r == '=' || r == '+' || r == '-' || r == '.':
		default:
			return fmt.Errorf("invalid character %q in type parameters", r)
		}
	}
	if inQuote {
		return errors.New("unterminated quote in type parameters")
	}
	return nil
}

// UnsupportedTypeError is returned when a column type is not supported by the database
type UnsupportedTypeError struct {
	TypeName     string
	DatabaseType string
}

func (e *UnsupportedTypeError) Error() string {
	return "data type: " + e.TypeName + " not supported by: " + e.DatabaseType
}

// parseTypeForValidation extracts the base type and whether it has parameters
func parseTypeForValidation(typeName string) (baseType string, hasParams bool) {
	typeName = strings.TrimSpace(typeName)
	before, _, ok := strings.Cut(typeName, "(")
	if !ok {
		return typeName, false
	}
	return before, true
}
