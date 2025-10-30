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
	"github.com/clidey/whodb/core/src/log"
	"go.mongodb.org/mongo-driver/bson"
)

type tableRelation struct {
	Table1       string
	Table2       string
	Relation     string
	SourceColumn string
	TargetColumn string
}

func (p *MongoDBPlugin) GetGraph(config *engine.PluginConfig, database string) ([]engine.GraphUnit, error) {
	ctx := context.Background()
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]any{
			"hostname": config.Credentials.Hostname,
			"database": database,
		}).Error("Failed to connect to MongoDB for graph generation")
		return nil, err
	}
	defer client.Disconnect(ctx)

	db := client.Database(database)
	cursor, err := db.ListCollections(ctx, bson.M{})
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]any{
			"hostname": config.Credentials.Hostname,
			"database": database,
		}).Error("Failed to list MongoDB collections for graph generation")
		return nil, err
	}
	defer cursor.Close(ctx)

	collections := []string{}
	collectionTypes := make(map[string]string)

	for cursor.Next(ctx) {
		var collectionInfo bson.M
		if err := cursor.Decode(&collectionInfo); err != nil {
			log.Logger.WithError(err).WithFields(map[string]any{
				"hostname": config.Credentials.Hostname,
				"database": database,
			}).Error("Failed to decode MongoDB collection info for graph generation")
			return nil, err
		}

		name, _ := collectionInfo["name"].(string)
		collectionType, _ := collectionInfo["type"].(string)

		collections = append(collections, name)
		collectionTypes[name] = collectionType
	}

	uniqueRelations := make(map[string]bool)
	relations := []tableRelation{}

	log.Logger.WithFields(map[string]any{
		"database":    database,
		"collections": collections,
	}).Info("MongoDB Graph: Starting relationship detection")

	for _, collectionName := range collections {
		collectionType := collectionTypes[collectionName]

		if collectionType == "view" {
			log.Logger.WithField("collection", collectionName).Info("MongoDB Graph: Skipping view")
			continue
		}

		collection := db.Collection(collectionName)

		var sampleDoc bson.M
		err := collection.FindOne(ctx, bson.M{}).Decode(&sampleDoc)
		if err != nil {
			log.Logger.WithFields(map[string]any{
				"collection": collectionName,
				"error":      err.Error(),
			}).Warn("MongoDB Graph: No documents found or error fetching sample")
			continue
		}

		fields := []string{}
		for f := range sampleDoc {
			fields = append(fields, f)
		}
		log.Logger.WithFields(map[string]any{
			"collection": collectionName,
			"fields":     fields,
		}).Info("MongoDB Graph: Found fields")

		foreignKeys := make(map[string]string)

		for fieldName := range sampleDoc {
			if fieldName == "_id" {
				continue
			}

			for _, otherCollection := range collections {
				if otherCollection == collectionName {
					continue
				}

				singularName := strings.TrimSuffix(otherCollection, "s")
				pluralName := otherCollection
				if !strings.HasSuffix(otherCollection, "s") {
					pluralName = otherCollection + "s"
				}

				lowerField := strings.ToLower(fieldName)
				if lowerField == strings.ToLower(singularName)+"_id" ||
					lowerField == strings.ToLower(singularName)+"id" ||
					lowerField == strings.ToLower(otherCollection)+"_id" ||
					lowerField == strings.ToLower(otherCollection)+"id" ||
					lowerField == strings.ToLower(pluralName)+"_id" ||
					lowerField == strings.ToLower(pluralName)+"id" {
					foreignKeys[otherCollection] = fieldName
					log.Logger.WithFields(map[string]any{
						"collection": collectionName,
						"field":      fieldName,
						"references": otherCollection,
					}).Info("MongoDB Graph: FOUND FK RELATIONSHIP")
					break
				}
			}
		}

		for fk, fieldName := range foreignKeys {
			relKey1 := collectionName + ":" + fk

			if !uniqueRelations[relKey1+":ManyToOne"] {
				uniqueRelations[relKey1+":ManyToOne"] = true
				relations = append(relations, tableRelation{
					Table1:       collectionName,
					Table2:       fk,
					Relation:     "ManyToOne",
					SourceColumn: fieldName,
					TargetColumn: "_id",
				})
				log.Logger.WithFields(map[string]any{
					"from":         collectionName,
					"to":           fk,
					"sourceColumn": fieldName,
					"targetColumn": "_id",
				}).Info("MongoDB Graph: ADDED RELATION")
			}
		}
	}

	log.Logger.WithFields(map[string]any{
		"database":       database,
		"relationsCount": len(relations),
	}).Info("MongoDB Graph: Finished relationship detection")

	tableMap := make(map[string][]engine.GraphUnitRelationship)
	for _, tr := range relations {
		sourceCol := tr.SourceColumn
		targetCol := tr.TargetColumn
		tableMap[tr.Table1] = append(tableMap[tr.Table1], engine.GraphUnitRelationship{
			Name:             tr.Table2,
			RelationshipType: engine.GraphUnitRelationshipType(tr.Relation),
			SourceColumn:     &sourceCol,
			TargetColumn:     &targetCol,
		})
	}

	storageUnits, err := p.GetStorageUnits(config, database)
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]any{
			"hostname": config.Credentials.Hostname,
			"database": database,
		}).Error("Failed to get MongoDB storage units for graph generation")
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
