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
	"strings"
	
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

func (p *DuckDBPlugin) EscapeSpecificIdentifier(identifier string) string {
	// DuckDB uses double quotes for identifiers, similar to PostgreSQL
	identifier = strings.Replace(identifier, "\"", "\"\"", -1)
	return identifier
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
		columnName := p.EscapeSpecificIdentifier(column.Key)
		columnType := column.Value
		
		// Validate and normalize column type for DuckDB
		normalizedType := p.normalizeColumnType(columnType)
		columnDefs = append(columnDefs, fmt.Sprintf("\"%s\" %s", columnName, normalizedType))
	}
	
	tableName := p.EscapeSpecificIdentifier(storageUnit)
	return fmt.Sprintf("CREATE TABLE \"%s\" (%s)", tableName, strings.Join(columnDefs, ", "))
}

// normalizeColumnType ensures the column type is valid for DuckDB
func (p *DuckDBPlugin) normalizeColumnType(columnType string) string {
	upperType := strings.ToUpper(strings.TrimSpace(columnType))
	
	// Map common SQL types to DuckDB equivalents
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
		// Return as-is if it's already a valid DuckDB type
		if p.GetSupportedColumnDataTypes().Contains(upperType) {
			return upperType
		}
		// Default to VARCHAR for unknown types
		return "VARCHAR"
	}
}