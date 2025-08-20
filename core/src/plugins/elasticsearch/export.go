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
	"github.com/clidey/whodb/core/src/log"
)

// ExportData exports ElasticSearch index data to tabular format
func (p *ElasticSearchPlugin) ExportData(config *engine.PluginConfig, schema string, storageUnit string, writer func([]string) error, selectedRows []map[string]any) error {
	// ElasticSearch doesn't support exporting selected rows from frontend
	if len(selectedRows) > 0 {
		err := fmt.Errorf("exporting selected rows is not supported for ElasticSearch")
		return err
	}
	db, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to connect to ElasticSearch while exporting data")
		return err
	}

	// Get mapping to understand fields
	mapping, err := db.Indices.GetMapping(
		db.Indices.GetMapping.WithIndex(storageUnit),
	)
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to get ElasticSearch index mapping for export")
		return fmt.Errorf("failed to get index mapping: %v", err)
	}
	defer mapping.Body.Close()

	var mappingResponse map[string]any
	if err := json.NewDecoder(mapping.Body).Decode(&mappingResponse); err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to decode ElasticSearch mapping response for export")
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
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to write CSV headers during ElasticSearch export")
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
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to execute ElasticSearch scroll search for export")
		return fmt.Errorf("failed to search index: %v", err)
	}
	defer res.Body.Close()

	var searchResult map[string]any
	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to decode ElasticSearch search result during export")
		return fmt.Errorf("failed to decode search result: %v", err)
	}

	scrollID := searchResult["_scroll_id"].(string)
	rowCount := 0

	for {
		hits := searchResult["hits"].(map[string]any)["hits"].([]any)
		if len(hits) == 0 {
			break
		}

		for _, hit := range hits {
			doc := hit.(map[string]any)["_source"].(map[string]any)

			row := make([]string, len(fieldNames))
			for i, field := range fieldNames {
				if val, exists := p.getNestedValue(doc, field); exists {
					row[i] = p.formatElasticValue(val)
				} else {
					row[i] = ""
				}
			}
			if err := writer(row); err != nil {
				log.Logger.WithError(err).WithField("storageUnit", storageUnit).WithField("rowCount", rowCount).Error("Failed to write row during ElasticSearch export")
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
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Error during ElasticSearch scroll operation, breaking export loop")
			break
		}
		defer res.Body.Close()

		searchResult = make(map[string]any)
		if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Error decoding ElasticSearch scroll response, breaking export loop")
			break
		}
	}

	return nil
}

// Helper functions

func (p *ElasticSearchPlugin) extractFieldNames(mapping map[string]any, indexName string) []string {
	fields := make(map[string]bool)

	if indexData, ok := mapping[indexName].(map[string]any); ok {
		if mappings, ok := indexData["mappings"].(map[string]any); ok {
			if properties, ok := mappings["properties"].(map[string]any); ok {
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

func (p *ElasticSearchPlugin) extractFieldsRecursive(properties map[string]any, prefix string, fields map[string]bool) {
	for name, prop := range properties {
		fullName := name
		if prefix != "" {
			fullName = prefix + "." + name
		}

		fields[fullName] = true

		if propMap, ok := prop.(map[string]any); ok {
			if subProps, ok := propMap["properties"].(map[string]any); ok {
				p.extractFieldsRecursive(subProps, fullName, fields)
			}
		}
	}
}

func (p *ElasticSearchPlugin) getNestedValue(doc map[string]any, field string) (any, bool) {
	parts := strings.Split(field, ".")
	current := doc

	for i, part := range parts {
		if i == len(parts)-1 {
			val, exists := current[part]
			return val, exists
		}

		if next, ok := current[part].(map[string]any); ok {
			current = next
		} else {
			return nil, false
		}
	}

	return nil, false
}

func (p *ElasticSearchPlugin) formatElasticValue(val any) string {
	if val == nil {
		return ""
	}

	var strVal string
	switch v := val.(type) {
	case string:
		strVal = v
	case []any, map[string]any:
		data, err := json.Marshal(v)
		if err != nil {
			log.Logger.WithError(err).Error("Failed to marshal ElasticSearch value to JSON during export formatting")
			strVal = fmt.Sprintf("%v", v)
		} else {
			strVal = string(data)
		}
	default:
		strVal = fmt.Sprintf("%v", v)
	}

	return common.EscapeFormula(strVal)
}
