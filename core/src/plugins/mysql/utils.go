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
	"fmt"
	"strconv"

	"gorm.io/gorm"
)

func getPrimaryKeyColumns(db *gorm.DB, schema string, tableName string) ([]string, error) {
	var primaryKeys []string
	query := `
		SELECT k.column_name
		FROM information_schema.table_constraints t
		JOIN information_schema.key_column_usage k
		USING (constraint_name, table_schema, table_name)
		WHERE t.constraint_type = 'PRIMARY KEY'
		AND t.table_schema = ?
		AND t.table_name = ?;
	`
	rows, err := db.Raw(query, schema, tableName).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var pkColumn string
		if err := rows.Scan(&pkColumn); err != nil {
			return nil, err
		}
		primaryKeys = append(primaryKeys, pkColumn)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(primaryKeys) == 0 {
		return nil, fmt.Errorf("no primary key found for table %s", tableName)
	}

	return primaryKeys, nil
}

func getColumnTypes(db *gorm.DB, schema, tableName string) (map[string]string, error) {
	columnTypes := make(map[string]string)
	query := `
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?;
	`
	rows, err := db.Raw(query, schema, tableName).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var columnName, dataType string
		if err := rows.Scan(&columnName, &dataType); err != nil {
			return nil, err
		}
		columnTypes[columnName] = dataType
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columnTypes, nil
}

func convertStringValue(value, columnType string) (interface{}, error) {
	switch columnType {
	case "int", "bigint", "smallint", "tinyint", "mediumint":
		return strconv.Atoi(value)
	case "boolean", "bit":
		return strconv.ParseBool(value)
	case "float", "double", "decimal":
		return strconv.ParseFloat(value, 64)
	default:
		return value, nil
	}
}
