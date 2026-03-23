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
	"regexp"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// commandPattern matches MongoDB shell-style commands: db.collection.operation(args)
// It also supports chained methods like .limit(n) and .sort({...})
var commandPattern = regexp.MustCompile(`^db\.(\w+)\.(\w+)\(([\s\S]*)\)(?:\.(limit|sort)\(([\s\S]*?)\))?(?:\.(limit|sort)\(([\s\S]*?)\))?$`)

// RawExecute parses and executes MongoDB shell-style commands.
// Supported patterns:
//   - db.collection.find(filter)
//   - db.collection.find(filter).limit(n)
//   - db.collection.find(filter).sort(sortDoc)
//   - db.collection.aggregate([pipeline])
//   - db.collection.countDocuments(filter)
//   - db.collection.distinct(field, filter)
//   - db.collection.insertOne(doc)
//   - db.collection.updateOne(filter, update)
//   - db.collection.deleteOne(filter)
func (p *MongoDBPlugin) RawExecute(config *engine.PluginConfig, query string, _ ...any) (*engine.GetRowsResult, error) {
	query = strings.TrimSpace(query)

	matches := commandPattern.FindStringSubmatch(query)
	if matches == nil {
		return nil, fmt.Errorf("unsupported MongoDB command format — expected db.collection.operation(...)")
	}

	collectionName := matches[1]
	operation := matches[2]
	argsStr := matches[3]

	// Parse chained methods (limit, sort)
	var limitVal int64
	var sortDoc bson.D
	for i := 4; i <= 6; i += 2 {
		if i >= len(matches) || matches[i] == "" {
			continue
		}
		chainMethod := matches[i]
		chainArg := matches[i+1]
		switch chainMethod {
		case "limit":
			n, err := strconv.ParseInt(strings.TrimSpace(chainArg), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid limit value: %s", chainArg)
			}
			limitVal = n
		case "sort":
			parsed, err := parseBSONArg(strings.TrimSpace(chainArg))
			if err != nil {
				return nil, fmt.Errorf("invalid sort document: %v", err)
			}
			if m, ok := parsed.(bson.D); ok {
				sortDoc = m
			}
		}
	}

	client, err := DB(config)
	if err != nil {
		log.WithError(err).Error("Failed to connect to MongoDB for raw execute")
		return nil, err
	}
	ctx, cancel := opCtx()
	defer cancel()
	defer disconnectClient(client)

	coll := client.Database(config.Credentials.Database).Collection(collectionName)

	switch operation {
	case "find":
		filter, err := parseFilterArg(argsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid filter: %v", err)
		}
		findOpts := options.Find()
		if limitVal > 0 {
			findOpts.SetLimit(limitVal)
		}
		if sortDoc != nil {
			findOpts.SetSort(sortDoc)
		}
		cursor, err := coll.Find(ctx, filter, findOpts)
		if err != nil {
			return nil, fmt.Errorf("find failed: %v", err)
		}
		defer cursor.Close(ctx)

		var docs []bson.M
		if err := cursor.All(ctx, &docs); err != nil {
			return nil, fmt.Errorf("failed to read results: %v", err)
		}
		return documentsToResult(docs), nil

	case "aggregate":
		pipeline, err := parsePipelineArg(argsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid pipeline: %v", err)
		}
		cursor, err := coll.Aggregate(ctx, pipeline)
		if err != nil {
			return nil, fmt.Errorf("aggregate failed: %v", err)
		}
		defer cursor.Close(ctx)

		var docs []bson.M
		if err := cursor.All(ctx, &docs); err != nil {
			return nil, fmt.Errorf("failed to read results: %v", err)
		}
		return documentsToResult(docs), nil

	case "countDocuments":
		filter, err := parseFilterArg(argsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid filter: %v", err)
		}
		count, err := coll.CountDocuments(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("countDocuments failed: %v", err)
		}
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "count", Type: "int"}},
			Rows:    [][]string{{strconv.FormatInt(count, 10)}},
		}, nil

	case "distinct":
		args, err := splitTopLevelArgs(argsStr)
		if err != nil || len(args) < 1 {
			return nil, fmt.Errorf("distinct requires at least a field name argument")
		}
		fieldName := strings.Trim(strings.TrimSpace(args[0]), `"'`)
		filter := bson.M{}
		if len(args) > 1 {
			parsed, err := parseBSONArg(strings.TrimSpace(args[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid filter for distinct: %v", err)
			}
			if m, ok := parsed.(bson.D); ok {
				filter = bsonDToM(m)
			}
		}
		values, err := coll.Distinct(ctx, fieldName, filter)
		if err != nil {
			return nil, fmt.Errorf("distinct failed: %v", err)
		}
		rows := make([][]string, 0, len(values))
		for _, v := range values {
			rows = append(rows, []string{fmt.Sprintf("%v", v)})
		}
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "value", Type: "string"}},
			Rows:    rows,
		}, nil

	case "insertOne":
		doc, err := parseBSONArg(strings.TrimSpace(argsStr))
		if err != nil {
			return nil, fmt.Errorf("invalid document: %v", err)
		}
		result, err := coll.InsertOne(ctx, doc)
		if err != nil {
			return nil, fmt.Errorf("insertOne failed: %v", err)
		}
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "result", Type: "string"}},
			Rows:    [][]string{{fmt.Sprintf("Inserted ID: %v", result.InsertedID)}},
		}, nil

	case "updateOne":
		args, err := splitTopLevelArgs(argsStr)
		if err != nil || len(args) < 2 {
			return nil, fmt.Errorf("updateOne requires filter and update arguments")
		}
		filter, err := parseBSONArg(strings.TrimSpace(args[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid filter: %v", err)
		}
		update, err := parseBSONArg(strings.TrimSpace(args[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid update: %v", err)
		}
		result, err := coll.UpdateOne(ctx, filter, update)
		if err != nil {
			return nil, fmt.Errorf("updateOne failed: %v", err)
		}
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "result", Type: "string"}},
			Rows:    [][]string{{fmt.Sprintf("Matched: %d, Modified: %d", result.MatchedCount, result.ModifiedCount)}},
		}, nil

	case "deleteOne":
		filter, err := parseFilterArg(argsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid filter: %v", err)
		}
		result, err := coll.DeleteOne(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("deleteOne failed: %v", err)
		}
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "result", Type: "string"}},
			Rows:    [][]string{{fmt.Sprintf("Deleted: %d", result.DeletedCount)}},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported MongoDB operation: %s", operation)
	}
}

// parseFilterArg parses a filter argument string into a bson.M.
// An empty argument string returns an empty filter (match all).
func parseFilterArg(argsStr string) (bson.M, error) {
	argsStr = strings.TrimSpace(argsStr)
	if argsStr == "" {
		return bson.M{}, nil
	}
	parsed, err := parseBSONArg(argsStr)
	if err != nil {
		return nil, err
	}
	if d, ok := parsed.(bson.D); ok {
		return bsonDToM(d), nil
	}
	return bson.M{}, nil
}

// parsePipelineArg parses a JSON array string into a MongoDB aggregation pipeline.
func parsePipelineArg(argsStr string) ([]bson.D, error) {
	argsStr = strings.TrimSpace(argsStr)
	normalized := normalizeJSON(argsStr)

	var rawPipeline []json.RawMessage
	if err := json.Unmarshal([]byte(normalized), &rawPipeline); err != nil {
		return nil, fmt.Errorf("invalid pipeline array: %v", err)
	}

	pipeline := make([]bson.D, 0, len(rawPipeline))
	for _, raw := range rawPipeline {
		var stage bson.D
		if err := bson.UnmarshalExtJSON(raw, false, &stage); err != nil {
			return nil, fmt.Errorf("invalid pipeline stage: %v", err)
		}
		pipeline = append(pipeline, stage)
	}
	return pipeline, nil
}

// parseBSONArg parses a JSON-like string into a bson.D (preserving key order).
func parseBSONArg(s string) (any, error) {
	s = normalizeJSON(s)
	var doc bson.D
	if err := bson.UnmarshalExtJSON([]byte(s), false, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// normalizeJSON converts MongoDB shell-style relaxed JSON (unquoted keys) to strict JSON.
func normalizeJSON(s string) string {
	// Replace unquoted keys with quoted keys
	// Matches: start of object or comma, optional whitespace, then an unquoted key followed by colon
	re := regexp.MustCompile(`([{,]\s*)([a-zA-Z_$][a-zA-Z0-9_$]*)\s*:`)
	return re.ReplaceAllString(s, `$1"$2":`)
}

// splitTopLevelArgs splits a comma-separated argument list respecting nested braces/brackets.
func splitTopLevelArgs(s string) ([]string, error) {
	var args []string
	depth := 0
	start := 0
	inString := false
	escape := false

	for i, ch := range s {
		if escape {
			escape = false
			continue
		}
		if ch == '\\' {
			escape = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch ch {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		case ',':
			if depth == 0 {
				args = append(args, s[start:i])
				start = i + 1
			}
		}
	}
	args = append(args, s[start:])
	return args, nil
}

// bsonDToM converts a bson.D to bson.M for use in MongoDB driver methods that accept bson.M.
func bsonDToM(d bson.D) bson.M {
	m := bson.M{}
	for _, e := range d {
		m[e.Key] = e.Value
	}
	return m
}

// documentsToResult converts a slice of MongoDB documents to a GetRowsResult
// with a single "document" column containing JSON strings, matching the GetRows format.
func documentsToResult(docs []bson.M) *engine.GetRowsResult {
	result := &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "document", Type: "Document"}},
		Rows:    [][]string{},
	}
	for _, doc := range docs {
		jsonBytes, err := json.Marshal(doc)
		if err != nil {
			continue
		}
		result.Rows = append(result.Rows, []string{string(jsonBytes)})
	}
	return result
}
