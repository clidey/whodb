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

		schema = p.EscapeIdentifier(schema)
		storageUnit = p.EscapeIdentifier(storageUnit)

		columns := []engine.Record{}
		for _, fieldType := range fields {
			if !p.GetSupportedColumnDataTypes().Contains(fieldType.Value) {
				return false, fmt.Errorf("data type: %s not supported by: %s", fieldType.Value, p.Plugin.Type)
			}

			fieldName := p.EscapeIdentifier(fieldType.Key)
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

			columns = append(columns, engine.Record{
				Key:   fieldName,
				Value: fieldType.Value,
				Extra: map[string]string{
					"primary":  strconv.FormatBool(primaryKey),
					"nullable": strconv.FormatBool(nullable),
				},
			})
		}

		createTableQuery := p.GetCreateTableQuery(schema, storageUnit, columns)

		if err := db.Exec(createTableQuery).Error; err != nil {
			log.Logger.WithError(err).Error(fmt.Sprintf("Failed to create table %s.%s with query: %s", schema, storageUnit, createTableQuery))
			return false, err
		}
		return true, nil
	})
}

func (p *GormPlugin) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (bool, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		if len(values) == 0 {
			return false, errors.New("no values provided to insert into the table")
		}

		schema = p.EscapeIdentifier(schema)
		storageUnit = p.EscapeIdentifier(storageUnit)
		fullTableName := p.FormTableName(schema, storageUnit)

		valuesToAdd, err := p.ConvertRecordValuesToMap(values)
		if err != nil {
			log.Logger.WithError(err).Error(fmt.Sprintf("Failed to convert record values for insertion into table %s.%s", schema, storageUnit))
			return false, err
		}

		result := db.Table(fullTableName).Create(valuesToAdd)

		if result.Error != nil {
			log.Logger.WithError(result.Error).Error(fmt.Sprintf("Failed to insert row into table %s.%s", schema, storageUnit))
			return false, result.Error
		}

		return true, nil
	})
}
