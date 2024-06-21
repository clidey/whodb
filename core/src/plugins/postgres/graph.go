package postgres

import (
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
)

type tableRelations struct {
	Table1   string
	Table2   string
	Relation string
}

const GraphQuery = `
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
        AND tc.table_schema = '%v'
        AND ccu.table_schema = '%v'
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
        AND tc.table_schema = '%v'
        AND ccu.table_schema = '%v'
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
        AND tc.table_schema = '%v'
        AND ccu.table_schema = '%v'
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
        AND kcu1.table_schema = '%v'
        AND kcu2.table_schema = '%v'
)
SELECT * FROM fk_constraints
UNION
SELECT * FROM pk_constraints
UNION
SELECT * FROM unique_constraints
UNION
SELECT * FROM many_to_many_constraints
`

func (p *PostgresPlugin) GetGraph(config *engine.PluginConfig, schema string) ([]engine.GraphUnit, error) {
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

	query := fmt.Sprintf(GraphQuery, schema, schema, schema, schema, schema, schema, schema, schema)
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
