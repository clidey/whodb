package elasticsearch

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
)

func (p *ElasticSearchPlugin) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, fields map[string]string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}

	res, err := client.Indices.Create(storageUnit)
	if err != nil {
		return false, err
	}

	defer res.Body.Close()

	if res.IsError() {
		return false, fmt.Errorf("failed to create index: %s", res.String())
	}

	return true, nil
}

func (p *ElasticSearchPlugin) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}

	jsonValue := map[string]string{}
	for _, value := range values {
		jsonValue[value.Key] = value.Value
	}

	documentBytes, err := json.Marshal(jsonValue)
	if err != nil {
		return false, fmt.Errorf("error marshaling document to JSON: %v", err)
	}

	documentReader := bytes.NewReader(documentBytes)

	res, err := client.Index(storageUnit, documentReader)
	if err != nil {
		return false, fmt.Errorf("error indexing document: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return false, fmt.Errorf("failed to index document: %s", res.String())
	}

	return true, nil
}
