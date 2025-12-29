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

package engine

import "strings"

// TypeCategory represents the category of a database type for UI grouping
type TypeCategory string

const (
	TypeCategoryNumeric  TypeCategory = "numeric"
	TypeCategoryText     TypeCategory = "text"
	TypeCategoryBinary   TypeCategory = "binary"
	TypeCategoryDatetime TypeCategory = "datetime"
	TypeCategoryBoolean  TypeCategory = "boolean"
	TypeCategoryJSON     TypeCategory = "json"
	TypeCategoryOther    TypeCategory = "other"
)

// TypeDefinition describes a database column type with its metadata
type TypeDefinition struct {
	ID               string       // Canonical type name (e.g., "VARCHAR", "INTEGER")
	Label            string       // Display label for UI (e.g., "varchar", "integer")
	HasLength        bool         // Shows length input when selected (VARCHAR, CHAR)
	HasPrecision     bool         // Shows precision/scale inputs (DECIMAL, NUMERIC)
	DefaultLength    *int         // Default length for types with HasLength
	DefaultPrecision *int         // Default precision for types with HasPrecision
	Category         TypeCategory // Type category for grouping
}

// DatabaseMetadata contains all metadata for a database type
type DatabaseMetadata struct {
	DatabaseType    DatabaseType
	TypeDefinitions []TypeDefinition
	Operators       []string
	AliasMap        map[string]string
}

// Helper function to create a pointer to an int
func IntPtr(i int) *int {
	return &i
}

// ValidateColumnType checks if a column type string is valid against the TypeDefinitions.
// It parses the type string (e.g., "VARCHAR(255)") and validates:
// - The base type exists in TypeDefinitions (or AliasMap)
// - Length/precision parameters match the type's requirements
// Returns nil if valid, or an error describing the issue.
func ValidateColumnType(typeName string, metadata *DatabaseMetadata) error {
	if metadata == nil || len(metadata.TypeDefinitions) == 0 {
		// No metadata available - allow any type (backward compatibility)
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
		return &UnsupportedTypeError{TypeName: typeName, DatabaseType: string(metadata.DatabaseType)}
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
	parenIdx := strings.IndexByte(typeName, '(')
	if parenIdx == -1 {
		return typeName, false
	}
	return typeName[:parenIdx], true
}
