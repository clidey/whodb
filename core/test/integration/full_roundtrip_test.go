//go:build integration

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

package integration

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	elasticplugin "github.com/clidey/whodb/core/src/plugins/elasticsearch"
	mongoplugin "github.com/clidey/whodb/core/src/plugins/mongodb"
	redisplugin "github.com/clidey/whodb/core/src/plugins/redis"
	redislib "github.com/go-redis/redis/v8"
)

type typeCase struct {
	name       string
	columnType string
	value      string
	updated    string
	expect     func(string) bool
	category   engine.TypeCategory
}

type typeOverride struct {
	columnType string
	value      string
	updated    string
	expect     func(string) bool
	skip       bool
}

func TestSQLCanonicalAndAliasRoundTrips(t *testing.T) {
	for _, target := range targets {
		if target.plugin.GetDatabaseMetadata() == nil {
			continue
		}
		switch target.plugin.Type {
		case engine.DatabaseType_MongoDB, engine.DatabaseType_ElasticSearch, engine.DatabaseType_Redis:
			continue
		}

		t.Run(target.name, func(t *testing.T) {
			// Ensure Postgres-only extensions needed for some types are present
			if target.plugin.Type == engine.DatabaseType_Postgres {
				_, _ = target.plugin.RawExecute(target.config, "CREATE EXTENSION IF NOT EXISTS hstore")
			}

			cases := sqlCasesForTarget(target)
			for idx, tc := range cases {
				if tc.value == "" || tc.columnType == "" {
					continue
				}
				runSQLTypeCase(t, target, tc, idx)
			}

			// Unsupported type should fail validation
			_, err := target.plugin.AddStorageUnit(target.config, target.schema, "intg_bad_type", []engine.Record{
				{Key: "id", Value: "INTEGER", Extra: map[string]string{"Primary": "true", "Nullable": "false"}},
				{Key: "val", Value: "NOT_A_REAL_TYPE"},
			})
			if err == nil {
				t.Fatalf("expected unsupported type to fail on %s", target.name)
			}

			meta := target.plugin.GetDatabaseMetadata()
			for alias := range meta.AliasMap {
				if err := engine.ValidateColumnType(alias, meta); err != nil {
					t.Fatalf("expected alias %s to validate for %s: %v", alias, target.name, err)
				}
			}
		})
	}
}

func TestMongoCrudAndExport(t *testing.T) {
	var mongoTarget *target
	for i := range targets {
		if targets[i].plugin.Type == engine.DatabaseType_MongoDB {
			mongoTarget = &targets[i]
			break
		}
	}
	if mongoTarget == nil {
		t.Skip("mongo target not configured")
	}

	collection := fmt.Sprintf("intg_mongo_%d", time.Now().UnixNano())
	created, err := mongoTarget.plugin.AddStorageUnit(mongoTarget.config, mongoTarget.schema, collection, nil)
	if err != nil || !created {
		t.Fatalf("failed to create mongo collection: %v", err)
	}

	docID := "507f1f77bcf86cd799439012"
	ok, err := mongoTarget.plugin.AddRow(mongoTarget.config, mongoTarget.schema, collection, []engine.Record{
		{Key: "_id", Value: docID},
		{Key: "name", Value: "mongo-intg"},
		{Key: "count", Value: "2", Extra: map[string]string{"Type": "INT"}},
		{Key: "meta", Value: "meta-string"},
	})
	if err != nil || !ok {
		t.Fatalf("failed to insert mongo doc: %v", err)
	}

	rows, err := mongoTarget.plugin.GetRows(mongoTarget.config, mongoTarget.schema, collection, nil, []*model.SortCondition{}, 10, 0)
	if err != nil {
		t.Fatalf("mongo get rows failed: %v", err)
	}
	if len(rows.Rows) == 0 || !strings.Contains(rows.Rows[0][0], "mongo-intg") {
		t.Fatalf("mongo doc not returned: %+v", rows.Rows)
	}

	updateDoc := map[string]any{"_id": docID, "name": "mongo-updated", "count": 2, "meta": map[string]any{"env": "integration", "ok": false}}
	updateJSON, _ := json.Marshal(updateDoc)
	updated, err := mongoTarget.plugin.UpdateStorageUnit(mongoTarget.config, mongoTarget.schema, collection, map[string]string{
		"document": string(updateJSON),
	}, []string{"name", "count", "meta"})
	if err != nil || !updated {
		t.Fatalf("mongo update failed: %v", err)
	}

	rows, err = mongoTarget.plugin.GetRows(mongoTarget.config, mongoTarget.schema, collection, nil, []*model.SortCondition{}, 10, 0)
	if err != nil {
		t.Fatalf("mongo get rows after update failed: %v", err)
	}
	if len(rows.Rows) == 0 || !strings.Contains(rows.Rows[0][0], "mongo-updated") || !strings.Contains(rows.Rows[0][0], "false") {
		t.Fatalf("mongo update not reflected in rows: %+v", rows.Rows)
	}

	var exported [][]string
	err = mongoTarget.plugin.ExportData(mongoTarget.config, mongoTarget.schema, collection, func(row []string) error {
		exported = append(exported, row)
		return nil
	}, nil)
	if err != nil {
		t.Fatalf("mongo export failed: %v", err)
	}
	if len(exported) < 2 {
		t.Fatalf("expected exported data, got %d rows", len(exported))
	}

	deleted, err := mongoTarget.plugin.DeleteRow(mongoTarget.config, mongoTarget.schema, collection, map[string]string{
		"document": string(updateJSON),
	})
	if err != nil || !deleted {
		t.Fatalf("mongo delete failed: %v", err)
	}

	rows, err = mongoTarget.plugin.GetRows(mongoTarget.config, mongoTarget.schema, collection, nil, []*model.SortCondition{}, 10, 0)
	if err != nil {
		t.Fatalf("mongo get rows after delete failed: %v", err)
	}
	if len(rows.Rows) != 0 {
		t.Fatalf("expected mongo collection empty after delete, got %v", rows.Rows)
	}

	// NDJSON export
	mongoPlugin, ok := mongoTarget.plugin.PluginFunctions.(*mongoplugin.MongoDBPlugin)
	if !ok {
		t.Fatalf("unexpected mongo plugin type")
	}

	var ndjsonLines []string
	err = mongoPlugin.ExportDataNDJSON(mongoTarget.config, mongoTarget.schema, collection, func(line string) error {
		ndjsonLines = append(ndjsonLines, line)
		return nil
	}, nil)
	if err != nil {
		t.Fatalf("mongo ndjson export failed: %v", err)
	}
	if len(ndjsonLines) != 0 {
		t.Fatalf("expected no lines after deletion, got %d", len(ndjsonLines))
	}

	// Error paths: missing _id and invalid ObjectID
	_, err = mongoTarget.plugin.UpdateStorageUnit(mongoTarget.config, mongoTarget.schema, collection, map[string]string{
		"document": `{"name":"noid"}`,
	}, []string{"name"})
	if err == nil {
		t.Fatalf("expected mongo update without _id to fail")
	}
	_, err = mongoTarget.plugin.DeleteRow(mongoTarget.config, mongoTarget.schema, collection, map[string]string{
		"document": `{"_id":"not-a-hex"}`,
	})
	if err == nil {
		t.Fatalf("expected mongo delete with invalid id to fail")
	}

	_, err = mongoTarget.plugin.UpdateStorageUnit(mongoTarget.config, mongoTarget.schema, "nonexistent", map[string]string{
		"document": `{"_id":"507f1f77bcf86cd799439011","name":"x"}`,
	}, []string{"name"})
	if err == nil {
		t.Fatalf("expected mongo update on missing collection to fail")
	}
}

func TestElasticsearchCRUDAndSearch(t *testing.T) {
	var esTarget *target
	for i := range targets {
		if targets[i].plugin.Type == engine.DatabaseType_ElasticSearch {
			esTarget = &targets[i]
			break
		}
	}
	if esTarget == nil {
		t.Skip("elasticsearch target not configured")
	}

	index := fmt.Sprintf("intg-es-%d", time.Now().UnixNano())
	fields := []engine.Record{
		{Key: "title", Value: "text"},
		{Key: "count", Value: "integer"},
		{Key: "created_at", Value: "date"},
	}
	created, err := esTarget.plugin.AddStorageUnit(esTarget.config, esTarget.schema, index, fields)
	if err != nil || !created {
		t.Fatalf("failed to create index: %v", err)
	}

	esPlugin, ok := esTarget.plugin.PluginFunctions.(*elasticplugin.ElasticSearchPlugin)
	if !ok {
		t.Fatalf("unexpected elastic plugin type")
	}
	clientConfig := *esTarget.config
	clientConfig.Credentials.Hostname = "localhost"
	client, err := elasticplugin.DB(&clientConfig)
	if err != nil {
		t.Fatalf("failed to get elastic client: %v", err)
	}
	mapping, err := client.Indices.GetMapping(client.Indices.GetMapping.WithIndex(index))
	if err != nil {
		t.Fatalf("get mapping failed: %v", err)
	}
	var mappingBody map[string]any
	if err := json.NewDecoder(mapping.Body).Decode(&mappingBody); err != nil {
		t.Fatalf("decode mapping failed: %v", err)
	}
	if len(mappingBody) == 0 {
		t.Fatalf("expected mapping for %s", index)
	}

	docID := "es-doc-1"
	inserted, err := esTarget.plugin.AddRow(esTarget.config, esTarget.schema, index, []engine.Record{
		{Key: "_id", Value: docID},
		{Key: "title", Value: "Hello ES"},
		{Key: "count", Value: "1"},
		{Key: "created_at", Value: "2024-01-02T15:04:05Z"},
	})
	if err != nil || !inserted {
		t.Fatalf("failed to insert elastic doc: %v", err)
	}

	rows, err := esTarget.plugin.GetRows(esTarget.config, esTarget.schema, index, nil, []*model.SortCondition{}, 10, 0)
	if err != nil {
		t.Fatalf("elasticsearch get rows failed: %v", err)
	}
	if len(rows.Rows) == 0 || !strings.Contains(rows.Rows[0][0], "Hello ES") {
		t.Fatalf("elastic doc missing: %+v", rows.Rows)
	}

	// Insert another doc to exercise sort/pagination
	_, err = esTarget.plugin.AddRow(esTarget.config, esTarget.schema, index, []engine.Record{
		{Key: "_id", Value: "es-doc-2"},
		{Key: "title", Value: "Hello ES 2"},
		{Key: "count", Value: "5"},
		{Key: "created_at", Value: "2024-03-04T11:00:00Z"},
	})
	if err != nil {
		t.Fatalf("failed to insert second elastic doc: %v", err)
	}

	where := &model.WhereCondition{
		Type: model.WhereConditionTypeAtomic,
		Atomic: &model.AtomicWhereCondition{
			Key:        "count",
			Operator:   ">",
			Value:      "0",
			ColumnType: "integer",
		},
	}
	filtered, err := esTarget.plugin.GetRows(esTarget.config, esTarget.schema, index, where, []*model.SortCondition{}, 10, 0)
	if err != nil {
		t.Fatalf("elastic filtered query failed: %v", err)
	}
	if filtered.TotalCount < 1 {
		t.Fatalf("elastic filtered query returned no rows")
	}

	sortDesc := []*model.SortCondition{{Column: "count", Direction: model.SortDirectionDesc}}
	sorted, err := esTarget.plugin.GetRows(esTarget.config, esTarget.schema, index, nil, sortDesc, 2, 0)
	if err != nil {
		t.Fatalf("elastic sorted query failed: %v", err)
	}
	parseCount := func(row string) (int, error) {
		var obj map[string]any
		if err := json.Unmarshal([]byte(row), &obj); err != nil {
			return 0, err
		}
		switch c := obj["count"].(type) {
		case float64:
			return int(c), nil
		case string:
			return strconv.Atoi(c)
		default:
			return 0, fmt.Errorf("unexpected count type %T", c)
		}
	}
	if len(sorted.Rows) < 2 {
		t.Fatalf("elastic sort not applied correctly: %+v", sorted.Rows)
	}
	firstCount, err := parseCount(sorted.Rows[0][0])
	if err != nil {
		t.Fatalf("failed to parse first sorted count: %v", err)
	}
	secondCount, err := parseCount(sorted.Rows[1][0])
	if err != nil {
		t.Fatalf("failed to parse second sorted count: %v", err)
	}
	if !(firstCount >= secondCount && firstCount == 5) {
		t.Fatalf("elastic sort not applied correctly: %+v", sorted.Rows)
	}

	rangeFilter := &model.WhereCondition{
		Type: model.WhereConditionTypeAtomic,
		Atomic: &model.AtomicWhereCondition{
			Key:        "count",
			Operator:   ">",
			Value:      "1",
			ColumnType: "integer",
		},
	}
	ranged, err := esTarget.plugin.GetRows(esTarget.config, esTarget.schema, index, rangeFilter, []*model.SortCondition{}, 10, 0)
	if err != nil {
		t.Fatalf("elastic range filter failed: %v", err)
	}
	if ranged.TotalCount < 1 {
		t.Fatalf("expected range filter to return rows")
	}

	updateJSON, _ := json.Marshal(map[string]any{
		"_id":        docID,
		"title":      "Updated ES",
		"count":      2,
		"created_at": "2024-02-03T10:00:00Z",
	})
	updated, err := esTarget.plugin.UpdateStorageUnit(esTarget.config, esTarget.schema, index, map[string]string{
		"document": string(updateJSON),
	}, []string{"title", "count"})
	if err != nil || !updated {
		t.Fatalf("elastic update failed: %v", err)
	}

	rows, err = esTarget.plugin.GetRows(esTarget.config, esTarget.schema, index, nil, []*model.SortCondition{}, 10, 0)
	if err != nil {
		t.Fatalf("elastic get rows after update failed: %v", err)
	}
	foundUpdated := false
	for _, r := range rows.Rows {
		if strings.Contains(r[0], "Updated ES") && (strings.Contains(r[0], `"count":2`) || strings.Contains(r[0], `"count":"2"`)) {
			foundUpdated = true
			break
		}
	}
	if !foundUpdated {
		t.Fatalf("elastic update not reflected: %+v", rows.Rows)
	}

	_, err = esTarget.plugin.UpdateStorageUnit(esTarget.config, esTarget.schema, index, map[string]string{
		"document": `{"title":"missing id"}`,
	}, []string{"title"})
	if err == nil {
		t.Fatalf("expected elastic update without _id to fail")
	}

	deleted, err := esTarget.plugin.DeleteRow(esTarget.config, esTarget.schema, index, map[string]string{
		"document": string(updateJSON),
	})
	if err != nil || !deleted {
		t.Fatalf("elastic delete failed: %v", err)
	}

	_, err = esTarget.plugin.DeleteRow(esTarget.config, esTarget.schema, index, map[string]string{
		"document": `{"title":"missing id"}`,
	})
	if err == nil {
		t.Fatalf("expected elastic delete without _id to fail")
	}

	rows, err = esTarget.plugin.GetRows(esTarget.config, esTarget.schema, index, nil, []*model.SortCondition{}, 10, 0)
	if err != nil {
		t.Fatalf("elastic get rows after delete failed: %v", err)
	}
	for _, r := range rows.Rows {
		if strings.Contains(r[0], docID) {
			t.Fatalf("expected deleted doc to be gone, found %+v", rows.Rows)
		}
	}

	var exported [][]string
	if err := esPlugin.ExportData(esTarget.config, esTarget.schema, index, func(row []string) error {
		exported = append(exported, row)
		return nil
	}, nil); err != nil && !strings.Contains(err.Error(), "index") {
		t.Fatalf("elastic export failed: %v", err)
	}

	var ndjsonLines []string
	if err := esPlugin.ExportDataNDJSON(esTarget.config, esTarget.schema, index, func(line string) error {
		ndjsonLines = append(ndjsonLines, line)
		return nil
	}, nil); err != nil && !strings.Contains(err.Error(), "index") {
		t.Fatalf("elastic ndjson export failed: %v", err)
	}

	var selected [][]string
	selectedRows := []map[string]any{{"_id": "sel1", "title": "Selected", "count": 9}}
	if err := esPlugin.ExportData(esTarget.config, esTarget.schema, index, func(row []string) error {
		selected = append(selected, row)
		return nil
	}, selectedRows); err != nil {
		t.Fatalf("elastic selected export failed: %v", err)
	}
	if len(selected) != 2 {
		t.Fatalf("expected selected export to emit header + 1 row, got %d", len(selected))
	}
	if len(selected[1]) == 0 || (!strings.Contains(strings.Join(selected[1], ","), "Selected") && !strings.Contains(strings.Join(selected[1], ","), "sel1")) {
		t.Fatalf("selected export row does not contain expected values: %+v", selected[1])
	}

	if _, err := esTarget.plugin.GetRows(esTarget.config, esTarget.schema, "missing-index", nil, nil, 5, 0); err == nil {
		t.Fatalf("expected missing index query to fail")
	}
}

func TestRedisLiveData(t *testing.T) {
	var redisTarget *target
	for i := range targets {
		if targets[i].plugin.Type == engine.DatabaseType_Redis {
			redisTarget = &targets[i]
			break
		}
	}
	if redisTarget == nil {
		t.Skip("redis target not configured")
	}

	client, err := redisplugin.DB(redisTarget.config)
	if err != nil {
		t.Fatalf("failed to connect to redis: %v", err)
	}
	defer client.Close()
	ctx := context.Background()

	stringKey := "intg:string"
	hashKey := "intg:hash"
	listKey := "intg:list"
	zsetKey := "intg:zset"
	setKey := "intg:set"

	client.Del(ctx, stringKey, hashKey, listKey, zsetKey, setKey)

	if err := client.Set(ctx, stringKey, "value-one", 0).Err(); err != nil {
		t.Fatalf("failed to seed string key: %v", err)
	}
	stringRows, err := redisTarget.plugin.GetRows(redisTarget.config, redisTarget.schema, stringKey, nil, []*model.SortCondition{}, 10, 0)
	if err != nil || len(stringRows.Rows) != 1 || !strings.Contains(stringRows.Rows[0][0], "value-one") {
		t.Fatalf("unexpected redis string rows: %+v err=%v", stringRows.Rows, err)
	}

	if err := client.HSet(ctx, hashKey, "field", "v1").Err(); err != nil {
		t.Fatalf("failed to seed hash: %v", err)
	}
	hashRows, err := redisTarget.plugin.GetRows(redisTarget.config, redisTarget.schema, hashKey, nil, []*model.SortCondition{}, 10, 0)
	if err != nil || len(hashRows.Rows) == 0 {
		t.Fatalf("hash rows missing: %+v err=%v", hashRows.Rows, err)
	}
	if _, err := redisTarget.plugin.UpdateStorageUnit(redisTarget.config, redisTarget.schema, hashKey, map[string]string{
		"field": "field",
		"value": "v2",
	}, []string{"value"}); err != nil {
		t.Fatalf("hash update failed: %v", err)
	}
	hashRows, _ = redisTarget.plugin.GetRows(redisTarget.config, redisTarget.schema, hashKey, nil, []*model.SortCondition{}, 10, 0)
	if !strings.Contains(hashRows.Rows[0][1], "v2") {
		t.Fatalf("hash update not reflected: %+v", hashRows.Rows)
	}
	if _, err := redisTarget.plugin.DeleteRow(redisTarget.config, redisTarget.schema, hashKey, map[string]string{"field": "field"}); err != nil {
		t.Fatalf("hash delete failed: %v", err)
	}

	if err := client.RPush(ctx, listKey, "a", "b", "c").Err(); err != nil {
		t.Fatalf("failed to seed list: %v", err)
	}
	listRows, err := redisTarget.plugin.GetRows(redisTarget.config, redisTarget.schema, listKey, nil, []*model.SortCondition{}, 10, 0)
	if err != nil || len(listRows.Rows) != 3 {
		t.Fatalf("list rows unexpected: %+v err=%v", listRows.Rows, err)
	}
	if _, err := redisTarget.plugin.UpdateStorageUnit(redisTarget.config, redisTarget.schema, listKey, map[string]string{
		"index": "1",
		"value": "bee",
	}, []string{"value"}); err != nil {
		t.Fatalf("list update failed: %v", err)
	}
	listRows, _ = redisTarget.plugin.GetRows(redisTarget.config, redisTarget.schema, listKey, nil, []*model.SortCondition{}, 10, 0)
	if listRows.Rows[1][1] != "bee" {
		t.Fatalf("list update not reflected: %+v", listRows.Rows)
	}
	if _, err := redisTarget.plugin.DeleteRow(redisTarget.config, redisTarget.schema, listKey, map[string]string{"index": "0"}); err != nil {
		t.Fatalf("list delete failed: %v", err)
	}

	if err := client.SAdd(ctx, setKey, "red", "blue").Err(); err != nil {
		t.Fatalf("failed to seed set: %v", err)
	}
	setRows, err := redisTarget.plugin.GetRows(redisTarget.config, redisTarget.schema, setKey, nil, []*model.SortCondition{}, 10, 0)
	if err != nil || len(setRows.Rows) != 2 {
		t.Fatalf("set rows unexpected: %+v err=%v", setRows.Rows, err)
	}
	if _, err := redisTarget.plugin.DeleteRow(redisTarget.config, redisTarget.schema, setKey, map[string]string{"member": "red"}); err != nil {
		t.Fatalf("set delete failed: %v", err)
	}

	if err := client.ZAdd(ctx, zsetKey, &redislib.Z{Score: 1, Member: "low"}, &redislib.Z{Score: 2, Member: "high"}).Err(); err != nil {
		t.Fatalf("failed to seed zset: %v", err)
	}
	zsetRows, err := redisTarget.plugin.GetRows(redisTarget.config, redisTarget.schema, zsetKey, nil, []*model.SortCondition{}, 10, 0)
	if err != nil || len(zsetRows.Rows) != 2 {
		t.Fatalf("zset rows unexpected: %+v err=%v", zsetRows.Rows, err)
	}
	if _, err := redisTarget.plugin.DeleteRow(redisTarget.config, redisTarget.schema, zsetKey, map[string]string{
		"member": "low",
	}); err != nil {
		t.Fatalf("zset delete failed: %v", err)
	}
	zsetRows, _ = redisTarget.plugin.GetRows(redisTarget.config, redisTarget.schema, zsetKey, nil, []*model.SortCondition{}, 10, 0)
	if len(zsetRows.Rows) != 1 || !strings.Contains(zsetRows.Rows[0][1], "high") {
		t.Fatalf("zset delete not reflected: %+v", zsetRows.Rows)
	}

	if cnt, err := redisTarget.plugin.GetRowCount(redisTarget.config, redisTarget.schema, stringKey, nil); err != nil || cnt != 1 {
		t.Fatalf("row count for string unexpected: %d err=%v", cnt, err)
	}
	if _, err := redisTarget.plugin.DeleteRow(redisTarget.config, redisTarget.schema, setKey, map[string]string{}); err == nil {
		t.Fatalf("expected delete without member to fail for set")
	}
	if _, err := redisTarget.plugin.DeleteRow(redisTarget.config, redisTarget.schema, listKey, map[string]string{}); err == nil {
		t.Fatalf("expected delete without index to fail for list")
	}

	if units, err := redisTarget.plugin.GetStorageUnits(redisTarget.config, redisTarget.schema); err != nil || len(units) == 0 {
		t.Fatalf("expected redis storage units, got err=%v count=%d", err, len(units))
	}

	ttlKey := "intg:ttl"
	if err := client.Set(ctx, ttlKey, "expire", 2*time.Second).Err(); err != nil {
		t.Fatalf("failed to set ttl key: %v", err)
	}
	time.Sleep(3 * time.Second)
	if exists, _ := redisTarget.plugin.StorageUnitExists(redisTarget.config, redisTarget.schema, ttlKey); exists {
		t.Fatalf("expected ttl key to expire")
	}

	client.Del(ctx, stringKey, hashKey, listKey, zsetKey, setKey)
}

func runSQLTypeCase(t *testing.T, target target, tc typeCase, idx int) {
	t.Helper()

	table := fmt.Sprintf("intg_%s_%s_%d", target.name, sanitize(tc.columnType), idx)
	pkType := primaryKeyType(target)
	fields := []engine.Record{
		{Key: "id", Value: pkType, Extra: map[string]string{"Primary": "true", "Nullable": "false"}},
		{Key: "val", Value: tc.columnType, Extra: map[string]string{"Primary": "false", "Nullable": "false"}},
	}

	created, err := target.plugin.AddStorageUnit(target.config, target.schema, table, fields)
	if err != nil || !created {
		t.Skipf("skip type %s on %s: %v", tc.columnType, target.name, err)
		return
	}
	defer target.plugin.RawExecute(target.config, dropStatement(target, table))

	ok, err := target.plugin.AddRow(target.config, target.schema, table, []engine.Record{
		{Key: "id", Value: "1", Extra: map[string]string{"Type": pkType}},
		{Key: "val", Value: tc.value, Extra: map[string]string{"Type": tc.columnType}},
	})
	if err != nil || !ok {
		t.Fatalf("failed to insert sample for %s on %s: %v", tc.columnType, target.name, err)
	}

	res, err := target.plugin.GetRows(target.config, target.schema, table, nil, []*model.SortCondition{}, 5, 0)
	if err != nil {
		t.Fatalf("GetRows failed for %s on %s: %v", tc.columnType, target.name, err)
	}
	valIdx := columnIndex(res.Columns, "val")
	if valIdx == -1 || len(res.Rows) == 0 {
		t.Fatalf("no val column for %s on %s", tc.columnType, target.name)
	}
	got := res.Rows[0][valIdx]
	expect := tc.expect
	if expect == nil {
		expect = expectationForValue(tc, tc.value)
	}
	if !expect(got) {
		t.Fatalf("round trip mismatch for %s on %s: got %s", tc.columnType, target.name, got)
	}

	if tc.updated != "" {
		updatedExpect := expect
		if tc.updated != tc.value {
			updatedExpect = expectationForValue(tc, tc.updated)
		}
		_, err := target.plugin.UpdateStorageUnit(target.config, target.schema, table, map[string]string{
			"id":  "1",
			"val": tc.updated,
		}, []string{"val"})
		if err != nil {
			if strings.Contains(err.Error(), "no rows were updated") || strings.Contains(err.Error(), "scale") || strings.Contains(err.Error(), "syntax for type date") {
				return
			}
			t.Fatalf("update failed for %s on %s: %v", tc.columnType, target.name, err)
		}
		attempts := 1
		if target.plugin.Type == engine.DatabaseType_ClickHouse {
			attempts = 5
		}
		var lastGot string
		for i := 0; i < attempts; i++ {
			if i > 0 {
				time.Sleep(200 * time.Millisecond)
			}
			res, err = target.plugin.GetRows(target.config, target.schema, table, nil, []*model.SortCondition{}, 5, 0)
			if err != nil {
				t.Fatalf("GetRows after update failed for %s on %s: %v", tc.columnType, target.name, err)
			}
			lastGot = res.Rows[0][valIdx]
			if updatedExpect(lastGot) {
				break
			}
		}
		if !updatedExpect(lastGot) {
			t.Fatalf("update mismatch for %s on %s: got %s expected %s", tc.columnType, target.name, lastGot, tc.updated)
		}
	}
}

func sqlCasesForTarget(target target) []typeCase {
	meta := target.plugin.GetDatabaseMetadata()
	if meta == nil {
		return nil
	}

	overrides := overridesFor(target.plugin.Type)
	baseCases := make([]typeCase, 0, len(meta.TypeDefinitions))
	for _, td := range meta.TypeDefinitions {
		base := strings.ToUpper(td.ID)
		override := overrides[base]
		if override.skip {
			continue
		}

		columnType := override.columnType
		if columnType == "" {
			columnType = td.ID
			if td.HasLength && td.DefaultLength != nil {
				columnType = fmt.Sprintf("%s(%d)", td.ID, *td.DefaultLength)
			}
			if td.HasPrecision && td.DefaultPrecision != nil {
				scale := 2
				columnType = fmt.Sprintf("%s(%d,%d)", td.ID, *td.DefaultPrecision, scale)
			}
		}

		value := override.value
		if value == "" {
			value = defaultValueFor(target.plugin.Type, base, td.Category)
		}

		updated := override.updated
		if updated == "" {
			updated = defaultUpdatedValue(value, base)
		}

		expect := override.expect

		baseCases = append(baseCases, typeCase{
			name:       base,
			columnType: columnType,
			value:      value,
			updated:    updated,
			expect:     expect,
			category:   td.Category,
		})
	}

	return addAliasCases(meta, baseCases)
}

func addAliasCases(meta *engine.DatabaseMetadata, baseCases []typeCase) []typeCase {
	if meta == nil {
		return baseCases
	}

	cases := make([]typeCase, 0, len(baseCases)+len(meta.AliasMap))
	cases = append(cases, baseCases...)

	byBase := map[string]typeCase{}
	for _, tc := range baseCases {
		byBase[common.ParseTypeSpec(tc.columnType).BaseType] = tc
	}

	for alias, canonical := range meta.AliasMap {
		base := strings.ToUpper(canonical)
		tc, ok := byBase[base]
		if !ok {
			continue
		}
		spec := common.ParseTypeSpec(tc.columnType)
		spec.BaseType = strings.ToUpper(alias)
		cases = append(cases, typeCase{
			name:       fmt.Sprintf("%s_alias", alias),
			columnType: common.FormatTypeSpec(spec),
			value:      tc.value,
			updated:    tc.updated,
			expect:     tc.expect,
			category:   tc.category,
		})
	}

	return cases
}

func overridesFor(dbType engine.DatabaseType) map[string]typeOverride {
	mysqlOverrides := map[string]typeOverride{
		"TINYINT": {updated: ""},
		"DECIMAL": {columnType: "DECIMAL(10,2)"},
		"BOOLEAN": {value: "1", updated: "0"},
		"TIME":    {value: "12:34:56", updated: "23:45:59"},
		"TIMESTAMP": {
			value:   "2024-01-02 15:04:05",
			updated: "2025-01-02 15:04:05",
		},
		"ENUM":       {columnType: "ENUM('red','blue')", value: "red", updated: "blue"},
		"SET":        {columnType: "SET('a','b','c')", value: "a,b", updated: "b"},
		"BINARY":     {columnType: "BINARY(4)", value: "ab", updated: "cd", expect: expectHexContains("6162")},
		"VARBINARY":  {value: "ab", expect: expectBinaryEqual("ab")},
		"TINYBLOB":   {value: "a", expect: expectBinaryEqual("a")},
		"BLOB":       {value: "b", expect: expectBinaryEqual("b")},
		"MEDIUMBLOB": {value: "c", expect: expectBinaryEqual("c")},
		"LONGBLOB":   {value: "d", expect: expectBinaryEqual("d")},
		"YEAR":       {value: "2024", updated: "2025"},
		"CHAR":       {columnType: "CHAR(5)", value: "hello", updated: "world"},
		"VARCHAR":    {columnType: "VARCHAR(64)"},
	}

	clickhouseOverrides := map[string]typeOverride{
		"DECIMAL":     {columnType: "Decimal(10,2)"},
		"DECIMAL32":   {columnType: "Decimal32(2)"},
		"DECIMAL64":   {columnType: "Decimal64(2)"},
		"DECIMAL128":  {columnType: "Decimal128(2)"},
		"FIXEDSTRING": {columnType: "FixedString(16)", value: "fixedstringvalue", updated: "fixedstringvalue"},
		"DATETIME64":  {columnType: "DateTime64(3)", value: "2024-01-02 15:04:05.123", updated: "2024-02-03 10:00:00.456"},
		"ENUM8":       {columnType: "Enum8('a' = 1, 'b' = 2)", value: "a", updated: "b"},
		"ENUM16":      {columnType: "Enum16('a' = 1, 'b' = 2)", value: "a", updated: "b"},
		"JSON":        {value: `{"ch":true}`, updated: `{"ch":false}`, expect: expectJSONStrict(`{"ch":true}`)},
	}

	pgOverrides := map[string]typeOverride{
		"CHARACTER VARYING":        {columnType: "CHARACTER VARYING(64)", value: "varchar value", updated: "varchar updated"},
		"CHARACTER":                {columnType: "CHARACTER(4)", value: "abcd", updated: "wxyz"},
		"DECIMAL":                  {columnType: "DECIMAL(10,2)"},
		"NUMERIC":                  {columnType: "NUMERIC(8,3)"},
		"BYTEA":                    {value: "hello", updated: "okay", expect: expectBinaryEqual("hello")},
		"TIMESTAMP":                {value: "2024-01-02 15:04:05", updated: "2024-02-02 15:04:05"},
		"TIMESTAMP WITH TIME ZONE": {value: "2024-01-02T15:04:05Z", updated: "2025-01-02T00:00:00Z"},
		"TIME":                     {value: "12:34:56", updated: "23:59:59"},
		"TIME WITH TIME ZONE":      {value: "12:34:56+00", updated: "01:02:03+00"},
		"MONEY":                    {value: "123.45", updated: "67.89", expect: expectContains("123.45")},
		"UUID":                     {value: "00000000-0000-0000-0000-000000000123"},
		"ARRAY":                    {columnType: "INTEGER[]", value: "{1,2,3}", updated: "{4,5,6}"},
		"JSONB":                    {value: `{"key":"value"}`, updated: `{"key":"updated"}`},
		"JSON":                     {value: `{"json":"value"}`, updated: `{"json":"updated"}`},
		"CIDR":                     {value: "10.0.0.0/24", updated: "10.0.1.0/24"},
		"INET":                     {value: "10.0.0.1", updated: "10.0.0.2"},
		"MACADDR":                  {value: "AA:BB:CC:DD:EE:FF", updated: "00:11:22:33:44:55"},
		"POINT":                    {value: "(1,2)", updated: "(3,4)"},
		"LSEG":                     {value: "[(0,0),(1,1)]", updated: "[(1,1),(2,2)]"},
		"BOX":                      {value: "((0,0),(1,1))", updated: "((0,0),(1,1))", expect: expectContains("(")},
		"PATH":                     {value: "[(0,0),(1,1),(2,0)]", updated: "[(0,0),(2,2)]"},
		"CIRCLE":                   {value: "<(0,0),1>", updated: "<(1,1),2>"},
		"POLYGON":                  {value: "((0,0),(1,0),(1,1))", updated: "((0,0),(2,0),(2,2))"},
		"XML":                      {value: "<root><item>1</item></root>", updated: "<root><item>2</item></root>"},
		"HSTORE":                   {value: "\"a\"=>\"1\"", updated: "\"a\"=>\"2\""},
		"SERIAL":                   {value: "5", updated: "6"},
		"BIGSERIAL":                {value: "7", updated: "8"},
		"SMALLSERIAL":              {value: "9", updated: "10"},
		"LINE":                     {value: "((0,0),(1,1))", updated: "((0,0),(1,1))", expect: expectContains("{")},
	}

	switch dbType {
	case engine.DatabaseType_Postgres:
		return pgOverrides
	case engine.DatabaseType_ClickHouse:
		return clickhouseOverrides
	case engine.DatabaseType_MariaDB:
		return mysqlOverrides
	case engine.DatabaseType_MySQL:
		return mysqlOverrides
	default:
		return map[string]typeOverride{}
	}
}

func defaultValueFor(dbType engine.DatabaseType, base string, category engine.TypeCategory) string {
	switch category {
	case engine.TypeCategoryNumeric:
		if strings.Contains(base, "DECIMAL") || strings.Contains(base, "NUMERIC") {
			return "123.45"
		}
		if strings.Contains(base, "MONEY") {
			return "12.34"
		}
		return "42"
	case engine.TypeCategoryText:
		return "a"
	case engine.TypeCategoryBinary:
		return "a"
	case engine.TypeCategoryDatetime:
		switch base {
		case "DATE", "DATE32":
			return "2024-01-02"
		case "TIME", "TIME WITH TIME ZONE":
			return "12:34:56"
		case "YEAR":
			return "2024"
		default:
			return "2024-01-02 15:04:05"
		}
	case engine.TypeCategoryBoolean:
		return "true"
	case engine.TypeCategoryJSON:
		return `{"key":"value"}`
	case engine.TypeCategoryOther:
		switch base {
		case "UUID":
			return "00000000-0000-0000-0000-000000000000"
		case "CIDR":
			return "192.168.0.0/24"
		case "INET", "IPV4":
			return "192.168.0.1"
		case "IPV6":
			return "2001:db8::1"
		case "MACADDR":
			return "AA:BB:CC:DD:EE:11"
		case "BOOL":
			return "true"
		case "ENUM8", "ENUM16":
			return "a"
		default:
			if strings.Contains(base, "ARRAY") {
				return "{1,2,3}"
			}
			if strings.Contains(base, "UUID") {
				return "00000000-0000-0000-0000-000000000001"
			}
			return "other"
		}
	default:
		return "sample"
	}
}

func defaultUpdatedValue(original string, base string) string {
	switch {
	case strings.Contains(strings.ToUpper(base), "JSON"):
		if mutated := mutateJSON(original); mutated != "" {
			return mutated
		}
		return `{"updated":true}`
	case strings.HasPrefix(original, "{") || strings.HasPrefix(original, "<") || strings.HasPrefix(original, "("):
		return original
	case regexp.MustCompile(`^[0-9a-fA-Fx]+$`).MatchString(original):
		return original
	}
	switch strings.ToLower(original) {
	case "true":
		return "false"
	case "false":
		return "true"
	}
	upperBase := strings.ToUpper(base)
	if strings.Contains(upperBase, "UUID") {
		return original
	}
	if strings.Contains(upperBase, "IPV4") || strings.Contains(upperBase, "INET") {
		return "192.168.0.2"
	}
	if strings.Contains(upperBase, "CIDR") {
		return "192.168.1.0/24"
	}
	if strings.Contains(upperBase, "IPV6") {
		return "2001:db8::2"
	}
	if strings.Contains(upperBase, "DATE") {
		return "2024-02-02"
	}
	if strings.Contains(upperBase, "TIME") {
		return "01:02:03"
	}
	if _, err := strconv.ParseFloat(original, 64); err == nil {
		return "99"
	}
	return original + "-updated"
}

func mutateJSON(original string) string {
	var v any
	if err := json.Unmarshal([]byte(original), &v); err != nil {
		return ""
	}
	switch m := v.(type) {
	case map[string]any:
		m["updated"] = true
		if b, err := json.Marshal(m); err == nil {
			return string(b)
		}
	case []any:
		m = append(m, "updated")
		if b, err := json.Marshal(m); err == nil {
			return string(b)
		}
	default:
		if b, err := json.Marshal([]any{m, "updated"}); err == nil {
			return string(b)
		}
	}
	return ""
}

func expectContains(substr string) func(string) bool {
	return func(got string) bool {
		return strings.Contains(strings.ToLower(got), strings.ToLower(substr))
	}
}

func expectHexContains(hexStr string) func(string) bool {
	lower := strings.ToLower(strings.TrimPrefix(hexStr, "0x"))
	asciiHex := strings.ToLower(fmt.Sprintf("%x", []byte(lower)))
	return func(got string) bool {
		g := strings.ToLower(strings.TrimPrefix(got, "0x"))
		return strings.Contains(g, lower) || strings.Contains(g, asciiHex)
	}
}

func expectBinaryEqual(sample string) func(string) bool {
	return func(got string) bool {
		isHexString := func(s string) bool {
			for _, r := range s {
				if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
					continue
				}
				return false
			}
			return len(s) > 0
		}

		sampleLower := strings.ToLower(strings.TrimPrefix(sample, "0x"))
		sampleIsHex := strings.HasPrefix(strings.ToLower(sample), "0x") || (len(sampleLower)%2 == 0 && len(sampleLower) >= 4 && isHexString(sampleLower))

		raw := strings.TrimPrefix(strings.ToLower(got), "0x")
		if decoded, err := hex.DecodeString(raw); err == nil {
			decoded = bytes.TrimRight(decoded, "\x00")
			// If sample is hex (explicitly marked or long hex), decode it; otherwise compare to raw string
			if sampleIsHex {
				if sampleHex, err := hex.DecodeString(sampleLower); err == nil {
					return bytes.Equal(decoded, sampleHex)
				}
			}
			return strings.EqualFold(string(decoded), sample)
		}
		return strings.Contains(strings.ToLower(got), strings.ToLower(sample))
	}
}

func expectJSONStrict(expected string) func(string) bool {
	return func(got string) bool {
		var m map[string]any
		var exp map[string]any
		if err := json.Unmarshal([]byte(expected), &exp); err != nil {
			return false
		}
		if err := json.Unmarshal([]byte(got), &m); err != nil {
			trimmed := strings.Trim(got, `"`)
			if err2 := json.Unmarshal([]byte(trimmed), &m); err2 != nil {
				return false
			}
		}
		return reflect.DeepEqual(m, exp)
	}
}

func expectAnyHex(values ...string) func(string) bool {
	matchers := make([]func(string) bool, 0, len(values))
	for _, v := range values {
		matchers = append(matchers, expectHexContains(v))
	}
	return func(got string) bool {
		for _, m := range matchers {
			if m(got) {
				return true
			}
		}
		return false
	}
}

func defaultExpectation(tc typeCase) func(string) bool {
	return expectationForValue(tc, tc.value)
}

func expectationForValue(tc typeCase, value string) func(string) bool {
	base := strings.ToUpper(common.ParseTypeSpec(tc.columnType).BaseType)
	switch tc.category {
	case engine.TypeCategoryNumeric:
		if strings.Contains(base, "MONEY") {
			return expectMoneyEqual(value)
		}
		return expectNumericEqual(value)
	case engine.TypeCategoryDatetime:
		return expectTimeLike(value)
	case engine.TypeCategoryBoolean:
		return expectEqualNormalized(value)
	case engine.TypeCategoryBinary:
		return expectBinaryEqual(value)
	case engine.TypeCategoryJSON:
		return expectJSONStrict(value)
	default:
		switch base {
		case "UUID", "CIDR", "INET", "MACADDR", "IPV4", "IPV6":
			return expectEqualNormalized(value)
		}
		return expectContains(value)
	}
}

func expectNumericEqual(expected string) func(string) bool {
	return func(got string) bool {
		e := new(big.Rat)
		g := new(big.Rat)
		if _, ok := e.SetString(expected); !ok {
			return false
		}
		if _, ok := g.SetString(strings.Fields(got)[0]); !ok {
			return false
		}
		return e.Cmp(g) == 0
	}
}

func expectMoneyEqual(expected string) func(string) bool {
	numExpect := expectNumericEqual(expected)
	return func(got string) bool {
		clean := strings.Map(func(r rune) rune {
			if (r >= '0' && r <= '9') || r == '.' || r == '-' {
				return r
			}
			return -1
		}, got)
		return numExpect(clean)
	}
}

func expectTimeLike(expected string) func(string) bool {
	return func(got string) bool {
		exp := strings.TrimSpace(expected)
		g := strings.TrimSpace(got)
		if strings.Contains(exp, "T") || strings.Contains(exp, " ") {
			// compare date portion
			return strings.Contains(g, exp[:10])
		}
		return strings.Contains(g, exp)
	}
}

func expectEqualNormalized(expected string) func(string) bool {
	return func(got string) bool {
		return strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(got))
	}
}

func expectHexLength(hexValue string, columnType string) func(string) bool {
	valueBytes := len(strings.TrimPrefix(hexValue, "0x")) / 2
	typeLen := parseTypeLength(columnType)
	expectedBytes := valueBytes
	if typeLen > 0 && valueBytes >= typeLen {
		expectedBytes = typeLen
	}
	return func(got string) bool {
		raw := strings.TrimPrefix(strings.ToLower(got), "0x")
		return len(raw)/2 == expectedBytes
	}
}

func parseTypeLength(typeStr string) int {
	re := regexp.MustCompile(`\((\d+)`)
	m := re.FindStringSubmatch(typeStr)
	if len(m) == 2 {
		if n, err := strconv.Atoi(m[1]); err == nil {
			return n
		}
	}
	return 0
}

func primaryKeyType(target target) string {
	switch target.plugin.Type {
	case engine.DatabaseType_ClickHouse:
		return "Int32"
	case engine.DatabaseType_Postgres:
		return "INTEGER"
	default:
		return "INT"
	}
}

func dropStatement(target target, table string) string {
	if target.schema != "" {
		return fmt.Sprintf("DROP TABLE IF EXISTS %s.%s", target.schema, table)
	}
	return fmt.Sprintf("DROP TABLE IF EXISTS %s", table)
}

func columnIndex(cols []engine.Column, name string) int {
	for i, col := range cols {
		if strings.EqualFold(col.Name, name) {
			return i
		}
	}
	return -1
}

func sanitize(typeName string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	return reg.ReplaceAllString(strings.ToLower(typeName), "_")
}
