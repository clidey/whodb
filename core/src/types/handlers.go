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
	"strings"

	"github.com/spf13/cast"
)

// BaseTypeHandler provides default database type handling
type BaseTypeHandler struct {
	dbType      string
	customTypes map[string]func(string) (any, error)
}

// NewBaseTypeHandler creates a new base type handler
func NewBaseTypeHandler(dbType string) *BaseTypeHandler {
	return &BaseTypeHandler{
		dbType:      dbType,
		customTypes: make(map[string]func(string) (any, error)),
	}
}

// CanHandle checks if this handler can handle the type
func (h *BaseTypeHandler) CanHandle(sqlType string) bool {
	upperType := strings.ToUpper(sqlType)
	_, exists := h.customTypes[upperType]
	return exists
}

// ConvertFromString converts string to database type
func (h *BaseTypeHandler) ConvertFromString(value string, sqlType string) (any, error) {
	upperType := strings.ToUpper(sqlType)
	if converter, exists := h.customTypes[upperType]; exists {
		return converter(value)
	}
	return nil, fmt.Errorf("type %s not handled by %s handler", sqlType, h.dbType)
}

// ConvertToString converts database type to string
func (h *BaseTypeHandler) ConvertToString(value any, sqlType string) (string, error) {
	return cast.ToStringE(value)
}

// RegisterCustomType registers a custom type converter
func (h *BaseTypeHandler) RegisterCustomType(sqlType string, converter func(string) (any, error)) {
	upperType := strings.ToUpper(sqlType)
	h.customTypes[upperType] = converter
}

// PostgreSQLHandler handles PostgreSQL-specific types
type PostgreSQLHandler struct {
	*BaseTypeHandler
	parser *PostgreSQLArrayParserSimple
}

// NewPostgreSQLHandler creates a PostgreSQL type handler
func NewPostgreSQLHandler() *PostgreSQLHandler {
	handler := &PostgreSQLHandler{
		BaseTypeHandler: NewBaseTypeHandler("postgresql"),
		parser:          NewPostgreSQLArrayParserSimple(),
	}

	// Register all PostgreSQL array types using pgx parser
	handler.registerArrayTypes()

	return handler
}

// registerArrayTypes registers handlers for various PostgreSQL array types
func (h *PostgreSQLHandler) registerArrayTypes() {
	// Integer arrays
	arrayTypes := []string{
		"_INT2", "_INT4", "_INT8", // Integer arrays
		"_FLOAT4", "_FLOAT8", // Float arrays
		"_TEXT", "_VARCHAR", "_CHAR", // Text arrays
		"_BOOL",                      // Boolean arrays
		"_UUID",                      // UUID arrays
		"_DATE",                      // Date arrays
		"_TIMESTAMP", "_TIMESTAMPTZ", // Timestamp arrays
		"_JSON", "_JSONB", // JSON arrays
		"_NUMERIC", "_DECIMAL", // Numeric arrays
		"_BYTEA", // Binary arrays
	}

	for _, arrayType := range arrayTypes {
		typeCopy := arrayType // Capture for closure
		h.RegisterCustomType(typeCopy, func(s string) (any, error) {
			return h.parser.ParseArraySimple(s, typeCopy)
		})
	}
}

// CanHandle checks if this handler can handle the type
func (h *PostgreSQLHandler) CanHandle(sqlType string) bool {
	upperType := strings.ToUpper(sqlType)

	// Check for PostgreSQL array types (underscore prefix)
	if strings.HasPrefix(upperType, "_") {
		return true
	}

	// Check base handler
	return h.BaseTypeHandler.CanHandle(sqlType)
}

// ConvertToString converts PostgreSQL types to string
func (h *PostgreSQLHandler) ConvertToString(value any, sqlType string) (string, error) {
	// Handle array types
	if strings.HasPrefix(strings.ToUpper(sqlType), "_") {
		return h.parser.FormatArraySimple(value), nil
	}

	// Use base handler for other types
	return h.BaseTypeHandler.ConvertToString(value, sqlType)
}

// MySQLHandler handles MySQL-specific types
type MySQLHandler struct {
	*BaseTypeHandler
}

// NewMySQLHandler creates a MySQL type handler
func NewMySQLHandler() *MySQLHandler {
	handler := &MySQLHandler{
		BaseTypeHandler: NewBaseTypeHandler("mysql"),
	}

	// Register MySQL-specific types
	handler.RegisterCustomType("YEAR", func(s string) (any, error) {
		return cast.ToInt64E(s)
	})
	handler.RegisterCustomType("SET", func(s string) (any, error) {
		// MySQL SET type - return as comma-separated string
		return s, nil
	})
	handler.RegisterCustomType("ENUM", func(s string) (any, error) {
		return s, nil
	})

	return handler
}

// ClickHouseHandler handles ClickHouse-specific types
type ClickHouseHandler struct {
	*BaseTypeHandler
	parser *ClickHouseArrayParser
}

// NewClickHouseHandler creates a ClickHouse type handler
func NewClickHouseHandler() *ClickHouseHandler {
	handler := &ClickHouseHandler{
		BaseTypeHandler: NewBaseTypeHandler("clickhouse"),
		parser:          &ClickHouseArrayParser{},
	}

	// Register array type handlers
	handler.registerArrayTypes()

	return handler
}

// registerArrayTypes registers handlers for various ClickHouse array types
func (h *ClickHouseHandler) registerArrayTypes() {
	// Common array types
	arrayTypes := []string{
		"ARRAY(INT8)", "ARRAY(INT16)", "ARRAY(INT32)", "ARRAY(INT64)",
		"ARRAY(UINT8)", "ARRAY(UINT16)", "ARRAY(UINT32)", "ARRAY(UINT64)",
		"ARRAY(FLOAT32)", "ARRAY(FLOAT64)",
		"ARRAY(STRING)", "ARRAY(FIXEDSTRING(16))",
		"ARRAY(DATE)", "ARRAY(DATETIME)",
		"ARRAY(UUID)",
		// Nested arrays
		"ARRAY(ARRAY(INT32))", "ARRAY(ARRAY(STRING))",
	}

	for _, arrayType := range arrayTypes {
		typeCopy := arrayType // Capture for closure
		h.RegisterCustomType(typeCopy, func(s string) (any, error) {
			return h.parser.ParseArray(s, extractInnerType(typeCopy))
		})
	}
}

// CanHandle checks if this handler can handle the type
func (h *ClickHouseHandler) CanHandle(sqlType string) bool {
	upperType := strings.ToUpper(sqlType)

	// Check for array types
	if strings.HasPrefix(upperType, "ARRAY(") {
		return true
	}

	// Check for other ClickHouse-specific types
	if strings.HasPrefix(upperType, "TUPLE(") ||
		strings.HasPrefix(upperType, "MAP(") ||
		strings.HasPrefix(upperType, "NESTED(") ||
		strings.HasPrefix(upperType, "LOWCARDINALITY(") ||
		strings.HasPrefix(upperType, "NULLABLE(") {
		return true
	}

	// Check base handler
	return h.BaseTypeHandler.CanHandle(sqlType)
}

// ConvertToString converts ClickHouse types to string
func (h *ClickHouseHandler) ConvertToString(value any, sqlType string) (string, error) {
	// Handle array types
	if strings.HasPrefix(strings.ToUpper(sqlType), "ARRAY(") {
		return h.parser.FormatArray(value), nil
	}

	// Use base handler for other types
	return h.BaseTypeHandler.ConvertToString(value, sqlType)
}

// SQLiteHandler handles SQLite-specific types
type SQLiteHandler struct {
	*BaseTypeHandler
}

// NewSQLiteHandler creates a SQLite type handler
func NewSQLiteHandler() *SQLiteHandler {
	handler := &SQLiteHandler{
		BaseTypeHandler: NewBaseTypeHandler("sqlite"),
	}

	// SQLite has fewer custom types since it's type-flexible
	// Most conversions are handled by the base converter

	return handler
}
