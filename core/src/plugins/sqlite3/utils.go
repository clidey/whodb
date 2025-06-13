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

package sqlite3

import (
	"strings"
)

func (p *Sqlite3Plugin) ConvertStringValueDuringMap(value, columnType string) (interface{}, error) {
	return value, nil
}

func (p *Sqlite3Plugin) GetPrimaryKeyColQuery() string {
	return `
		SELECT p.name AS pk_column
		FROM sqlite_master m,
			 pragma_table_info(m.name) p
		WHERE m.type = 'table'
		  AND m.name = ?
		  AND m.name NOT LIKE 'sqlite_%'
		  AND p.pk > 0
		ORDER BY m.name, p.pk;`
}

func (p *Sqlite3Plugin) GetColTypeQuery() string {
	return `
		SELECT p.name AS column_name,
			   p.type AS data_type
		FROM sqlite_master m,
			 pragma_table_info(m.name) p
		WHERE m.type = 'table'
		  AND m.name NOT LIKE 'sqlite_%';
	`
}

func (p *Sqlite3Plugin) EscapeSpecificIdentifier(identifier string) string {
	identifier = strings.Replace(identifier, "\"", "\"\"", -1)
	return "\"" + identifier + "\""
}
