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
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
	"strconv"
)

func (p *GormPlugin) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, fields []engine.Record) (bool, error) {
	return plugins.WithConnection[bool](config, p.DB, func(db *gorm.DB) (bool, error) {
		if len(fields) == 0 {
			return false, errors.New("no fields provided for table creation")
		}

		schema = p.EscapeIdentifier(schema)
		storageUnit = p.EscapeIdentifier(storageUnit)

		columns := []engine.Record{}
		for _, fieldType := range fields {
			if !p.GetSupportedColumnDataTypes().Contains(fieldType.Value) {
				return false, fmt.Errorf("data type: %s not supported by: %s", fieldType, p.Plugin.Type)
			}

			fieldName := p.EscapeIdentifier(fieldType.Key)
			primaryKey, err := strconv.ParseBool(fieldType.Extra["Primary"])
			if err != nil {
				primaryKey = false
			}
			nullable, err := strconv.ParseBool(fieldType.Extra["Nullable"])
			if err != nil {
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
			return false, err
		}
		return true, nil
	})
}

func (p *GormPlugin) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (bool, error) {
	return plugins.WithConnection[bool](config, p.DB, func(db *gorm.DB) (bool, error) {
		if len(values) == 0 {
			return false, errors.New("no values provided to insert into the table")
		}

		schema = p.EscapeIdentifier(schema)
		storageUnit = p.EscapeIdentifier(storageUnit)
		fullTableName := p.FormTableName(schema, storageUnit)

		valuesToAdd, err := p.ConvertRecordValuesToMap(values)
		if err != nil {
			return false, err
		}

		result := db.Table(fullTableName).Create(valuesToAdd)

		if result.Error != nil {
			return false, result.Error
		}

		return true, nil
	})
}
