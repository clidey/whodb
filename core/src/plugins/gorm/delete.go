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

func (p *GormPlugin) DeleteRow(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	return plugins.WithConnection[bool](config, p.DB, func(db *gorm.DB) (bool, error) {
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
		for column, strValue := range values {
			columnType, exists := columnTypes[column]
			if !exists {
				return false, fmt.Errorf("column '%s' does not exist in table %s", column, storageUnit)
			}

			convertedValue, err := p.ConvertStringValue(strValue, columnType)
			if err != nil {
				convertedValue = strValue // use string value if conversion fails?
			}

			targetColumn := column
			if common.ContainsString(pkColumns, column) {
				conditions[targetColumn] = convertedValue
			} else {
				convertedValues[targetColumn] = convertedValue
			}
		}

		schema = p.EscapeIdentifier(schema)
		storageUnit = p.EscapeIdentifier(storageUnit)
		tableName := p.FormTableName(schema, storageUnit)

		var result *gorm.DB
		if len(conditions) == 0 {
			result = db.Table(tableName).Where(convertedValues).Delete(nil)
		} else {
			result = db.Table(tableName).Where(conditions).Delete(nil)
		}

		if result.Error != nil {
			return false, result.Error
		}

		// todo: investigate why the clickhouse driver doesnt show any updated rows after a delete
		if p.Type != engine.DatabaseType_ClickHouse && result.RowsAffected == 0 {
			return false, errors.New("no rows were deleted")
		}

		return true, nil
	})
}
