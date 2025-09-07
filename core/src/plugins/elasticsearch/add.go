/*
 * Copyright 2025 Clidey, Inc.
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
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

func (p *ElasticSearchPlugin) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, fields []engine.Record) (bool, error) {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to connect to ElasticSearch while adding storage unit")
		return false, err
	}

	res, err := client.Indices.Create(storageUnit)
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to create ElasticSearch index")
		return false, err
	}

	defer res.Body.Close()

	if res.IsError() {
		err := fmt.Errorf("failed to create index: %s", res.String())
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("ElasticSearch index creation API returned error")
		return false, err
	}

	return true, nil
}

func (p *ElasticSearchPlugin) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (bool, error) {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to connect to ElasticSearch while adding row")
		return false, err
	}

	jsonValue := map[string]string{}
	for _, value := range values {
		jsonValue[value.Key] = value.Value
	}

	documentBytes, err := json.Marshal(jsonValue)
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to marshal ElasticSearch document to JSON")
		return false, fmt.Errorf("error marshaling document to JSON: %v", err)
	}

	documentReader := bytes.NewReader(documentBytes)

	res, err := client.Index(
		storageUnit,
		documentReader,
		client.Index.WithRefresh("true"), // Ensure the document is immediately visible
	)
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to index document in ElasticSearch")
		return false, fmt.Errorf("error indexing document: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		err := fmt.Errorf("failed to index document: %s", res.String())
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("ElasticSearch document indexing API returned error")
		return false, err
	}

	return true, nil
}
