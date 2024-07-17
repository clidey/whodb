package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
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

func (p *ElasticSearchPlugin) GetDatabases() ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (p *ElasticSearchPlugin) GetSchema(config *engine.PluginConfig) ([]string, error) {
	return nil, errors.ErrUnsupported
}

func (p *ElasticSearchPlugin) GetStorageUnits(config *engine.PluginConfig, database string) ([]engine.StorageUnit, error) {
	client, err := DB(config)
	if err != nil {
		return nil, err
	}

	res, err := client.Indices.Stats()
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error getting stats for indices: %s", res.String())
	}

	var stats map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&stats); err != nil {
		return nil, err
	}

	indicesStats := stats["indices"].(map[string]interface{})
	storageUnits := make([]engine.StorageUnit, 0, len(indicesStats))

	for indexName, indexStatsInterface := range indicesStats {
		indexStats := indexStatsInterface.(map[string]interface{})
		primaries := indexStats["primaries"].(map[string]interface{})
		docs := primaries["docs"].(map[string]interface{})
		store := primaries["store"].(map[string]interface{})

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

func (p *ElasticSearchPlugin) GetRows(config *engine.PluginConfig, database, collection, filter string, pageSize, pageOffset int) (*engine.GetRowsResult, error) {
	client, err := DB(config)
	if err != nil {
		return nil, err
	}

	var elasticSearchConditions map[string]interface{}
	if len(filter) > 0 {
		if err := json.Unmarshal([]byte(filter), &elasticSearchConditions); err != nil {
			return nil, fmt.Errorf("invalid filter format: %v", err)
		}
	}

	query := map[string]interface{}{
		"from": pageOffset,
		"size": pageSize,
	}

	for key, value := range elasticSearchConditions {
		query[key] = value
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}

	res, err := client.Search(
		client.Search.WithContext(context.Background()),
		client.Search.WithIndex(collection),
		client.Search.WithBody(&buf),
		client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error searching documents: %s", res.String())
	}

	var searchResult map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		return nil, err
	}

	hits := searchResult["hits"].(map[string]interface{})["hits"].([]interface{})
	result := &engine.GetRowsResult{
		Columns: []engine.Column{
			{Name: "document", Type: "Document"},
		},
		Rows: [][]string{},
	}

	for _, hit := range hits {
		hitMap := hit.(map[string]interface{})
		source := hitMap["_source"].(map[string]interface{})
		id := hitMap["_id"]
		source["_id"] = id
		jsonBytes, err := json.Marshal(source)
		if err != nil {
			return nil, err
		}
		result.Rows = append(result.Rows, []string{string(jsonBytes)})
	}

	return result, nil
}

func (p *ElasticSearchPlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return nil, errors.New("unsupported operation")
}

func NewElasticSearchPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_ElasticSearch,
		PluginFunctions: &ElasticSearchPlugin{},
	}
}
