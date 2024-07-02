package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/elastic/go-elasticsearch/esapi"
)

func (p *ElasticSearchPlugin) UpdateStorageUnit(config *engine.PluginConfig, database string, storageUnit string, values map[string]string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}

	documentJSON, ok := values["document"]
	if !ok {
		return false, errors.New("missing 'document' key in values map")
	}

	var jsonValues map[string]interface{}
	if err := json.Unmarshal([]byte(documentJSON), &jsonValues); err != nil {
		return false, err
	}

	id, ok := jsonValues["_id"]
	if !ok {
		return false, errors.New("missing '_id' field in the document")
	}

	idStr, ok := id.(string)
	if !ok {
		return false, errors.New("invalid '_id' field; not a valid string")
	}

	delete(jsonValues, "_id")

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(jsonValues); err != nil {
		return false, err
	}

	req := esapi.UpdateRequest{
		Index:      storageUnit,
		DocumentID: idStr,
		Body:       &buf,
		Refresh:    "true",
	}

	res, err := req.Do(context.Background(), client)
	if err != nil {
		return false, err
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
