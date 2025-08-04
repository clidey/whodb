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
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ExportCSV exports MongoDB collection data to CSV format
func (p *MongoDBPlugin) ExportCSV(config *engine.PluginConfig, schema string, storageUnit string, writer func([]string) error, progressCallback func(int)) error {
	client, err := DB(config)
	if err != nil {
		return err
	}

	db := client.Database(schema)
	collection := db.Collection(storageUnit)
	
	// First, get all field names from a sample of documents
	fieldNames, err := p.getCollectionFields(collection)
	if err != nil {
		return fmt.Errorf("failed to get collection fields: %v", err)
	}

	// Write headers with type information
	headers := make([]string, len(fieldNames))
	for i, field := range fieldNames {
		headers[i] = common.FormatCSVHeader(field, "BSON")
	}
	if err := writer(headers); err != nil {
		return fmt.Errorf("failed to write headers: %v", err)
	}

	// Export all documents
	cursor, err := collection.Find(context.Background(), bson.D{})
	if err != nil {
		return fmt.Errorf("failed to query collection: %v", err)
	}
	defer cursor.Close(context.Background())

	rowCount := 0
	for cursor.Next(context.Background()) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return fmt.Errorf("failed to decode document: %v", err)
		}

		row := make([]string, len(fieldNames))
		for i, field := range fieldNames {
			if val, exists := doc[field]; exists {
				row[i] = p.formatBSONValue(val)
			} else {
				row[i] = ""
			}
		}

		if err := writer(row); err != nil {
			return fmt.Errorf("failed to write row: %v", err)
		}

		rowCount++
		if progressCallback != nil && rowCount%1000 == 0 {
			progressCallback(rowCount)
		}
	}

	if progressCallback != nil {
		progressCallback(rowCount)
	}

	return cursor.Err()
}

// ImportCSV imports CSV data into MongoDB collection
func (p *MongoDBPlugin) ImportCSV(config *engine.PluginConfig, schema string, storageUnit string, reader func() ([]string, error), mode engine.ImportMode, progressCallback func(engine.ImportProgress)) error {
	client, err := DB(config)
	if err != nil {
		return err
	}

	db := client.Database(schema)
	collection := db.Collection(storageUnit)

	// Read headers
	headers, err := reader()
	if err != nil {
		return fmt.Errorf("failed to read headers: %v", err)
	}

	// Parse column names from headers
	columnNames, _, err := common.ParseCSVHeaders(headers)
	if err != nil {
		return err
	}

	// Handle override mode
	if mode == engine.ImportModeOverride {
		if err := collection.Drop(context.Background()); err != nil {
			return fmt.Errorf("failed to clear collection: %v", err)
		}
	}

	// Process rows
	rowCount := 0
	var documents []interface{}
	batchSize := 1000

	for {
		row, err := reader()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("failed to read row %d: %v", rowCount+1, err)
		}

		// Create document from row
		doc := bson.M{}
		for i, colName := range columnNames {
			if i < len(row) {
				doc[colName] = p.parseBSONValue(row[i])
			}
		}

		documents = append(documents, doc)

		// Insert in batches
		if len(documents) >= batchSize {
			if _, err := collection.InsertMany(context.Background(), documents); err != nil {
				return fmt.Errorf("failed to insert documents at row %d: %v", rowCount+1, err)
			}
			documents = documents[:0]
		}

		rowCount++
		if progressCallback != nil && rowCount%100 == 0 {
			progressCallback(engine.ImportProgress{
				ProcessedRows: rowCount,
				Status:        "importing",
			})
		}
	}

	// Insert remaining documents
	if len(documents) > 0 {
		if _, err := collection.InsertMany(context.Background(), documents); err != nil {
			return fmt.Errorf("failed to insert final batch: %v", err)
		}
	}

	if progressCallback != nil {
		progressCallback(engine.ImportProgress{
			ProcessedRows: rowCount,
			Status:        "completed",
		})
	}

	return nil
}

// Helper functions

func (p *MongoDBPlugin) getCollectionFields(collection *mongo.Collection) ([]string, error) {
	// Sample documents to get field names
	opts := options.Find().SetLimit(100)
	cursor, err := collection.Find(context.Background(), bson.D{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

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
	
	// Ensure _id is first if present
	for i, field := range fields {
		if field == "_id" && i != 0 {
			fields[0], fields[i] = fields[i], fields[0]
			break
		}
	}

	return fields, nil
}

func (p *MongoDBPlugin) formatBSONValue(val interface{}) string {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case string:
		return v
	case []interface{}, bson.A, bson.M, map[string]interface{}:
		// Convert complex types to JSON
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(data)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (p *MongoDBPlugin) parseBSONValue(val string) interface{} {
	if val == "" {
		return nil
	}

	// Try to parse as JSON for complex types
	if strings.HasPrefix(val, "{") || strings.HasPrefix(val, "[") {
		var parsed interface{}
		if err := json.Unmarshal([]byte(val), &parsed); err == nil {
			return parsed
		}
	}

	// Return as string by default
	return val
}