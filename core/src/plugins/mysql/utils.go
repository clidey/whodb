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

package mysql

func (p *MySQLPlugin) ConvertStringValueDuringMap(value, columnType string) (interface{}, error) {
	return value, nil
}

// Identifier quoting handled by GORM Dialector

func (p *MySQLPlugin) GetPrimaryKeyColQuery() string {
	return `
		SELECT k.column_name
		FROM information_schema.table_constraints t
		JOIN information_schema.key_column_usage k
		USING (constraint_name, table_schema, table_name)
		WHERE t.constraint_type = 'PRIMARY KEY'
		AND t.table_schema = ?
		AND t.table_name = ?;
	`
}

func (p *MySQLPlugin) GetColTypeQuery() string {
	return `
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?;
	`
}
