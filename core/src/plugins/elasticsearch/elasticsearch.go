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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

var (
	supportedOperators = map[string]string{
		"match": "match", "match_phrase": "match_phrase", "match_phrase_prefix": "match_phrase_prefix", "multi_match": "multi_match",
		"bool": "bool", "term": "term", "terms": "terms", "range": "range", "exists": "exists", "prefix": "prefix", "wildcard": "wildcard",
		"regexp": "regexp", "fuzzy": "fuzzy", "ids": "ids", "constant_score": "constant_score", "function_score": "function_score",
		"dis_max": "dis_max", "nested": "nested", "has_child": "has_child", "has_parent": "has_parent",
	}
)

type ElasticSearchPlugin struct{}

func (p *ElasticSearchPlugin) IsAvailable(config *engine.PluginConfig) bool {
	client, err := DB(config)
	if err != nil {
		return false
	}
	res, err := client.Info()
	if err != nil || res.IsError() {
		return false
	}
	return true
}

func (p *ElasticSearchPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (p *ElasticSearchPlugin) GetAllSchemas(config *engine.PluginConfig) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (p *ElasticSearchPlugin) GetStorageUnits(config *engine.PluginConfig, database string) ([]engine.StorageUnit, error) {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).Error("Failed to connect to ElasticSearch while getting storage units")
		return nil, err
	}

	res, err := client.Indices.Stats()
	if err != nil {
		log.Logger.WithError(err).Error("Failed to get ElasticSearch indices stats")
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		err := fmt.Errorf("error getting stats for indices: %s", res.String())
		log.Logger.WithError(err).Error("ElasticSearch indices stats API returned error")
		return nil, err
	}

	var stats map[string]any
	if err := json.NewDecoder(res.Body).Decode(&stats); err != nil {
		log.Logger.WithError(err).Error("Failed to decode ElasticSearch indices stats response")
		return nil, err
	}

	indicesStats, ok := stats["indices"].(map[string]any)
	if !ok {
		log.Logger.Error("Unexpected indices stats format from ElasticSearch")
		return nil, fmt.Errorf("unexpected indices stats format")
	}

	storageUnits := make([]engine.StorageUnit, 0, len(indicesStats))

	for indexName, indexStatsInterface := range indicesStats {
		indexStats, ok := indexStatsInterface.(map[string]any)
		if !ok {
			log.Logger.Warnf("Skipping index %s: unexpected stats format", indexName)
			continue
		}

		primaries, ok := indexStats["primaries"].(map[string]any)
		if !ok {
			log.Logger.Warnf("Skipping index %s: missing primaries data", indexName)
			continue
		}

		docs, ok := primaries["docs"].(map[string]any)
		if !ok {
			log.Logger.Warnf("Skipping index %s: missing docs data", indexName)
			continue
		}

		store, ok := primaries["store"].(map[string]any)
		if !ok {
			log.Logger.Warnf("Skipping index %s: missing store data", indexName)
			continue
		}

		storageUnit := engine.StorageUnit{
			Name: indexName,
			Attributes: []engine.Record{
				{Key: "Storage Size", Value: fmt.Sprintf("%v", store["size_in_bytes"])},
				{Key: "Count", Value: fmt.Sprintf("%v", docs["count"])},
			},
		}
		storageUnits = append(storageUnits, storageUnit)
	}

	return storageUnits, nil
}

func (p *ElasticSearchPlugin) GetRows(config *engine.PluginConfig, database, collection string, where *model.WhereCondition, sort []*model.SortCondition, pageSize, pageOffset int) (*engine.GetRowsResult, error) {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithField("collection", collection).Error("Failed to connect to ElasticSearch while getting rows")
		return nil, err
	}

	// Convert the where condition to an Elasticsearch filter
	elasticSearchConditions, err := convertWhereConditionToES(where)
	if err != nil {
		log.Logger.WithError(err).WithField("collection", collection).Error("Failed to convert where condition to ElasticSearch query")
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
	if len(sort) > 0 {
		sortArray := []map[string]any{}
		for _, s := range sort {
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
		query["sort"] = sortArray
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		log.Logger.WithError(err).WithField("collection", collection).Error("Failed to encode ElasticSearch query to JSON")
		return nil, err
	}

	res, err := client.Search(
		client.Search.WithContext(context.Background()),
		client.Search.WithIndex(collection),
		client.Search.WithBody(&buf),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Logger.WithError(err).WithField("collection", collection).Error("Failed to execute ElasticSearch search query")
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		err := fmt.Errorf("error searching documents: %s", res.String())
		log.Logger.WithError(err).WithField("collection", collection).Error("ElasticSearch search API returned error")
		return nil, err
	}

	var searchResult map[string]any
	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		log.Logger.WithError(err).WithField("collection", collection).Error("Failed to decode ElasticSearch search response")
		return nil, err
	}

	hits, ok := searchResult["hits"].(map[string]any)["hits"].([]any)
	if !ok {
		err := fmt.Errorf("invalid response structure")
		log.Logger.WithError(err).WithField("collection", collection).Error("ElasticSearch search response has invalid structure")
		return nil, err
	}

	result := &engine.GetRowsResult{
		Columns: []engine.Column{
			{Name: "document", Type: "Document"},
		},
		Rows: [][]string{},
	}

	for _, hit := range hits {
		hitMap := hit.(map[string]any)
		source := hitMap["_source"].(map[string]any)
		id := hitMap["_id"]
		source["_id"] = id
		jsonBytes, err := json.Marshal(source)
		if err != nil {
			log.Logger.WithError(err).WithField("collection", collection).Error("Failed to marshal ElasticSearch document source to JSON")
			return nil, err
		}
		result.Rows = append(result.Rows, []string{string(jsonBytes)})
	}

	return result, nil
}

func (p *ElasticSearchPlugin) GetColumnsForTable(config *engine.PluginConfig, schema string, storageUnit string) ([]engine.Column, error) {
	// Elasticsearch doesn't have a traditional column structure, it returns documents
	return []engine.Column{
		{Name: "document", Type: "Document"},
	}, nil
}

// convertAtomicConditionToES converts an atomic where condition to an Elasticsearch query clause
func convertAtomicConditionToES(atomic *model.AtomicWhereCondition) (map[string]any, error) {
	if atomic == nil {
		return nil, fmt.Errorf("atomic condition is nil")
	}

	// Handle different operators
	switch atomic.Operator {
	case "match", "MATCH":
		// Full-text search
		return map[string]any{
			"match": map[string]any{
				atomic.Key: atomic.Value,
			},
		}, nil

	case "=", "eq", "EQ", "equals", "EQUALS":
		// Exact match
		return map[string]any{
			"term": map[string]any{
				atomic.Key: atomic.Value,
			},
		}, nil

	case "!=", "ne", "NE", "not equals", "NOT EQUALS":
		// Not equal
		return map[string]any{
			"bool": map[string]any{
				"must_not": []map[string]any{
					{
						"term": map[string]any{
							atomic.Key: atomic.Value,
						},
					},
				},
			},
		}, nil

	case "exists", "EXISTS":
		// Field exists
		return map[string]any{
			"exists": map[string]any{
				"field": atomic.Key,
			},
		}, nil

	case "not exists", "NOT EXISTS":
		// Field does not exist
		return map[string]any{
			"bool": map[string]any{
				"must_not": []map[string]any{
					{
						"exists": map[string]any{
							"field": atomic.Key,
						},
					},
				},
			},
		}, nil

	case ">", "gt", "GT":
		// Greater than
		return map[string]any{
			"range": map[string]any{
				atomic.Key: map[string]any{
					"gt": atomic.Value,
				},
			},
		}, nil

	case ">=", "gte", "GTE":
		// Greater than or equal
		return map[string]any{
			"range": map[string]any{
				atomic.Key: map[string]any{
					"gte": atomic.Value,
				},
			},
		}, nil

	case "<", "lt", "LT":
		// Less than
		return map[string]any{
			"range": map[string]any{
				atomic.Key: map[string]any{
					"lt": atomic.Value,
				},
			},
		}, nil

	case "<=", "lte", "LTE":
		// Less than or equal
		return map[string]any{
			"range": map[string]any{
				atomic.Key: map[string]any{
					"lte": atomic.Value,
				},
			},
		}, nil

	case "like", "LIKE", "contains", "CONTAINS":
		// Wildcard search
		return map[string]any{
			"wildcard": map[string]any{
				atomic.Key: map[string]any{
					"value": fmt.Sprintf("*%v*", atomic.Value),
				},
			},
		}, nil

	case "prefix", "PREFIX", "starts with", "STARTS WITH":
		// Prefix search
		return map[string]any{
			"prefix": map[string]any{
				atomic.Key: atomic.Value,
			},
		}, nil

	default:
		// Default to match query for unknown operators
		log.Logger.WithField("operator", atomic.Operator).Warn("Unknown operator, defaulting to match query")
		return map[string]any{
			"match": map[string]any{
				atomic.Key: atomic.Value,
			},
		}, nil
	}
}

func convertWhereConditionToES(where *model.WhereCondition) (map[string]any, error) {
	if where == nil {
		return map[string]any{}, nil
	}

	switch where.Type {
	case model.WhereConditionTypeAtomic:
		if where.Atomic == nil {
			err := fmt.Errorf("atomic condition must have an atomicwherecondition")
			log.Logger.WithError(err).Error("Invalid atomic where condition: missing atomic condition")
			return nil, err
		}

		// Convert atomic condition based on operator
		clause, err := convertAtomicConditionToES(where.Atomic)
		if err != nil {
			return nil, err
		}

		return map[string]any{
			"must": []map[string]any{clause},
		}, nil

	case model.WhereConditionTypeAnd:
		if where.And == nil || len(where.And.Children) == 0 {
			return map[string]any{}, nil
		}
		mustClauses := []map[string]any{}
		for _, child := range where.And.Children {
			// Handle child conditions based on their type
			if child.Type == model.WhereConditionTypeAtomic && child.Atomic != nil {
				// For atomic children, convert based on operator
				clause, err := convertAtomicConditionToES(child.Atomic)
				if err != nil {
					log.Logger.WithError(err).Error("Failed to convert atomic condition in AND clause to ElasticSearch query")
					return nil, err
				}
				mustClauses = append(mustClauses, clause)
			} else {
				// For nested AND/OR, we need to wrap them in a bool query
				childCondition, err := convertWhereConditionToES(child)
				if err != nil {
					log.Logger.WithError(err).Error("Failed to convert child condition in AND clause to ElasticSearch query")
					return nil, err
				}
				// Wrap the child condition in a bool query
				mustClauses = append(mustClauses, map[string]any{
					"bool": childCondition,
				})
			}
		}
		return map[string]any{"must": mustClauses}, nil

	case model.WhereConditionTypeOr:
		if where.Or == nil || len(where.Or.Children) == 0 {
			return map[string]any{}, nil
		}
		shouldClauses := []map[string]any{}
		for _, child := range where.Or.Children {
			// Handle child conditions based on their type
			if child.Type == model.WhereConditionTypeAtomic && child.Atomic != nil {
				// For atomic children, convert based on operator
				clause, err := convertAtomicConditionToES(child.Atomic)
				if err != nil {
					log.Logger.WithError(err).Error("Failed to convert atomic condition in OR clause to ElasticSearch query")
					return nil, err
				}
				shouldClauses = append(shouldClauses, clause)
			} else {
				// For nested AND/OR, we need to wrap them in a bool query
				childCondition, err := convertWhereConditionToES(child)
				if err != nil {
					log.Logger.WithError(err).Error("Failed to convert child condition in OR clause to ElasticSearch query")
					return nil, err
				}
				// Wrap the child condition in a bool query
				shouldClauses = append(shouldClauses, map[string]any{
					"bool": childCondition,
				})
			}
		}
		return map[string]any{
			"should":               shouldClauses,
			"minimum_should_match": 1, // Ensures at least one condition matches
		}, nil

	default:
		err := fmt.Errorf("unknown whereconditiontype: %v", where.Type)
		return nil, err
	}
}

func (p *ElasticSearchPlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return nil, errors.New("unsupported operation")
}

func (p *ElasticSearchPlugin) Chat(config *engine.PluginConfig, schema string, model string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
	return nil, errors.ErrUnsupported
}

func (p *ElasticSearchPlugin) FormatValue(val any) string {
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

// GetColumnConstraints - not supported for ElasticSearch
func (p *ElasticSearchPlugin) GetColumnConstraints(config *engine.PluginConfig, schema string, storageUnit string) (map[string]map[string]any, error) {
	return make(map[string]map[string]any), nil
}

// ClearTableData - not supported for ElasticSearch
func (p *ElasticSearchPlugin) ClearTableData(config *engine.PluginConfig, schema string, storageUnit string) (bool, error) {
	return false, errors.ErrUnsupported
}

// WithTransaction executes the operation directly since ElasticSearch doesn't support transactions
func (p *ElasticSearchPlugin) WithTransaction(config *engine.PluginConfig, operation func(tx any) error) error { //nolint:unused
	// ElasticSearch doesn't support transactions
	// For now, just execute the operation directly
	return operation(nil)
}

func (p *ElasticSearchPlugin) GetSupportedOperators() map[string]string { //nolint:unused
	return supportedOperators
}

func NewElasticSearchPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_ElasticSearch,
		PluginFunctions: &ElasticSearchPlugin{},
	}
}
