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

package gorm_plugin

import (
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
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
		var tableRelations []tableRelations

		// TODO: BIG EDGE CASE - ClickHouse uses database name instead of schema
		if p.Type == engine.DatabaseType_ClickHouse {
			schema = config.Credentials.Database
		}
		if err := p.GetGraphQueryDB(db, schema).Scan(&tableRelations).Error; err != nil {
			log.Logger.WithError(err).Error(fmt.Sprintf("Failed to execute graph query for schema: %s", schema))
			return nil, err
		}

		tableMap := make(map[string][]engine.GraphUnitRelationship)
		for _, tr := range tableRelations {
			tableMap[tr.Table1] = append(tableMap[tr.Table1], engine.GraphUnitRelationship{Name: tr.Table2, RelationshipType: engine.GraphUnitRelationshipType(tr.Relation)})
		}

		storageUnits, err := p.GetStorageUnits(config, schema)
		if err != nil {
			log.Logger.WithError(err).Error(fmt.Sprintf("Failed to get storage units for graph generation in schema: %s", schema))
			return nil, err
		}

		storageUnitsMap := map[string]engine.StorageUnit{}
		for _, storageUnit := range storageUnits {
			storageUnitsMap[storageUnit.Name] = storageUnit
		}

		var tables []engine.GraphUnit
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
