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

package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/elastic/go-elasticsearch/v8/esutil"
)

// ExportCSV exports ElasticSearch index data to CSV format
func (p *ElasticSearchPlugin) ExportCSV(config *engine.PluginConfig, schema string, storageUnit string, writer func([]string) error, selectedRows []map[string]interface{}) error {
	// ElasticSearch doesn't support exporting selected rows from frontend
	if len(selectedRows) > 0 {
		return fmt.Errorf("exporting selected rows is not supported for ElasticSearch")
	}
	db, err := DB(config)
	if err != nil {
		return err
	}

	// Get mapping to understand fields
	mapping, err := db.Indices.GetMapping(
		db.Indices.GetMapping.WithIndex(storageUnit),
	)
	if err != nil {
		return fmt.Errorf("failed to get index mapping: %v", err)
	}
	defer mapping.Body.Close()

	var mappingResponse map[string]interface{}
	if err := json.NewDecoder(mapping.Body).Decode(&mappingResponse); err != nil {
		return fmt.Errorf("failed to decode mapping: %v", err)
	}

	// Extract field names from mapping
	fieldNames := p.extractFieldNames(mappingResponse, storageUnit)

	// Write headers
	headers := make([]string, len(fieldNames))
	for i, field := range fieldNames {
		headers[i] = common.FormatCSVHeader(field, "JSON")
	}
	if err := writer(headers); err != nil {
		return fmt.Errorf("failed to write headers: %v", err)
	}

	// Scroll through all documents
	res, err := db.Search(
		db.Search.WithContext(context.Background()),
		db.Search.WithIndex(storageUnit),
		db.Search.WithScroll(5*60*1000), // 5 minutes
		db.Search.WithSize(1000),
	)
	if err != nil {
		return fmt.Errorf("failed to search index: %v", err)
	}
	defer res.Body.Close()

	var searchResult map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		return fmt.Errorf("failed to decode search result: %v", err)
	}

	scrollID := searchResult["_scroll_id"].(string)
	rowCount := 0

	for {
		hits := searchResult["hits"].(map[string]interface{})["hits"].([]interface{})
		if len(hits) == 0 {
			break
		}

		for _, hit := range hits {
			doc := hit.(map[string]interface{})["_source"].(map[string]interface{})

			row := make([]string, len(fieldNames))
			for i, field := range fieldNames {
				if val, exists := p.getNestedValue(doc, field); exists {
					row[i] = p.formatElasticValue(val)
				} else {
					row[i] = ""
				}
			}

			if err := writer(row); err != nil {
				return fmt.Errorf("failed to write row: %v", err)
			}

			rowCount++
		}

		// Get next batch
		res, err = db.Scroll(
			db.Scroll.WithScrollID(scrollID),
			db.Scroll.WithScroll(5*60*1000),
		)
		if err != nil {
			break
		}
		defer res.Body.Close()

		searchResult = make(map[string]interface{})
		if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
			break
		}
	}


	return nil
}

// ImportCSV imports CSV data into ElasticSearch index
func (p *ElasticSearchPlugin) ImportCSV(config *engine.PluginConfig, schema string, storageUnit string, reader func() ([]string, error), mode engine.ImportMode, progressCallback func(engine.ImportProgress)) error {
	db, err := DB(config)
	if err != nil {
		return err
	}

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
		res, err := db.DeleteByQuery(
			[]string{storageUnit},
			strings.NewReader(`{"query": {"match_all": {}}}`),
		)
		if err != nil {
			return fmt.Errorf("failed to clear index: %v", err)
		}
		defer res.Body.Close()
	}

	// Create bulk indexer
	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client: db,
		Index:  storageUnit,
	})
	if err != nil {
		return fmt.Errorf("failed to create bulk indexer: %v", err)
	}

	// Process rows
	rowCount := 0
	for {
		row, err := reader()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("failed to read row %d: %v", rowCount+1, err)
		}

		// Create document from row
		doc := make(map[string]interface{})
		for i, colName := range columnNames {
			if i < len(row) {
				doc[colName] = p.parseElasticValue(row[i])
			}
		}

		data, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("failed to marshal document at row %d: %v", rowCount+1, err)
		}

		// Add to bulk indexer
		err = bi.Add(
			context.Background(),
			esutil.BulkIndexerItem{
				Action: "index",
				Body:   strings.NewReader(string(data)),
			},
		)
		if err != nil {
			return fmt.Errorf("failed to add document to bulk at row %d: %v", rowCount+1, err)
		}

		rowCount++
		if progressCallback != nil && rowCount%100 == 0 {
			progressCallback(engine.ImportProgress{
				ProcessedRows: rowCount,
				Status:        "importing",
			})
		}
	}

	// Close bulk indexer
	if err := bi.Close(context.Background()); err != nil {
		return fmt.Errorf("failed to close bulk indexer: %v", err)
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

func (p *ElasticSearchPlugin) extractFieldNames(mapping map[string]interface{}, indexName string) []string {
	fields := make(map[string]bool)

	if indexData, ok := mapping[indexName].(map[string]interface{}); ok {
		if mappings, ok := indexData["mappings"].(map[string]interface{}); ok {
			if properties, ok := mappings["properties"].(map[string]interface{}); ok {
				p.extractFieldsRecursive(properties, "", fields)
			}
		}
	}

	// Convert to slice
	result := make([]string, 0, len(fields))
	for field := range fields {
		result = append(result, field)
	}
	return result
}

func (p *ElasticSearchPlugin) extractFieldsRecursive(properties map[string]interface{}, prefix string, fields map[string]bool) {
	for name, prop := range properties {
		fullName := name
		if prefix != "" {
			fullName = prefix + "." + name
		}

		fields[fullName] = true

		if propMap, ok := prop.(map[string]interface{}); ok {
			if subProps, ok := propMap["properties"].(map[string]interface{}); ok {
				p.extractFieldsRecursive(subProps, fullName, fields)
			}
		}
	}
}

func (p *ElasticSearchPlugin) getNestedValue(doc map[string]interface{}, field string) (interface{}, bool) {
	parts := strings.Split(field, ".")
	current := doc

	for i, part := range parts {
		if i == len(parts)-1 {
			val, exists := current[part]
			return val, exists
		}

		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return nil, false
		}
	}

	return nil, false
}

func (p *ElasticSearchPlugin) formatElasticValue(val interface{}) string {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case string:
		return v
	case []interface{}, map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(data)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (p *ElasticSearchPlugin) parseElasticValue(val string) interface{} {
	if val == "" {
		return nil
	}

	// Try to parse as JSON
	if strings.HasPrefix(val, "{") || strings.HasPrefix(val, "[") {
		var parsed interface{}
		if err := json.Unmarshal([]byte(val), &parsed); err == nil {
			return parsed
		}
	}

	return val
}
