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
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

func (p *MongoDBPlugin) UpdateStorageUnit(config *engine.PluginConfig, database string, storageUnit string, values map[string]string, updatedColumns []string) (bool, error) {
	ctx, cancel := opCtx(config)
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

	replacement, objectID, err := parseMongoReplacementDocument(documentJSON)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"database":    database,
			"storageUnit": storageUnit,
		}).Error("Failed to parse MongoDB replacement document")
		return false, err
	}

	// codeql[go/sql-injection]: MongoDB row updates intentionally apply the user-authored document body to the selected document.
	result, err := collection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: objectID}}, replacement)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":       config.Credentials.Hostname,
			"database":       database,
			"storageUnit":    storageUnit,
			"objectID":       formatMongoIDForLog(objectID),
			"updatedColumns": updatedColumns,
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

	objectID := normalizeMongoID(idValue)
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
	case bson.ObjectID:
		return v.Hex()
	default:
		return fmt.Sprintf("%v", v)
	}
}
