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
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
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

// GetColumnTypes gets column types using Migrator's ColumnTypes
func (m *MigratorHelper) GetColumnTypes(tableName string) (map[string]string, error) {
	columnTypes := make(map[string]string)

	// Try to use GORM's Migrator ColumnTypes method
	types, err := m.migrator.ColumnTypes(tableName)
	if err != nil {
		// Fall back to raw SQL if Migrator doesn't work
		return m.getColumnTypesRaw(tableName)
	}

	for _, col := range types {
		columnTypes[col.Name()] = strings.ToUpper(col.DatabaseTypeName())
	}

	return columnTypes, nil
}

// GetOrderedColumns returns columns in their definition order
func (m *MigratorHelper) GetOrderedColumns(tableName string) ([]engine.Column, error) {
	// Try to use GORM's Migrator ColumnTypes method
	types, err := m.migrator.ColumnTypes(tableName)
	if err != nil {
		// Fall back to raw SQL if Migrator doesn't work
		return m.getOrderedColumnsRaw(tableName)
	}

	columns := make([]engine.Column, 0, len(types))
	for _, col := range types {
		columns = append(columns, engine.Column{
			Name: col.Name(),
			Type: strings.ToUpper(col.DatabaseTypeName()),
		})
	}

	return columns, nil
}

// getOrderedColumnsRaw falls back to raw SQL for ordered columns
func (m *MigratorHelper) getOrderedColumnsRaw(tableName string) ([]engine.Column, error) {
	var columns []engine.Column

	// Extract schema and table name
	parts := strings.Split(tableName, ".")
	var schema, table string
	if len(parts) == 2 {
		schema = parts[0]
		table = parts[1]
	} else {
		table = tableName
	}

	query := m.plugin.GetColTypeQuery()

	var rows *sql.Rows
	var err error

	if m.plugin.GetDatabaseType() == engine.DatabaseType_Sqlite3 {
		rows, err = m.db.Raw(query, table).Rows()
	} else {
		rows, err = m.db.Raw(query, schema, table).Rows()
	}

	if err != nil {
		log.Logger.WithError(err).WithField("table", tableName).Error("Failed to execute column types query")
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var columnName, dataType string
		if err := rows.Scan(&columnName, &dataType); err != nil {
			log.Logger.WithError(err).Error("Failed to scan column type")
			return nil, err
		}
		columns = append(columns, engine.Column{
			Name: columnName,
			Type: strings.ToUpper(dataType),
		})
	}

	return columns, nil
}

// getColumnTypesRaw falls back to raw SQL for column types
func (m *MigratorHelper) getColumnTypesRaw(tableName string) (map[string]string, error) {
	columnTypes := make(map[string]string)

	// Extract schema and table name
	parts := strings.Split(tableName, ".")
	var schema, table string
	if len(parts) == 2 {
		schema = parts[0]
		table = parts[1]
	} else {
		table = tableName
	}

	query := m.plugin.GetColTypeQuery()

	var rows *sql.Rows
	var err error

	if m.plugin.GetDatabaseType() == engine.DatabaseType_Sqlite3 {
		rows, err = m.db.Raw(query, table).Rows()
	} else {
		rows, err = m.db.Raw(query, schema, table).Rows()
	}

	if err != nil {
		log.Logger.WithError(err).WithField("table", tableName).Error("Failed to execute column types query")
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var columnName, dataType string
		if err := rows.Scan(&columnName, &dataType); err != nil {
			log.Logger.WithError(err).WithField("table", tableName).Error("Failed to scan column type data")
			return nil, err
		}
		columnTypes[columnName] = dataType
	}

	if err := rows.Err(); err != nil {
		log.Logger.WithError(err).WithField("table", tableName).Error("Row iteration error while getting column types")
		return nil, err
	}

	return columnTypes, nil
}
