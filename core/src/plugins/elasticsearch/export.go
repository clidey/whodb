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
func (p *ElasticSearchPlugin) ExportData(config *engine.PluginConfig, schema string, storageUnit string, writer engine.ExportWriter, selection *engine.ExportSelection) error {
	// Selected rows are the grid's own cells, so they are written as they are.
	if selection.HasRows() {
		return selection.Write(writer, nil)
	}
	db, err := DB(config)
	if err != nil {
		log.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to connect to ElasticSearch while exporting data")
		return err
	}

	// Get mapping to understand fields
	mapping, err := db.Indices.GetMapping(
		db.Indices.GetMapping.WithIndex(storageUnit),
	)
	if err != nil {
		log.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to get ElasticSearch index mapping for export")
		return fmt.Errorf("failed to get index mapping: %w", err)
	}
	defer func() { _ = mapping.Body.Close() }()

	var mappingResponse map[string]any
	if err := json.NewDecoder(mapping.Body).Decode(&mappingResponse); err != nil {
		log.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to decode ElasticSearch mapping response for export")
		return fmt.Errorf("failed to decode mapping: %w", err)
	}

	// Extract field names from mapping
	fieldNames := p.extractFieldNames(mappingResponse, storageUnit)

	// Write headers
	headers := make([]string, len(fieldNames))
	for i, field := range fieldNames {
		headers[i] = common.FormatCSVHeader(field, "JSON")
	}
	if err := writer(headers, nil); err != nil {
		log.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to write CSV headers during ElasticSearch export")
		return fmt.Errorf("failed to write headers: %w", err)
	}

	// Scroll through all documents
	res, err := db.Search(
		db.Search.WithContext(context.Background()),
		db.Search.WithIndex(storageUnit),
		db.Search.WithScroll(5*60*1000), // 5 minutes
		db.Search.WithSize(1000),
	)
	if err != nil {
		log.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to execute ElasticSearch scroll search for export")
		return fmt.Errorf("failed to search index: %w", err)
	}

	var searchResult map[string]any
	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		_ = res.Body.Close()
		log.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to decode ElasticSearch search result during export")
		return fmt.Errorf("failed to decode search result: %w", err)
	}
	_ = res.Body.Close()

	scrollID, _ := searchResult["_scroll_id"].(string)
	rowCount := 0

	for {
		hits, err := responseHits(searchResult)
		if err != nil {
			log.WithError(err).WithField("storageUnit", storageUnit).Error("ElasticSearch search response has invalid structure during export")
			return err
		}
		if len(hits) == 0 {
			break
		}

		for _, hit := range hits {
			doc, ok := hitSource(hit)
			if !ok {
				continue
			}

			row := make([]string, len(fieldNames))
			text := make([]bool, len(fieldNames))
			for i, field := range fieldNames {
				if val, exists := p.getNestedValue(doc, field); exists {
					row[i], text[i] = formatElasticValue(val)
				}
			}
			if err := writer(row, text); err != nil {
				log.WithError(err).WithField("storageUnit", storageUnit).WithField("rowCount", rowCount).Error("Failed to write row during ElasticSearch export")
				return fmt.Errorf("failed to write row: %w", err)
			}

			rowCount++
		}

		if scrollID == "" {
			break
		}

		// Get next batch
		res, err = db.Scroll(
			db.Scroll.WithScrollID(scrollID),
			db.Scroll.WithScroll(5*60*1000),
		)
		if err != nil {
			log.WithError(err).WithField("storageUnit", storageUnit).Error("Error during ElasticSearch scroll operation")
			return fmt.Errorf("scroll failed: %w", err)
		}

		searchResult = make(map[string]any)
		if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
			_ = res.Body.Close()
			log.WithError(err).WithField("storageUnit", storageUnit).Error("Error decoding ElasticSearch scroll response")
			return fmt.Errorf("failed to decode scroll response: %w", err)
		}
		_ = res.Body.Close()

		if nextScrollID, ok := searchResult["_scroll_id"].(string); ok && nextScrollID != "" {
			scrollID = nextScrollID
		}
	}

	return nil
}

// ExportDataNDJSON streams Elasticsearch data as NDJSON.
func (p *ElasticSearchPlugin) ExportDataNDJSON(config *engine.PluginConfig, schema string, storageUnit string, writer func(string) error, selectedRows []map[string]any) error {
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

	db, err := DB(config)
	if err != nil {
		return err
	}

	res, err := db.Search(
		db.Search.WithContext(context.Background()),
		db.Search.WithIndex(storageUnit),
		db.Search.WithScroll(5*60*1000), // 5 minutes
		db.Search.WithSize(1000),
	)
	if err != nil {
		return fmt.Errorf("failed to search index: %w", err)
	}

	var searchResult map[string]any
	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		_ = res.Body.Close()
		return fmt.Errorf("failed to decode search result: %w", err)
	}
	_ = res.Body.Close()

	scrollID, _ := searchResult["_scroll_id"].(string)

	for {
		hits, err := responseHits(searchResult)
		if err != nil {
			return err
		}
		if len(hits) == 0 {
			break
		}

		for _, hit := range hits {
			doc, ok := hitSource(hit)
			if !ok {
				continue
			}
			line, err := json.Marshal(doc)
			if err != nil {
				return err
			}
			if err := writer(string(line)); err != nil {
				return err
			}
		}

		if scrollID == "" {
			break
		}

		res, err = db.Scroll(
			db.Scroll.WithScrollID(scrollID),
			db.Scroll.WithScroll(5*60*1000),
		)
		if err != nil {
			return fmt.Errorf("scroll failed: %w", err)
		}

		searchResult = make(map[string]any)
		if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
			_ = res.Body.Close()
			return fmt.Errorf("failed to decode scroll response: %w", err)
		}
		_ = res.Body.Close()

		if nextScrollID, ok := searchResult["_scroll_id"].(string); ok && nextScrollID != "" {
			scrollID = nextScrollID
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

// formatElasticValue renders one document field the way the grid's document
// JSON shows it, and reports whether it is free text. Strings are written
// bare; every other value is its JSON.
func formatElasticValue(val any) (string, bool) {
	switch v := val.(type) {
	case nil:
		return "", false
	case string:
		return v, true
	}

	data, err := json.Marshal(val)
	if err != nil {
		log.WithError(err).Error("Failed to marshal ElasticSearch value to JSON during export formatting")
		return fmt.Sprintf("%v", val), true
	}
	return string(data), false
}
