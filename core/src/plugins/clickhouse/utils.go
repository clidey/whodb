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

import "strings"

// Identifier quoting handled by GORM Dialector

func (p *ClickHousePlugin) GetPrimaryKeyColQuery() string {
	return `
		SELECT name
		FROM system.columns
		WHERE database = ? AND table = ? AND is_in_primary_key = 1
	`
}

// isMultiStatement checks if a SQL string contains multiple statements by
// stripping ClickHouse comment syntax and counting semicolons.
// (https://clickhouse.com/docs/sql-reference/syntax#comments):
func isMultiStatement(query string) bool {
	var b strings.Builder
	inBlock := false
	for i := 0; i < len(query); i++ {
		if inBlock {
			if i+1 < len(query) && query[i] == '*' && query[i+1] == '/' {
				inBlock = false
				i++
			}
			continue
		}
		if i+1 < len(query) && query[i] == '/' && query[i+1] == '*' {
			inBlock = true
			i++
			continue
		}
		if i+1 < len(query) && query[i] == '-' && query[i+1] == '-' {
			for i < len(query) && query[i] != '\n' {
				i++
			}
			continue
		}
		if query[i] == '#' {
			for i < len(query) && query[i] != '\n' {
				i++
			}
			continue
		}
		b.WriteByte(query[i])
	}
	return strings.Count(strings.TrimSpace(b.String()), ";") > 1
}
