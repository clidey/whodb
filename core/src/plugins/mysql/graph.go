package mysql

import "gorm.io/gorm"

const graphQuery = `
WITH ForeignKeyRelations AS (
    SELECT 
        rc.TABLE_NAME AS Table1,
        rc.REFERENCED_TABLE_NAME AS Table2
    FROM 
        INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS rc
    WHERE 
        rc.CONSTRAINT_SCHEMA = ?
)
SELECT DISTINCT
    fkr.Table1,
    fkr.Table2,
    CASE
        WHEN pk2.CONSTRAINT_TYPE = 'PRIMARY KEY' THEN 'ManyToOne'  -- Table1 has a FK referencing a PK in Table2
        ELSE 'Unknown'
    END AS Relation
FROM 
    ForeignKeyRelations fkr
    JOIN INFORMATION_SCHEMA.TABLE_CONSTRAINTS pk1 
        ON fkr.Table1 = pk1.TABLE_NAME 
        AND pk1.CONSTRAINT_TYPE = 'FOREIGN KEY'
    JOIN INFORMATION_SCHEMA.TABLE_CONSTRAINTS pk2 
        ON fkr.Table2 = pk2.TABLE_NAME 
        AND pk2.CONSTRAINT_TYPE = 'PRIMARY KEY'
WHERE 
    fkr.Table1 <> fkr.Table2;
`

func (p *MySQLPlugin) GetGraphQueryDB(db *gorm.DB, schema string) *gorm.DB {
	return db.Raw(graphQuery, schema)
}
