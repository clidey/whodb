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
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

func (p *GormPlugin) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, fields []engine.Record) (bool, error) {
	return p.CreateStorageUnit(config, schema, engine.RecordsToObjectDefinition(storageUnit, fields))
}

// CreateStorageUnit creates a table from a normalized object definition.
func (p *GormPlugin) CreateStorageUnit(config *engine.PluginConfig, schema string, definition engine.ObjectDefinition) (bool, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		if len(definition.Columns) == 0 {
			return false, errors.New("no fields provided for table creation")
		}

		migrator := NewMigratorHelper(db, p.GormPluginFunctions)
		fullTableName := p.FormTableName(schema, definition.Name)
		if migrator.TableExists(fullTableName) {
			return false, fmt.Errorf("table %s already exists", fullTableName)
		}

		var columns []engine.Record
		metadata, _ := sourcecatalog.ResolveSessionMetadata(string(p.Type))
		for _, column := range definition.Columns {
			if err := engine.ValidateColumnType(column.Type, string(p.Type), metadata); err != nil {
				return false, err
			}

			if column.Nullable != nil && column.Primary && *column.Nullable {
				return false, fmt.Errorf("column %s cannot be both primary key and nullable", column.Name)
			}

			columns = append(columns, engine.ColumnDefinitionToRecord(column))
		}

		createTableQuery := p.GetCreateTableQuery(db, schema, definition.Name, columns)

		// codeql[go/sql-injection]: AddStorageUnit intentionally executes user-authored DDL for the requested table definition.
		if err := db.Exec(createTableQuery).Error; err != nil {
			log.WithError(err).Error(fmt.Sprintf("Failed to create table %s.%s with query: %s", schema, definition.Name, createTableQuery))
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
		return errors.New("storage unit name cannot be empty")
	}

	// Fetch column types to ensure proper type conversion
	columnTypeInfos, err := p.GormPluginFunctions.GetColumnTypes(db, schema, storageUnit)
	if err != nil {
		log.WithError(err).WithField("schema", schema).WithField("storageUnit", storageUnit).
			Warn("Failed to fetch column types, continuing without type information")
		columnTypeInfos = make(map[string]ColumnTypeInfo)
	}

	for i, value := range values {
		if values[i].Extra == nil {
			values[i].Extra = make(map[string]string)
		}
		if values[i].Extra["Type"] == "" {
			if colInfo, ok := columnTypeInfos[value.Key]; ok {
				values[i].Extra["Type"] = colInfo.Type
				if colInfo.IsNullable {
					values[i].Extra["IsNullable"] = "true"
				}
			}
		}
	}

	builder := p.GormPluginFunctions.CreateSQLBuilder(db)

	valuesToAdd, err := p.ConvertRecordValuesToMap(values)
	if err != nil {
		return err
	}

	return builder.InsertRow(schema, storageUnit, valuesToAdd)
}

func (p *GormPlugin) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (bool, error) {
	if p.errorHandler == nil {
		p.InitPlugin()
	}

	if storageUnit == "" {
		log.Error("AddRow called with empty storageUnit name")
		return false, errors.New("storage unit name cannot be empty")
	}

	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		err := p.addRowWithDB(db, schema, storageUnit, values)
		if err != nil {
			err = p.errorHandler.HandleError(err, "AddRow", map[string]any{
				"schema":      schema,
				"storageUnit": storageUnit,
				"valueCount":  len(values),
			})
		}
		return err == nil, err
	})
}

// AddRowReturningID inserts a row and returns the auto-generated ID.
// Returns 0 if the table has no auto-increment column or the database doesn't support it.
func (p *GormPlugin) AddRowReturningID(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (int64, error) {
	if p.errorHandler == nil {
		p.InitPlugin()
	}

	if storageUnit == "" {
		log.Error("AddRowReturningID called with empty storageUnit name")
		return 0, errors.New("storage unit name cannot be empty")
	}

	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (int64, error) {
		var lastID int64

		// Use a transaction to ensure INSERT and lastval() use the same connection.
		// This is critical for PostgreSQL where lastval() is session-specific.
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := p.addRowWithDB(tx, schema, storageUnit, values); err != nil {
				return err
			}

			var err error
			lastID, err = p.GormPluginFunctions.GetLastInsertID(tx)
			if err != nil {
				log.WithError(err).Warn("Failed to get last insert ID")
				return nil
			}

			return nil
		})

		if err != nil {
			err = p.errorHandler.HandleError(err, "AddRowReturningID", map[string]any{
				"schema":      schema,
				"storageUnit": storageUnit,
				"valueCount":  len(values),
			})
			return 0, err
		}

		return lastID, nil
	})
}
