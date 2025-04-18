/*
 * Copyright 2025 Clidey, Inc.
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
	"context"
	"errors"

	"github.com/clidey/whodb/core/src/engine"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (p *MongoDBPlugin) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, fields []engine.Record) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}
	defer client.Disconnect(context.Background())

	database := client.Database(schema)

	err = createCollectionIfNotExists(database, storageUnit)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (p *MongoDBPlugin) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}
	defer client.Disconnect(context.Background())

	collection := client.Database(schema).Collection(storageUnit)

	document := make(map[string]interface{})
	for _, value := range values {
		document[value.Key] = value.Value
	}

	_, err = collection.InsertOne(context.Background(), document)
	if err != nil {
		return false, err
	}

	return true, nil
}

func createCollectionIfNotExists(database *mongo.Database, collectionName string) error {
	collections, err := database.ListCollectionNames(context.Background(), bson.D{})
	if err != nil {
		return err
	}

	for _, col := range collections {
		if col == collectionName {
			return errors.New("collection already exists")
		}
	}

	err = database.CreateCollection(context.Background(), collectionName)
	if err != nil {
		return err
	}

	return nil
}
