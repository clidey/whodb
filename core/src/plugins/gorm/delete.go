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

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

func (p *GormPlugin) DeleteRow(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		pkColumns, err := p.GormPluginFunctions.GetPrimaryKeyColumns(db, schema, storageUnit)
		if err != nil {
			log.Logger.WithError(err).Error(fmt.Sprintf("Failed to get primary key columns for table %s.%s during delete operation", schema, storageUnit))
			pkColumns = []string{}
		}

		columnTypes, err := p.GetColumnTypes(db, schema, storageUnit)
		if err != nil {
			log.Logger.WithError(err).Error(fmt.Sprintf("Failed to get column types for table %s.%s during delete operation", schema, storageUnit))
			return false, err
		}

		conditions := make(map[string]interface{})
		convertedValues := make(map[string]interface{})
		for column, strValue := range values {
			columnType, exists := columnTypes[column]
			if !exists {
				return false, fmt.Errorf("column '%s' does not exist in table %s", column, storageUnit)
			}

			convertedValue, err := p.GormPluginFunctions.ConvertStringValue(strValue, columnType)
			if err != nil {
				log.Logger.WithError(err).Error(fmt.Sprintf("Failed to convert string value '%s' for column '%s' during delete from table %s.%s", strValue, column, schema, storageUnit))
				convertedValue = strValue // use string value if conversion fails?
			}

			// GORM handles identifier escaping automatically
			if common.ContainsString(pkColumns, column) {
				conditions[column] = convertedValue
			} else {
				convertedValues[column] = convertedValue
			}
		}

		// Use SQL builder for consistent delete operations
		builder := p.GormPluginFunctions.CreateSQLBuilder(db)

		var whereConditions map[string]any
		if len(conditions) == 0 {
			whereConditions = convertedValues
		} else {
			whereConditions = conditions
		}

		result := builder.DeleteQuery(schema, storageUnit, whereConditions)

		if result.Error != nil {
			log.Logger.WithError(result.Error).Error(fmt.Sprintf("Failed to delete rows from table %s.%s", schema, storageUnit))
			return false, result.Error
		}

		// TODO: BIG EDGE CASE - ClickHouse driver doesn't report affected rows properly for DELETE
		if p.Type != engine.DatabaseType_ClickHouse && result.RowsAffected == 0 {
			return false, errors.New("no rows were deleted")
		}

		return true, nil
	})
}
