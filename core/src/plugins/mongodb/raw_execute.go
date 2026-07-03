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
	"fmt"
	"strconv"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/clidey/whodb/core/src/engine"
)

// RawExecute executes a bounded MongoDB shell-style command and returns rows.
func (p *MongoDBPlugin) RawExecute(config *engine.PluginConfig, query string, _ ...any) (*engine.GetRowsResult, error) {
	cmd, err := parseMongoShellCommand(query)
	if err != nil {
		return nil, err
	}
	args, err := parseMongoShellArgs(cmd.RawArgs)
	if err != nil {
		return nil, err
	}

	client, err := DB(config)
	if err != nil {
		return nil, err
	}
	ctx, cancel := opCtx(config)
	defer cancel()
	defer disconnectClient(client)

	database := client.Database(config.Credentials.Database)
	if cmd.Collection == "" {
		return execMongoDatabaseCommand(ctx, database, cmd)
	}

	collection := database.Collection(cmd.Collection)
	switch cmd.Method {
	case "find":
		return execMongoFind(ctx, collection, args)
	case "findOne":
		return execMongoFindOne(ctx, collection, args)
	case "insertOne":
		return execMongoInsertOne(ctx, collection, args)
	case "insertMany":
		return execMongoInsertMany(ctx, collection, args)
	case "updateOne":
		return execMongoUpdate(ctx, collection, args, false)
	case "updateMany":
		return execMongoUpdate(ctx, collection, args, true)
	case "deleteOne":
		return execMongoDelete(ctx, collection, args, false)
	case "deleteMany":
		return execMongoDelete(ctx, collection, args, true)
	case "countDocuments":
		return execMongoCountDocuments(ctx, collection, args)
	case "aggregate":
		return execMongoAggregate(ctx, collection, args)
	case "distinct":
		return execMongoDistinct(ctx, collection, args)
	case "createIndex":
		return execMongoCreateIndex(ctx, collection, args)
	case "drop":
		return execMongoDrop(ctx, collection)
	default:
		return nil, fmt.Errorf("unsupported method %q; supported methods: find, findOne, insertOne, insertMany, updateOne, updateMany, deleteOne, deleteMany, countDocuments, aggregate, distinct, createIndex, drop", cmd.Method)
	}
}

func execMongoDatabaseCommand(ctx context.Context, database *mongo.Database, cmd *mongoShellCommand) (*engine.GetRowsResult, error) {
	switch cmd.Method {
	case "dropDatabase":
		if err := database.Drop(ctx); err != nil {
			return nil, err
		}
		return mongoMutationResult(map[string]string{"acknowledged": "true", "database": database.Name()}), nil
	default:
		return nil, fmt.Errorf("unsupported database method %q; supported methods: dropDatabase", cmd.Method)
	}
}

func execMongoFind(ctx context.Context, collection *mongo.Collection, args []any) (*engine.GetRowsResult, error) {
	filter := mongoArgDocument(args, 0)
	findOptions := options.Find().SetLimit(1000)
	if projection, ok := mongoArgDocumentOK(args, 1); ok && len(projection) > 0 {
		findOptions.SetProjection(projection)
	}
	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()
	return mongoCursorToRows(ctx, cursor)
}

func execMongoFindOne(ctx context.Context, collection *mongo.Collection, args []any) (*engine.GetRowsResult, error) {
	filter := mongoArgDocument(args, 0)
	var doc bson.D
	err := collection.FindOne(ctx, filter).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return mongoDocumentsToRows(nil)
	}
	if err != nil {
		return nil, err
	}
	return mongoDocumentsToRows([]bson.D{doc})
}

func execMongoInsertOne(ctx context.Context, collection *mongo.Collection, args []any) (*engine.GetRowsResult, error) {
	if len(args) == 0 {
		return nil, errors.New("insertOne requires a document argument")
	}
	result, err := collection.InsertOne(ctx, args[0])
	if err != nil {
		return nil, err
	}
	return mongoMutationResult(map[string]string{"insertedId": formatMongoValueForCell(result.InsertedID)}), nil
}

func execMongoInsertMany(ctx context.Context, collection *mongo.Collection, args []any) (*engine.GetRowsResult, error) {
	if len(args) == 0 {
		return nil, errors.New("insertMany requires an array argument")
	}
	documents, ok := args[0].(bson.A)
	if !ok {
		return nil, errors.New("insertMany: first argument must be an array of documents")
	}
	result, err := collection.InsertMany(ctx, []any(documents))
	if err != nil {
		return nil, err
	}
	return mongoMutationResult(map[string]string{"insertedCount": strconv.Itoa(len(result.InsertedIDs))}), nil
}

func execMongoUpdate(ctx context.Context, collection *mongo.Collection, args []any, multi bool) (*engine.GetRowsResult, error) {
	if len(args) < 2 {
		return nil, errors.New("update requires filter and update arguments")
	}
	filter := mongoArgDocument(args, 0)
	update := mongoArgDocument(args, 1)
	var result *mongo.UpdateResult
	var err error
	if multi {
		result, err = collection.UpdateMany(ctx, filter, update)
	} else {
		result, err = collection.UpdateOne(ctx, filter, update)
	}
	if err != nil {
		return nil, err
	}
	values := map[string]string{
		"matchedCount":  strconv.FormatInt(result.MatchedCount, 10),
		"modifiedCount": strconv.FormatInt(result.ModifiedCount, 10),
	}
	if result.UpsertedCount > 0 {
		values["upsertedId"] = formatMongoValueForCell(result.UpsertedID)
	}
	return mongoMutationResult(values), nil
}

func execMongoDelete(ctx context.Context, collection *mongo.Collection, args []any, multi bool) (*engine.GetRowsResult, error) {
	filter := mongoArgDocument(args, 0)
	var count int64
	var err error
	if multi {
		result, deleteErr := collection.DeleteMany(ctx, filter)
		if result != nil {
			count = result.DeletedCount
		}
		err = deleteErr
	} else {
		result, deleteErr := collection.DeleteOne(ctx, filter)
		if result != nil {
			count = result.DeletedCount
		}
		err = deleteErr
	}
	if err != nil {
		return nil, err
	}
	return mongoMutationResult(map[string]string{"deletedCount": strconv.FormatInt(count, 10)}), nil
}

func execMongoCountDocuments(ctx context.Context, collection *mongo.Collection, args []any) (*engine.GetRowsResult, error) {
	count, err := collection.CountDocuments(ctx, mongoArgDocument(args, 0))
	if err != nil {
		return nil, err
	}
	return mongoMutationResult(map[string]string{"count": strconv.FormatInt(count, 10)}), nil
}

func execMongoAggregate(ctx context.Context, collection *mongo.Collection, args []any) (*engine.GetRowsResult, error) {
	if len(args) == 0 {
		return nil, errors.New("aggregate requires a pipeline array")
	}
	pipeline, ok := args[0].(bson.A)
	if !ok {
		return nil, errors.New("aggregate: first argument must be an array")
	}
	if err := rejectDangerousPipelineStages(pipeline); err != nil {
		return nil, err
	}
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()
	return mongoCursorToRows(ctx, cursor)
}

// dangerousPipelineOperators are aggregation operators/stages that allow
// server-side JavaScript execution ($where/$function/$accumulator) or writes
// to other collections ($out/$merge). They are rejected anywhere in a
// user-supplied aggregation pipeline.
var dangerousPipelineOperators = map[string]struct{}{
	"$where":       {},
	"$function":    {},
	"$accumulator": {},
	"$out":         {},
	"$merge":       {},
}

// rejectDangerousPipelineStages walks a decoded aggregation pipeline and returns
// an error if any dangerous operator/stage key is present at any depth.
func rejectDangerousPipelineStages(v any) error {
	switch val := v.(type) {
	case bson.A:
		for _, item := range val {
			if err := rejectDangerousPipelineStages(item); err != nil {
				return err
			}
		}
	case []any:
		for _, item := range val {
			if err := rejectDangerousPipelineStages(item); err != nil {
				return err
			}
		}
	case bson.D:
		for _, e := range val {
			if _, bad := dangerousPipelineOperators[e.Key]; bad {
				return fmt.Errorf("aggregation operator %q is not allowed", e.Key)
			}
			if err := rejectDangerousPipelineStages(e.Value); err != nil {
				return err
			}
		}
	case bson.M:
		for k, sub := range val {
			if _, bad := dangerousPipelineOperators[k]; bad {
				return fmt.Errorf("aggregation operator %q is not allowed", k)
			}
			if err := rejectDangerousPipelineStages(sub); err != nil {
				return err
			}
		}
	case map[string]any:
		for k, sub := range val {
			if _, bad := dangerousPipelineOperators[k]; bad {
				return fmt.Errorf("aggregation operator %q is not allowed", k)
			}
			if err := rejectDangerousPipelineStages(sub); err != nil {
				return err
			}
		}
	}
	return nil
}

func execMongoDistinct(ctx context.Context, collection *mongo.Collection, args []any) (*engine.GetRowsResult, error) {
	if len(args) == 0 {
		return nil, errors.New("distinct requires a field name")
	}
	fieldName, ok := args[0].(string)
	if !ok {
		return nil, errors.New("distinct: first argument must be a field name")
	}
	result := collection.Distinct(ctx, fieldName, mongoArgDocument(args, 1))
	if err := result.Err(); err != nil {
		return nil, err
	}
	var values bson.A
	if err := result.Decode(&values); err != nil {
		return nil, err
	}
	rows := make([][]string, len(values))
	for i, value := range values {
		rows[i] = []string{formatMongoValueForCell(value)}
	}
	return &engine.GetRowsResult{
		Columns:       []engine.Column{{Name: "value", Type: mongoTypeString}},
		Rows:          rows,
		DisableUpdate: true,
		TotalCount:    int64(len(rows)),
	}, nil
}

func execMongoCreateIndex(ctx context.Context, collection *mongo.Collection, args []any) (*engine.GetRowsResult, error) {
	if len(args) == 0 {
		return nil, errors.New("createIndex requires a key document")
	}
	name, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: mongoArgDocument(args, 0)})
	if err != nil {
		return nil, err
	}
	return mongoMutationResult(map[string]string{"index": name}), nil
}

func execMongoDrop(ctx context.Context, collection *mongo.Collection) (*engine.GetRowsResult, error) {
	if err := collection.Drop(ctx); err != nil {
		return nil, err
	}
	return mongoMutationResult(map[string]string{"acknowledged": "true", mongoFieldCollection: collection.Name()}), nil
}

func mongoCursorToRows(ctx context.Context, cursor *mongo.Cursor) (*engine.GetRowsResult, error) {
	documents := []bson.D{}
	for cursor.Next(ctx) {
		var doc bson.D
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		documents = append(documents, doc)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return mongoDocumentsToRows(documents)
}

func mongoDocumentsToRows(documents []bson.D) (*engine.GetRowsResult, error) {
	result := &engine.GetRowsResult{
		Columns:       []engine.Column{{Name: "document", Type: "Document"}},
		Rows:          [][]string{},
		DisableUpdate: true,
		TotalCount:    int64(len(documents)),
	}
	for _, doc := range documents {
		jsonBytes, err := marshalMongoDocumentJSON(doc)
		if err != nil {
			return nil, err
		}
		result.Rows = append(result.Rows, []string{string(jsonBytes)})
	}
	return result, nil
}

func mongoMutationResult(values map[string]string) *engine.GetRowsResult {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	// Stable output keeps tests and UI rendering predictable.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	rows := make([][]string, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, []string{key, values[key]})
	}
	return &engine.GetRowsResult{
		Columns:       []engine.Column{{Name: "field", Type: mongoTypeString}, {Name: "value", Type: mongoTypeString}},
		Rows:          rows,
		DisableUpdate: true,
		TotalCount:    int64(len(rows)),
	}
}

func formatMongoValueForCell(value any) string {
	jsonBytes, err := bson.MarshalExtJSON(value, false, false)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(jsonBytes)
}

func mongoArgDocument(args []any, index int) bson.D {
	document, _ := mongoArgDocumentOK(args, index)
	return document
}

func mongoArgDocumentOK(args []any, index int) (bson.D, bool) {
	if index >= len(args) || args[index] == nil {
		return bson.D{}, false
	}
	switch value := args[index].(type) {
	case bson.D:
		return value, true
	case bson.M:
		document := make(bson.D, 0, len(value))
		for key, item := range value {
			document = append(document, bson.E{Key: key, Value: item})
		}
		return document, true
	default:
		return bson.D{}, false
	}
}
