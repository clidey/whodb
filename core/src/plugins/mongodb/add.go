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
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (p *MongoDBPlugin) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, fields []engine.Record) (bool, error) {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"schema":      schema,
			"storageUnit": storageUnit,
		}).Error("Failed to connect to MongoDB for adding storage unit")
		return false, err
	}
	defer client.Disconnect(context.Background())

	database := client.Database(schema)

	err = createCollectionIfNotExists(database, storageUnit, fields)
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"schema":      schema,
			"storageUnit": storageUnit,
		}).Error("Failed to create MongoDB collection")
		return false, err
	}

	return true, nil
}

func (p *MongoDBPlugin) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (bool, error) {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"schema":      schema,
			"storageUnit": storageUnit,
		}).Error("Failed to connect to MongoDB for adding row")
		return false, err
	}
	defer client.Disconnect(context.Background())

	collection := client.Database(schema).Collection(storageUnit)

	document := make(map[string]any)
	for _, value := range values {
		// Skip null values - MongoDB treats missing fields as null for optional fields
		if value.Extra["IsNull"] == "true" {
			continue
		}
		typeHint := value.Extra["Type"]
		document[value.Key] = coerceMongoValue(value.Key, value.Value, typeHint)
	}

	// If _id provided as hex string, convert to ObjectID for proper identity handling
	if rawID, exists := document["_id"]; exists {
		if id, err := normalizeMongoID(rawID); err == nil {
			document["_id"] = id
		}
	}

	_, err = collection.InsertOne(context.Background(), document)
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]any{
			"hostname":       config.Credentials.Hostname,
			"schema":         schema,
			"storageUnit":    storageUnit,
			"documentFields": len(values),
		}).Error("Failed to insert document into MongoDB collection")
		return false, handleMongoError(err)
	}

	return true, nil
}

// AddRowReturningID is not supported for MongoDB (document IDs are generated client-side).
// Returns 0 as MongoDB uses ObjectIDs, not auto-increment integers.
func (p *MongoDBPlugin) AddRowReturningID(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (int64, error) {
	_, err := p.AddRow(config, schema, storageUnit, values)
	if err != nil {
		return 0, err
	}
	return 0, nil
}

func (p *MongoDBPlugin) BulkAddRows(config *engine.PluginConfig, schema string, storageUnit string, rows [][]engine.Record) (bool, error) {
	if len(rows) == 0 {
		return true, nil
	}

	client, err := DB(config)
	if err != nil {
		return false, err
	}
	defer client.Disconnect(context.Background())

	collection := client.Database(schema).Collection(storageUnit)

	documents := make([]any, len(rows))
	for i, row := range rows {
		document := make(map[string]any)
		for _, value := range row {
			// Skip null values - MongoDB treats missing fields as null for optional fields
			if value.Extra["IsNull"] == "true" {
				continue
			}
			typeHint := value.Extra["Type"]
			document[value.Key] = coerceMongoValue(value.Key, value.Value, typeHint)
		}
		if rawID, exists := document["_id"]; exists {
			if id, err := normalizeMongoID(rawID); err == nil {
				document["_id"] = id
			}
		}
		documents[i] = document
	}

	_, err = collection.InsertMany(context.Background(), documents)
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]any{
			"schema":      schema,
			"storageUnit": storageUnit,
			"rowCount":    len(rows),
		}).Error("Failed to bulk insert documents into MongoDB collection")
		return false, handleMongoError(err)
	}

	return true, nil
}

func createCollectionIfNotExists(database *mongo.Database, collectionName string, fields []engine.Record) error {
	collections, err := database.ListCollectionNames(context.Background(), bson.D{})
	if err != nil {
		log.Logger.WithError(err).WithField("collectionName", collectionName).Error("Failed to list MongoDB collection names")
		return err
	}

	for _, col := range collections {
		if col == collectionName {
			return errors.New("collection already exists")
		}
	}

	opts := options.CreateCollection()
	if validator := buildValidator(fields); validator != nil {
		opts.SetValidator(validator)
	}

	err = database.CreateCollection(context.Background(), collectionName, opts)
	if err != nil {
		log.Logger.WithError(err).WithField("collectionName", collectionName).Error("Failed to create MongoDB collection")
		return err
	}

	return nil
}

// buildValidator creates a simple JSON Schema validator from provided field definitions.
// It is best-effort: if no fields are provided, returns nil to keep default collection behavior.
func buildValidator(fields []engine.Record) bson.M {
	if len(fields) == 0 {
		return nil
	}

	properties := bson.M{}
	var required []string

	for _, f := range fields {
		bsonType := mapMongoFieldType(f.Value)
		properties[f.Key] = bson.M{"bsonType": bsonType}

		nullable, err := strconv.ParseBool(f.Extra["Nullable"])
		if err != nil {
			nullable = false
		}
		if !nullable {
			required = append(required, f.Key)
		}
	}

	schema := bson.M{
		"bsonType":   "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	return bson.M{
		"$jsonSchema": schema,
	}
}

// mapMongoFieldType maps an arbitrary field type string to a BSON type for validator use.
func mapMongoFieldType(typeStr string) string {
	lower := strings.ToLower(typeStr)
	switch {
	case strings.Contains(lower, "objectid"):
		return "objectId"
	case strings.Contains(lower, "int"):
		return "int"
	case strings.Contains(lower, "long"):
		return "long"
	case strings.Contains(lower, "double"), strings.Contains(lower, "float"), strings.Contains(lower, "decimal"):
		return "double"
	case strings.Contains(lower, "bool"):
		return "bool"
	case strings.Contains(lower, "date"), strings.Contains(lower, "time"):
		return "date"
	case strings.Contains(lower, "array"):
		return "array"
	case strings.Contains(lower, "object"), strings.Contains(lower, "json"):
		return "object"
	default:
		return "string"
	}
}
