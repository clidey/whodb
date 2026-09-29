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

package gorm_plugin

import (
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/sqlident"
)

// MigratorHelper provides schema metadata operations for safely scoped GORM table queries.
type MigratorHelper struct {
	plugin GormPluginFunctions
}

type quotedColumnTypeProvider interface {
	QuotedColumnTypes(db *gorm.DB, schema, table string) ([]gorm.ColumnType, error)
}

// NewMigratorHelper creates a new migrator helper
func NewMigratorHelper(plugin GormPluginFunctions) *MigratorHelper {
	return &MigratorHelper{
		plugin: plugin,
	}
}

// ReadSQLColumnTypes returns driver metadata from an already safely scoped table query.
func ReadSQLColumnTypes(query *gorm.DB) (map[string]*sql.ColumnType, error) {
	rows, err := query.Limit(1).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	result := make(map[string]*sql.ColumnType, len(columnTypes))
	for _, columnType := range columnTypes {
		result[columnType.Name()] = columnType
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// TableExists checks whether a safely scoped table query can be opened.
func (m *MigratorHelper) TableExists(query *gorm.DB) bool {
	rows, err := query.Limit(1).Rows()
	if err != nil {
		return false
	}
	defer func() { _ = rows.Close() }()
	_ = rows.Next()
	return rows.Err() == nil
}

// GetConstraints gets table constraints using column metadata.
func (m *MigratorHelper) GetConstraints(query *gorm.DB, schema, table string) (map[string][]gorm.ColumnType, error) {
	// GORM's Migrator doesn't directly expose constraints
	// We can get column types which include some constraint info
	columnTypes, err := m.columnTypes(query, schema, table)
	if err != nil {
		return nil, err
	}

	constraints := make(map[string][]gorm.ColumnType)
	for _, col := range columnTypes {
		// Check for various constraints
		if primary, ok := col.PrimaryKey(); ok && primary {
			constraints["PRIMARY"] = append(constraints["PRIMARY"], col)
		}
		if unique, ok := col.Unique(); ok && unique {
			constraints["UNIQUE"] = append(constraints["UNIQUE"], col)
		}
		if nullable, ok := col.Nullable(); ok && !nullable {
			constraints["NOT_NULL"] = append(constraints["NOT_NULL"], col)
		}
	}

	return constraints, nil
}

// GetColumnTypes gets column types and nullability using driver metadata.
// Returns types with length info when available (e.g., "VARCHAR(255)").
func (m *MigratorHelper) GetColumnTypes(query *gorm.DB, schema, table string) (map[string]ColumnTypeInfo, error) {
	columnTypes := make(map[string]ColumnTypeInfo)

	types, err := m.columnTypes(query, schema, table)
	if err != nil {
		return nil, err
	}

	for _, col := range types {
		fullType := m.buildFullTypeName(col)
		normalizedType := m.plugin.NormalizeType(fullType)

		isNullable := true // safe default: assume nullable
		if nullable, ok := col.Nullable(); ok {
			isNullable = nullable
		}

		columnTypes[col.Name()] = ColumnTypeInfo{
			Type:       normalizedType,
			IsNullable: isNullable,
		}
	}

	return columnTypes, nil
}

// ColumnTypeInfo holds a column's type string alongside its nullability.
// Used by CRUD operations that need both pieces of information for value conversion.
type ColumnTypeInfo struct {
	Type       string
	IsNullable bool
}

// typesWithLength lists types where showing length is meaningful to users.
// These are types where users can specify a length/size when creating columns.
var typesWithLength = map[string]bool{
	// Character types
	"VARCHAR": true, "CHAR": true, "CHARACTER": true, "CHARACTER VARYING": true,
	"VARCHAR2": true, "NVARCHAR": true, "NVARCHAR2": true, "NCHAR": true, "BPCHAR": true,
	// Binary types
	"BINARY": true, "VARBINARY": true, "BIT": true, "BIT VARYING": true, "VARBIT": true,
	// ClickHouse string types
	"FIXEDSTRING": true,
}

// typesWithPrecision lists types where showing precision/scale is meaningful.
var typesWithPrecision = map[string]bool{
	"DECIMAL": true, "NUMERIC": true, "NUMBER": true,
	// Some databases use FLOAT/DOUBLE with precision
	"FLOAT": true, "DOUBLE": true, "REAL": true,
}

// buildFullTypeName constructs the full type name including length/precision.
// Only appends length for types where it's user-specifiable (VARCHAR, CHAR, etc.)
// to avoid showing internal storage sizes for types like BOX or POLYGON.
func (m *MigratorHelper) buildFullTypeName(col gorm.ColumnType) string {
	baseName := strings.ToUpper(col.DatabaseTypeName())

	// Only show length for types where users can specify it
	if typesWithLength[baseName] {
		if length, ok := col.Length(); ok && length > 0 {
			return fmt.Sprintf("%s(%d)", baseName, length)
		}
	}

	// show precision/scale for decimal-like types
	if typesWithPrecision[baseName] {
		if precision, scale, ok := col.DecimalSize(); ok && precision > 0 {
			return fmt.Sprintf("%s(%d,%d)", baseName, precision, scale)
		}
	}

	return baseName
}

// GetOrderedColumns returns columns in their definition order.
// Returns types with length info when available and normalized to canonical form.
func (m *MigratorHelper) GetOrderedColumns(query *gorm.DB, schema, table string) ([]engine.Column, error) {
	types, err := m.columnTypes(query, schema, table)
	if err != nil {
		return nil, err
	}

	columns := make([]engine.Column, 0, len(types))
	for _, col := range types {
		fullType := m.buildFullTypeName(col)
		normalizedType := m.plugin.NormalizeType(fullType)
		baseName := strings.ToUpper(col.DatabaseTypeName())

		isAutoIncr, _ := col.AutoIncrement()

		isNullable := true // safe default: assume nullable
		if nullable, ok := col.Nullable(); ok {
			isNullable = nullable
		}

		column := engine.Column{
			Name:            col.Name(),
			Type:            normalizedType,
			IsNullable:      isNullable,
			IsAutoIncrement: isAutoIncr,
		}

		// Only extract length for types where it's user-specifiable
		if typesWithLength[baseName] {
			if length, ok := col.Length(); ok && length > 0 {
				l := int(length)
				column.Length = &l
			}
		}

		// Only extract precision/scale for decimal-like types
		// Always include scale (even if 0) because NUMBER(10,0) and NUMBER(10) are semantically different
		if typesWithPrecision[baseName] {
			if precision, scale, ok := col.DecimalSize(); ok && precision > 0 {
				p := int(precision)
				column.Precision = &p
				s := int(scale)
				column.Scale = &s
			}
		}

		columns = append(columns, column)
	}

	return columns, nil
}

func (m *MigratorHelper) columnTypes(query *gorm.DB, schema, table string) ([]gorm.ColumnType, error) {
	if sqlident.IsSimple(schema) && sqlident.IsSimple(table) {
		return query.Session(&gorm.Session{NewDB: true}).Migrator().ColumnTypes(m.plugin.FormTableName(schema, table))
	}
	provider, ok := m.plugin.(quotedColumnTypeProvider)
	if !ok {
		return nil, fmt.Errorf("column metadata requires connector-native lookup for quoted identifier %q", m.plugin.FormTableName(schema, table))
	}
	return provider.QuotedColumnTypes(query.Session(&gorm.Session{NewDB: true}), schema, table)
}
