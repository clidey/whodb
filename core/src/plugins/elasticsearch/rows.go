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

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common/graphutil"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

func (p *ElasticSearchPlugin) GetRows(config *engine.PluginConfig, req *engine.GetRowsRequest) (*engine.GetRowsResult, error) {
	collection := req.StorageUnit
	where, sortConds, pageSize, pageOffset := req.Where, req.Sort, req.PageSize, req.PageOffset
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithField("collection", collection).Error("Failed to connect to ElasticSearch while getting rows")
		return nil, err
	}

	// Convert the where condition to an Elasticsearch filter
	elasticSearchConditions, err := convertWhereConditionToES(where)
	if err != nil {
		log.WithError(err).WithField("collection", collection).Error("Failed to convert where condition to ElasticSearch query")
		return nil, fmt.Errorf("error converting where condition: %v", err)
	}

	query := map[string]any{
		"from": pageOffset,
		"size": pageSize,
		"query": map[string]any{
			"bool": elasticSearchConditions,
		},
	}

	// Apply sorting if provided
	// Skip "document" column as it's a virtual column representing the entire JSON document
	if len(sortConds) > 0 {
		sortArray := []map[string]any{}
		for _, s := range sortConds {
			if s.Column == "document" {
				continue
			}
			order := "asc"
			if s.Direction == model.SortDirectionDesc {
				order = "desc"
			}
			sortArray = append(sortArray, map[string]any{
				s.Column: map[string]any{
					"order": order,
				},
			})
		}
		if len(sortArray) > 0 {
			query["sort"] = sortArray
		}
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		log.WithError(err).WithField("collection", collection).Error("Failed to encode ElasticSearch query to JSON")
		return nil, err
	}

	res, err := client.Search(
		client.Search.WithContext(context.Background()),
		client.Search.WithIndex(collection),
		client.Search.WithBody(&buf),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.WithError(err).WithField("collection", collection).Error("Failed to execute ElasticSearch search query")
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		err := fmt.Errorf("error searching documents: %s", res.String())
		log.WithError(err).WithField("collection", collection).Error("ElasticSearch search API returned error")
		return nil, err
	}

	var searchResult map[string]any
	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		log.WithError(err).WithField("collection", collection).Error("Failed to decode ElasticSearch search response")
		return nil, err
	}

	hits, ok := searchResult["hits"].(map[string]any)["hits"].([]any)
	if !ok {
		err := fmt.Errorf("invalid response structure")
		log.WithError(err).WithField("collection", collection).Error("ElasticSearch search response has invalid structure")
		return nil, err
	}

	result := &engine.GetRowsResult{
		Columns: []engine.Column{
			{Name: "document", Type: "Document"},
		},
		Rows: [][]string{},
	}

	// Extract total count from the response (already tracked via WithTrackTotalHits)
	if hitsMap, ok := searchResult["hits"].(map[string]any); ok {
		if total, ok := hitsMap["total"].(map[string]any); ok {
			if value, ok := total["value"].(float64); ok {
				result.TotalCount = int64(value)
			}
		}
	}

	for _, hit := range hits {
		hitMap := hit.(map[string]any)
		source := hitMap["_source"].(map[string]any)
		id := hitMap["_id"]
		source["_id"] = id
		jsonBytes, err := json.Marshal(source)
		if err != nil {
			log.WithError(err).WithField("collection", collection).Error("Failed to marshal ElasticSearch document source to JSON")
			return nil, err
		}
		result.Rows = append(result.Rows, []string{string(jsonBytes)})
	}

	return result, nil
}

func (p *ElasticSearchPlugin) GetRowCount(config *engine.PluginConfig, database, index string, where *model.WhereCondition) (int64, error) {
	client, err := DB(config)
	if err != nil {
		return 0, err
	}

	elasticSearchConditions, err := convertWhereConditionToES(where)
	if err != nil {
		return 0, fmt.Errorf("error converting where condition: %v", err)
	}

	query := map[string]any{
		"query": map[string]any{
			"bool": elasticSearchConditions,
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return 0, err
	}

	res, err := client.Count(
		client.Count.WithContext(context.Background()),
		client.Count.WithIndex(index),
		client.Count.WithBody(&buf),
	)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return 0, fmt.Errorf("error counting documents: %s", res.String())
	}

	var countResult map[string]any
	if err := json.NewDecoder(res.Body).Decode(&countResult); err != nil {
		return 0, err
	}

	count, ok := countResult["count"].(float64)
	if !ok {
		return 0, fmt.Errorf("unexpected count response format")
	}

	return int64(count), nil
}

func (p *ElasticSearchPlugin) GetColumnsForTable(config *engine.PluginConfig, schema string, storageUnit string) ([]engine.Column, error) {
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"index": storageUnit,
		}).Error("Failed to connect to ElasticSearch for column inference")
		return nil, err
	}

	var buf bytes.Buffer
	query := map[string]any{
		"size": 100,
		"query": map[string]any{
			"match_all": map[string]any{},
		},
	}
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		log.WithError(err).WithField("index", storageUnit).Error("Failed to encode query for column inference")
		return nil, err
	}

	res, err := client.Search(
		client.Search.WithContext(context.Background()),
		client.Search.WithIndex(storageUnit),
		client.Search.WithBody(&buf),
	)
	if err != nil {
		log.WithError(err).WithField("index", storageUnit).Error("Failed to search for sample document")
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		log.WithField("index", storageUnit).Warn("No documents found, returning empty schema")
		return []engine.Column{}, nil
	}

	var searchResult map[string]any
	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		log.WithError(err).WithField("index", storageUnit).Error("Failed to decode search result")
		return nil, err
	}

	hits := searchResult["hits"].(map[string]any)["hits"].([]any)
	if len(hits) == 0 {
		return []engine.Column{}, nil
	}

	fieldTypes := make(map[string]string)
	for _, h := range hits {
		hitMap, ok := h.(map[string]any)
		if !ok {
			continue
		}
		source, ok := hitMap["_source"].(map[string]any)
		if !ok {
			continue
		}
		for fieldName, fieldValue := range source {
			fieldType := inferElasticSearchType(fieldValue)
			fieldTypes[fieldName] = mergeElasticTypes(fieldTypes[fieldName], fieldType)
		}
	}

	indicesRes, err := client.Indices.Stats()
	if err != nil {
		log.WithError(err).Error("Failed to get ElasticSearch indices for FK detection")
		return nil, err
	}
	defer indicesRes.Body.Close()

	if indicesRes.IsError() {
		log.Error("ElasticSearch indices stats API returned error for FK detection")
		return nil, fmt.Errorf("error getting indices: %s", indicesRes.String())
	}

	var stats map[string]any
	if err := json.NewDecoder(indicesRes.Body).Decode(&stats); err != nil {
		log.WithError(err).Error("Failed to decode ElasticSearch indices stats")
		return nil, err
	}

	indicesStats := stats["indices"].(map[string]any)
	var indices []string
	for indexName := range indicesStats {
		indices = append(indices, indexName)
	}

	if len(fieldTypes) == 0 {
		return []engine.Column{}, nil
	}

	fieldNames := make([]string, 0, len(fieldTypes))
	for name := range fieldTypes {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	// Infer FK relationships using shared heuristics
	fkMap := graphutil.InferForeignKeys(storageUnit, fieldNames, indices)
	fieldToRef := make(map[string]string, len(fkMap))
	for refUnit, field := range fkMap {
		fieldToRef[field] = refUnit
	}

	columns := []engine.Column{
		{
			Name:         "_id",
			Type:         "keyword",
			IsPrimary:    true,
			IsForeignKey: false,
		},
	}

	for _, fieldName := range fieldNames {
		fieldType := fieldTypes[fieldName]

		var isForeignKey bool
		var referencedTable *string
		if ref, ok := fieldToRef[fieldName]; ok {
			isForeignKey = true
			referencedTable = &ref
		}

		columns = append(columns, engine.Column{
			Name:            fieldName,
			Type:            fieldType,
			IsPrimary:       false,
			IsForeignKey:    isForeignKey,
			ReferencedTable: referencedTable,
		})
	}

	return columns, nil
}

func inferElasticSearchType(value any) string {
	if value == nil {
		return "null"
	}

	switch value.(type) {
	case string:
		return "text"
	case float64, float32:
		return "float"
	case int, int32, int64:
		return "long"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "keyword"
	}
}

// mergeElasticTypes combines inferred types; conflicting types become "mixed".
func mergeElasticTypes(current, next string) string {
	if current == "" {
		return next
	}
	if current == next {
		return current
	}
	return "mixed"
}
