// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

var script = `
for (entry in params.entrySet()) {
	ctx._source[entry.getKey()] = entry.getValue();
}
for (key in ctx._source.keySet().toArray()) {
	if (!params.containsKey(key)) {
		ctx._source.remove(key);
	}
}
`

func (p *ElasticSearchPlugin) UpdateStorageUnit(config *engine.PluginConfig, database string, storageUnit string, values map[string]string, updatedColumns []string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to connect to ElasticSearch while updating storage unit")
		return false, err
	}

	documentJSON, ok := values["document"]
	if !ok {
		err := errors.New("missing 'document' key in values map")
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Missing document key in update request values")
		return false, err
	}

	var jsonValues map[string]interface{}
	if err := json.Unmarshal([]byte(documentJSON), &jsonValues); err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to unmarshal document JSON for update")
		return false, err
	}

	id, ok := jsonValues["_id"]
	if !ok {
		err := errors.New("missing '_id' field in the document")
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Missing _id field in document for update")
		return false, err
	}

	delete(jsonValues, "_id")

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(map[string]interface{}{
		"script": map[string]interface{}{
			"source": script,
			"lang":   "painless",
			"params": jsonValues,
		},
		"upsert": jsonValues,
	}); err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).WithField("documentId", id).Error("Failed to encode ElasticSearch update request")
		return false, err
	}

	res, err := client.Update(
		storageUnit,
		id.(string),
		&buf,
		client.Update.WithContext(context.Background()),
		client.Update.WithRefresh("true"),
	)
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).WithField("documentId", id).Error("Failed to execute ElasticSearch update operation")
		return false, fmt.Errorf("failed to execute update: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		err := fmt.Errorf("error updating document: %s", res.String())
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).WithField("documentId", id).Error("ElasticSearch update API returned error")
		return false, err
	}

	var updateResponse map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&updateResponse); err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).WithField("documentId", id).Error("Failed to decode ElasticSearch update response")
		return false, err
	}

	if result, ok := updateResponse["result"].(string); ok && result == "noop" {
		err := errors.New("no documents were updated")
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).WithField("documentId", id).WithField("result", result).Error("ElasticSearch update operation did not update any documents")
		return false, err
	}

	return true, nil
}
