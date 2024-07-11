package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
)

type JsonSourceMap struct {
	Id     string          `json:"_id"`
	Source json.RawMessage `json:"source"`
}

func (p *ElasticSearchPlugin) UpdateStorageUnit(config *engine.PluginConfig, database string, storageUnit string, values map[string]string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}

	documentJSON, ok := values["document"]
	if !ok {
		return false, errors.New("missing 'document' key in values map")
	}

	jsonSourceMap := &JsonSourceMap{}
	if err := json.Unmarshal([]byte(documentJSON), jsonSourceMap); err != nil {
		return false, errors.New("source is not correctly formatted")
	}

	var jsonValues map[string]interface{}
	if err := json.Unmarshal(jsonSourceMap.Source, &jsonValues); err != nil {
		return false, err
	}

	script := `
		for (entry in params.entrySet()) {
			ctx._source[entry.getKey()] = entry.getValue();
		}
		for (key in ctx._source.keySet().toArray()) {
			if (!params.containsKey(key)) {
				ctx._source.remove(key);
			}
		}
	`
	params := jsonValues

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(map[string]interface{}{
		"script": map[string]interface{}{
			"source": script,
			"lang":   "painless",
			"params": params,
		},
		"upsert": jsonValues,
	}); err != nil {
		return false, err
	}

	res, err := client.Update(
		storageUnit,
		jsonSourceMap.Id,
		&buf,
		client.Update.WithContext(context.Background()),
		client.Update.WithRefresh("true"),
	)
	if err != nil {
		return false, fmt.Errorf("failed to execute update: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return false, fmt.Errorf("error updating document: %s", res.String())
	}

	var updateResponse map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&updateResponse); err != nil {
		return false, err
	}

	if result, ok := updateResponse["result"].(string); ok && result == "noop" {
		return false, errors.New("no documents were updated")
	}

	return true, nil
}
