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
	collections, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	relations := []tableRelation{}

	for _, collectionName := range collections {
		collection := db.Collection(collectionName)
		indexes, err := collection.Indexes().List(ctx)
		if err != nil {
			return nil, err
		}

		for indexes.Next(ctx) {
			var index bson.M
			if err := indexes.Decode(&index); err != nil {
				return nil, err
			}

			keys, ok := index["key"].(bson.M)
			if !ok {
				continue
			}

			for key := range keys {
				for _, otherCollection := range collections {
					if otherCollection != collectionName {
						singularName := strings.TrimSuffix(otherCollection, "s")
						if key == singularName+"_id" || key == otherCollection+"_id" {
							relations = append(relations, tableRelation{
								Table1:   collectionName,
								Table2:   otherCollection,
								Relation: "ManyToOne",
							})
						}
					}
				}
			}
		}

		var doc bson.M
		err = collection.FindOne(ctx, bson.M{}).Decode(&doc)
		if err != nil {
			continue
		}

		for key := range doc {
			for _, otherCollection := range collections {
				singularName := strings.TrimSuffix(otherCollection, "s")
				if key == singularName+"_id" || key == otherCollection+"_id" {
					relations = append(relations, tableRelation{
						Table1:   collectionName,
						Table2:   otherCollection,
						Relation: "ManyToMany",
					})
				}
			}
		}
	}

	tableMap := make(map[string][]engine.GraphUnitRelationship)
	for _, tr := range relations {
		tableMap[tr.Table1] = append(tableMap[tr.Table1], engine.GraphUnitRelationship{Name: tr.Table2, RelationshipType: engine.GraphUnitRelationshipType(tr.Relation)})
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
		tables = append(tables, engine.GraphUnit{Unit: storageUnit, Relations: relations})
	}

	return tables, nil
}
