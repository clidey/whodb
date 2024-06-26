package mysql

import (
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
)

type tableRelations struct {
	Table1   string
	Table2   string
	Relation string
}

const graphQuery = `
SELECT DISTINCT
    rc.TABLE_NAME AS Table1,
    rc.REFERENCED_TABLE_NAME AS Table2,
    CASE
        -- OneToOne: Both sides have unique constraints on the key columns
        WHEN tc1.CONSTRAINT_TYPE = 'PRIMARY KEY' AND tc2.CONSTRAINT_TYPE = 'PRIMARY KEY' THEN 'OneToOne'
        
        -- OneToMany: Table1 has a unique constraint on the key columns, but Table2 does not
        WHEN tc1.CONSTRAINT_TYPE = 'PRIMARY KEY' AND tc2.CONSTRAINT_TYPE = 'FOREIGN KEY' THEN 'OneToMany'
        
        -- ManyToOne: Table2 has a unique constraint on the key columns, but Table1 does not
        WHEN tc1.CONSTRAINT_TYPE = 'FOREIGN KEY' AND tc2.CONSTRAINT_TYPE = 'PRIMARY KEY' THEN 'ManyToOne'
        
        -- ManyToMany: Typically involves an additional join table (simplified assumption here)
        ELSE 'ManyToMany'
    END AS Relation
FROM
    INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS rc
    JOIN INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc1 ON rc.CONSTRAINT_NAME = tc1.CONSTRAINT_NAME
    JOIN INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc2 ON rc.UNIQUE_CONSTRAINT_NAME = tc2.CONSTRAINT_NAME
WHERE
    tc1.TABLE_SCHEMA = '%v'
    AND tc2.TABLE_SCHEMA = '%v'
    AND tc1.TABLE_NAME <> tc2.TABLE_NAME;
`

func (p *MySQLPlugin) GetGraph(config *engine.PluginConfig, schema string) ([]engine.GraphUnit, error) {
	db, err := DB(config)
	if err != nil {
		return nil, err
	}
	sqlDb, err := db.DB()
	if err != nil {
		return nil, err
	}
	defer sqlDb.Close()

	tableRelations := []tableRelations{}

	query := fmt.Sprintf(graphQuery, schema, schema)
	if err := db.Raw(query).Scan(&tableRelations).Error; err != nil {
		return nil, err
	}

	tableMap := make(map[string][]engine.GraphUnitRelationship)
	for _, tr := range tableRelations {
		tableMap[tr.Table1] = append(tableMap[tr.Table1], engine.GraphUnitRelationship{Name: tr.Table2, RelationshipType: engine.GraphUnitRelationshipType(tr.Relation)})
	}

	storageUnits, err := p.GetStorageUnits(config, schema)
	if err != nil {
		return nil, err
	}

	storageUnitsMap := map[string]engine.StorageUnit{}
	for _, storageUnit := range storageUnits {
		storageUnitsMap[storageUnit.Name] = storageUnit
	}

	tables := []engine.GraphUnit{}
	for _, storageUnit := range storageUnits {
		foundTable, ok := tableMap[storageUnit.Name]
		var relations []engine.GraphUnitRelationship
		if ok {
			relations = foundTable
		}
		tables = append(tables, engine.GraphUnit{Unit: storageUnit, Relations: relations})
	}

	return tables, nil
}
