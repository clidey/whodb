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
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	supportedOperators = map[string]string{
		"eq": "eq", "ne": "ne", "gt": "gt", "gte": "gte", "lt": "lt", "lte": "lte",
		"in": "in", "nin": "nin", "and": "and", "or": "or", "not": "not", "nor": "nor",
		"exists": "exists", "type": "type", "regex": "regex", "expr": "expr", "mod": "mod",
		"all": "all", "elemMatch": "elemMatch", "size": "size", "bitsAllClear": "bitsAllClear",
		"bitsAllSet": "bitsAllSet", "bitsAnyClear": "bitsAnyClear", "bitsAnySet": "bitsAnySet",
		"geoIntersects": "geoIntersects", "geoWithin": "geoWithin", "near": "near", "nearSphere": "nearSphere",
	}
)

type MongoDBPlugin struct {
	engine.BasePlugin
}

func (p *MongoDBPlugin) IsAvailable(ctx context.Context, config *engine.PluginConfig) bool {
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Failed to connect to MongoDB for availability check")
		return false
	}
	defer disconnectClient(client)
	return true
}

func (p *MongoDBPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Failed to connect to MongoDB for database listing")
		return nil, err
	}
	ctx, cancel := opCtx()
	defer cancel()
	defer disconnectClient(client)

	databases, err := client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		log.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Failed to list MongoDB database names")
		return nil, err
	}

	return databases, nil
}

func (p *MongoDBPlugin) GetAllSchemas(config *engine.PluginConfig) ([]string, error) {
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Failed to connect to MongoDB for schema listing")
		return nil, err
	}
	ctx, cancel := opCtx()
	defer cancel()
	defer disconnectClient(client)

	databases, err := client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		log.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Failed to list MongoDB database names")
		return nil, err
	}
	return databases, nil
}

func (p *MongoDBPlugin) GetStorageUnits(config *engine.PluginConfig, database string) ([]engine.StorageUnit, error) {
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname": config.Credentials.Hostname,
			"database": database,
		}).Error("Failed to connect to MongoDB for storage unit listing")
		return nil, err
	}
	ctx, cancel := opCtx()
	defer cancel()
	defer disconnectClient(client)

	db := client.Database(database)
	listOpts := options.ListCollections().SetAuthorizedCollections(true)
	cursor, err := db.ListCollections(ctx, bson.M{}, listOpts)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname": config.Credentials.Hostname,
			"database": database,
		}).Error("Failed to list MongoDB collections")
		return nil, err
	}
	defer cursor.Close(ctx)

	storageUnits := []engine.StorageUnit{}
	for cursor.Next(ctx) {
		var collectionInfo bson.M
		if err := cursor.Decode(&collectionInfo); err != nil {
			log.WithError(err).WithFields(map[string]any{
				"hostname": config.Credentials.Hostname,
				"database": database,
			}).Error("Failed to decode MongoDB collection info")
			return nil, err
		}

		collectionName, _ := collectionInfo["name"].(string)
		collectionType, _ := collectionInfo["type"].(string)

		// Skip MongoDB system collections (e.g., system.views, system.profile)
		if strings.HasPrefix(collectionName, "system.") {
			continue
		}

		storageUnit := engine.StorageUnit{Name: collectionName}

		if collectionType == "view" {
			viewOn, _ := collectionInfo["options"].(bson.M)["viewOn"].(string)

			storageUnit.Attributes = []engine.Record{
				{Key: "Type", Value: "View"},
				{Key: "View On", Value: viewOn},
			}
		} else {
			stats := bson.M{}
			err := db.RunCommand(ctx, bson.D{{Key: "collStats", Value: collectionName}}).Decode(&stats)
			if err != nil {
				log.WithError(err).WithFields(map[string]any{
					"hostname":   config.Credentials.Hostname,
					"database":   database,
					"collection": collectionName,
				}).Error("Failed to get MongoDB collection statistics")
				return nil, err
			}

			storageUnit.Attributes = []engine.Record{
				{Key: "Type", Value: "Collection"},
				{Key: "Storage Size", Value: fmt.Sprintf("%v", stats["storageSize"])},
				{Key: "Count", Value: fmt.Sprintf("%v", stats["count"])},
			}
		}

		storageUnits = append(storageUnits, storageUnit)
	}

	if err := cursor.Err(); err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname": config.Credentials.Hostname,
			"database": database,
		}).Error("MongoDB cursor error while listing collections")
		return nil, err
	}

	return storageUnits, nil
}

func (p *MongoDBPlugin) StorageUnitExists(config *engine.PluginConfig, database string, collection string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}
	ctx, cancel := opCtx()
	defer cancel()
	defer disconnectClient(client)

	db := client.Database(database)
	names, err := db.ListCollectionNames(ctx, bson.M{"name": collection})
	if err != nil {
		return false, err
	}
	return len(names) > 0, nil
}

func (p *MongoDBPlugin) FormatValue(val any) string {
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

// GetColumnConstraints retrieves MongoDB schema validation rules and maps them to the constraint format
// used by the mock data generator. Supports $jsonSchema validator with:
// - required fields → nullable: false
// - enum → check_values
// - minimum/maximum → check_min/check_max
// - minLength/maxLength → length (uses maxLength)
// - pattern → pattern (stored for reference)
func (p *MongoDBPlugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error) {
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"schema":      schema,
			"storageUnit": storageUnit,
		}).Error("Failed to connect to MongoDB for column constraints")
		return make(map[string]map[string]any), nil
	}
	ctx, cancel := opCtx()
	defer cancel()
	defer disconnectClient(client)

	db := client.Database(schema)

	// Get collection info with validator
	filter := bson.M{"name": storageUnit}
	cursor, err := db.ListCollections(ctx, filter)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"schema":      schema,
			"storageUnit": storageUnit,
		}).Debug("Failed to list collections for schema validation")
		return make(map[string]map[string]any), nil
	}
	defer cursor.Close(ctx)

	var collInfo bson.M
	if !cursor.Next(ctx) {
		return make(map[string]map[string]any), nil
	}
	if err := cursor.Decode(&collInfo); err != nil {
		log.WithError(err).WithField("collection", storageUnit).Debug("Failed to decode collection info")
		return make(map[string]map[string]any), nil
	}

	// Extract validator from collection options
	opts, ok := collInfo["options"].(bson.M)
	if !ok {
		return make(map[string]map[string]any), nil
	}

	validator, ok := opts["validator"].(bson.M)
	if !ok {
		return make(map[string]map[string]any), nil
	}

	// Extract $jsonSchema
	jsonSchema, ok := validator["$jsonSchema"].(bson.M)
	if !ok {
		return make(map[string]map[string]any), nil
	}

	constraints := parseMongoDBJsonSchema(jsonSchema)
	log.WithFields(map[string]any{
		"collection":      storageUnit,
		"constraintCount": len(constraints),
	}).Debug("Parsed MongoDB schema validation constraints")

	return constraints, nil
}

// parseMongoDBJsonSchema extracts constraints from a MongoDB $jsonSchema validator.
func parseMongoDBJsonSchema(schema bson.M) map[string]map[string]any {
	constraints := make(map[string]map[string]any)

	// Get required fields
	requiredFields := make(map[string]bool)
	if required, ok := schema["required"].(bson.A); ok {
		for _, field := range required {
			if fieldName, ok := field.(string); ok {
				requiredFields[fieldName] = true
			}
		}
	}

	// Get properties
	properties, ok := schema["properties"].(bson.M)
	if !ok {
		return constraints
	}

	for fieldName, fieldSchemaRaw := range properties {
		fieldSchema, ok := fieldSchemaRaw.(bson.M)
		if !ok {
			continue
		}

		colConstraints := make(map[string]any)

		// Set nullable based on required array
		colConstraints["nullable"] = !requiredFields[fieldName]

		// Get bsonType
		if bsonType, ok := fieldSchema["bsonType"].(string); ok {
			colConstraints["type"] = bsonType
		}

		// Get enum values → check_values
		if enum, ok := fieldSchema["enum"].(bson.A); ok {
			values := make([]string, 0, len(enum))
			for _, v := range enum {
				if s, ok := v.(string); ok {
					values = append(values, s)
				}
			}
			if len(values) > 0 {
				colConstraints["check_values"] = values
			}
		}

		// Get minimum/maximum → check_min/check_max
		// MongoDB may return int32 or float64 depending on how the schema was defined
		if minVal := toFloat64(fieldSchema["minimum"]); minVal != nil {
			colConstraints["check_min"] = *minVal
		}
		if maxVal := toFloat64(fieldSchema["maximum"]); maxVal != nil {
			colConstraints["check_max"] = *maxVal
		}

		// Get maxLength → length (mock data generator uses this to limit string length)
		if maxLen := toFloat64(fieldSchema["maxLength"]); maxLen != nil {
			colConstraints["length"] = int(*maxLen)
		}

		// Store pattern for reference (could be used for custom validation in future)
		if pattern, ok := fieldSchema["pattern"].(string); ok {
			colConstraints["pattern"] = pattern
		}

		if len(colConstraints) > 0 {
			constraints[fieldName] = colConstraints
		}
	}

	return constraints
}

// toFloat64 converts common BSON numeric types to float64.
func toFloat64(v any) *float64 {
	var result float64
	switch val := v.(type) {
	case float64:
		result = val
	case float32:
		result = float64(val)
	case int:
		result = float64(val)
	case int32:
		result = float64(val)
	case int64:
		result = float64(val)
	default:
		return nil
	}
	return &result
}

// ClearTableData deletes all documents from a MongoDB collection.
// This is used by the mock data generator when overwrite mode is enabled.
func (p *MongoDBPlugin) ClearTableData(config *engine.PluginConfig, schema string, storageUnit string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"schema":      schema,
			"storageUnit": storageUnit,
		}).Error("Failed to connect to MongoDB to clear collection")
		return false, err
	}
	ctx, cancel := opCtx()
	defer cancel()
	defer disconnectClient(client)

	collection := client.Database(schema).Collection(storageUnit)
	result, err := collection.DeleteMany(ctx, bson.M{})
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"schema":      schema,
			"storageUnit": storageUnit,
		}).Error("Failed to clear MongoDB collection")
		return false, err
	}

	log.WithFields(map[string]any{
		"schema":       schema,
		"storageUnit":  storageUnit,
		"deletedCount": result.DeletedCount,
	}).Info("Cleared MongoDB collection for mock data generation")

	return true, nil
}

// GetDatabaseMetadata returns MongoDB metadata for frontend configuration.
// MongoDB is a document database without traditional type definitions.
func (p *MongoDBPlugin) GetDatabaseMetadata() *engine.DatabaseMetadata {
	operators := make([]string, 0, len(supportedOperators))
	for op := range supportedOperators {
		operators = append(operators, op)
	}
	return &engine.DatabaseMetadata{
		DatabaseType: engine.DatabaseType_MongoDB,
		TypeDefinitions: []engine.TypeDefinition{
			{ID: "string", Label: "string", Category: engine.TypeCategoryText},
			{ID: "int", Label: "int", Category: engine.TypeCategoryNumeric},
			{ID: "double", Label: "double", Category: engine.TypeCategoryNumeric},
			{ID: "bool", Label: "bool", Category: engine.TypeCategoryBoolean},
			{ID: "date", Label: "date", Category: engine.TypeCategoryDatetime},
			{ID: "objectId", Label: "objectId", Category: engine.TypeCategoryOther},
			{ID: "array", Label: "array", Category: engine.TypeCategoryOther},
			{ID: "object", Label: "object", Category: engine.TypeCategoryOther},
			{ID: "mixed", Label: "mixed", Category: engine.TypeCategoryOther},
		},
		Operators: operators,
		AliasMap:  map[string]string{},
		Capabilities: engine.Capabilities{
			SupportsDatabaseSwitch: true,
		},
	}
}

func NewMongoDBPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_MongoDB,
		PluginFunctions: &MongoDBPlugin{},
	}
}
