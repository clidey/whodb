package clickhouse

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
SELECT
    table AS table1,
    'None' AS table2,
    'None' AS relation
FROM
    system.tables
WHERE
    database = '%v'
`

func (p *ClickHousePlugin) GetGraph(config *engine.PluginConfig, schema string) ([]engine.GraphUnit, error) {
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

	query := fmt.Sprintf(graphQuery, schema)
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
