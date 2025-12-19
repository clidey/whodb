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
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

type tableRelation struct {
	Table1       string
	Table2       string
	Relation     string
	SourceColumn string
	TargetColumn string
}

func (p *ElasticSearchPlugin) GetGraph(config *engine.PluginConfig, database string) ([]engine.GraphUnit, error) {
	client, err := DB(config)
	if err != nil {
		log.Logger.WithError(err).Error("Failed to connect to ElasticSearch while getting graph")
		return nil, err
	}

	res, err := client.Indices.Stats()
	if err != nil {
		log.Logger.WithError(err).Error("Failed to get ElasticSearch indices stats for graph generation")
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		err := fmt.Errorf("error getting indices: %s", res.String())
		log.Logger.WithError(err).Error("ElasticSearch indices stats API returned error for graph generation")
		return nil, err
	}

	var stats map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&stats); err != nil {
		log.Logger.WithError(err).Error("Failed to decode ElasticSearch indices stats for graph generation")
		return nil, err
	}

	indicesStats := stats["indices"].(map[string]interface{})

	indices := []string{}
	for indexName := range indicesStats {
		indices = append(indices, indexName)
	}

	relations := []tableRelation{}
	uniqueRelations := make(map[string]bool)

	for indexName := range indicesStats {
		var buf bytes.Buffer
		query := map[string]interface{}{
			"size": 100,
			"query": map[string]interface{}{
				"match_all": map[string]interface{}{},
			},
		}
		if err := json.NewEncoder(&buf).Encode(query); err != nil {
			log.Logger.WithError(err).WithField("indexName", indexName).Error("Failed to encode ElasticSearch query for graph generation")
			return nil, err
		}

		res, err := client.Search(
			client.Search.WithContext(context.Background()),
			client.Search.WithIndex(indexName),
			client.Search.WithBody(&buf),
		)
		if err != nil {
			log.Logger.WithError(err).WithField("indexName", indexName).Error("Failed to execute ElasticSearch search query for graph generation")
			return nil, err
		}
		defer res.Body.Close()

		if res.IsError() {
			err := fmt.Errorf("error searching documents: %s", formatElasticError(res))
			log.Logger.WithError(err).WithField("indexName", indexName).Error("ElasticSearch search API returned error for graph generation")
			return nil, err
		}

		var searchResult map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
			log.Logger.WithError(err).WithField("indexName", indexName).Error("Failed to decode ElasticSearch search result for graph generation")
			return nil, err
		}

		hits := searchResult["hits"].(map[string]interface{})["hits"].([]interface{})
		if len(hits) > 0 {
			foreignKeys := make(map[string]string)

			for _, h := range hits {
				doc, ok := h.(map[string]any)["_source"].(map[string]any)
				if !ok {
					continue
				}

				for fieldName := range doc {
					if fieldName == "_id" {
						continue
					}

					// Check for explicit relation hints
					if strings.Contains(strings.ToLower(fieldName), ".id") {
						for _, otherIndex := range indices {
							if otherIndex == indexName {
								continue
							}
							if strings.Contains(strings.ToLower(fieldName), strings.ToLower(otherIndex)) {
								foreignKeys[otherIndex] = fieldName
								break
							}
						}
					}

					for _, otherIndex := range indices {
						if otherIndex == indexName {
							continue
						}

						singularName := strings.TrimSuffix(otherIndex, "s")
						pluralName := otherIndex
						if !strings.HasSuffix(otherIndex, "s") {
							pluralName = otherIndex + "s"
						}

						lowerField := strings.ToLower(fieldName)
						if lowerField == strings.ToLower(singularName)+"_id" ||
							lowerField == strings.ToLower(singularName)+"id" ||
							lowerField == strings.ToLower(otherIndex)+"_id" ||
							lowerField == strings.ToLower(otherIndex)+"id" ||
							lowerField == strings.ToLower(pluralName)+"_id" ||
							lowerField == strings.ToLower(pluralName)+"id" {
							foreignKeys[otherIndex] = fieldName
							break
						}
					}
				}
			}

			for fk, fieldName := range foreignKeys {
				relKey := indexName + ":" + fk
				if !uniqueRelations[relKey+":ManyToOne"] {
					uniqueRelations[relKey+":ManyToOne"] = true
					relations = append(relations, tableRelation{
						Table1:       indexName,
						Table2:       fk,
						Relation:     "ManyToOne",
						SourceColumn: fieldName,
						TargetColumn: "_id",
					})
				}
			}
		}
	}

	tableMap := make(map[string][]engine.GraphUnitRelationship)
	for _, tr := range relations {
		sourceCol := tr.SourceColumn
		targetCol := tr.TargetColumn
		tableMap[tr.Table1] = append(tableMap[tr.Table1], engine.GraphUnitRelationship{
			Name:             tr.Table2,
			RelationshipType: engine.GraphUnitRelationshipType(tr.Relation),
			SourceColumn:     &sourceCol,
			TargetColumn:     &targetCol,
		})
	}

	storageUnits, err := p.GetStorageUnits(config, database)
	if err != nil {
		log.Logger.WithError(err).Error("Failed to get storage units for ElasticSearch graph generation")
		return nil, err
	}

	storageUnitsMap := map[string]engine.StorageUnit{}
	for _, storageUnit := range storageUnits {
		storageUnitsMap[storageUnit.Name] = storageUnit
	}

	tables := []engine.GraphUnit{}
	for _, storageUnit := range storageUnits {
		foundTable, ok := tableMap[storageUnit.Name]
		var relations []engine.GraphUnitRelationship
		if ok {
			relations = foundTable
		}
		tables = append(tables, engine.GraphUnit{Unit: storageUnit, Relations: relations})
	}

	return tables, nil
}
