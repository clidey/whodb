/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mongodb

import (
	"github.com/clidey/whodb/core/src/common/graphutil"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (p *MongoDBPlugin) GetGraph(config *engine.PluginConfig, database string) ([]engine.GraphUnit, error) {
	ctx, cancel := opCtx()
	defer cancel()
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname": config.Credentials.Hostname,
			"database": database,
		}).Error("Failed to connect to MongoDB for graph generation")
		return nil, err
	}
	defer disconnectClient(client)

	db := client.Database(database)
	cursor, err := db.ListCollections(ctx, bson.M{})
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
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
			log.WithError(err).WithFields(map[string]any{
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
	var relations []graphutil.Relation

	log.WithFields(map[string]any{
		"database":    database,
		"collections": collections,
	}).Info("MongoDB Graph: Starting relationship detection")

	for _, collectionName := range collections {
		collectionType := collectionTypes[collectionName]

		if collectionType == "view" {
			log.WithField("collection", collectionName).Info("MongoDB Graph: Skipping view")
			continue
		}

		collection := db.Collection(collectionName)

		cursorSample, err := collection.Find(ctx, bson.M{}, options.Find().SetLimit(100))
		if err != nil {
			log.WithError(err).WithField("collection", collectionName).Warn("MongoDB Graph: Unable to sample documents")
			continue
		}

		fieldFrequency := make(map[string]int)
		fieldSamples := make(map[string]any)

		for cursorSample.Next(ctx) {
			var doc bson.M
			if err := cursorSample.Decode(&doc); err != nil {
				continue
			}
			for k, v := range doc {
				fieldFrequency[k]++
				// store a representative value for id detection
				if _, exists := fieldSamples[k]; !exists {
					fieldSamples[k] = v
				}
			}
		}
		cursorSample.Close(ctx)

		if len(fieldFrequency) == 0 {
			log.WithField("collection", collectionName).Warn("MongoDB Graph: No documents found or empty collection")
			continue
		}

		fieldNames := make([]string, 0, len(fieldFrequency))
		for fieldName := range fieldFrequency {
			fieldNames = append(fieldNames, fieldName)
		}

		foreignKeys := graphutil.InferForeignKeys(collectionName, fieldNames, collections)

		for fk, fieldName := range foreignKeys {
			log.WithFields(map[string]any{
				"collection": collectionName,
				"field":      fieldName,
				"references": fk,
			}).Info("MongoDB Graph: FOUND FK RELATIONSHIP")

			relKey := collectionName + ":" + fk + ":ManyToOne"
			if !uniqueRelations[relKey] {
				uniqueRelations[relKey] = true
				relations = append(relations, graphutil.Relation{
					Table1:       collectionName,
					Table2:       fk,
					Relation:     "ManyToOne",
					SourceColumn: fieldName,
					TargetColumn: "_id",
				})
				log.WithFields(map[string]any{
					"from":         collectionName,
					"to":           fk,
					"sourceColumn": fieldName,
					"targetColumn": "_id",
				}).Info("MongoDB Graph: ADDED RELATION")
			}
		}
	}

	log.WithFields(map[string]any{
		"database":       database,
		"relationsCount": len(relations),
	}).Info("MongoDB Graph: Finished relationship detection")

	storageUnits, err := p.GetStorageUnits(config, database)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname": config.Credentials.Hostname,
			"database": database,
		}).Error("Failed to get MongoDB storage units for graph generation")
		return nil, err
	}

	return graphutil.BuildGraphUnits(relations, storageUnits), nil
}
