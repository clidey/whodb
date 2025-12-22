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
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/gorm"
)

// MigratorHelper provides schema operations using GORM's Migrator interface
// NOTE: Most methods are not yet used but are prepared for future schema modification features
type MigratorHelper struct {
	db       *gorm.DB
	plugin   GormPluginFunctions
	migrator gorm.Migrator
}

// NewMigratorHelper creates a new migrator helper
func NewMigratorHelper(db *gorm.DB, plugin GormPluginFunctions) *MigratorHelper {
	return &MigratorHelper{
		db:       db,
		plugin:   plugin,
		migrator: db.Migrator(),
	}
}

// TableExists checks if a table exists using Migrator
func (m *MigratorHelper) TableExists(tableName string) bool {
	return m.migrator.HasTable(tableName)
}

// GetConstraints gets table constraints using Migrator
func (m *MigratorHelper) GetConstraints(tableName string) (map[string][]gorm.ColumnType, error) {
	// GORM's Migrator doesn't directly expose constraints
	// We can get column types which include some constraint info
	columnTypes, err := m.migrator.ColumnTypes(tableName)
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

// GetColumnTypes gets column types using Migrator's ColumnTypes.
// Returns types with length info when available (e.g., "VARCHAR(255)").
func (m *MigratorHelper) GetColumnTypes(tableName string) (map[string]string, error) {
	columnTypes := make(map[string]string)

	types, err := m.migrator.ColumnTypes(tableName)
	if err != nil {
		return nil, err
	}

	for _, col := range types {
		fullType := m.buildFullTypeName(col)
		// Normalize the type using the plugin's normalization
		normalizedType := m.plugin.NormalizeType(fullType)
		columnTypes[col.Name()] = normalizedType
	}

	return columnTypes, nil
}

// typesWithLength lists types where showing length is meaningful to users.
// These are types where users can specify a length/size when creating columns.
var typesWithLength = map[string]bool{
	// Character types
	"VARCHAR": true, "CHAR": true, "CHARACTER": true, "CHARACTER VARYING": true,
	"NVARCHAR": true, "NCHAR": true, "BPCHAR": true,
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
func (m *MigratorHelper) GetOrderedColumns(tableName string) ([]engine.Column, error) {
	types, err := m.migrator.ColumnTypes(tableName)
	if err != nil {
		return nil, err
	}

	columns := make([]engine.Column, 0, len(types))
	for _, col := range types {
		fullType := m.buildFullTypeName(col)
		normalizedType := m.plugin.NormalizeType(fullType)
		baseName := strings.ToUpper(col.DatabaseTypeName())

		column := engine.Column{
			Name: col.Name(),
			Type: normalizedType,
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
