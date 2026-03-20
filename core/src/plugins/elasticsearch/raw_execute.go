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

package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// RawExecute parses and executes Elasticsearch queries.
// Expected format: INDEX_NAME | QUERY_JSON
// The query JSON is passed directly to the Elasticsearch Search API.
func (p *ElasticSearchPlugin) RawExecute(config *engine.PluginConfig, query string, _ ...any) (*engine.GetRowsResult, error) {
	query = strings.TrimSpace(query)

	parts := strings.SplitN(query, " | ", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("expected format: INDEX_NAME | QUERY_JSON")
	}

	indexName := strings.TrimSpace(parts[0])
	queryJSON := strings.TrimSpace(parts[1])

	if indexName == "" {
		return nil, fmt.Errorf("index name cannot be empty")
	}

	client, err := DB(config)
	if err != nil {
		log.WithError(err).Error("Failed to connect to Elasticsearch for raw execute")
		return nil, err
	}

	var queryMap map[string]any
	if err := json.Unmarshal([]byte(queryJSON), &queryMap); err != nil {
		return nil, fmt.Errorf("invalid query JSON: %v", err)
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(queryMap); err != nil {
		return nil, fmt.Errorf("failed to encode query: %v", err)
	}

	res, err := client.Search(
		client.Search.WithContext(context.Background()),
		client.Search.WithIndex(indexName),
		client.Search.WithBody(&buf),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("search failed: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("Elasticsearch error: %s", res.String())
	}

	var searchResult map[string]any
	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	_, hasAggs := queryMap["aggs"]
	if !hasAggs {
		_, hasAggs = queryMap["aggregations"]
	}

	if hasAggs {
		return formatAggregationResult(searchResult)
	}

	return formatSearchResult(searchResult)
}

// formatSearchResult converts Elasticsearch search hits into a GetRowsResult
// with a single "document" column containing JSON strings, matching the GetRows format.
func formatSearchResult(searchResult map[string]any) (*engine.GetRowsResult, error) {
	result := &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "document", Type: "Document"}},
		Rows:    [][]string{},
	}

	hitsMap, ok := searchResult["hits"].(map[string]any)
	if !ok {
		return result, nil
	}

	hits, ok := hitsMap["hits"].([]any)
	if !ok {
		return result, nil
	}

	for _, hit := range hits {
		hitMap, ok := hit.(map[string]any)
		if !ok {
			continue
		}
		source, ok := hitMap["_source"].(map[string]any)
		if !ok {
			continue
		}
		source["_id"] = hitMap["_id"]
		jsonBytes, err := json.Marshal(source)
		if err != nil {
			continue
		}
		result.Rows = append(result.Rows, []string{string(jsonBytes)})
	}

	return result, nil
}

// formatAggregationResult converts Elasticsearch aggregation results into a key-value GetRowsResult.
func formatAggregationResult(searchResult map[string]any) (*engine.GetRowsResult, error) {
	aggs, ok := searchResult["aggregations"].(map[string]any)
	if !ok {
		return &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "result", Type: "string"}},
			Rows:    [][]string{{"No aggregation results"}},
		}, nil
	}

	result := &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "key", Type: "string"}, {Name: "value", Type: "string"}},
		Rows:    [][]string{},
	}

	// Sort aggregation names for deterministic output
	aggNames := make([]string, 0, len(aggs))
	for name := range aggs {
		aggNames = append(aggNames, name)
	}
	sort.Strings(aggNames)

	for _, aggName := range aggNames {
		aggData := aggs[aggName]
		aggMap, ok := aggData.(map[string]any)
		if !ok {
			continue
		}

		// Handle bucket aggregations (terms, histogram, etc.)
		if buckets, ok := aggMap["buckets"].([]any); ok {
			for _, bucket := range buckets {
				bucketMap, ok := bucket.(map[string]any)
				if !ok {
					continue
				}
				key := fmt.Sprintf("%v", bucketMap["key"])
				docCount := fmt.Sprintf("%v", bucketMap["doc_count"])
				result.Rows = append(result.Rows, []string{key, docCount})
			}
			continue
		}

		// Handle metric aggregations (avg, sum, min, max, etc.)
		if value, ok := aggMap["value"]; ok {
			result.Rows = append(result.Rows, []string{aggName, fmt.Sprintf("%v", value)})
			continue
		}

		// Fallback: serialize the whole aggregation as JSON
		jsonBytes, err := json.Marshal(aggMap)
		if err != nil {
			continue
		}
		result.Rows = append(result.Rows, []string{aggName, string(jsonBytes)})
	}

	return result, nil
}
