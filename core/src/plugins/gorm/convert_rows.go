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

package gorm_plugin

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/dromara/carbon/v2"
	"gorm.io/gorm"
)

// ConvertRawToRows converts raw SQL rows into the engine's GetRowsResult format.
// Handles column metadata extraction (type, length, precision) and per-row value formatting
// including binary, geometry, time, and plugin-specific custom types.
func (p *GormPlugin) ConvertRawToRows(rows *sql.Rows) (*engine.GetRowsResult, error) {
	columns, err := rows.Columns()
	if err != nil {
		log.WithError(err).Error("Failed to get column names from result set")
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		log.WithError(err).Error("Failed to get column types from result set")
		return nil, err
	}

	// Create a map for faster column type lookup
	typeMap := make(map[string]*sql.ColumnType, len(columnTypes))
	for _, colType := range columnTypes {
		typeMap[colType.Name()] = colType
	}

	result := &engine.GetRowsResult{
		Columns: make([]engine.Column, 0, len(columns)),
		Rows:    make([][]string, 0, 100),
	}

	for _, col := range columns {
		if colType, exists := typeMap[col]; exists {
			colTypeName := colType.DatabaseTypeName()
			// Let plugins handle array type display (e.g., PostgreSQL _int4 -> []int4)
			if p.GormPluginFunctions.IsArrayType(colTypeName) {
				colTypeName = strings.Replace(colTypeName, "_", "[]", 1)
			}
			if customName := p.GormPluginFunctions.GetCustomColumnTypeName(col, colTypeName); customName != "" {
				colTypeName = customName
			}

			column := engine.Column{Name: col, Type: colTypeName}
			baseTypeName := strings.ToUpper(colType.DatabaseTypeName())

			// Only extract length for types where it's user-specifiable
			if typesWithLength[baseTypeName] {
				if length, ok := colType.Length(); ok && length > 0 {
					l := int(length)
					column.Length = &l
					// Include length in type name for display
					colTypeName = fmt.Sprintf("%s(%d)", colTypeName, length)
					column.Type = colTypeName
				}
			}

			// Only extract precision/scale for decimal-like types
			if typesWithPrecision[baseTypeName] {
				if precision, scale, ok := colType.DecimalSize(); ok && precision > 0 {
					prec := int(precision)
					column.Precision = &prec
					if scale > 0 {
						s := int(scale)
						column.Scale = &s
						colTypeName = fmt.Sprintf("%s(%d,%d)", colType.DatabaseTypeName(), precision, scale)
					} else {
						colTypeName = fmt.Sprintf("%s(%d)", colType.DatabaseTypeName(), precision)
					}
					column.Type = colTypeName
				}
			}

			result.Columns = append(result.Columns, column)
		}
	}

	for rows.Next() {
		columnPointers := make([]any, len(columns))
		row := make([]string, len(columns))

		for i, col := range columns {
			colType := typeMap[col]
			typeName := colType.DatabaseTypeName()

			if p.GormPluginFunctions.ShouldHandleColumnType(typeName) {
				columnPointers[i] = p.GormPluginFunctions.GetColumnScanner(typeName)
			} else {
				switch typeName {
				case "VARBINARY", "BINARY", "IMAGE", "BYTEA", "BLOB", "HIERARCHYID",
					"GEOMETRY", "POINT", "LINESTRING", "POLYGON", "GEOGRAPHY",
					"MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON":
					columnPointers[i] = new(sql.RawBytes)
				default:
					columnPointers[i] = new(sql.NullString)
				}
			}
		}

		if err := rows.Scan(columnPointers...); err != nil {
			log.WithError(err).Error("Failed to scan row data")
			return nil, err
		}

		for i, colPtr := range columnPointers {
			colType := typeMap[columns[i]]
			typeName := colType.DatabaseTypeName()

			if p.GormPluginFunctions.ShouldHandleColumnType(typeName) {
				value, err := p.GormPluginFunctions.FormatColumnValue(typeName, colPtr)
				if err != nil {
					row[i] = "ERROR: " + err.Error()
				} else {
					row[i] = value
				}
			} else {
				switch typeName {
				case "VARBINARY", "BINARY", "IMAGE", "BYTEA", "BLOB":
					rawBytes := colPtr.(*sql.RawBytes)
					if rawBytes == nil || len(*rawBytes) == 0 {
						row[i] = ""
					} else {
						row[i] = "0x" + hex.EncodeToString(*rawBytes)
					}
				case "GEOMETRY", "POINT", "LINESTRING", "POLYGON", "GEOGRAPHY",
					"MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON":
					rawBytes := colPtr.(*sql.RawBytes)
					if rawBytes == nil || len(*rawBytes) == 0 {
						row[i] = ""
					} else if formatted := p.GormPluginFunctions.FormatGeometryValue(*rawBytes, typeName); formatted != "" {
						row[i] = formatted
					} else {
						row[i] = "0x" + hex.EncodeToString(*rawBytes)
					}
				case "TIME":
					// TIME columns are returned as full datetime strings with zero date (e.g., "0001-01-01T12:00:00Z")
					// Extract just the time portion for display
					val := colPtr.(*sql.NullString)
					if val.Valid {
						row[i] = formatTimeOnly(val.String)
					} else {
						row[i] = ""
					}
				default:
					val := colPtr.(*sql.NullString)
					if val.Valid {
						row[i] = val.String
					} else {
						row[i] = ""
					}
				}
			}
		}

		result.Rows = append(result.Rows, row)
	}

	result.TotalCount = int64(len(result.Rows))
	return result, nil
}

// FindMissingDataType resolves unknown column types by querying system catalogs.
func (p *GormPlugin) FindMissingDataType(db *gorm.DB, columnType string) string {
	if p.Type == engine.DatabaseType_Postgres {
		var typname string
		if err := db.Table("pg_type").
			Select("typname").
			Where("oid = ?", columnType).
			Scan(&typname).Error; err != nil {
			log.WithError(err).Error(fmt.Sprintf("Failed to find PostgreSQL type name for OID: %s", columnType))
			typname = columnType
		}
		return strings.ToUpper(typname)
	}
	return columnType
}

// GetRowsOrderBy returns the ORDER BY clause for pagination queries.
// Default implementation returns empty string (no ordering).
func (p *GormPlugin) GetRowsOrderBy(db *gorm.DB, schema string, storageUnit string) string {
	return ""
}

// ShouldHandleColumnType returns false by default.
func (p *GormPlugin) ShouldHandleColumnType(columnType string) bool {
	return false
}

// GetColumnScanner returns nil by default.
func (p *GormPlugin) GetColumnScanner(columnType string) any {
	return nil
}

// FormatColumnValue returns empty string by default.
func (p *GormPlugin) FormatColumnValue(columnType string, scanner any) (string, error) {
	return "", nil
}

// GetCustomColumnTypeName returns empty string by default.
func (p *GormPlugin) GetCustomColumnTypeName(columnName string, defaultTypeName string) string {
	return ""
}

// IsGeometryType returns true for common geometry type names.
func (p *GormPlugin) IsGeometryType(columnType string) bool {
	switch columnType {
	case "GEOMETRY", "POINT", "LINESTRING", "POLYGON", "GEOGRAPHY",
		"MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON", "GEOMETRYCOLLECTION":
		return true
	default:
		return false
	}
}

// FormatGeometryValue returns empty string by default (use hex formatting).
func (p *GormPlugin) FormatGeometryValue(rawBytes []byte, columnType string) string {
	return ""
}

// formatTimeOnly extracts just the time portion from a datetime string.
// Database drivers return TIME columns as full datetime with zero date (e.g., "0001-01-01T12:00:00Z").
// This function extracts just the time portion for cleaner display.
func formatTimeOnly(value string) string {
	c := carbon.Parse(value)
	if c.Error != nil || c.IsInvalid() {
		// If carbon can't parse it, return as-is
		return value
	}

	// Check if it has sub-second precision
	if c.Nanosecond() > 0 {
		return c.ToTimeMilliString()
	}
	return c.ToTimeString()
}

// HandleCustomDataType returns false by default (no custom handling).
func (p *GormPlugin) HandleCustomDataType(value string, columnType string, isNullable bool) (any, bool, error) {
	return nil, false, nil
}

// IsArrayType returns false by default.
// PostgreSQL overrides this to detect underscore-prefixed array types.
func (p *GormPlugin) IsArrayType(columnType string) bool {
	return false
}

// ResolveGraphSchema returns the schema parameter unchanged by default.
// ClickHouse overrides this to return the database name.
func (p *GormPlugin) ResolveGraphSchema(config *engine.PluginConfig, schema string) string {
	return schema
}

// ShouldCheckRowsAffected returns true by default.
// ClickHouse overrides this to return false since its GORM driver
// doesn't report affected rows for mutations.
func (p *GormPlugin) ShouldCheckRowsAffected() bool {
	return true
}
