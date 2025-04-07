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

package mongodb

import (
	"context"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"go.mongodb.org/mongo-driver/bson"
)

type tableRelation struct {
	Table1   string
	Table2   string
	Relation string
}

func (p *MongoDBPlugin) GetGraph(config *engine.PluginConfig, database string) ([]engine.GraphUnit, error) {
	ctx := context.Background()
	client, err := DB(config)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(ctx)

	db := client.Database(database)
	cursor, err := db.ListCollections(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	collections := []string{}
	collectionTypes := make(map[string]string)

	for cursor.Next(ctx) {
		var collectionInfo bson.M
		if err := cursor.Decode(&collectionInfo); err != nil {
			return nil, err
		}

		name, _ := collectionInfo["name"].(string)
		collectionType, _ := collectionInfo["type"].(string)

		collections = append(collections, name)
		collectionTypes[name] = collectionType
	}

	uniqueRelations := make(map[string]bool)
	relations := []tableRelation{}

	for _, collectionName := range collections {
		collectionType := collectionTypes[collectionName]

		if collectionType == "view" {
			continue
		}

		collection := db.Collection(collectionName)
		indexes, err := collection.Indexes().List(ctx)
		if err != nil {
			return nil, err
		}

		foreignKeys := make(map[string]bool)
		uniqueKeys := make(map[string]bool)

		for indexes.Next(ctx) {
			var index bson.M
			if err := indexes.Decode(&index); err != nil {
				return nil, err
			}

			keys, ok := index["key"].(bson.M)
			if !ok {
				continue
			}

			unique, _ := index["unique"].(bool)

			for key := range keys {
				for _, otherCollection := range collections {
					singularName := strings.TrimSuffix(otherCollection, "s")
					if key == singularName+"_id" || key == otherCollection+"_id" {
						foreignKeys[otherCollection] = true
						if unique {
							uniqueKeys[otherCollection] = true
						}
					}
				}
			}
		}

		for fk := range foreignKeys {
			relKey1 := collectionName + ":" + fk
			relKey2 := fk + ":" + collectionName

			if uniqueKeys[fk] {
				if !uniqueRelations[relKey1+":OneToOne"] {
					uniqueRelations[relKey1+":OneToOne"] = true
					relations = append(relations, tableRelation{
						Table1:   collectionName,
						Table2:   fk,
						Relation: "OneToOne",
					})
				}
			} else {
				if !uniqueRelations[relKey1+":OneToMany"] {
					uniqueRelations[relKey1+":OneToMany"] = true
					relations = append(relations, tableRelation{
						Table1:   fk,
						Table2:   collectionName,
						Relation: "OneToMany",
					})
				}

				if !uniqueRelations[relKey2+":ManyToOne"] {
					uniqueRelations[relKey2+":ManyToOne"] = true
					relations = append(relations, tableRelation{
						Table1:   collectionName,
						Table2:   fk,
						Relation: "ManyToOne",
					})
				}
			}
		}

		// todo: figure out Many-to-Many (Junction Table)
	}

	tableMap := make(map[string][]engine.GraphUnitRelationship)
	for _, tr := range relations {
		tableMap[tr.Table1] = append(tableMap[tr.Table1], engine.GraphUnitRelationship{
			Name:             tr.Table2,
			RelationshipType: engine.GraphUnitRelationshipType(tr.Relation),
		})
	}

	storageUnits, err := p.GetStorageUnits(config, database)
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
		tables = append(tables, engine.GraphUnit{
			Unit:      storageUnit,
			Relations: relations,
		})
	}

	return tables, nil
}
