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

package postgres

import (
	"errors"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
)

func (p *PostgresPlugin) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, fields map[string]string) (bool, error) {
	db, err := DB(config)
	if err != nil {
		return false, err
	}

	sqlDb, err := db.DB()
	if err != nil {
		return false, err
	}
	defer sqlDb.Close()

	var columns []string
	for field, fieldType := range fields {
		columns = append(columns, fmt.Sprintf("%s %s", field, fieldType))
	}

	createTableSQL := fmt.Sprintf("CREATE TABLE %s.%s (%s);", schema, storageUnit, strings.Join(columns, ", "))

	if err := db.Exec(createTableSQL).Error; err != nil {
		return false, err
	}

	return true, nil
}

func (p *PostgresPlugin) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (bool, error) {
	db, err := DB(config)
	if err != nil {
		return false, err
	}

	sqlDb, err := db.DB()
	if err != nil {
		return false, err
	}
	defer sqlDb.Close()

	if len(values) == 0 {
		return false, errors.New("no values provided to insert into the table")
	}

	columns := make([]string, 0, len(values))
	placeholders := make([]string, 0, len(values))
	args := make([]interface{}, 0, len(values))

	for _, value := range values {
		columns = append(columns, value.Key)
		if value.Extra["Config"] == "sql" {
			placeholders = append(placeholders, value.Value)
		} else {
			placeholders = append(placeholders, fmt.Sprintf("?::%v", value.Extra["Type"]))
			args = append(args, value.Value)
		}
	}

	insertSQL := fmt.Sprintf(
		"INSERT INTO %s.%s (%s) VALUES (%s);",
		schema, storageUnit,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	if err := db.Exec(insertSQL, args...).Error; err != nil {
		return false, err
	}

	return true, nil
}
