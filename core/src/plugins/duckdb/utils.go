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

package duckdb

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	
	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/gorm"
)

func (p *DuckDBPlugin) ConvertStringValueDuringMap(value, columnType string) (interface{}, error) {
	return value, nil
}

func (p *DuckDBPlugin) GetPrimaryKeyColQuery() string {
	return `
		SELECT 
			column_name as pk_column
		FROM 
			information_schema.key_column_usage
		WHERE 
			table_name = ? 
			AND table_schema = 'main'
			AND constraint_name LIKE '%_pkey'
		ORDER BY 
			ordinal_position;`
}

func (p *DuckDBPlugin) GetColTypeQuery() string {
	return `
		SELECT 
			column_name AS column_name,
			data_type AS data_type
		FROM 
			information_schema.columns
		WHERE 
			table_name = ?
			AND table_schema = 'main'
		ORDER BY 
			ordinal_position;`
}

// DuckDB reserved keywords that require quoting or are forbidden
var duckdbReservedKeywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "INSERT": true, "UPDATE": true, "DELETE": true,
	"CREATE": true, "DROP": true, "ALTER": true, "TABLE": true, "INDEX": true, "VIEW": true,
	"DATABASE": true, "SCHEMA": true, "COLUMN": true, "CONSTRAINT": true, "PRIMARY": true,
	"FOREIGN": true, "KEY": true, "REFERENCES": true, "UNIQUE": true, "NOT": true, "NULL": true,
	"DEFAULT": true, "CHECK": true, "UNION": true, "JOIN": true, "INNER": true, "LEFT": true,
	"RIGHT": true, "FULL": true, "OUTER": true, "ON": true, "USING": true, "GROUP": true,
	"ORDER": true, "BY": true, "HAVING": true, "LIMIT": true, "OFFSET": true, "AS": true,
	"DISTINCT": true, "ALL": true, "EXISTS": true, "IN": true, "BETWEEN": true, "LIKE": true,
	"ILIKE": true, "IS": true, "AND": true, "OR": true, "CASE": true, "WHEN": true, "THEN": true,
	"ELSE": true, "END": true, "CAST": true, "EXTRACT": true, "SUBSTRING": true, "TRIM": true,
	"COALESCE": true, "NULLIF": true, "GREATEST": true, "LEAST": true, "ARRAY": true, "STRUCT": true,
	"MAP": true, "UNION": true, "EXCEPT": true, "INTERSECT": true, "WITH": true, "RECURSIVE": true,
	"RETURNING": true, "CONFLICT": true, "DO": true, "NOTHING": true, "UPSERT": true,
}

// validateIdentifier validates that an identifier is safe for use in SQL
func (p *DuckDBPlugin) validateIdentifier(identifier string) error {
	// Check length limits (DuckDB supports up to 64 characters for identifiers)
	if len(identifier) == 0 {
		return fmt.Errorf("identifier cannot be empty")
	}
	if len(identifier) > 64 {
		return fmt.Errorf("identifier exceeds maximum length of 64 characters")
	}
	
	// Check for null bytes and other dangerous control characters
	if strings.Contains(identifier, "\x00") {
		return fmt.Errorf("identifier contains null byte")
	}
	
	// Check for dangerous characters that could enable injection
	for _, char := range identifier {
		if char < 32 && char != 9 && char != 10 && char != 13 { // Allow tab, newline, carriage return
			return fmt.Errorf("identifier contains invalid control character: %U", char)
		}
	}
	
	// Check for SQL injection patterns
	suspiciousPatterns := []string{
		"--", "/*", "*/", ";", "xp_", "sp_", "@@", "EXEC", "EXECUTE", 
		"SCRIPT", "JAVASCRIPT", "VBSCRIPT", "ONLOAD", "ONERROR",
	}
	upperIdentifier := strings.ToUpper(identifier)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(upperIdentifier, pattern) {
			return fmt.Errorf("identifier contains suspicious pattern: %s", pattern)
		}
	}
	
	return nil
}

// EscapeSpecificIdentifier properly escapes and validates identifiers for DuckDB
func (p *DuckDBPlugin) EscapeSpecificIdentifier(identifier string) string {
	// First validate the identifier for security
	if err := p.validateIdentifier(identifier); err != nil {
		// If validation fails, create a safe fallback identifier
		// This prevents injection while maintaining functionality
		safeIdentifier := p.createSafeIdentifier(identifier)
		return safeIdentifier
	}
	
	// Check if identifier needs quoting (contains special chars or is reserved)
	needsQuoting := p.identifierNeedsQuoting(identifier)
	
	if needsQuoting {
		// Escape double quotes by doubling them, then wrap in quotes
		escaped := strings.Replace(identifier, "\"", "\"\"", -1)
		return "\"" + escaped + "\""
	}
	
	// Return identifier as-is if it doesn't need quoting
	return identifier
}

// identifierNeedsQuoting determines if an identifier needs to be quoted
func (p *DuckDBPlugin) identifierNeedsQuoting(identifier string) bool {
	// Check if it's a reserved keyword
	if duckdbReservedKeywords[strings.ToUpper(identifier)] {
		return true
	}
	
	// Check if it starts with a number
	if len(identifier) > 0 && unicode.IsDigit(rune(identifier[0])) {
		return true
	}
	
	// Check if it contains special characters that require quoting
	for _, char := range identifier {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '_' {
			return true
		}
	}
	
	return false
}

// createSafeIdentifier creates a safe fallback identifier when validation fails
func (p *DuckDBPlugin) createSafeIdentifier(original string) string {
	// Create a safe identifier by removing dangerous characters
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	safe := reg.ReplaceAllString(original, "_")
	
	// Ensure it doesn't start with a number
	if len(safe) > 0 && unicode.IsDigit(rune(safe[0])) {
		safe = "col_" + safe
	}
	
	// Ensure it's not empty
	if safe == "" {
		safe = "safe_identifier"
	}
	
	// Ensure it's not too long
	if len(safe) > 64 {
		safe = safe[:64]
	}
	
	// Ensure it's not a reserved keyword
	if duckdbReservedKeywords[strings.ToUpper(safe)] {
		safe = safe + "_col"
	}
	
	// Always quote safe identifiers since they may have been modified
	return "\"" + safe + "\""
}

// GetGraphQueryDB returns the database connection for graph queries
func (p *DuckDBPlugin) GetGraphQueryDB(db *gorm.DB, schema string) *gorm.DB {
	// For DuckDB, we don't need schema-specific handling since it uses 'main' schema
	return db
}

// GetCreateTableQuery generates a CREATE TABLE statement for DuckDB
func (p *DuckDBPlugin) GetCreateTableQuery(schema string, storageUnit string, columns []engine.Record) string {
	var columnDefs []string
	
	for _, column := range columns {
		// Use secure identifier escaping (handles quoting automatically)
		columnName := p.EscapeSpecificIdentifier(column.Key)
		columnType := column.Value
		
		// Validate and normalize column type for DuckDB
		normalizedType := p.normalizeColumnType(columnType)
		
		// EscapeSpecificIdentifier now handles quoting, so don't add extra quotes
		columnDefs = append(columnDefs, fmt.Sprintf("%s %s", columnName, normalizedType))
	}
	
	// Use secure identifier escaping for table name (handles quoting automatically)
	tableName := p.EscapeSpecificIdentifier(storageUnit)
	return fmt.Sprintf("CREATE TABLE %s (%s)", tableName, strings.Join(columnDefs, ", "))
}

// normalizeColumnType ensures the column type is valid for DuckDB and prevents injection
func (p *DuckDBPlugin) normalizeColumnType(columnType string) string {
	// First sanitize the input to prevent SQL injection through column types
	columnType = strings.TrimSpace(columnType)
	
	// Enhanced security validation with comprehensive pattern detection
	if err := p.validateColumnType(columnType); err != nil {
		// If validation fails, return safe default and log the attempt
		return "VARCHAR"
	}
	
	upperType := strings.ToUpper(columnType)
	
	// Map common SQL types to DuckDB equivalents (simple types first)
	switch upperType {
	case "INT", "INT4":
		return "INTEGER"
	case "INT8":
		return "BIGINT"
	case "INT2":
		return "SMALLINT"
	case "INT1":
		return "TINYINT"
	case "FLOAT4":
		return "REAL"
	case "FLOAT8":
		return "DOUBLE"
	case "BOOL":
		return "BOOLEAN"
	case "STRING":
		return "VARCHAR"
	default:
		// Check if it's a valid DuckDB type with parameters (e.g., VARCHAR(255))
		validatedType := p.parseAndValidateParameterizedType(upperType)
		if validatedType != "" {
			return validatedType
		}
		
		// Check simple types without parameters
		if p.isValidSimpleDuckDBType(upperType) {
			return upperType
		}
		
		// Default to VARCHAR for unknown types
		return "VARCHAR"
	}
}

// validateColumnType performs comprehensive security validation on column types
func (p *DuckDBPlugin) validateColumnType(columnType string) error {
	// Check length limits
	if len(columnType) == 0 {
		return fmt.Errorf("column type cannot be empty")
	}
	if len(columnType) > 100 {
		return fmt.Errorf("column type exceeds maximum length")
	}
	
	// Check for null bytes and dangerous control characters
	if strings.Contains(columnType, "\x00") {
		return fmt.Errorf("column type contains null byte")
	}
	
	// Check for dangerous characters that could enable injection
	for _, char := range columnType {
		if char < 32 && char != 9 && char != 10 && char != 13 {
			return fmt.Errorf("column type contains invalid control character")
		}
	}
	
	// Enhanced suspicious pattern detection
	suspiciousPatterns := []string{
		"--", "/*", "*/", ";", "DROP", "DELETE", "INSERT", "UPDATE", "CREATE", "ALTER",
		"EXEC", "EXECUTE", "SCRIPT", "XP_", "SP_", "@@", "UNION", "SELECT", "FROM",
		"WHERE", "ORDER", "GROUP", "HAVING", "DECLARE", "SET", "GRANT", "REVOKE",
		"TRUNCATE", "MERGE", "CALL", "RETURN", "THROW", "TRY", "CATCH", "BEGIN",
		"END", "IF", "ELSE", "WHILE", "FOR", "CURSOR", "PROCEDURE", "FUNCTION",
		"TRIGGER", "VIEW", "SCHEMA", "DATABASE", "BACKUP", "RESTORE", "SHUTDOWN",
	}
	
	upperType := strings.ToUpper(columnType)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(upperType, pattern) {
			return fmt.Errorf("column type contains suspicious pattern: %s", pattern)
		}
	}
	
	// Validate that column type only contains strictly allowed characters
	// More restrictive: only letters, numbers, parentheses, spaces, comma, underscore
	allowedChars := regexp.MustCompile(`^[a-zA-Z0-9()\s,_]+$`)
	if !allowedChars.MatchString(columnType) {
		return fmt.Errorf("column type contains invalid characters")
	}
	
	return nil
}

// parseAndValidateParameterizedType handles types with parameters like VARCHAR(255) or DECIMAL(10,2)
func (p *DuckDBPlugin) parseAndValidateParameterizedType(typeStr string) string {
	// Match pattern: TYPE(param1) or TYPE(param1,param2)
	paramPattern := regexp.MustCompile(`^([A-Z]+)\s*\(\s*(\d+)(?:\s*,\s*(\d+))?\s*\)$`)
	matches := paramPattern.FindStringSubmatch(typeStr)
	
	if len(matches) == 0 {
		return "" // Not a parameterized type
	}
	
	baseType := matches[1]
	param1 := matches[2]
	param2 := ""
	if len(matches) > 3 && matches[3] != "" {
		param2 = matches[3]
	}
	
	// Validate base type is allowed
	if !p.isValidSimpleDuckDBType(baseType) {
		return ""
	}
	
	// Validate parameters are within reasonable bounds
	if !p.validateTypeParameters(baseType, param1, param2) {
		return ""
	}
	
	// Reconstruct the validated type
	if param2 != "" {
		return fmt.Sprintf("%s(%s,%s)", baseType, param1, param2)
	} else {
		return fmt.Sprintf("%s(%s)", baseType, param1)
	}
}

// validateTypeParameters ensures type parameters are within valid ranges
func (p *DuckDBPlugin) validateTypeParameters(baseType, param1, param2 string) bool {
	// Convert parameters to integers for validation
	p1, err1 := fmt.Atoi(param1)
	if err1 != nil {
		return false
	}
	
	var p2 int
	var err2 error
	if param2 != "" {
		p2, err2 = fmt.Atoi(param2)
		if err2 != nil {
			return false
		}
	}
	
	// Validate parameter ranges based on type
	switch baseType {
	case "VARCHAR", "CHAR", "TEXT":
		// Length parameter should be reasonable (1 to 65535)
		return p1 > 0 && p1 <= 65535 && param2 == ""
	case "DECIMAL", "NUMERIC":
		// Precision and scale parameters
		if param2 == "" {
			// Only precision specified
			return p1 > 0 && p1 <= 38
		} else {
			// Both precision and scale
			return p1 > 0 && p1 <= 38 && p2 >= 0 && p2 <= p1
		}
	case "FLOAT", "DOUBLE", "REAL":
		// Precision parameter for float types
		return p1 > 0 && p1 <= 53 && param2 == ""
	default:
		// For other types, be conservative and don't allow parameters
		return false
	}
}

// isValidSimpleDuckDBType checks if a type string is a valid simple DuckDB type (no parameters)
func (p *DuckDBPlugin) isValidSimpleDuckDBType(typeStr string) bool {
	// Comprehensive list of valid DuckDB base types
	validTypes := map[string]bool{
		"BOOLEAN": true, "TINYINT": true, "SMALLINT": true, "INTEGER": true, "BIGINT": true, 
		"HUGEINT": true, "UTINYINT": true, "USMALLINT": true, "UINTEGER": true, "UBIGINT": true, 
		"REAL": true, "DOUBLE": true, "DECIMAL": true, "NUMERIC": true, "VARCHAR": true, 
		"CHAR": true, "TEXT": true, "STRING": true, "BLOB": true, "BYTEA": true, 
		"DATE": true, "TIME": true, "TIMESTAMP": true, "TIMESTAMPTZ": true, "INTERVAL": true, 
		"UUID": true, "JSON": true, "ARRAY": true, "LIST": true, "STRUCT": true, "MAP": true, 
		"UNION": true, "ENUM": true,
	}
	
	return validTypes[typeStr]
}

