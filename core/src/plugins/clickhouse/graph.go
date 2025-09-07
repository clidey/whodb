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

import "gorm.io/gorm"

// TODO: clickhouse doesn't have relations. need to revisit this query.
// todo: right now it's meant to just recognize based on key names but that's for debug purposes.
const graphQuery = `
WITH candidate_keys AS (
    SELECT
        database,
        table,
        name AS column_name
    FROM system.columns
    WHERE database = ?
      AND (name LIKE '%_id' OR name = 'id')
),
relations AS (
    -- Many-to-One + One-to-One candidates
    SELECT
        fk.table AS table1,
        pk.table AS table2,
        fk.column_name AS fk_column,
        pk.column_name AS pk_column,
        CASE
            WHEN fk.column_name LIKE pk.table || '_id' THEN 'ManyToOne'
            ELSE 'PossibleRelation'
        END AS relation
    FROM candidate_keys fk
    JOIN candidate_keys pk
        ON fk.column_name LIKE pk.table || '_id'
       AND pk.column_name = 'id'
       AND fk.table != pk.table
),
join_tables AS (
    -- Many-to-Many candidates (tables with 2+ *_id columns and no "id" column)
    SELECT
        table AS table1,
        arrayJoin(arrayFilter(x -> x != 'id', groupArray(column_name))) AS fk_column,
        'JoinTable' AS relation
    FROM candidate_keys
    GROUP BY database, table
    HAVING countIf(column_name LIKE '%_id') >= 2
       AND countIf(column_name = 'id') = 0
)
SELECT *
FROM relations
UNION ALL
SELECT 
    jt.table1,
    splitByString('_id', jt.fk_column)[1] AS table2, -- infer target table from column name
    jt.fk_column AS fk_column,
    'id' AS pk_column,
    'ManyToMany' AS relation
FROM join_tables jt
ORDER BY relation, table1, table2;
`

func (p *ClickHousePlugin) GetGraphQueryDB(db *gorm.DB, schema string) *gorm.DB {
	return db.Raw(graphQuery, schema)
}
