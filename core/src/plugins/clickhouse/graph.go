package clickhouse

import (
	"github.com/clidey/whodb/core/src/engine"
)

func (p *ClickHousePlugin) GetGraph(config *engine.PluginConfig, schema string) ([]engine.GraphUnit, error) {
	// ClickHouse doesn't have built-in support for relationships
	// We'll return a simple graph representation based on table structure
	storageUnits, err := p.GetStorageUnits(config, schema)
	if err != nil {
		return nil, err
	}

	var graphUnits []engine.GraphUnit
	for _, unit := range storageUnits {
		graphUnits = append(graphUnits, engine.GraphUnit{
			Unit:      unit,
			Relations: []engine.GraphUnitRelationship{},
		})
	}

	return graphUnits, nil
}
