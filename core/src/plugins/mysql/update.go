// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package mysql

import (
	"errors"
	"fmt"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

func (p *MySQLPlugin) UpdateStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	db, err := DB(config)
	if err != nil {
		return false, err
	}

	sqlDb, err := db.DB()
	if err != nil {
		return false, err
	}
	defer sqlDb.Close()

	pkColumns, err := getPrimaryKeyColumns(db, schema, storageUnit)
	if err != nil {
		return false, err
	}

	columnTypes, err := getColumnTypes(db, schema, storageUnit)
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

		convertedValue, err := convertStringValue(strValue, columnType)
		if err != nil {
			return false, fmt.Errorf("failed to convert value for column '%s': %v", column, err)
		}

		if common.ContainsString(pkColumns, column) {
			conditions[column] = convertedValue
		} else {
			convertedValues[column] = convertedValue
		}
	}

	tableName := fmt.Sprintf("%s.%s", schema, storageUnit)
	dbConditions := db.Table(tableName)
	for key, value := range conditions {
		dbConditions = dbConditions.Where(fmt.Sprintf("`%s` = ?", key), value)
	}

	result := dbConditions.Table(tableName).Updates(convertedValues)
	if result.Error != nil {
		return false, result.Error
	}

	if result.RowsAffected == 0 {
		return false, errors.New("no rows were updated")
	}

	return true, nil
}
