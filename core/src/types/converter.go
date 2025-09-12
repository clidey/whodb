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
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/guregu/null/v5"
	"github.com/spf13/cast"
)

// Column represents a database column with type information
type Column struct {
	Name string
	Type string
}

// TypeConverter provides type conversion functionality
type TypeConverter interface {
	// Core conversions
	ConvertFromString(value string, dataType string) (any, error)
	ConvertToString(value any, dataType string) (string, error)

	// Bulk operations
	ConvertRow(values []string, columns []Column) ([]any, error)
	ConvertResults(rows [][]any, columns []Column) ([][]string, error)

	// Type information
	GetTypeDefinition(sqlType string) (*TypeDefinition, bool)
	IsNullable(sqlType string) bool
	GetBaseType(sqlType string) string

	// Registration
	RegisterDatabaseHandler(dbType string, handler DatabaseTypeHandler)
}

// DatabaseTypeHandler handles database-specific type conversions
type DatabaseTypeHandler interface {
	// Convert from string to database-specific type
	ConvertFromString(value string, sqlType string) (any, error)
	// Convert from database-specific type to string
	ConvertToString(value any, sqlType string) (string, error)
	// Check if this handler should handle this type
	CanHandle(sqlType string) bool
}

// UniversalTypeConverter implements TypeConverter using vetted libraries
type UniversalTypeConverter struct {
	registry *TypeRegistry
	dbType   string
	handlers map[string]DatabaseTypeHandler
}

// NewUniversalTypeConverter creates a new universal type converter
func NewUniversalTypeConverter(dbType string, registry *TypeRegistry) *UniversalTypeConverter {
	if registry == nil {
		registry = NewTypeRegistry()
		InitializeDefaultTypes(registry)
	}

	return &UniversalTypeConverter{
		registry: registry,
		dbType:   dbType,
		handlers: make(map[string]DatabaseTypeHandler),
	}
}

// RegisterDatabaseHandler registers a database-specific handler
func (c *UniversalTypeConverter) RegisterDatabaseHandler(dbType string, handler DatabaseTypeHandler) {
	c.handlers[dbType] = handler
}

// IsNullable checks if a SQL type is nullable
func (c *UniversalTypeConverter) IsNullable(sqlType string) bool {
	upperType := strings.ToUpper(sqlType)

	// ClickHouse nullable syntax
	if strings.HasPrefix(upperType, "NULLABLE(") {
		return true
	}

	// Check for NULL suffix or prefix
	if strings.Contains(upperType, "NULL") && !strings.Contains(upperType, "NOT NULL") {
		return true
	}

	return false
}

// GetBaseType extracts the base type from a nullable type
func (c *UniversalTypeConverter) GetBaseType(sqlType string) string {
	upperType := strings.ToUpper(sqlType)

	// Handle ClickHouse Nullable() wrapper
	if strings.HasPrefix(upperType, "NULLABLE(") {
		baseType := strings.TrimPrefix(upperType, "NULLABLE(")
		baseType = strings.TrimSuffix(baseType, ")")
		return baseType
	}

	// Handle LowCardinality() wrapper
	if strings.HasPrefix(upperType, "LOWCARDINALITY(") {
		baseType := strings.TrimPrefix(upperType, "LOWCARDINALITY(")
		baseType = strings.TrimSuffix(baseType, ")")
		return baseType
	}

	return sqlType
}

// ConvertFromString converts a string value to the appropriate type
func (c *UniversalTypeConverter) ConvertFromString(value string, dataType string) (any, error) {
	// Handle nullable types
	isNullable := c.IsNullable(dataType)
	baseType := c.GetBaseType(dataType)

	// Check for NULL values
	if isNullable && (value == "" || strings.EqualFold(value, "NULL")) {
		return c.getNullValue(baseType)
	}

	// Try database-specific handler first
	if handler, exists := c.handlers[c.dbType]; exists {
		if handler.CanHandle(baseType) {
			result, err := handler.ConvertFromString(value, baseType)
			if err == nil {
				if isNullable {
					return c.wrapNullable(result, baseType)
				}
				return result, nil
			}
		}
	}

	// Try type registry
	if typeDef, exists := c.registry.GetType(baseType); exists {
		if typeDef.FromString != nil {
			result, err := typeDef.FromString(value)
			if err != nil {
				return nil, err
			}
			if isNullable {
				return c.wrapNullable(result, baseType)
			}
			return result, nil
		}
	}

	// Fall back to spf13/cast for basic types
	result, err := c.convertWithCast(value, baseType)
	if err != nil {
		return nil, err
	}

	if isNullable {
		return c.wrapNullable(result, baseType)
	}

	return result, nil
}

// ConvertToString converts a value to string representation
func (c *UniversalTypeConverter) ConvertToString(value any, dataType string) (string, error) {
	if value == nil {
		return "", nil
	}

	// Handle guregu/null types
	switch v := value.(type) {
	case null.String:
		if !v.Valid {
			return "", nil
		}
		return v.String, nil
	case null.Int:
		if !v.Valid {
			return "", nil
		}
		return cast.ToString(v.Int64), nil
	case null.Float:
		if !v.Valid {
			return "", nil
		}
		return cast.ToString(v.Float64), nil
	case null.Bool:
		if !v.Valid {
			return "", nil
		}
		return cast.ToString(v.Bool), nil
	case null.Time:
		if !v.Valid {
			return "", nil
		}
		return c.formatTime(v.Time), nil
	}

	// Handle sql.Null types
	switch v := value.(type) {
	case sql.NullString:
		if !v.Valid {
			return "", nil
		}
		return v.String, nil
	case sql.NullInt64:
		if !v.Valid {
			return "", nil
		}
		return cast.ToString(v.Int64), nil
	case sql.NullFloat64:
		if !v.Valid {
			return "", nil
		}
		return cast.ToString(v.Float64), nil
	case sql.NullBool:
		if !v.Valid {
			return "", nil
		}
		return cast.ToString(v.Bool), nil
	case sql.NullTime:
		if !v.Valid {
			return "", nil
		}
		return c.formatTime(v.Time), nil
	}

	// Try database-specific handler
	baseType := c.GetBaseType(dataType)
	if handler, exists := c.handlers[c.dbType]; exists {
		if handler.CanHandle(baseType) {
			return handler.ConvertToString(value, baseType)
		}
	}

	// Try type registry
	if typeDef, exists := c.registry.GetType(baseType); exists {
		if typeDef.ToString != nil {
			return typeDef.ToString(value)
		}
	}

	// Use cast for everything else
	return cast.ToString(value), nil
}

// convertWithCast uses spf13/cast for basic type conversions
func (c *UniversalTypeConverter) convertWithCast(value string, dataType string) (any, error) {
	upperType := strings.ToUpper(dataType)
	category := c.registry.GetTypeCategory(upperType)

	switch category {
	case TypeCategoryNumeric:
		// Check if it's an integer type
		if c.isIntegerType(upperType) {
			return cast.ToInt64E(value)
		}
		// Otherwise treat as float
		return cast.ToFloat64E(value)

	case TypeCategoryBoolean:
		return cast.ToBoolE(value)

	case TypeCategoryDate:
		return cast.ToTimeE(value)

	case TypeCategoryText:
		return cast.ToStringE(value)

	default:
		// For unknown types, keep as string
		return value, nil
	}
}

// getNullValue returns appropriate null value for a type
func (c *UniversalTypeConverter) getNullValue(dataType string) (any, error) {
	upperType := strings.ToUpper(dataType)
	category := c.registry.GetTypeCategory(upperType)

	switch category {
	case TypeCategoryNumeric:
		if c.isIntegerType(upperType) {
			return null.IntFromPtr(nil), nil
		}
		return null.FloatFromPtr(nil), nil

	case TypeCategoryBoolean:
		return null.BoolFromPtr(nil), nil

	case TypeCategoryDate:
		return null.TimeFromPtr(nil), nil

	default:
		return null.StringFromPtr(nil), nil
	}
}

// wrapNullable wraps a value in a nullable type
func (c *UniversalTypeConverter) wrapNullable(value any, dataType string) (any, error) {
	if value == nil {
		return c.getNullValue(dataType)
	}

	upperType := strings.ToUpper(dataType)
	category := c.registry.GetTypeCategory(upperType)

	switch category {
	case TypeCategoryNumeric:
		if c.isIntegerType(upperType) {
			if v, err := cast.ToInt64E(value); err == nil {
				return null.IntFrom(v), nil
			}
		}
		if v, err := cast.ToFloat64E(value); err == nil {
			return null.FloatFrom(v), nil
		}

	case TypeCategoryBoolean:
		if v, err := cast.ToBoolE(value); err == nil {
			return null.BoolFrom(v), nil
		}

	case TypeCategoryDate:
		if v, err := cast.ToTimeE(value); err == nil {
			return null.TimeFrom(v), nil
		}

	default:
		if v, err := cast.ToStringE(value); err == nil {
			return null.StringFrom(v), nil
		}
	}

	return value, nil
}

// isIntegerType checks if a type is an integer type
func (c *UniversalTypeConverter) isIntegerType(dataType string) bool {
	upperType := strings.ToUpper(dataType)
	intTypes := []string{
		"INT", "INTEGER", "BIGINT", "SMALLINT", "TINYINT", "MEDIUMINT",
		"INT2", "INT4", "INT8", "INT16", "INT32", "INT64",
		"UINT", "UINT8", "UINT16", "UINT32", "UINT64",
	}

	for _, intType := range intTypes {
		if strings.Contains(upperType, intType) {
			return true
		}
	}
	return false
}

// formatTime formats a time value for string representation
func (c *UniversalTypeConverter) formatTime(t time.Time) string {
	// Check if it's a date-only value (no time component)
	if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0 {
		return t.Format("2006-01-02")
	}
	// Full datetime
	return t.Format("2006-01-02T15:04:05")
}

// GetTypeDefinition returns the type definition for a SQL type
func (c *UniversalTypeConverter) GetTypeDefinition(sqlType string) (*TypeDefinition, bool) {
	return c.registry.GetType(sqlType)
}

// ConvertRow converts a row of string values to appropriate types
func (c *UniversalTypeConverter) ConvertRow(values []string, columns []Column) ([]any, error) {
	result := make([]any, len(values))

	for i, value := range values {
		if i < len(columns) {
			converted, err := c.ConvertFromString(value, columns[i].Type)
			if err != nil {
				return nil, fmt.Errorf("column %s: %w", columns[i].Name, err)
			}
			result[i] = converted
		} else {
			result[i] = value
		}
	}

	return result, nil
}

// ConvertResults converts database results to string arrays
func (c *UniversalTypeConverter) ConvertResults(rows [][]any, columns []Column) ([][]string, error) {
	result := make([][]string, len(rows))

	for i, row := range rows {
		stringRow := make([]string, len(row))
		for j, value := range row {
			dataType := ""
			if j < len(columns) {
				dataType = columns[j].Type
			}

			str, err := c.ConvertToString(value, dataType)
			if err != nil {
				return nil, fmt.Errorf("row %d, column %d: %w", i, j, err)
			}
			stringRow[j] = str
		}
		result[i] = stringRow
	}

	return result, nil
}
