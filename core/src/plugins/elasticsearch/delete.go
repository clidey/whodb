// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package elasticsearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
)

func (p *ElasticSearchPlugin) DeleteRow(config *engine.PluginConfig, database string, storageUnit string, values map[string]string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		return false, err
	}

	// Extract the document JSON
	documentJSON, ok := values["document"]
	if !ok {
		return false, errors.New("missing 'document' key in values map")
	}

	// Unmarshal the JSON to extract the _id field
	var jsonValues map[string]interface{}
	if err := json.Unmarshal([]byte(documentJSON), &jsonValues); err != nil {
		return false, err
	}

	// Get the _id from the document
	id, ok := jsonValues["_id"]
	if !ok {
		return false, errors.New("missing '_id' field in the document")
	}

	// Delete the document by ID
	res, err := client.Delete(
		storageUnit,
		id.(string),
		client.Delete.WithContext(context.Background()),
		client.Delete.WithRefresh("true"), // Ensure the deletion is immediately visible
	)
	if err != nil {
		return false, fmt.Errorf("failed to execute delete: %w", err)
	}
	defer res.Body.Close()

	// Check if the response indicates an error
	if res.IsError() {
		return false, fmt.Errorf("error deleting document: %s", res.String())
	}

	// Decode the response to check the result
	var deleteResponse map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&deleteResponse); err != nil {
		return false, err
	}

	// Check if the deletion was successful
	if result, ok := deleteResponse["result"].(string); ok && result != "deleted" {
		return false, errors.New("no documents were deleted")
	}

	return true, nil
}
