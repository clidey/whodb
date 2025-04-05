package postgres

import "gorm.io/gorm"

const graphQuery = `
WITH fk_constraints AS (
    SELECT DISTINCT
        ccu.table_name AS table1,
        tc.table_name AS table2,
        'OneToMany' AS relation
    FROM 
        information_schema.table_constraints AS tc
    JOIN 
        information_schema.key_column_usage AS kcu
    ON 
        tc.constraint_name = kcu.constraint_name
    JOIN 
        information_schema.constraint_column_usage AS ccu
    ON 
        ccu.constraint_name = tc.constraint_name
    WHERE 
        tc.constraint_type = 'FOREIGN KEY'
        AND tc.table_schema = ?
        AND ccu.table_schema = ?
),
pk_constraints AS (
    SELECT DISTINCT
        tc.table_name AS table1,
        ccu.table_name AS table2,
        'OneToOne' AS relation
    FROM 
        information_schema.table_constraints AS tc
    JOIN 
        information_schema.key_column_usage AS kcu
    ON 
        tc.constraint_name = kcu.constraint_name
    JOIN 
        information_schema.constraint_column_usage AS ccu
    ON 
        ccu.constraint_name = tc.constraint_name
    WHERE 
        tc.constraint_type = 'PRIMARY KEY'
        AND tc.table_schema = ?
        AND ccu.table_schema = ?
        AND tc.table_name != ccu.table_name
),
unique_constraints AS (
    SELECT DISTINCT
        tc.table_name AS table1,
        ccu.table_name AS table2,
        'ManyToOne' AS relation
    FROM 
        information_schema.table_constraints AS tc
    JOIN 
        information_schema.key_column_usage AS kcu
    ON 
        tc.constraint_name = kcu.constraint_name
    JOIN 
        information_schema.constraint_column_usage AS ccu
    ON 
        ccu.constraint_name = tc.constraint_name
    WHERE 
        tc.constraint_type = 'UNIQUE'
        AND tc.table_schema = ?
        AND ccu.table_schema = ?
        AND tc.table_name != ccu.table_name
),
many_to_many_constraints AS (
    SELECT DISTINCT
        kcu1.table_name AS table1,
        kcu2.table_name AS table2,
        'ManyToMany' AS relation
    FROM
        information_schema.key_column_usage kcu1
    JOIN
        information_schema.referential_constraints rc
    ON
        kcu1.constraint_name = rc.constraint_name
    JOIN
        information_schema.key_column_usage kcu2
    ON
        kcu2.constraint_name = rc.unique_constraint_name
    WHERE
        kcu1.ordinal_position = 1 AND kcu2.ordinal_position = 2
        AND kcu1.table_schema = ?
        AND kcu2.table_schema = ?
)
SELECT * FROM fk_constraints
UNION
SELECT * FROM pk_constraints
UNION
SELECT * FROM unique_constraints
UNION
SELECT * FROM many_to_many_constraints
`

func (p *PostgresPlugin) GetGraphQueryDB(db *gorm.DB, schema string) *gorm.DB {
	return db.Raw(graphQuery, schema, schema, schema, schema, schema, schema, schema, schema)
}
