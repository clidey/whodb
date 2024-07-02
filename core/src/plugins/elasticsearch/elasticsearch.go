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
	return nil, errors.New("unsupported operation")
}

func (p *ElasticSearchPlugin) GetSchema(config *engine.PluginConfig) ([]string, error) {
	client, err := DB(config)
	if err != nil {
		return nil, err
	}

	res, err := client.Indices.Get([]string{})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error getting indices: %s", res.String())
	}

	var indices map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&indices); err != nil {
		return nil, err
	}

	databases := make([]string, 0, len(indices))
	for index := range indices {
		databases = append(databases, index)
	}

	return databases, nil
}

func (p *ElasticSearchPlugin) GetStorageUnits(config *engine.PluginConfig, database string) ([]engine.StorageUnit, error) {
	client, err := DB(config)
	if err != nil {
		return nil, err
	}

	res, err := client.Indices.Stats(client.Indices.Stats.WithIndex(database))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error getting stats for index %s: %s", database, res.String())
	}

	var stats map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&stats); err != nil {
		return nil, err
	}

	indicesStats := stats["indices"].(map[string]interface{})
	indexStats := indicesStats[database].(map[string]interface{})
	primaries := indexStats["primaries"].(map[string]interface{})
	docs := primaries["docs"].(map[string]interface{})
	store := primaries["store"].(map[string]interface{})

	storageUnits := []engine.StorageUnit{
		{
			Name: database,
			Attributes: []engine.Record{
				{Key: "Storage Size", Value: fmt.Sprintf("%v", store["size_in_bytes"])},
				{Key: "Count", Value: fmt.Sprintf("%v", docs["count"])},
			},
		},
	}

	return storageUnits, nil
}

func (p *ElasticSearchPlugin) GetRows(config *engine.PluginConfig, database, collection, filter string, pageSize, pageOffset int) (*engine.GetRowsResult, error) {
	client, err := DB(config)
	if err != nil {
		return nil, err
	}

	var esFilter map[string]interface{}
	if len(filter) > 0 {
		if err := json.Unmarshal([]byte(filter), &esFilter); err != nil {
			return nil, fmt.Errorf("invalid filter format: %v", err)
		}
	}

	query := map[string]interface{}{
		"from":  pageOffset,
		"size":  pageSize,
		"query": esFilter,
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
		doc := hit.(map[string]interface{})["_source"]
		jsonBytes, err := json.Marshal(doc)
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
