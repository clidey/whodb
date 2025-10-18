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

package sqlite3

import (
	"gorm.io/gorm"
)

const graphQuery = `
WITH fk_constraints AS (
    SELECT DISTINCT
        m.name AS table1,
        f."table" AS table2,
        'OneToMany' AS relation,
        f."from" AS source_column,
        f."to" AS target_column
    FROM
        sqlite_master m,
        pragma_foreign_key_list(m.name) f
),
pk_constraints AS (
    SELECT DISTINCT
        p.name AS table1,
        m.name AS table2,
        'OneToOne' AS relation,
        NULL AS source_column,
        NULL AS target_column
    FROM
        sqlite_master p,
        pragma_table_info(p.name) t
    JOIN
        sqlite_master m
    ON
        t.name = m.name
    WHERE
        t.pk = 1
        AND p.name != m.name
),
unique_constraints AS (
    SELECT DISTINCT
        p.name AS table1,
        i."table" AS table2,
        'ManyToOne' AS relation,
        NULL AS source_column,
        NULL AS target_column
    FROM
        sqlite_master p
    JOIN
        (SELECT m.name AS "table", i."unique" AS "unique"
         FROM sqlite_master m, pragma_index_list(m.name) i) i
    ON
        p.name = i."table"
    WHERE
        i."unique" = 1
        AND p.name != i."table"
),
many_to_many_constraints AS (
    SELECT DISTINCT
        k1."table" AS table1,
        k2."table" AS table2,
        'ManyToMany' AS relation,
        NULL AS source_column,
        NULL AS target_column
    FROM
        (SELECT f."table", f.seq
         FROM sqlite_master m, pragma_foreign_key_list(m.name) f) k1
    JOIN
        (SELECT f."table", f.seq
         FROM sqlite_master m, pragma_foreign_key_list(m.name) f) k2
    ON
        k1."table" = k2."table"
    WHERE
        k1.seq = 0 AND k2.seq = 1
)
SELECT * FROM fk_constraints
UNION ALL
SELECT * FROM pk_constraints
UNION ALL
SELECT * FROM unique_constraints
UNION ALL
SELECT * FROM many_to_many_constraints;
`

func (p *Sqlite3Plugin) GetGraphQueryDB(db *gorm.DB, schema string) *gorm.DB {
	return db.Raw(graphQuery)
}
