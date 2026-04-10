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
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

type ElasticSearchPlugin struct {
	engine.BasePlugin
}

func (p *ElasticSearchPlugin) IsAvailable(ctx context.Context, config *engine.PluginConfig) bool {
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

// GetDatabases lists all Elasticsearch indices (equivalent to databases).
// System indices (those starting with .) are filtered out.
func (p *ElasticSearchPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	client, err := DB(config)
	if err != nil {
		log.WithError(err).Error("Failed to connect to ElasticSearch while listing indices")
		return nil, err
	}

	// Use Cat Indices API for lightweight index listing
	res, err := client.Cat.Indices(
		client.Cat.Indices.WithFormat("json"),
	)
	if err != nil {
		log.WithError(err).Error("Failed to get ElasticSearch indices")
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		err := fmt.Errorf("error getting indices: %s", res.String())
		log.WithError(err).Error("ElasticSearch Cat Indices API returned error")
		return nil, err
	}

	var indices []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&indices); err != nil {
		log.WithError(err).Error("Failed to decode ElasticSearch indices response")
		return nil, err
	}

	databases := make([]string, 0, len(indices))
	for _, idx := range indices {
		indexName, ok := idx["index"].(string)
		if !ok {
			continue
		}

		// Skip hidden/system indices (those starting with .)
		if strings.HasPrefix(indexName, ".") {
			continue
		}

		databases = append(databases, indexName)
	}

	return databases, nil
}

func (p *ElasticSearchPlugin) GetStorageUnits(config *engine.PluginConfig, database string) ([]engine.StorageUnit, error) {
	client, err := DB(config)
	if err != nil {
		log.WithError(err).Error("Failed to connect to ElasticSearch while getting storage units")
		return nil, err
	}

	res, err := client.Indices.Stats()
	if err != nil {
		log.WithError(err).Error("Failed to get ElasticSearch indices stats")
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		err := fmt.Errorf("error getting stats for indices: %s", res.String())
		log.WithError(err).Error("ElasticSearch indices stats API returned error")
		return nil, err
	}

	var stats map[string]any
	if err := json.NewDecoder(res.Body).Decode(&stats); err != nil {
		log.WithError(err).Error("Failed to decode ElasticSearch indices stats response")
		return nil, err
	}

	indicesStats, ok := stats["indices"].(map[string]any)
	if !ok {
		log.WithField("stats", stats).Error("Unexpected indices stats format from ElasticSearch")
		return nil, fmt.Errorf("unexpected indices stats format")
	}

	storageUnits := make([]engine.StorageUnit, 0, len(indicesStats))

	for indexName, indexStatsInterface := range indicesStats {
		// Skip hidden/system indices (those starting with .)
		if strings.HasPrefix(indexName, ".") {
			continue
		}

		indexStats, ok := indexStatsInterface.(map[string]any)
		if !ok {
			log.Warnf("Skipping index %s: unexpected stats format", indexName)
			continue
		}

		primaries, ok := indexStats["primaries"].(map[string]any)
		if !ok {
			log.Warnf("Skipping index %s: missing primaries data", indexName)
			continue
		}

		docs, ok := primaries["docs"].(map[string]any)
		if !ok {
			log.Warnf("Skipping index %s: missing docs data", indexName)
			continue
		}

		store, ok := primaries["store"].(map[string]any)
		if !ok {
			log.Warnf("Skipping index %s: missing store data", indexName)
			continue
		}

		storageUnit := engine.StorageUnit{
			Name: indexName,
			Attributes: []engine.Record{
				{Key: "Type", Value: "Index"},
				{Key: "Storage Size", Value: fmt.Sprintf("%v", store["size_in_bytes"])},
				{Key: "Count", Value: fmt.Sprintf("%v", docs["count"])},
			},
		}
		storageUnits = append(storageUnits, storageUnit)
	}

	return storageUnits, nil
}

func (p *ElasticSearchPlugin) StorageUnitExists(config *engine.PluginConfig, database string, index string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}

	res, err := client.Indices.Exists([]string{index})
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	return res.StatusCode == 200, nil
}

func (p *ElasticSearchPlugin) FormatValue(val any) string {
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

// GetDatabaseMetadata returns ElasticSearch metadata for frontend configuration.
// ElasticSearch is a search engine without traditional type definitions.
func (p *ElasticSearchPlugin) GetDatabaseMetadata() *engine.DatabaseMetadata {
	operators := make([]string, 0, len(supportedOperators))
	for op := range supportedOperators {
		operators = append(operators, op)
	}
	return &engine.DatabaseMetadata{
		DatabaseType: engine.DatabaseType_ElasticSearch,
		TypeDefinitions: []engine.TypeDefinition{
			{ID: "text", Label: "text", Category: engine.TypeCategoryText},
			{ID: "keyword", Label: "keyword", Category: engine.TypeCategoryText},
			{ID: "boolean", Label: "boolean", Category: engine.TypeCategoryBoolean},
			{ID: "long", Label: "long", Category: engine.TypeCategoryNumeric},
			{ID: "double", Label: "double", Category: engine.TypeCategoryNumeric},
			{ID: "date", Label: "date", Category: engine.TypeCategoryDatetime},
			{ID: "object", Label: "object", Category: engine.TypeCategoryOther},
			{ID: "array", Label: "array", Category: engine.TypeCategoryOther},
			{ID: "geo_point", Label: "geo_point", Category: engine.TypeCategoryOther},
			{ID: "nested", Label: "nested", Category: engine.TypeCategoryOther},
			{ID: "mixed", Label: "mixed", Category: engine.TypeCategoryOther},
		},
		Operators: operators,
		AliasMap:  map[string]string{},
	}
}

func init() {
	engine.RegisterPlugin(NewElasticSearchPlugin())
}

func NewElasticSearchPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_ElasticSearch,
		PluginFunctions: &ElasticSearchPlugin{},
	}
}
