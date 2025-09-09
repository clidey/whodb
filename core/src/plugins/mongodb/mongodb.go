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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

type MongoDBPlugin struct{}

func (p *MongoDBPlugin) IsAvailable(config *engine.PluginConfig) bool {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Failed to connect to MongoDB for availability check")
		return false
	}
	defer client.Disconnect(context.TODO())
	return true
}

func (p *MongoDBPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Failed to connect to MongoDB for database listing")
		return nil, err
	}
	defer client.Disconnect(context.TODO())

	databases, err := client.ListDatabaseNames(context.TODO(), bson.M{})
	if err != nil {
		log.Logger.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Failed to list MongoDB database names")
		return nil, err
	}

	return databases, nil
}

func (p *MongoDBPlugin) GetAllSchemas(config *engine.PluginConfig) ([]string, error) {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Failed to connect to MongoDB for schema listing")
		return nil, err
	}
	defer client.Disconnect(context.TODO())

	databases, err := client.ListDatabaseNames(context.TODO(), bson.M{})
	if err != nil {
		log.Logger.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Failed to list MongoDB database names")
		return nil, err
	}
	return databases, nil
}

func (p *MongoDBPlugin) GetStorageUnits(config *engine.PluginConfig, database string) ([]engine.StorageUnit, error) {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname": config.Credentials.Hostname,
			"database": database,
		}).Error("Failed to connect to MongoDB for storage unit listing")
		return nil, err
	}
	defer client.Disconnect(context.TODO())

	db := client.Database(database)
	cursor, err := db.ListCollections(context.TODO(), bson.M{})
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname": config.Credentials.Hostname,
			"database": database,
		}).Error("Failed to list MongoDB collections")
		return nil, err
	}
	defer cursor.Close(context.TODO())

	storageUnits := []engine.StorageUnit{}
	for cursor.Next(context.TODO()) {
		var collectionInfo bson.M
		if err := cursor.Decode(&collectionInfo); err != nil {
			log.Logger.WithError(err).WithFields(map[string]interface{}{
				"hostname": config.Credentials.Hostname,
				"database": database,
			}).Error("Failed to decode MongoDB collection info")
			return nil, err
		}

		collectionName, _ := collectionInfo["name"].(string)
		collectionType, _ := collectionInfo["type"].(string)

		storageUnit := engine.StorageUnit{Name: collectionName}

		if collectionType == "view" {
			viewOn, _ := collectionInfo["options"].(bson.M)["viewOn"].(string)

			storageUnit.Attributes = []engine.Record{
				{Key: "Type", Value: "View"},
				{Key: "View On", Value: viewOn},
			}
		} else {
			stats := bson.M{}
			err := db.RunCommand(context.TODO(), bson.D{{Key: "collStats", Value: collectionName}}).Decode(&stats)
			if err != nil {
				log.Logger.WithError(err).WithFields(map[string]interface{}{
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
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname": config.Credentials.Hostname,
			"database": database,
		}).Error("MongoDB cursor error while listing collections")
		return nil, err
	}

	return storageUnits, nil
}

func (p *MongoDBPlugin) GetRows(config *engine.PluginConfig, database, collection string, where *model.WhereCondition, sort []*model.SortCondition, pageSize, pageOffset int) (*engine.GetRowsResult, error) {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname":   config.Credentials.Hostname,
			"database":   database,
			"collection": collection,
		}).Error("Failed to connect to MongoDB for row retrieval")
		return nil, err
	}
	defer client.Disconnect(context.TODO())

	db := client.Database(database)
	coll := db.Collection(collection)

	bsonFilter, err := convertWhereConditionToMongoDB(where)
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname":   config.Credentials.Hostname,
			"database":   database,
			"collection": collection,
		}).Error("Failed to convert where condition to MongoDB filter")
		return nil, fmt.Errorf("error converting where condition: %v", err)
	}

	findOptions := options.Find()
	findOptions.SetLimit(int64(pageSize))
	findOptions.SetSkip(int64(pageOffset))

	// Apply sorting if provided
	if len(sort) > 0 {
		sortMap := bson.D{}
		for _, s := range sort {
			direction := 1 // ASC
			if s.Direction == model.SortDirectionDesc {
				direction = -1 // DESC
			}
			sortMap = append(sortMap, bson.E{Key: s.Column, Value: direction})
		}
		findOptions.SetSort(sortMap)
	}

	cursor, err := coll.Find(context.TODO(), bsonFilter, findOptions)
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname":   config.Credentials.Hostname,
			"database":   database,
			"collection": collection,
			"pageSize":   pageSize,
			"pageOffset": pageOffset,
		}).Error("Failed to execute MongoDB find query")
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var rowsResult []bson.M
	if err = cursor.All(context.TODO(), &rowsResult); err != nil {
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname":   config.Credentials.Hostname,
			"database":   database,
			"collection": collection,
		}).Error("Failed to decode MongoDB query results")
		return nil, err
	}

	result := &engine.GetRowsResult{
		Columns: []engine.Column{
			{Name: "document", Type: "Document"},
		},
		Rows: [][]string{},
	}

	for _, doc := range rowsResult {
		jsonBytes, err := json.Marshal(doc)
		if err != nil {
			log.Logger.WithError(err).WithFields(map[string]interface{}{
				"hostname":   config.Credentials.Hostname,
				"database":   database,
				"collection": collection,
			}).Error("Failed to marshal MongoDB document to JSON")
			return nil, err
		}
		result.Rows = append(result.Rows, []string{string(jsonBytes)})
	}

	return result, nil
}

func (p *MongoDBPlugin) GetColumnsForTable(config *engine.PluginConfig, schema string, storageUnit string) ([]engine.Column, error) {
	// MongoDB doesn't have a traditional column structure, it returns documents
	return []engine.Column{
		{Name: "document", Type: "Document"},
	}, nil
}

func convertWhereConditionToMongoDB(where *model.WhereCondition) (bson.M, error) {
	if where == nil {
		return bson.M{}, nil
	}

	switch where.Type {
	case model.WhereConditionTypeAtomic:
		if where.Atomic == nil {
			return nil, fmt.Errorf("atomic condition must have an atomicwherecondition")
		}

		operatorMap := map[string]string{
			"eq":  "$eq",
			"ne":  "$ne",
			"gt":  "$gt",
			"gte": "$gte",
			"lt":  "$lt",
			"lte": "$lte",
		}

		mongoOperator, exists := operatorMap[where.Atomic.Operator]
		if !exists {
			return nil, fmt.Errorf("unsupported operator: %s", where.Atomic.Operator)
		}

		// Handle _id field specially - convert string to ObjectID
		var value any = where.Atomic.Value
		if where.Atomic.Key == "_id" {
			objectID, err := primitive.ObjectIDFromHex(where.Atomic.Value)
			if err != nil {
				// If it's not a valid ObjectID, use the string value as-is
				// This allows querying by non-ObjectID _id values if they exist
				value = where.Atomic.Value
			} else {
				value = objectID
			}
		}

		return bson.M{where.Atomic.Key: bson.M{mongoOperator: value}}, nil

	case model.WhereConditionTypeAnd:
		if where.And == nil || len(where.And.Children) == 0 {
			return bson.M{}, nil
		}

		andConditions := []bson.M{}
		for _, child := range where.And.Children {
			childCondition, err := convertWhereConditionToMongoDB(child)
			if err != nil {
				log.Logger.WithError(err).Error("Failed to convert child AND condition to MongoDB filter")
				return nil, err
			}
			andConditions = append(andConditions, childCondition)
		}

		return bson.M{"$and": andConditions}, nil

	case model.WhereConditionTypeOr:
		if where.Or == nil || len(where.Or.Children) == 0 {
			return bson.M{}, nil
		}

		orConditions := []bson.M{}
		for _, child := range where.Or.Children {
			childCondition, err := convertWhereConditionToMongoDB(child)
			if err != nil {
				log.Logger.WithError(err).Error("Failed to convert child OR condition to MongoDB filter")
				return nil, err
			}
			orConditions = append(orConditions, childCondition)
		}

		return bson.M{"$or": orConditions}, nil

	default:
		return nil, fmt.Errorf("unknown whereconditiontype: %v", where.Type)
	}
}

func (p *MongoDBPlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return nil, errors.ErrUnsupported
}

func (p *MongoDBPlugin) Chat(config *engine.PluginConfig, schema string, model string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
	return nil, errors.ErrUnsupported
}

func (p *MongoDBPlugin) FormatValue(val any) string {
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

func (p *MongoDBPlugin) GetSupportedOperators() map[string]string {
	return supportedOperators
}

// GetColumnConstraints returns empty constraints for MongoDB since mock data generation doesn't apply to NoSQL databases
func (p *MongoDBPlugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error) {
	return make(map[string]map[string]any), nil
}

// ClearTableData returns not supported error for MongoDB since mock data generation doesn't apply to NoSQL databases
func (p *MongoDBPlugin) ClearTableData(config *engine.PluginConfig, schema string, storageUnit string) (bool, error) {
	return false, errors.New("mock data generation is not supported for MongoDB")
}

// WithTransaction executes the operation directly since MongoDB doesn't support transactions in the same way as SQL databases
func (p *MongoDBPlugin) WithTransaction(config *engine.PluginConfig, operation func(tx any) error) error {
	// MongoDB doesn't support transactions in the same way as SQL databases
	// For now, just execute the operation directly
	return operation(nil)
}

func NewMongoDBPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_MongoDB,
		PluginFunctions: &MongoDBPlugin{},
	}
}
