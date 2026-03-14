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
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
	"slices"
)

func (p *GormPlugin) UpdateStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string, updatedColumns []string) (bool, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		pkColumns, err := p.GormPluginFunctions.GetPrimaryKeyColumns(db, schema, storageUnit)
		if err != nil {
			log.WithError(err).Error(fmt.Sprintf("Failed to get primary key columns for table %s.%s during update operation", schema, storageUnit))
			pkColumns = []string{}
		}

		columnTypeInfos, err := p.GormPluginFunctions.GetColumnTypes(db, schema, storageUnit)
		if err != nil {
			log.WithError(err).Error(fmt.Sprintf("Failed to get column types for table %s.%s during update operation", schema, storageUnit))
			return false, err
		}

		conditions := make(map[string]any)
		convertedValues := make(map[string]any)
		unchangedValues := make(map[string]any)

		for column, strValue := range values {
			isPK := slices.Contains(pkColumns, column)
			isUpdated := slices.Contains(updatedColumns, column)

			// Only convert columns we actually need:
			// - PK columns (for WHERE clause)
			// - Updated columns (for SET clause)
			// - Unchanged columns only if no PKs exist (fallback WHERE)
			needsConversion := isPK || isUpdated || len(pkColumns) == 0

			if !needsConversion {
				continue
			}

			colInfo, exists := columnTypeInfos[column]
			if !exists {
				return false, fmt.Errorf("column '%s' does not exist in table %s", column, storageUnit)
			}

			convertedValue, err := p.GormPluginFunctions.ConvertStringValue(strValue, colInfo.Type, colInfo.IsNullable)
			if err != nil {
				log.WithError(err).Error(fmt.Sprintf("Failed to convert string value '%s' for column '%s' during update of table %s.%s", strValue, column, schema, storageUnit))
				convertedValue = strValue
			}

			// GORM handles identifier escaping automatically
			if isPK {
				conditions[column] = convertedValue
			} else if isUpdated {
				convertedValues[column] = convertedValue
			} else {
				// Store unchanged values for WHERE clause if no PKs
				unchangedValues[column] = convertedValue
			}
		}

		// If no columns to update, return early
		if len(convertedValues) == 0 {
			return true, nil
		}

		// Use SQL builder for update query
		builder := p.GormPluginFunctions.CreateSQLBuilder(db)

		// Use SQLBuilder for consistent behavior for all database types.
		var whereConditions map[string]any
		if len(conditions) == 0 {
			whereConditions = unchangedValues
		} else {
			whereConditions = conditions
		}

		result := builder.UpdateQuery(schema, storageUnit, convertedValues, whereConditions)

		if result.Error != nil {
			log.WithError(result.Error).Error(fmt.Sprintf("Failed to update rows in table %s.%s", schema, storageUnit))
			return false, result.Error
		}

		// ClickHouse GORM driver doesn't report affected rows for UPDATE mutations
		if p.Type != engine.DatabaseType_ClickHouse && result.RowsAffected == 0 {
			return false, errors.New("no rows were updated")
		}

		return true, nil
	})
}
