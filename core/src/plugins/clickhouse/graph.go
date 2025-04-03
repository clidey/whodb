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
