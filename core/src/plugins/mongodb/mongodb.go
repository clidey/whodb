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
	"sort"
	"strconv"
	"strings"

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
		log.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Failed to connect to MongoDB for availability check")
		return false
	}
	ctx, cancel := opCtx()
	defer cancel()
	defer client.Disconnect(ctx)
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
	defer client.Disconnect(ctx)

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
	defer client.Disconnect(ctx)

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
	defer client.Disconnect(ctx)

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
	defer client.Disconnect(ctx)

	db := client.Database(database)
	names, err := db.ListCollectionNames(ctx, bson.M{"name": collection})
	if err != nil {
		return false, err
	}
	return len(names) > 0, nil
}

func (p *MongoDBPlugin) GetRows(config *engine.PluginConfig, database, collection string, where *model.WhereCondition, sort []*model.SortCondition, pageSize, pageOffset int) (*engine.GetRowsResult, error) {
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
	defer client.Disconnect(ctx)

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
	defer client.Disconnect(ctx)

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
	defer client.Disconnect(ctx)

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

	columns := []engine.Column{}
	for _, fieldName := range fieldNames {
		fieldType := fieldTypes[fieldName]
		isPrimary := fieldName == "_id"

		var isForeignKey bool
		var referencedTable *string

		if fieldName != "_id" {
			lowerField := strings.ToLower(fieldName)
			for _, otherCollection := range collections {
				if otherCollection == storageUnit {
					continue
				}

				singularName := strings.TrimSuffix(otherCollection, "s")
				pluralName := otherCollection
				if !strings.HasSuffix(otherCollection, "s") {
					pluralName = otherCollection + "s"
				}

				if lowerField == strings.ToLower(singularName)+"_id" ||
					lowerField == strings.ToLower(singularName)+"id" ||
					lowerField == strings.ToLower(otherCollection)+"_id" ||
					lowerField == strings.ToLower(otherCollection)+"id" ||
					lowerField == strings.ToLower(pluralName)+"_id" ||
					lowerField == strings.ToLower(pluralName)+"id" {
					isForeignKey = true
					referencedTable = &otherCollection
					break
				}
			}
		}

		columns = append(columns, engine.Column{
			Name:            fieldName,
			Type:            fieldType,
			IsPrimary:       isPrimary,
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
	case primitive.ObjectID:
		return "ObjectId"
	case string:
		return "string"
	case int, int32, int64:
		return "int"
	case float32, float64:
		return "double"
	case bool:
		return "bool"
	case primitive.DateTime:
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

func convertWhereConditionToMongoDB(where *model.WhereCondition) (bson.M, error) {
	if where == nil {
		return bson.M{}, nil
	}

	// Normalize operator to lower for comparisons
	getOp := func(op string) string { return strings.ToLower(op) }

	switch where.Type {
	case model.WhereConditionTypeAtomic:
		if where.Atomic == nil {
			return nil, fmt.Errorf("atomic condition must have an atomicwherecondition")
		}

		operator := getOp(where.Atomic.Operator)

		switch operator {
		case "eq", "ne", "gt", "gte", "lt", "lte":
			mongoOperator := "$" + operator
			value := convertMongoValue(where.Atomic.Key, where.Atomic.Value)
			return bson.M{where.Atomic.Key: bson.M{mongoOperator: value}}, nil

		case "in", "nin":
			values := parseCommaSeparatedValues(where.Atomic.Key, where.Atomic.Value)
			mongoOperator := "$" + operator
			return bson.M{where.Atomic.Key: bson.M{mongoOperator: values}}, nil

		case "regex":
			return bson.M{where.Atomic.Key: bson.M{"$regex": where.Atomic.Value}}, nil

		case "exists":
			exists, err := strconv.ParseBool(where.Atomic.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid exists value: %s", where.Atomic.Value)
			}
			return bson.M{where.Atomic.Key: bson.M{"$exists": exists}}, nil

		case "type":
			return bson.M{where.Atomic.Key: bson.M{"$type": where.Atomic.Value}}, nil

		case "expr":
			// Expect a JSON expression; try to decode
			var expr any
			if err := json.Unmarshal([]byte(where.Atomic.Value), &expr); err != nil {
				return nil, fmt.Errorf("invalid expr payload: %w", err)
			}
			return bson.M{"$expr": expr}, nil

		case "mod":
			parts := strings.Split(where.Atomic.Value, ",")
			if len(parts) != 2 {
				return nil, fmt.Errorf("mod expects 'divisor,remainder'")
			}
			div, err1 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
			rem, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("mod operands must be integers")
			}
			return bson.M{where.Atomic.Key: bson.M{"$mod": []int64{div, rem}}}, nil

		case "all":
			values := parseCommaSeparatedValues(where.Atomic.Key, where.Atomic.Value)
			return bson.M{where.Atomic.Key: bson.M{"$all": values}}, nil

		case "elemmatch":
			var elem bson.M
			if err := json.Unmarshal([]byte(where.Atomic.Value), &elem); err != nil {
				return nil, fmt.Errorf("elemMatch expects JSON object: %w", err)
			}
			return bson.M{where.Atomic.Key: bson.M{"$elemMatch": elem}}, nil

		case "size":
			size, err := strconv.ParseInt(where.Atomic.Value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("size expects integer: %w", err)
			}
			return bson.M{where.Atomic.Key: bson.M{"$size": size}}, nil

		case "bitsallclear", "bitsallset", "bitsanyclear", "bitsanyset":
			mask, err := strconv.ParseInt(where.Atomic.Value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%s expects integer bitmask", operator)
			}
			return bson.M{where.Atomic.Key: bson.M{"$" + operator: mask}}, nil

		case "geointersects", "geowithin", "near", "nearsphere":
			var payload any
			if err := json.Unmarshal([]byte(where.Atomic.Value), &payload); err != nil {
				return nil, fmt.Errorf("%s expects JSON payload: %w", operator, err)
			}
			return bson.M{where.Atomic.Key: bson.M{"$" + operator: payload}}, nil

		default:
			return nil, fmt.Errorf("unsupported operator: %s", where.Atomic.Operator)
		}

	case model.WhereConditionTypeAnd:
		if where.And == nil || len(where.And.Children) == 0 {
			return bson.M{}, nil
		}

		andConditions := []bson.M{}
		for _, child := range where.And.Children {
			childCondition, err := convertWhereConditionToMongoDB(child)
			if err != nil {
				log.WithError(err).Error("Failed to convert child AND condition to MongoDB filter")
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
				log.WithError(err).Error("Failed to convert child OR condition to MongoDB filter")
				return nil, err
			}
			orConditions = append(orConditions, childCondition)
		}

		return bson.M{"$or": orConditions}, nil

	default:
		return nil, fmt.Errorf("unknown whereconditiontype: %v", where.Type)
	}
}

// convertMongoValue handles ObjectID conversion for _id and basic numeric/bool coercion.
// Used for query building where type hints are not available.
func convertMongoValue(key string, raw string) any {
	if key == "_id" {
		id, err := normalizeMongoID(raw)
		if err == nil {
			return id
		}
		return raw
	}
	return coerceMongoValue(key, raw, "") // No type hint for queries
}

func parseCommaSeparatedValues(key string, raw string) []any {
	if strings.TrimSpace(raw) == "" {
		return []any{}
	}
	parts := strings.Split(raw, ",")
	values := make([]any, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		values = append(values, convertMongoValue(key, v))
	}
	return values
}

func (p *MongoDBPlugin) RawExecute(config *engine.PluginConfig, query string, params ...any) (*engine.GetRowsResult, error) {
	return nil, errors.ErrUnsupported
}

func (p *MongoDBPlugin) Chat(config *engine.PluginConfig, schema string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
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
	defer client.Disconnect(ctx)

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
func (p *MongoDBPlugin) NullifyFKColumn(_ *engine.PluginConfig, _, _, _ string) error { return nil }

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
	defer client.Disconnect(ctx)

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

// WithTransaction executes the operation directly since MongoDB doesn't support transactions in the same way as SQL databases
func (p *MongoDBPlugin) WithTransaction(config *engine.PluginConfig, operation func(tx any) error) error {
	// MongoDB doesn't support transactions in the same way as SQL databases
	// For now, just execute the operation directly
	return operation(nil)
}

func (p *MongoDBPlugin) GetForeignKeyRelationships(config *engine.PluginConfig, schema string, storageUnit string) (map[string]*engine.ForeignKeyRelationship, error) {
	return make(map[string]*engine.ForeignKeyRelationship), nil
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
	}
}

func NewMongoDBPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_MongoDB,
		PluginFunctions: &MongoDBPlugin{},
	}
}
