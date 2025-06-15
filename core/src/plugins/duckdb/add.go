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

package duckdb

import (
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

func (p *DuckDBPlugin) AddStorageUnitRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) error {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) error {
		tableName := p.FormTableName(schema, storageUnit)
		
		data, err := p.ConvertRecordValuesToMap(values)
		if err != nil {
			return err
		}
		
		return db.Table(tableName).Create(data).Error
	})
}

func (p *DuckDBPlugin) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, columns []engine.Record) error {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) error {
		query := p.GetCreateTableQuery(schema, storageUnit, columns)
		return db.Exec(query).Error
	})
}