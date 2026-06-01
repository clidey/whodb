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

	"github.com/dromara/carbon/v2"
	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// ColumnCodec provides custom scanner allocation and value formatting for a raw SQL column type.
type ColumnCodec interface {
	Scanner() any
	Format(scanner any) (string, error)
}

// ColumnCodecFuncs adapts two small functions into a ColumnCodec.
type ColumnCodecFuncs struct {
	NewScanner  func() any
	FormatValue func(scanner any) (string, error)
}

// Scanner allocates a scan target for the codec.
func (c ColumnCodecFuncs) Scanner() any {
	if c.NewScanner == nil {
		return nil
	}
	return c.NewScanner()
}

// Format renders a scanned value for UI display.
func (c ColumnCodecFuncs) Format(scanner any) (string, error) {
	if c.FormatValue == nil {
		return "", nil
	}
	return c.FormatValue(scanner)
}

// ConvertRawToRows converts raw SQL rows into the engine's GetRowsResult format.
// Handles column metadata extraction (type, length, precision) and per-row value formatting
// including binary, geometry, time, and plugin-specific custom types.
func (p *GormPlugin) ConvertRawToRows(rows *sql.Rows) (*engine.GetRowsResult, error) {
	columns, typeMap, resultColumns, err := p.describeRawColumns(rows)
	if err != nil {
		return nil, err
	}

	result := &engine.GetRowsResult{
		Columns: resultColumns,
		Rows:    make([][]string, 0, 100),
	}

	for rows.Next() {
		row, err := p.scanRawRow(rows, columns, typeMap)
		if err != nil {
			return nil, err
		}

		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		log.WithError(err).Error("Failed while iterating result rows")
		return nil, err
	}

	result.TotalCount = int64(len(result.Rows))
	return result, nil
}

func (p *GormPlugin) describeRawColumns(rows *sql.Rows) ([]string, map[string]*sql.ColumnType, []engine.Column, error) {
	columns, err := rows.Columns()
	if err != nil {
		log.WithError(err).Error("Failed to get column names from result set")
		return nil, nil, nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		log.WithError(err).Error("Failed to get column types from result set")
		return nil, nil, nil, err
	}

	typeMap := make(map[string]*sql.ColumnType, len(columnTypes))
	for _, colType := range columnTypes {
		typeMap[colType.Name()] = colType
	}

	resultColumns := make([]engine.Column, 0, len(columns))
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

			if typesWithLength[baseTypeName] {
				if length, ok := colType.Length(); ok && length > 0 {
					l := int(length)
					column.Length = &l
					colTypeName = fmt.Sprintf("%s(%d)", colTypeName, length)
					column.Type = colTypeName
				}
			}

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

			resultColumns = append(resultColumns, column)
		}
	}

	return columns, typeMap, resultColumns, nil
}

func (p *GormPlugin) scanRawRow(rows *sql.Rows, columns []string, typeMap map[string]*sql.ColumnType) ([]string, error) {
	columnPointers := make([]any, len(columns))
	row := make([]string, len(columns))

	for i, col := range columns {
		colType := typeMap[col]
		typeName := ""
		if colType != nil {
			typeName = colType.DatabaseTypeName()
		}

		columnPointers[i] = CreateRawColumnScanner(p.GormPluginFunctions, typeName)
	}

	if err := rows.Scan(columnPointers...); err != nil {
		log.WithError(err).Error("Failed to scan row data")
		return nil, err
	}

	for i, colPtr := range columnPointers {
		colType := typeMap[columns[i]]
		typeName := ""
		if colType != nil {
			typeName = colType.DatabaseTypeName()
		}

		value, err := FormatScannedRawColumnValue(p.GormPluginFunctions, typeName, colPtr)
		if err != nil {
			row[i] = "ERROR: " + err.Error()
			continue
		}
		row[i] = value
	}

	return row, nil
}

// CreateRawColumnScanner returns the scan target for a raw SQL column type.
func CreateRawColumnScanner(plugin GormPluginFunctions, columnType string) any {
	if codec := plugin.GetColumnCodec(columnType); codec != nil {
		return codec.Scanner()
	}

	switch columnType {
	case "VARBINARY", "BINARY", "IMAGE", "BYTEA", "BLOB", "HIERARCHYID":
		return new(sql.RawBytes)
	default:
		if plugin.IsGeometryType(columnType) {
			return new(sql.RawBytes)
		}
		return new(sql.NullString)
	}
}

// FormatScannedRawColumnValue formats a scanned raw SQL column value for display.
func FormatScannedRawColumnValue(plugin GormPluginFunctions, columnType string, scanner any) (string, error) {
	if codec := plugin.GetColumnCodec(columnType); codec != nil {
		return codec.Format(scanner)
	}

	switch columnType {
	case "VARBINARY", "BINARY", "IMAGE", "BYTEA", "BLOB":
		rawBytes := scanner.(*sql.RawBytes)
		if rawBytes == nil || len(*rawBytes) == 0 {
			return "", nil
		}
		return "0x" + hex.EncodeToString(*rawBytes), nil
	case "TIME":
		val := scanner.(*sql.NullString)
		if !val.Valid {
			return "", nil
		}
		return formatTimeOnly(val.String), nil
	default:
		if plugin.IsGeometryType(columnType) {
			rawBytes := scanner.(*sql.RawBytes)
			if rawBytes == nil || len(*rawBytes) == 0 {
				return "", nil
			}
			if formatted := plugin.FormatGeometryValue(*rawBytes, columnType); formatted != "" {
				return formatted, nil
			}
			return "0x" + hex.EncodeToString(*rawBytes), nil
		}

		val := scanner.(*sql.NullString)
		if !val.Valid {
			return "", nil
		}
		return val.String, nil
	}
}

// FindMissingDataType resolves unknown column types by querying system catalogs.
func (p *GormPlugin) FindMissingDataType(db *gorm.DB, columnType string) string {
	if p.Type == engine.DatabaseType_Postgres {
		var typname string
		if err := db.Table("pg_type").
			Select("typname").
			Where("oid = ?", columnType).
			Scan(&typname).Error; err != nil {
			log.WithError(err).Error("Failed to find PostgreSQL type name for OID: " + columnType)
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

// GetColumnCodec returns nil by default.
func (p *GormPlugin) GetColumnCodec(columnType string) ColumnCodec {
	return nil
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
