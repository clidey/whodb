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
	"fmt"
	"sort"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common/graphutil"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (p *MongoDBPlugin) GetRows(config *engine.PluginConfig, req *engine.GetRowsRequest) (*engine.GetRowsResult, error) {
	database, collection := req.Schema, req.StorageUnit
	where, sortConds, pageSize, pageOffset := req.Where, req.Sort, req.PageSize, req.PageOffset
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":   config.Credentials.Hostname,
			"database":   database,
			"collection": collection,
		}).Error("Failed to connect to MongoDB for row retrieval")
		return nil, err
	}
	ctx, cancel := opCtx()
	defer cancel()
	defer disconnectClient(client)

	db := client.Database(database)
	coll := db.Collection(collection)

	bsonFilter, err := convertWhereConditionToMongoDB(where)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":   config.Credentials.Hostname,
			"database":   database,
			"collection": collection,
		}).Error("Failed to convert where condition to MongoDB filter")
		return nil, fmt.Errorf("error converting where condition: %v", err)
	}

	// Start count query in parallel
	var totalCount int64
	countDone := make(chan error, 1)
	go func() {
		var countErr error
		// codeql[go/nosql-injection]: collection name validated by StorageUnitExists before reaching this code
		totalCount, countErr = coll.CountDocuments(ctx, bsonFilter)
		countDone <- countErr
	}()

	findOptions := options.Find()
	findOptions.SetLimit(int64(pageSize))
	findOptions.SetSkip(int64(pageOffset))

	// Apply sorting if provided
	if len(sortConds) > 0 {
		sortMap := bson.D{}
		for _, s := range sortConds {
			direction := 1 // ASC
			if s.Direction == model.SortDirectionDesc {
				direction = -1 // DESC
			}
			sortMap = append(sortMap, bson.E{Key: s.Column, Value: direction})
		}
		findOptions.SetSort(sortMap)
	}

	cursor, err := coll.Find(ctx, bsonFilter, findOptions)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":   config.Credentials.Hostname,
			"database":   database,
			"collection": collection,
			"pageSize":   pageSize,
			"pageOffset": pageOffset,
		}).Error("Failed to execute MongoDB find query")
		return nil, err
	}
	defer cursor.Close(ctx)

	var rowsResult []bson.M
	if err = cursor.All(ctx, &rowsResult); err != nil {
		log.WithError(err).WithFields(map[string]any{
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
			log.WithError(err).WithFields(map[string]any{
				"hostname":   config.Credentials.Hostname,
				"database":   database,
				"collection": collection,
			}).Error("Failed to marshal MongoDB document to JSON")
			return nil, err
		}
		result.Rows = append(result.Rows, []string{string(jsonBytes)})
	}

	// Wait for count query to complete
	if countErr := <-countDone; countErr != nil {
		log.WithError(countErr).Warn("Failed to get MongoDB document count")
	} else {
		result.TotalCount = totalCount
	}

	return result, nil
}

func (p *MongoDBPlugin) GetRowCount(config *engine.PluginConfig, database, collection string, where *model.WhereCondition) (int64, error) {
	client, err := DB(config)
	if err != nil {
		return 0, err
	}
	ctx, cancel := opCtx()
	defer cancel()
	defer disconnectClient(client)

	db := client.Database(database)
	coll := db.Collection(collection)

	bsonFilter, err := convertWhereConditionToMongoDB(where)
	if err != nil {
		return 0, fmt.Errorf("error converting where condition: %v", err)
	}

	// codeql[go/nosql-injection]: collection name validated by StorageUnitExists before reaching this code
	count, err := coll.CountDocuments(ctx, bsonFilter)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (p *MongoDBPlugin) GetColumnsForTable(config *engine.PluginConfig, schema string, storageUnit string) ([]engine.Column, error) {
	ctx, cancel := opCtx()
	defer cancel()
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":   config.Credentials.Hostname,
			"database":   schema,
			"collection": storageUnit,
		}).Error("Failed to connect to MongoDB for column inference")
		return nil, err
	}
	defer disconnectClient(client)

	db := client.Database(schema)
	collection := db.Collection(storageUnit)

	// Sample up to 100 documents to build a merged schema view
	cursor, err := collection.Find(ctx, bson.M{}, options.Find().SetLimit(100))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	fieldTypes := make(map[string]string)
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		for fieldName, fieldValue := range doc {
			fieldType := inferMongoDBType(fieldValue)
			fieldTypes[fieldName] = mergeMongoTypes(fieldTypes[fieldName], fieldType)
		}
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	if len(fieldTypes) == 0 {
		log.WithField("collection", storageUnit).Warn("MongoDB GetColumns: No documents found, returning empty schema")
		return []engine.Column{}, nil
	}

	cursor, err = db.ListCollections(ctx, bson.M{})
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname": config.Credentials.Hostname,
			"database": schema,
		}).Error("Failed to list MongoDB collections for FK detection")
		return nil, err
	}
	defer cursor.Close(ctx)

	var collections []string
	for cursor.Next(ctx) {
		var collectionInfo bson.M
		if err := cursor.Decode(&collectionInfo); err != nil {
			continue
		}
		name, _ := collectionInfo["name"].(string)
		collections = append(collections, name)
	}

	fieldNames := make([]string, 0, len(fieldTypes))
	for name := range fieldTypes {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	// Infer FK relationships using shared heuristics
	fkMap := graphutil.InferForeignKeys(storageUnit, fieldNames, collections)
	fieldToRef := make(map[string]string, len(fkMap))
	for refUnit, field := range fkMap {
		fieldToRef[field] = refUnit
	}

	columns := []engine.Column{}
	for _, fieldName := range fieldNames {
		fieldType := fieldTypes[fieldName]

		var isForeignKey bool
		var referencedTable *string
		if ref, ok := fieldToRef[fieldName]; ok {
			isForeignKey = true
			referencedTable = &ref
		}

		columns = append(columns, engine.Column{
			Name:            fieldName,
			Type:            fieldType,
			IsPrimary:       fieldName == "_id",
			IsForeignKey:    isForeignKey,
			ReferencedTable: referencedTable,
		})
	}

	return columns, nil
}

func inferMongoDBType(value any) string {
	if value == nil {
		return "null"
	}

	switch value.(type) {
	case bson.ObjectID:
		return "ObjectId"
	case string:
		return "string"
	case int, int32, int64:
		return "int"
	case float32, float64:
		return "double"
	case bool:
		return "bool"
	case bson.DateTime:
		return "date"
	case []any:
		return "array"
	case map[string]any, bson.M:
		return "object"
	default:
		return "mixed"
	}
}

// mergeMongoTypes combines type hints; if conflicting, returns "mixed".
func mergeMongoTypes(current, next string) string {
	if current == "" {
		return next
	}
	if current == next {
		return current
	}
	return "mixed"
}
