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
	"strings"

	"github.com/clidey/whodb/core/src/source"
)

// Helper function to create a pointer to an int
//
//go:fix inline
func IntPtr(i int) *int {
	return new(i)
}

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

	// Validate parameters
	if (typeDef.HasLength || typeDef.HasPrecision) && !hasParams {
		// Type supports parameters but none provided - this is OK, defaults will be used
		return nil
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
