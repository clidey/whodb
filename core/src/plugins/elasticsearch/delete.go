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
	"errors"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

func (p *ElasticSearchPlugin) DeleteRow(config *engine.PluginConfig, database string, storageUnit string, values map[string]string) (bool, error) {
	client, err := DB(config)
	if err != nil {
		log.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to connect to ElasticSearch while deleting row")
		return false, err
	}

	// Extract the document JSON
	documentJSON, ok := values["document"]
	if !ok {
		err := errors.New("missing 'document' key in values map")
		log.WithError(err).WithField("storageUnit", storageUnit).Error("Missing document key in delete request values")
		return false, err
	}

	// Unmarshal the JSON to extract the _id field
	var jsonValues map[string]any
	if err := json.Unmarshal([]byte(documentJSON), &jsonValues); err != nil {
		log.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to unmarshal document JSON for deletion")
		return false, err
	}

	id, ok := jsonValues["_id"]
	if !ok {
		err := errors.New("missing '_id' field in the document")
		log.WithError(err).WithField("storageUnit", storageUnit).Error("Missing _id field in document for deletion")
		return false, err
	}

	idStr, ok := id.(string)
	if !ok || strings.TrimSpace(idStr) == "" {
		return false, fmt.Errorf("invalid '_id' field; expected non-empty string")
	}

	// Delete the document by ID
	res, err := client.Delete(
		storageUnit,
		idStr,
		client.Delete.WithContext(context.Background()),
		client.Delete.WithRefresh("true"), // Ensure the deletion is immediately visible
	)
	if err != nil {
		log.WithError(err).WithField("storageUnit", storageUnit).WithField("documentId", idStr).Error("Failed to execute ElasticSearch delete operation")
		return false, fmt.Errorf("failed to execute delete: %w", err)
	}
	defer res.Body.Close()

	// Check if the response indicates an error
	if res.IsError() {
		err := fmt.Errorf("error deleting document: %s", formatElasticError(res))
		log.WithError(err).WithField("storageUnit", storageUnit).WithField("documentId", idStr).Error("ElasticSearch delete API returned error")
		return false, err
	}

	// Decode the response to check the result
	var deleteResponse map[string]any
	if err := json.NewDecoder(res.Body).Decode(&deleteResponse); err != nil {
		log.WithError(err).WithField("storageUnit", storageUnit).WithField("documentId", id).Error("Failed to decode ElasticSearch delete response")
		return false, err
	}

	// Check if the deletion was successful
	if result, ok := deleteResponse["result"].(string); ok && result != "deleted" {
		err := errors.New("no documents were deleted")
		log.WithError(err).WithField("storageUnit", storageUnit).WithField("documentId", id).WithField("result", result).Error("ElasticSearch delete operation did not delete any documents")
		return false, err
	}

	return true, nil
}
