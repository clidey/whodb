package sqlite3

import (
	"gorm.io/gorm"
)

const graphQuery = `
WITH fk_constraints AS (
    SELECT DISTINCT
        p.name AS table1,
        f."table" AS table2,
        'OneToMany' AS relation
    FROM 
        sqlite_master p
    JOIN 
        (SELECT m.name AS "table", f."table" AS "table2"
         FROM sqlite_master m, pragma_foreign_key_list(m.name) f) f
    ON 
        p.name = f."table2"
),
pk_constraints AS (
    SELECT DISTINCT
        p.name AS table1,
        m.name AS table2,
        'OneToOne' AS relation
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
        'ManyToOne' AS relation
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
        'ManyToMany' AS relation
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
