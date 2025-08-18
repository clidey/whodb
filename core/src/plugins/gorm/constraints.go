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

// GetColumnConstraints is the base implementation - plugins should override for database-specific logic
func (p *GormPlugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error) {
	constraints := make(map[string]map[string]any)

	// Default implementation - just return empty constraints
	// Each database plugin should override this with their specific implementation
	return constraints, nil
}

// ClearTableData clears all data from a table
// clearTableDataWithDB performs the actual table data clearing using the provided database connection
func (p *GormPlugin) clearTableDataWithDB(db *gorm.DB, schema string, storageUnit string) error {
	tableName := p.FormTableName(schema, storageUnit)
	result := db.Table(tableName).Where("1=1").Delete(nil)
	return result.Error
}

func (p *GormPlugin) ClearTableData(config *engine.PluginConfig, schema string, storageUnit string) (bool, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		err := p.clearTableDataWithDB(db, schema, storageUnit)
		return err == nil, err
	})
}
