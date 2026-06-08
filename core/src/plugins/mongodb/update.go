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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (p *MongoDBPlugin) UpdateStorageUnit(config *engine.PluginConfig, database string, storageUnit string, values map[string]string, updatedColumns []string) (bool, error) {
	ctx, cancel := opCtx()
	defer cancel()
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"database":    database,
			"storageUnit": storageUnit,
		}).Error("Failed to connect to MongoDB for storage unit update")
		return false, err
	}
	defer disconnectClient(client)

	db := client.Database(database)
	collection := db.Collection(storageUnit)

	documentJSON, ok := values["document"]
	if !ok {
		log.WithFields(map[string]any{
			"hostname":       config.Credentials.Hostname,
			"database":       database,
			"storageUnit":    storageUnit,
			"availableKeys":  getMapKeys(values),
			"updatedColumns": updatedColumns,
		}).Error("Missing 'document' key in values map for MongoDB storage unit update")
		return false, errors.New("missing 'document' key in values map")
	}

	var jsonValues bson.M
	if err := json.Unmarshal([]byte(documentJSON), &jsonValues); err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":     config.Credentials.Hostname,
			"database":     database,
			"storageUnit":  storageUnit,
			"documentJSON": documentJSON,
		}).Error("Failed to unmarshal document JSON for MongoDB storage unit update")
		return false, err
	}

	id, ok := jsonValues["_id"]
	if !ok {
		log.WithFields(map[string]any{
			"hostname":       config.Credentials.Hostname,
			"database":       database,
			"storageUnit":    storageUnit,
			"documentFields": getDocumentFieldNames(jsonValues),
		}).Error("Missing '_id' field in document for MongoDB storage unit update")
		return false, errors.New("missing '_id' field in the document")
	}

	objectID, err := normalizeMongoID(id)
	if err != nil {
		return false, fmt.Errorf("invalid '_id' value: %w", err)
	}

	delete(jsonValues, "_id")

	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": jsonValues}

	var objectIDLog string
	switch v := objectID.(type) {
	case primitive.ObjectID:
		objectIDLog = v.Hex()
	default:
		objectIDLog = fmt.Sprintf("%v", v)
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":       config.Credentials.Hostname,
			"database":       database,
			"storageUnit":    storageUnit,
			"objectID":       objectIDLog,
			"updatedColumns": updatedColumns,
		}).Error("Failed to update document in MongoDB collection")
		return false, handleMongoError(err)
	}

	if result.MatchedCount == 0 {
		log.WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"database":    database,
			"storageUnit": storageUnit,
			"objectID":    objectIDLog,
		}).Warn("No documents matched the filter for MongoDB storage unit update")
		return false, errors.New("no documents matched the filter")
	}
	if result.ModifiedCount == 0 {
		log.WithFields(map[string]any{
			"hostname":     config.Credentials.Hostname,
			"database":     database,
			"storageUnit":  storageUnit,
			"objectID":     objectIDLog,
			"matchedCount": result.MatchedCount,
		}).Warn("No documents were modified during MongoDB storage unit update")
		return false, errors.New("no documents were updated")
	}

	return true, nil
}

// ReplaceRow replaces a MongoDB document while preserving the authored BSON field order.
func (p *MongoDBPlugin) ReplaceRow(config *engine.PluginConfig, database string, storageUnit string, values map[string]string) (bool, error) {
	ctx, cancel := opCtx()
	defer cancel()
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"database":    database,
			"storageUnit": storageUnit,
		}).Error("Failed to connect to MongoDB for row replacement")
		return false, err
	}
	defer disconnectClient(client)

	documentJSON, ok := values["document"]
	if !ok {
		log.WithFields(map[string]any{
			"hostname":      config.Credentials.Hostname,
			"database":      database,
			"storageUnit":   storageUnit,
			"availableKeys": getMapKeys(values),
		}).Error("Missing 'document' key in values map for MongoDB row replacement")
		return false, errors.New("missing 'document' key in values map")
	}

	replacement, objectID, err := parseMongoReplacementDocument(documentJSON)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"database":    database,
			"storageUnit": storageUnit,
		}).Error("Failed to parse MongoDB replacement document")
		return false, err
	}

	collection := client.Database(database).Collection(storageUnit)
	filter := bson.D{{Key: "_id", Value: objectID}}

	result, err := collection.ReplaceOne(ctx, filter, replacement)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"database":    database,
			"storageUnit": storageUnit,
			"objectID":    formatMongoIDForLog(objectID),
		}).Error("Failed to replace document in MongoDB collection")
		return false, handleMongoError(err)
	}

	if result.MatchedCount == 0 {
		log.WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"database":    database,
			"storageUnit": storageUnit,
			"objectID":    formatMongoIDForLog(objectID),
		}).Warn("No documents matched the filter for MongoDB row replacement")
		return false, errors.New("no documents matched the filter")
	}
	if result.ModifiedCount == 0 {
		log.WithFields(map[string]any{
			"hostname":     config.Credentials.Hostname,
			"database":     database,
			"storageUnit":  storageUnit,
			"objectID":     formatMongoIDForLog(objectID),
			"matchedCount": result.MatchedCount,
		}).Warn("No documents were modified during MongoDB row replacement")
		return false, errors.New("no documents were replaced")
	}

	return true, nil
}

func parseMongoReplacementDocument(documentJSON string) (bson.D, any, error) {
	var document bson.D
	if err := bson.UnmarshalExtJSON([]byte(documentJSON), false, &document); err != nil {
		return nil, nil, err
	}

	idIndex := -1
	var idValue any
	for index, element := range document {
		if element.Key == "_id" {
			idIndex = index
			idValue = element.Value
			break
		}
	}
	if idIndex < 0 {
		return nil, nil, errors.New("missing '_id' field in the document")
	}

	objectID, err := normalizeMongoID(idValue)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid '_id' value: %w", err)
	}

	document[idIndex].Value = objectID
	if idIndex > 0 {
		idElement := document[idIndex]
		copy(document[1:idIndex+1], document[0:idIndex])
		document[0] = idElement
	}

	return document, objectID, nil
}

func formatMongoIDForLog(id any) string {
	switch v := id.(type) {
	case primitive.ObjectID:
		return v.Hex()
	default:
		return fmt.Sprintf("%v", v)
	}
}
