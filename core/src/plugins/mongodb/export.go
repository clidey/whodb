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
	"time"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ExportData exports MongoDB collection data to tabular format
func (p *MongoDBPlugin) ExportData(config *engine.PluginConfig, schema string, storageUnit string, writer func([]string) error, selectedRows []map[string]any) error {
	// If selected rows are provided, export only those
	if len(selectedRows) > 0 {
		// Extract field names from selected rows
		fieldSet := make(map[string]bool)
		for _, row := range selectedRows {
			for k := range row {
				fieldSet[k] = true
			}
		}

		// Convert to sorted slice
		fieldNames := make([]string, 0, len(fieldSet))
		for field := range fieldSet {
			fieldNames = append(fieldNames, field)
		}
		sort.Strings(fieldNames)

		// Write headers
		headers := make([]string, len(fieldNames))
		for i, field := range fieldNames {
			headers[i] = common.FormatCSVHeader(field, "BSON")
		}
		if err := writer(headers); err != nil {
			return fmt.Errorf("failed to write headers: %v", err)
		}

		// Write selected rows
		for _, row := range selectedRows {
			rowData := make([]string, len(fieldNames))
			for i, field := range fieldNames {
				if val, exists := row[field]; exists {
					rowData[i] = p.formatBSONValue(val)
				} else {
					rowData[i] = ""
				}
			}
			if err := writer(rowData); err != nil {
				return fmt.Errorf("failed to write row: %v", err)
			}
		}
		return nil
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
		return fmt.Errorf("failed to get collection fields: %v", err)
	}

	// Write headers with type information
	headers := make([]string, len(fieldNames))
	for i, field := range fieldNames {
		headers[i] = common.FormatCSVHeader(field, "BSON")
	}
	if err := writer(headers); err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"schema":      schema,
			"storageUnit": storageUnit,
			"headerCount": len(headers),
		}).Error("Failed to write CSV headers for MongoDB export")
		return fmt.Errorf("failed to write headers: %v", err)
	}

	// Export all documents
	cursor, err := collection.Find(context.Background(), bson.D{})
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname":    config.Credentials.Hostname,
			"schema":      schema,
			"storageUnit": storageUnit,
		}).Error("Failed to query MongoDB collection for export")
		return fmt.Errorf("failed to query collection: %v", err)
	}
	defer cursor.Close(context.Background())

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
			log.WithError(err).WithFields(map[string]any{
				"hostname":    config.Credentials.Hostname,
				"schema":      schema,
				"storageUnit": storageUnit,
				"rowNumber":   rowCount + 1,
			}).Error("Failed to write CSV row for MongoDB export")
			return fmt.Errorf("failed to write row: %v", err)
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
	defer client.Disconnect(context.Background())

	collection := client.Database(schema).Collection(storageUnit)
	cursor, err := collection.Find(context.Background(), bson.D{})
	if err != nil {
		return err
	}
	defer cursor.Close(context.Background())

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

func (p *MongoDBPlugin) formatBSONValue(val any) string {
	if val == nil {
		return ""
	}

	var strVal string
	switch v := val.(type) {
	case string:
		strVal = v
	case bool:
		strVal = fmt.Sprintf("%t", v)
	case int, int8, int16, int32, int64:
		strVal = fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		strVal = fmt.Sprintf("%d", v)
	case float32, float64:
		strVal = fmt.Sprintf("%g", v)
	case time.Time:
		strVal = v.Format(time.RFC3339)
	case primitive.DateTime:
		strVal = v.Time().Format(time.RFC3339)
	case primitive.Timestamp:
		strVal = fmt.Sprintf("Timestamp(%d,%d)", v.T, v.I)
	case primitive.ObjectID:
		strVal = v.Hex()
	case primitive.Binary:
		strVal = fmt.Sprintf("Binary(subtype:%d,len:%d)", v.Subtype, len(v.Data))
	case primitive.Regex:
		strVal = fmt.Sprintf("/%s/%s", v.Pattern, v.Options)
	case primitive.DBPointer:
		strVal = fmt.Sprintf("DBPointer(%s,%s)", v.DB, v.Pointer.Hex())
	case primitive.JavaScript:
		strVal = string(v)
	case primitive.Symbol:
		strVal = string(v)
	case primitive.CodeWithScope:
		scopeJSON, _ := json.Marshal(v.Scope)
		strVal = fmt.Sprintf("CodeWithScope(%s, %s)", v.Code, string(scopeJSON))
	case primitive.Decimal128:
		strVal = v.String()
	case primitive.MinKey:
		strVal = "MinKey"
	case primitive.MaxKey:
		strVal = "MaxKey"
	case primitive.Null:
		strVal = "null"
	case primitive.Undefined:
		strVal = "undefined"
	case []any, bson.A:
		// Convert arrays to JSON
		data, err := json.Marshal(v)
		if err != nil {
			log.WithError(err).WithField("valueType", "array").Warn("Failed to marshal array value to JSON during MongoDB export, using string representation")
			strVal = fmt.Sprintf("%v", v)
		} else {
			strVal = string(data)
		}
	case bson.M, map[string]any, bson.D:
		// Convert documents to JSON
		data, err := json.Marshal(v)
		if err != nil {
			log.WithError(err).WithField("valueType", "document").Warn("Failed to marshal document value to JSON during MongoDB export, using string representation")
			strVal = fmt.Sprintf("%v", v)
		} else {
			strVal = string(data)
		}
	case bson.E:
		// Handle single BSON element
		strVal = fmt.Sprintf("%s: %v", v.Key, v.Value)
	default:
		// Fallback for any other types
		strVal = fmt.Sprintf("%v", v)
	}

	// Apply formula injection protection
	return common.EscapeFormula(strVal)
}
