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
	"errors"
	"fmt"
	"strconv"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

func (p *GormPlugin) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, fields []engine.Record) (bool, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		if len(fields) == 0 {
			return false, errors.New("no fields provided for table creation")
		}

		// Check if table already exists using Migrator
		migrator := NewMigratorHelper(db, p)
		var fullTableName string
		if schema != "" && p.Type != engine.DatabaseType_Sqlite3 {
			fullTableName = schema + "." + storageUnit
		} else {
			fullTableName = storageUnit
		}

		if migrator.TableExists(fullTableName) {
			return false, fmt.Errorf("table %s already exists", fullTableName)
		}

		var columns []engine.Record
		for _, fieldType := range fields {
			if !p.GetSupportedColumnDataTypes().Contains(fieldType.Value) {
				return false, fmt.Errorf("data type: %s not supported by: %s", fieldType.Value, p.Plugin.Type)
			}

			// Keep original field name without quoting for column definition
			fieldName := fieldType.Key
			primaryKey, err := strconv.ParseBool(fieldType.Extra["Primary"])
			if err != nil {
				log.Logger.WithError(err).Error(fmt.Sprintf("Failed to parse Primary key flag for field %s in table %s.%s", fieldType.Key, schema, storageUnit))
				primaryKey = false
			}
			nullable, err := strconv.ParseBool(fieldType.Extra["Nullable"])
			if err != nil {
				log.Logger.WithError(err).Error(fmt.Sprintf("Failed to parse Nullable flag for field %s in table %s.%s", fieldType.Key, schema, storageUnit))
				nullable = false
			}

			// Validate: Primary keys cannot be nullable
			if primaryKey && nullable {
				return false, fmt.Errorf("column %s cannot be both primary key and nullable", fieldType.Key)
			}

			columns = append(columns, engine.Record{
				Key:   fieldName,
				Value: fieldType.Value,
				Extra: map[string]string{
					"primary":  strconv.FormatBool(primaryKey),
					"nullable": strconv.FormatBool(nullable),
				},
			})
		}

		createTableQuery := p.GetCreateTableQuery(db, schema, storageUnit, columns)

		if err := db.Exec(createTableQuery).Error; err != nil {
			log.Logger.WithError(err).Error(fmt.Sprintf("Failed to create table %s.%s with query: %s", schema, storageUnit, createTableQuery))
			return false, err
		}
		return true, nil
	})
}

// addRowWithDB performs the actual row insertion using the provided database connection
func (p *GormPlugin) addRowWithDB(db *gorm.DB, schema string, storageUnit string, values []engine.Record) error {
	if len(values) == 0 {
		return errors.New("no values provided to insert into the table")
	}

	if storageUnit == "" {
		return fmt.Errorf("storage unit name cannot be empty")
	}

	// Fetch column types to ensure proper type conversion
	columnTypes, err := p.GetColumnTypes(db, schema, storageUnit)
	if err != nil {
		log.Logger.WithError(err).WithField("schema", schema).WithField("storageUnit", storageUnit).
			Warn("Failed to fetch column types, continuing without type information")
		columnTypes = make(map[string]string)
	}

	// Populate type information in values if not already present
	for i, value := range values {
		if values[i].Extra == nil {
			values[i].Extra = make(map[string]string)
		}
		if values[i].Extra["Type"] == "" {
			if colType, ok := columnTypes[value.Key]; ok {
				values[i].Extra["Type"] = colType
			}
		}
	}

	// Use SQL builder for consistent insert operations
	builder := p.GormPluginFunctions.CreateSQLBuilder(db)

	valuesToAdd, err := p.ConvertRecordValuesToMap(values)
	if err != nil {
		return err
	}

	// Use SQLBuilder's InsertRow method which uses GORM's native Create
	return builder.InsertRow(schema, storageUnit, valuesToAdd)
}

func (p *GormPlugin) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (bool, error) {
	// Initialize error handler if needed
	if p.errorHandler == nil {
		p.InitPlugin()
	}

	// Log for debugging
	if storageUnit == "" {
		log.Logger.Error("AddRow called with empty storageUnit name")
		return false, fmt.Errorf("storage unit name cannot be empty")
	}

	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		err := p.addRowWithDB(db, schema, storageUnit, values)
		if err != nil {
			// Use error handler for user-friendly error messages
			err = p.errorHandler.HandleError(err, "AddRow", map[string]any{
				"schema":      schema,
				"storageUnit": storageUnit,
				"valueCount":  len(values),
			})
		}
		return err == nil, err
	})
}
