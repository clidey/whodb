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
	"encoding/json"
	"errors"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (p *MongoDBPlugin) DeleteRow(config *engine.PluginConfig, database string, storageUnit string, values map[string]string) (bool, error) {
	ctx := context.Background()
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname": config.Credentials.Hostname,
			"database": database,
			"storageUnit": storageUnit,
		}).Error("Failed to connect to MongoDB for row deletion")
		return false, err
	}
	defer client.Disconnect(ctx)

	db := client.Database(database)
	collection := db.Collection(storageUnit)

	documentJSON, ok := values["document"]
	if !ok {
		log.Logger.WithFields(map[string]interface{}{
			"hostname": config.Credentials.Hostname,
			"database": database,
			"storageUnit": storageUnit,
			"availableKeys": getMapKeys(values),
		}).Error("Missing 'document' key in values map for MongoDB row deletion")
		return false, errors.New("missing 'document' key in values map")
	}

	var jsonValues bson.M
	if err := json.Unmarshal([]byte(documentJSON), &jsonValues); err != nil {
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname": config.Credentials.Hostname,
			"database": database,
			"storageUnit": storageUnit,
			"documentJSON": documentJSON,
		}).Error("Failed to unmarshal document JSON for MongoDB row deletion")
		return false, err
	}

	id, ok := jsonValues["_id"]
	if !ok {
		log.Logger.WithFields(map[string]interface{}{
			"hostname": config.Credentials.Hostname,
			"database": database,
			"storageUnit": storageUnit,
			"documentFields": getDocumentFieldNames(jsonValues),
		}).Error("Missing '_id' field in document for MongoDB row deletion")
		return false, errors.New("missing '_id' field in the document")
	}

	objectID, err := primitive.ObjectIDFromHex(id.(string))
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname": config.Credentials.Hostname,
			"database": database,
			"storageUnit": storageUnit,
			"idValue": id,
		}).Error("Invalid '_id' field; not a valid ObjectID for MongoDB row deletion")
		return false, errors.New("invalid '_id' field; not a valid ObjectID")
	}

	delete(jsonValues, "_id")

	filter := bson.M{"_id": objectID}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname": config.Credentials.Hostname,
			"database": database,
			"storageUnit": storageUnit,
			"objectID": objectID.Hex(),
		}).Error("Failed to delete document from MongoDB collection")
		return false, err
	}

	if result.DeletedCount == 0 {
		log.Logger.WithFields(map[string]interface{}{
			"hostname": config.Credentials.Hostname,
			"database": database,
			"storageUnit": storageUnit,
			"objectID": objectID.Hex(),
		}).Warn("No documents were deleted from MongoDB collection")
		return false, errors.New("no documents were deleted")
	}

	return true, nil
}

// Helper functions for logging
func getMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getDocumentFieldNames(doc bson.M) []string {
	fields := make([]string, 0, len(doc))
	for k := range doc {
		fields = append(fields, k)
	}
	return fields
}
