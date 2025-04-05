package gorm_plugin

import (
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

type tableRelations struct {
	Table1   string
	Table2   string
	Relation string
}

func (p *GormPlugin) GetGraph(config *engine.PluginConfig, schema string) ([]engine.GraphUnit, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]engine.GraphUnit, error) {
		tableRelations := []tableRelations{}

		if p.Type == engine.DatabaseType_ClickHouse {
			schema = config.Credentials.Database
		}
		if err := p.GetGraphQueryDB(db, schema).Scan(&tableRelations).Error; err != nil {
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
	})
}
