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
		client.Cat.Indices.WithContext(config.OperationContext()),
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

	res, err := client.Indices.Stats(client.Indices.Stats.WithContext(config.OperationContext()))
	if err != nil {
		log.WithError(err).Error("Failed to get ElasticSearch indices stats")
		return nil, err
	}
	defer res.Body.Close()

	// _stats requires the monitor cluster privilege. If it is denied (403) or
	// otherwise errors, fall back to listing indices without size/count so the
	// UI remains usable for read-only callers.
	if res.IsError() {
		log.Warnf("ElasticSearch indices stats API returned error (%d); falling back to index-only listing", res.StatusCode)
		return p.listIndicesWithoutStats(config)
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

		attrs := []engine.Record{{Key: "Type", Value: "Index"}}

		indexStats, ok := indexStatsInterface.(map[string]any)
		if !ok {
			log.Warnf("Skipping stats for index %s: unexpected format", indexName)
			storageUnits = append(storageUnits, engine.StorageUnit{Name: indexName, Attributes: attrs})
			continue
		}

		primaries, _ := indexStats["primaries"].(map[string]any)
		if store, ok := primaries["store"].(map[string]any); ok {
			if bytes, ok := toInt64(store["size_in_bytes"]); ok {
				attrs = append(attrs, engine.Record{Key: "Data Size", Value: fmt.Sprintf("%d", bytes)})
			}
		}
		if docs, ok := primaries["docs"].(map[string]any); ok {
			if count, ok := toInt64(docs["count"]); ok {
				attrs = append(attrs, engine.Record{Key: "Count", Value: fmt.Sprintf("%d", count)})
			}
		}

		storageUnits = append(storageUnits, engine.StorageUnit{Name: indexName, Attributes: attrs})
	}

	return storageUnits, nil
}

// listIndicesWithoutStats returns indices visible to the caller but without
// size/count attributes. Used when _stats is unavailable (e.g., 403 Forbidden).
func (p *ElasticSearchPlugin) listIndicesWithoutStats(config *engine.PluginConfig) ([]engine.StorageUnit, error) {
	names, err := p.GetDatabases(config)
	if err != nil {
		return nil, err
	}
	units := make([]engine.StorageUnit, 0, len(names))
	for _, name := range names {
		units = append(units, engine.StorageUnit{
			Name:       name,
			Attributes: []engine.Record{{Key: "Type", Value: "Index"}},
		})
	}
	return units, nil
}

// toInt64 extracts an int64 from a JSON-decoded value, handling the common
// numeric forms encoutered when encoding/json targets an any.
func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	}
	return 0, false
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

func init() {
	engine.RegisterPlugin(NewElasticSearchPlugin())
}

func NewElasticSearchPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_ElasticSearch,
		PluginFunctions: &ElasticSearchPlugin{},
	}
}
