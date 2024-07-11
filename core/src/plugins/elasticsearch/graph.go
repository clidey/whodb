package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
)

type tableRelation struct {
	Table1   string
	Table2   string
	Relation string
}

func (p *ElasticSearchPlugin) GetGraph(config *engine.PluginConfig, database string) ([]engine.GraphUnit, error) {
	client, err := DB(config)
	if err != nil {
		return nil, err
	}

	res, err := client.Indices.Stats()
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error getting indices: %s", res.String())
	}

	var stats map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&stats); err != nil {
		return nil, err
	}

	indicesStats := stats["indices"].(map[string]interface{})

	relations := []tableRelation{}
	for indexName := range indicesStats {
		var buf bytes.Buffer
		query := map[string]interface{}{
			"size": 1,
			"query": map[string]interface{}{
				"match_all": map[string]interface{}{},
			},
		}
		if err := json.NewEncoder(&buf).Encode(query); err != nil {
			return nil, err
		}

		res, err := client.Search(
			client.Search.WithContext(context.Background()),
			client.Search.WithIndex(indexName),
			client.Search.WithBody(&buf),
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
		if len(hits) > 0 {
			doc := hits[0].(map[string]interface{})["_source"].(map[string]interface{})

			for key := range doc {
				for otherIndexName := range indicesStats {
					singularName := strings.TrimSuffix(otherIndexName, "s")
					if key == singularName+"_id" || key == otherIndexName+"_id" {
						relations = append(relations, tableRelation{
							Table1:   indexName,
							Table2:   otherIndexName,
							Relation: "ManyToMany",
						})
					}
				}
			}
		}
	}

	tableMap := make(map[string][]engine.GraphUnitRelationship)
	for _, tr := range relations {
		tableMap[tr.Table1] = append(tableMap[tr.Table1], engine.GraphUnitRelationship{Name: tr.Table2, RelationshipType: engine.GraphUnitRelationshipType(tr.Relation)})
	}

	storageUnits, err := p.GetStorageUnits(config, database)
	if err != nil {
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
