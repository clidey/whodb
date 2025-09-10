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

package clickhouse

func (p *ClickHousePlugin) ConvertStringValueDuringMap(value, columnType string) (interface{}, error) {
	return p.ConvertStringValue(value, columnType)
}

// Identifier quoting handled by GORM Dialector

func (p *ClickHousePlugin) GetPrimaryKeyColQuery() string {
	return `
		SELECT name
		FROM system.columns
		WHERE database = ? AND table = ? AND is_in_primary_key = 1
	`
}

func (p *ClickHousePlugin) GetColTypeQuery() string {
	return `
		SELECT 
			name,
			type
		FROM system.columns
		WHERE database = ? AND table = ?`
}
