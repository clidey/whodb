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
		SELECT k.COLUMN_NAME
		FROM information_schema.TABLE_CONSTRAINTS t
		JOIN information_schema.KEY_COLUMN_USAGE k
		ON k.CONSTRAINT_NAME = t.CONSTRAINT_NAME
		AND k.TABLE_SCHEMA = t.TABLE_SCHEMA
		AND k.TABLE_NAME = t.TABLE_NAME
		WHERE t.CONSTRAINT_TYPE = 'PRIMARY KEY'
		AND t.TABLE_SCHEMA = ?
		AND t.TABLE_NAME = ?
		ORDER BY k.ORDINAL_POSITION;
	`
}

func (p *MySQLPlugin) GetColTypeQuery() string {
	return `
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?;
	`
}
