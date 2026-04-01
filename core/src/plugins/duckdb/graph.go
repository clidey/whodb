/*
 * Copyright 2026 Clidey, Inc.
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

package duckdb

import "gorm.io/gorm"

// DuckDB's information_schema.constraint_column_usage does not report FK constraints.
// Use duckdb_constraints() which has explicit referenced_table/referenced_column_names.
const graphQuery = `
SELECT DISTINCT
    dc.referenced_table AS table1,
    dc.table_name AS table2,
    'OneToMany' AS relation,
    unnest(dc.constraint_column_names) AS source_column,
    unnest(dc.referenced_column_names) AS target_column
FROM duckdb_constraints() dc
WHERE dc.constraint_type = 'FOREIGN KEY'
    AND dc.schema_name = ?
`

func (p *DuckDBPlugin) GetGraphQueryDB(db *gorm.DB, schema string) *gorm.DB {
	return db.Raw(graphQuery, schema)
}
