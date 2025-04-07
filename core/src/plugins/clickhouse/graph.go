// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clickhouse

import "gorm.io/gorm"

const graphQuery = `
WITH dependencies AS (
    SELECT 
        table AS Table1,
        referenced_table AS Table2,
        CASE
            WHEN constraint_type IN ('PRIMARY KEY', 'UNIQUE') 
            AND referenced_constraint_type IN ('PRIMARY KEY', 'UNIQUE') 
            THEN 'OneToOne'
            WHEN constraint_type IN ('PRIMARY KEY', 'UNIQUE') 
            THEN 'OneToMany'
            WHEN referenced_constraint_type IN ('PRIMARY KEY', 'UNIQUE') 
            THEN 'ManyToOne'
            ELSE 'ManyToMany'
        END AS Relation
    FROM system.tables t
    JOIN system.columns c ON t.database = c.database AND t.name = c.table
    WHERE c.type LIKE '%Nullable%'
    AND t.database = ?
)
SELECT DISTINCT * FROM dependencies
WHERE Table1 != Table2;
`

func (p *ClickHousePlugin) GetGraphQueryDB(db *gorm.DB, schema string) *gorm.DB {
	return db.Raw(graphQuery, schema)
}
