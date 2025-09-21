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
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

// GetColumnConstraints gets column constraints using GORM's Migrator
func (p *GormPlugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (map[string]map[string]any, error) {
		// Build full table name
		var fullTableName string
		if schema != "" && p.Type != engine.DatabaseType_Sqlite3 {
			fullTableName = schema + "." + storageUnit
		} else {
			fullTableName = storageUnit
		}

		// Use Migrator to get constraints
		migrator := NewMigratorHelper(db, p)
		migratorConstraints, err := migrator.GetConstraints(fullTableName)
		if err != nil {
			// Fall back to empty constraints if Migrator fails
			// This maintains backward compatibility
			return make(map[string]map[string]any), nil
		}

		// Convert Migrator constraints to the expected format
		constraints := make(map[string]map[string]any)

		// Process each constraint type
		for constraintType, columns := range migratorConstraints {
			for _, col := range columns {
				columnName := col.Name()
				if constraints[columnName] == nil {
					constraints[columnName] = make(map[string]any)
				}

				// Map constraint types to properties
				switch constraintType {
				case "PRIMARY":
					constraints[columnName]["primary"] = true
				case "UNIQUE":
					constraints[columnName]["unique"] = true
				case "NOT_NULL":
					constraints[columnName]["nullable"] = false
				}

				// Add data type information
				constraints[columnName]["type"] = col.DatabaseTypeName()

				// Add additional constraint info if available
				if nullable, ok := col.Nullable(); ok {
					constraints[columnName]["nullable"] = nullable
				}
				if unique, ok := col.Unique(); ok {
					constraints[columnName]["unique"] = unique
				}
				if autoIncrement, ok := col.AutoIncrement(); ok && autoIncrement {
					constraints[columnName]["auto_increment"] = true
				}
				if defaultValue, ok := col.DefaultValue(); ok && defaultValue != "" {
					constraints[columnName]["default"] = defaultValue
				}
				if comment, ok := col.Comment(); ok && comment != "" {
					constraints[columnName]["comment"] = comment
				}
				if length, ok := col.Length(); ok && length > 0 {
					constraints[columnName]["length"] = length
				}
				if precision, scale, ok := col.DecimalSize(); ok {
					constraints[columnName]["precision"] = precision
					if scale > 0 {
						constraints[columnName]["scale"] = scale
					}
				}
			}
		}

		// If no constraints found via Migrator, try database-specific implementation
		if len(constraints) == 0 {
			// Each database plugin can still override this method for custom logic
			return p.getColumnConstraintsRaw(db, schema, storageUnit)
		}

		return constraints, nil
	})
}

// getColumnConstraintsRaw is a fallback for when Migrator doesn't provide constraints
func (p *GormPlugin) getColumnConstraintsRaw(db *gorm.DB, schema string, storageUnit string) (map[string]map[string]any, error) {
	// Default implementation - return empty constraints
	// Database-specific plugins should override this method
	return make(map[string]map[string]any), nil
}

// ClearTableData clears all data from a table
// clearTableDataWithDB performs the actual table data clearing using the provided database connection
func (p *GormPlugin) clearTableDataWithDB(db *gorm.DB, schema string, storageUnit string) error {
	// Use SQL builder for consistent delete operations
	builder := p.GormPluginFunctions.CreateSQLBuilder(db)
	// Delete all rows (empty conditions map means delete all)
	result := builder.DeleteQuery(schema, storageUnit, map[string]any{"1": 1})
	return result.Error
}

func (p *GormPlugin) ClearTableData(config *engine.PluginConfig, schema string, storageUnit string) (bool, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		err := p.clearTableDataWithDB(db, schema, storageUnit)
		return err == nil, err
	})
}
