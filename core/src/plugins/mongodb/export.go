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
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// ExportData exports MongoDB collection data to tabular format
func (p *MongoDBPlugin) ExportData(config *engine.PluginConfig, schema string, storageUnit string, writer engine.ExportWriter, selection *engine.ExportSelection) error {
	// Selected rows are the grid's own cells, so they are written as they are.
	if selection.HasRows() {
		return selection.Write(writer, nil)
	}
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"schema":      schema,
			"storageUnit": storageUnit,
		}).Error("Failed to connect to MongoDB for data export")
		return err
	}

	db := client.Database(schema)
	collection := db.Collection(storageUnit)

	// First, get all field names from a sample of documents
	fieldNames, err := p.getCollectionFields(collection)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"schema":      schema,
			"storageUnit": storageUnit,
		}).Error("Failed to get MongoDB collection fields for export")
		return fmt.Errorf("failed to get collection fields: %w", err)
	}

	// Write headers with type information
	headers := make([]string, len(fieldNames))
	for i, field := range fieldNames {
		headers[i] = common.FormatCSVHeader(field, "BSON")
	}
	if err := writer(headers, nil); err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"schema":      schema,
			"storageUnit": storageUnit,
			"headerCount": len(headers),
		}).Error("Failed to write CSV headers for MongoDB export")
		return fmt.Errorf("failed to write headers: %w", err)
	}

	// Export all documents
	cursor, err := collection.Find(context.Background(), bson.D{})
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"schema":      schema,
			"storageUnit": storageUnit,
		}).Error("Failed to query MongoDB collection for export")
		return fmt.Errorf("failed to query collection: %w", err)
	}
	defer func() { _ = cursor.Close(context.Background()) }()

	rowCount := 0
	for cursor.Next(context.Background()) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			log.WithError(err).WithFields(map[string]any{
				"hostname":    config.Credentials.Hostname,
				"schema":      schema,
				"storageUnit": storageUnit,
				"rowNumber":   rowCount + 1,
			}).Error("Failed to decode MongoDB document during export")
			return fmt.Errorf("failed to decode document: %w", err)
		}

		row := make([]string, len(fieldNames))
		text := make([]bool, len(fieldNames))
		for i, field := range fieldNames {
			if val, exists := doc[field]; exists {
				row[i], text[i] = formatBSONValue(val)
			}
		}

		if err := writer(row, text); err != nil {
			log.WithError(err).WithFields(map[string]any{
				"hostname":    config.Credentials.Hostname,
				"schema":      schema,
				"storageUnit": storageUnit,
				"rowNumber":   rowCount + 1,
			}).Error("Failed to write CSV row for MongoDB export")
			return fmt.Errorf("failed to write row: %w", err)
		}

		rowCount++
	}

	return cursor.Err()
}

// ExportDataNDJSON streams MongoDB data as NDJSON.
func (p *MongoDBPlugin) ExportDataNDJSON(config *engine.PluginConfig, schema string, storageUnit string, writer func(string) error, selectedRows []map[string]any) error {
	if len(selectedRows) > 0 {
		for _, row := range selectedRows {
			line, err := json.Marshal(row)
			if err != nil {
				return err
			}
			if err := writer(string(line)); err != nil {
				return err
			}
		}
		return nil
	}

	client, err := DB(config)
	if err != nil {
		return err
	}
	defer disconnectClient(client)

	collection := client.Database(schema).Collection(storageUnit)
	cursor, err := collection.Find(context.Background(), bson.D{})
	if err != nil {
		return err
	}
	defer func() { _ = cursor.Close(context.Background()) }()

	for cursor.Next(context.Background()) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return err
		}
		line, err := json.Marshal(doc)
		if err != nil {
			return err
		}
		if err := writer(string(line)); err != nil {
			return err
		}
	}

	return cursor.Err()
}

// Helper functions

func (p *MongoDBPlugin) getCollectionFields(collection *mongo.Collection) ([]string, error) {
	// Sample documents to get field names
	opts := options.Find().SetLimit(100)
	cursor, err := collection.Find(context.Background(), bson.D{}, opts)
	if err != nil {
		log.WithError(err).WithField("collectionName", collection.Name()).Error("Failed to sample MongoDB collection documents for field extraction")
		return nil, err
	}
	defer func() { _ = cursor.Close(context.Background()) }()

	fieldSet := make(map[string]bool)
	for cursor.Next(context.Background()) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		for field := range doc {
			fieldSet[field] = true
		}
	}

	// Convert to sorted slice
	fields := make([]string, 0, len(fieldSet))
	for field := range fieldSet {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	// Ensure _id is first if present
	for i, field := range fields {
		if field == "_id" && i != 0 {
			fields[0], fields[i] = fields[i], fields[0]
			break
		}
	}

	return fields, nil
}

// formatBSONValue renders one document field the way the grid's document JSON
// shows it, and reports whether it is free text. Strings are written bare;
// ObjectIDs, dates and decimals are unwrapped from their one-key Extended JSON
// form; everything else is the field's Extended JSON.
func formatBSONValue(val any) (string, bool) {
	switch v := val.(type) {
	case nil:
		return "", false
	case string:
		return v, true
	case bson.JavaScript:
		return string(v), true
	case bson.Symbol:
		return string(v), true
	}

	data, err := bson.MarshalExtJSON(bson.D{{Key: "v", Value: val}}, false, false)
	if err != nil {
		log.WithError(err).Warn("Failed to marshal MongoDB value to Extended JSON during export, using string representation")
		return fmt.Sprintf("%v", val), true
	}
	rendered := strings.TrimSuffix(strings.TrimPrefix(string(data), `{"v":`), "}")

	var wrapped map[string]string
	if json.Unmarshal([]byte(rendered), &wrapped) == nil && len(wrapped) == 1 {
		for _, key := range []string{"$oid", "$date", "$numberDecimal"} {
			if inner, ok := wrapped[key]; ok {
				return inner, false
			}
		}
	}
	return rendered, false
}
