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
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/log"
)

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

	case "match_phrase_prefix", "MATCH_PHRASE_PREFIX":
		return map[string]any{
			"match_phrase_prefix": map[string]any{
				atomic.Key: atomic.Value,
			},
		}, nil

	case "=", "eq", "EQ", "equals", "EQUALS", "term", "TERM":
		// Special handling for _id field - use ids query
		if atomic.Key == "_id" {
			return map[string]any{
				"ids": map[string]any{
					"values": []any{atomic.Value},
				},
			}, nil
		}
		// Exact match for other fields
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

	case "terms", "TERMS":
		// Multiple exact matches
		values := parseCSVToSlice(atomic.Value)
		return map[string]any{
			"terms": map[string]any{
				atomic.Key: values,
			},
		}, nil

	case "ids", "IDS":
		var ids []any
		ids = parseCSVToSlice(atomic.Value)
		return map[string]any{
			"ids": map[string]any{
				"values": ids,
			},
		}, nil

	case "range", "RANGE":
		// Expect "min,max" (empty allowed)
		minBound, maxBound := parseRangeBounds(fmt.Sprintf("%v", atomic.Value))
		rangeClause := map[string]any{}
		if minBound != "" {
			rangeClause["gte"] = minBound
		}
		if maxBound != "" {
			rangeClause["lte"] = maxBound
		}
		return map[string]any{
			"range": map[string]any{
				atomic.Key: rangeClause,
			},
		}, nil

	case "match_phrase", "MATCH_PHRASE":
		// Phrase match
		return map[string]any{
			"match_phrase": map[string]any{
				atomic.Key: atomic.Value,
			},
		}, nil

	case "wildcard", "WILDCARD":
		// Wildcard pattern match
		return map[string]any{
			"wildcard": map[string]any{
				atomic.Key: map[string]any{
					"value": atomic.Value,
				},
			},
		}, nil

	case "regexp", "REGEXP":
		// Regular expression match
		return map[string]any{
			"regexp": map[string]any{
				atomic.Key: atomic.Value,
			},
		}, nil

	case "fuzzy", "FUZZY":
		// Fuzzy match
		return map[string]any{
			"fuzzy": map[string]any{
				atomic.Key: atomic.Value,
			},
		}, nil

	default:
		// Default to match query for unknown operators
		log.WithField("operator", atomic.Operator).Warn("Unknown operator, defaulting to match query")
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
			log.WithError(err).Error("Invalid atomic where condition: missing atomic condition")
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
					log.WithError(err).Error("Failed to convert atomic condition in AND clause to ElasticSearch query")
					return nil, err
				}
				mustClauses = append(mustClauses, clause)
			} else {
				// For nested AND/OR, we need to wrap them in a bool query
				childCondition, err := convertWhereConditionToES(child)
				if err != nil {
					log.WithError(err).Error("Failed to convert child condition in AND clause to ElasticSearch query")
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
					log.WithError(err).Error("Failed to convert atomic condition in OR clause to ElasticSearch query")
					return nil, err
				}
				shouldClauses = append(shouldClauses, clause)
			} else {
				// For nested AND/OR, we need to wrap them in a bool query
				childCondition, err := convertWhereConditionToES(child)
				if err != nil {
					log.WithError(err).Error("Failed to convert child condition in OR clause to ElasticSearch query")
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

func parseCSVToSlice(raw string) []any {
	if strings.TrimSpace(raw) == "" {
		return []any{}
	}
	parts := strings.Split(raw, ",")
	values := make([]any, 0, len(parts))
	for _, p := range parts {
		values = append(values, strings.TrimSpace(p))
	}
	return values
}

func parseRangeBounds(raw string) (string, string) {
	parts := strings.Split(raw, ",")
	if len(parts) == 1 {
		return strings.TrimSpace(parts[0]), ""
	}
	minBound := strings.TrimSpace(parts[0])
	maxBound := strings.TrimSpace(parts[1])
	return minBound, maxBound
}
