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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"go.mongodb.org/mongo-driver/bson"
)

func (p *MongoDBPlugin) DeleteRow(config *engine.PluginConfig, database string, storageUnit string, values map[string]string) (bool, error) {
	ctx, cancel := opCtx()
	defer cancel()
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"database":    database,
			"storageUnit": storageUnit,
		}).Error("Failed to connect to MongoDB for row deletion")
		return false, err
	}
	defer client.Disconnect(ctx)

	db := client.Database(database)
	collection := db.Collection(storageUnit)

	documentJSON, ok := values["document"]
	if !ok {
		log.WithFields(map[string]any{
			"hostname":      config.Credentials.Hostname,
			"database":      database,
			"storageUnit":   storageUnit,
			"availableKeys": getMapKeys(values),
		}).Error("Missing 'document' key in values map for MongoDB row deletion")
		return false, errors.New("missing 'document' key in values map")
	}

	var jsonValues bson.M
	if err := json.Unmarshal([]byte(documentJSON), &jsonValues); err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":     config.Credentials.Hostname,
			"database":     database,
			"storageUnit":  storageUnit,
			"documentJSON": documentJSON,
		}).Error("Failed to unmarshal document JSON for MongoDB row deletion")
		return false, err
	}

	id, ok := jsonValues["_id"]
	if !ok {
		log.WithFields(map[string]any{
			"hostname":       config.Credentials.Hostname,
			"database":       database,
			"storageUnit":    storageUnit,
			"documentFields": getDocumentFieldNames(jsonValues),
		}).Error("Missing '_id' field in document for MongoDB row deletion")
		return false, errors.New("missing '_id' field in the document")
	}

	objectID, err := normalizeMongoID(id)
	if err != nil {
		return false, fmt.Errorf("invalid '_id' value: %w", err)
	}

	delete(jsonValues, "_id")

	filter := bson.M{"_id": objectID}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"database":    database,
			"storageUnit": storageUnit,
			"objectID":    objectID,
		}).Error("Failed to delete document from MongoDB collection")
		return false, handleMongoError(err)
	}

	if result.DeletedCount == 0 {
		log.WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"database":    database,
			"storageUnit": storageUnit,
			"objectID":    objectID,
		}).Warn("No documents were deleted from MongoDB collection")
		return false, errors.New("no documents were deleted")
	}

	return true, nil
}
