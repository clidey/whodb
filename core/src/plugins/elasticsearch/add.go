/*
 * // Copyright 2025 Clidey, Inc.
 * //
 * // Licensed under the Apache License, Version 2.0 (the "License");
 * // you may not use this file except in compliance with the License.
 * // You may obtain a copy of the License at
 * //
 * //     http://www.apache.org/licenses/LICENSE-2.0
 * //
 * // Unless required by applicable law or agreed to in writing, software
 * // distributed under the License is distributed on an "AS IS" BASIS,
 * // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * // See the License for the specific language governing permissions and
 * // limitations under the License.
 */

package elasticsearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

func (p *ElasticSearchPlugin) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, fields []engine.Record) (bool, error) {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to connect to ElasticSearch while adding storage unit")
		return false, err
	}

	// Build mapping from provided fields, best-effort. If none provided, keep default.
	var body *bytes.Buffer
	if len(fields) > 0 {
		mapping := map[string]any{
			"mappings": map[string]any{
				"properties": buildElasticMappings(fields),
			},
		}
		body = new(bytes.Buffer)
		if err := json.NewEncoder(body).Encode(mapping); err != nil {
			log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to encode ElasticSearch mapping for index creation")
			return false, err
		}
	}

	req := client.Indices.Create
	var res *esapi.Response
	if body != nil {
		res, err = req(storageUnit, client.Indices.Create.WithBody(body))
	} else {
		res, err = req(storageUnit)
	}
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to create ElasticSearch index")
		return false, err
	}

	defer res.Body.Close()

	if res.IsError() {
		err := fmt.Errorf("failed to create index: %s", formatElasticError(res))
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

	docID := ""
	if id, ok := jsonValue["_id"]; ok {
		docID = id
		delete(jsonValue, "_id")
	}

	documentBytes, err := json.Marshal(jsonValue)
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to marshal ElasticSearch document to JSON")
		return false, fmt.Errorf("error marshaling document to JSON: %v", err)
	}

	documentReader := bytes.NewReader(documentBytes)

	indexOpts := []func(*esapi.IndexRequest){
		client.Index.WithRefresh("true"),
	}
	if strings.TrimSpace(docID) != "" {
		indexOpts = append(indexOpts, client.Index.WithDocumentID(docID))
	}

	res, err := client.Index(
		storageUnit,
		documentReader,
		indexOpts...,
	)
	if err != nil {
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("Failed to index document in ElasticSearch")
		return false, fmt.Errorf("error indexing document: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		err := fmt.Errorf("failed to index document: %s", formatElasticError(res))
		log.Logger.WithError(err).WithField("storageUnit", storageUnit).Error("ElasticSearch document indexing API returned error")
		return false, err
	}

	return true, nil
}

// buildElasticMappings converts field definitions into an Elasticsearch properties map.
func buildElasticMappings(fields []engine.Record) map[string]any {
	props := make(map[string]any, len(fields))
	for _, f := range fields {
		props[f.Key] = map[string]any{
			"type": mapElasticFieldType(f.Value),
		}
	}
	return props
}

// mapElasticFieldType maps an arbitrary type string into a reasonable ES field type.
func mapElasticFieldType(typeStr string) string {
	lower := strings.ToLower(typeStr)
	switch {
	case strings.Contains(lower, "keyword"):
		return "keyword"
	case strings.Contains(lower, "text"):
		return "text"
	case strings.Contains(lower, "bool"):
		return "boolean"
	case strings.Contains(lower, "date"), strings.Contains(lower, "time"):
		return "date"
	case strings.Contains(lower, "int"), strings.Contains(lower, "long"):
		return "long"
	case strings.Contains(lower, "float"), strings.Contains(lower, "double"), strings.Contains(lower, "decimal"):
		return "double"
	case strings.Contains(lower, "geo"):
		return "geo_point"
	case strings.Contains(lower, "object"), strings.Contains(lower, "json"):
		return "object"
	default:
		return "text"
	}
}
