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
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

func (p *GormPlugin) UpdateStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string, updatedColumns []string) (bool, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		pkColumns, err := p.GetPrimaryKeyColumns(db, schema, storageUnit)
		if err != nil {
			pkColumns = []string{}
		}

		columnTypes, err := p.GetColumnTypes(db, schema, storageUnit)
		if err != nil {
			return false, err
		}

		conditions := make(map[string]interface{})
		convertedValues := make(map[string]interface{})
		unchangedValues := make(map[string]interface{})

		for column, strValue := range values {
			columnType, exists := columnTypes[column]
			if !exists {
				return false, fmt.Errorf("column '%s' does not exist in table %s", column, storageUnit)
			}

			convertedValue, err := p.ConvertStringValue(strValue, columnType)
			if err != nil {
				convertedValue = strValue // use the original value if conversion fails?
			}

			targetColumn := column
			if common.ContainsString(pkColumns, column) {
				conditions[targetColumn] = convertedValue
			} else if common.ContainsString(updatedColumns, column) {
				convertedValues[targetColumn] = convertedValue
			} else {
				// Store unchanged values for WHERE clause if no PKs
				unchangedValues[targetColumn] = convertedValue
			}
		}

		// If no columns to update, return early
		if len(convertedValues) == 0 {
			return true, nil
		}

		schema = p.EscapeIdentifier(schema)
		storageUnit = p.EscapeIdentifier(storageUnit)
		tableName := p.FormTableName(schema, storageUnit)

		var result *gorm.DB
		if len(conditions) == 0 {
			if p.Type == engine.DatabaseType_MySQL || p.Type == engine.DatabaseType_MariaDB {
				result = p.executeUpdateWithWhereMap(db, tableName, unchangedValues, convertedValues)
			} else {
				result = db.Table(tableName).Where(unchangedValues).Updates(convertedValues)
			}
		} else {
			if p.Type == engine.DatabaseType_MySQL || p.Type == engine.DatabaseType_MariaDB {
				result = p.executeUpdateWithWhereMap(db, tableName, conditions, convertedValues)
			} else {
				result = db.Table(tableName).Where(conditions, nil).Updates(convertedValues)
			}
		}

		if result.Error != nil {
			return false, result.Error
		}

		// todo: investigate why the clickhouse driver doesnt show any updated rows after an update
		if p.Type != engine.DatabaseType_ClickHouse && result.RowsAffected == 0 {
			return false, errors.New("no rows were updated")
		}

		return true, nil
	})
}

// weird bug for mysql/mariadb driver where the where clause is not properly escaped, so have to do it manually below
// should be fine as it still uses the query builder
// todo: need to investigate underlying driver to see what's going on
func (p *GormPlugin) executeUpdateWithWhereMap(db *gorm.DB, tableName string, whereConditions map[string]interface{}, updateValues map[string]interface{}) *gorm.DB {
	query := db.Table(tableName)

	for column, value := range whereConditions {
		escapedColumn := p.EscapeIdentifier(column)
		query = query.Where(fmt.Sprintf("%s = ?", escapedColumn), value)
	}

	return query.Updates(updateValues)
}
