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

package redis

import (
	"testing"

	"github.com/clidey/whodb/core/src/common/ssl"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/query"
	"github.com/clidey/whodb/core/src/sourcecatalog"
	_ "github.com/clidey/whodb/core/src/sources/database"
)

func redisAtomicWhere(key, operator, value string) *query.WhereCondition {
	return &query.WhereCondition{
		Type: query.WhereConditionTypeAtomic,
		Atomic: &query.AtomicWhereCondition{
			Key:      key,
			Operator: operator,
			Value:    value,
		},
	}
}

func TestConvertWhereConditionToRedisFilter(t *testing.T) {
	filter, err := convertWhereConditionToRedisFilter(redisAtomicWhere(redisKeyValue, "contains", "alice"))
	if err != nil {
		t.Fatalf("expected atomic filter conversion to succeed, got %v", err)
	}
	if filter[redisKeyValue].Operator != "CONTAINS" || filter[redisKeyValue].Value != "alice" {
		t.Fatalf("unexpected redis filter: %#v", filter)
	}

	if _, err := convertWhereConditionToRedisFilter(&query.WhereCondition{Type: query.WhereConditionTypeOr}); err == nil {
		t.Fatal("expected compound where conditions to be rejected")
	}
}

func TestEvaluateRedisConditionOperators(t *testing.T) {
	testCases := []struct {
		name     string
		value    string
		operator string
		target   string
		want     bool
	}{
		{name: "equals", value: "paid", operator: "=", target: "paid", want: true},
		{name: "not equals", value: "paid", operator: "!=", target: "draft", want: true},
		{name: "greater than", value: "20", operator: ">", target: "10", want: true},
		{name: "contains", value: "alice@example.com", operator: "CONTAINS", target: "@example.com", want: true},
		{name: "starts with", value: "customer_42", operator: "STARTS WITH", target: "customer_", want: true},
		{name: "ends with", value: "invoice.pdf", operator: "ENDS WITH", target: ".pdf", want: true},
		{name: "in", value: "draft", operator: "IN", target: "pending, draft, paid", want: true},
		{name: "not in", value: "archived", operator: "NOT IN", target: "pending, draft, paid", want: true},
		{name: "unknown operator", value: "x", operator: "BOGUS", target: "x", want: false},
	}

	for _, tc := range testCases {
		if got := evaluateRedisCondition(tc.value, tc.operator, tc.target); got != tc.want {
			t.Fatalf("%s: expected %t, got %t", tc.name, tc.want, got)
		}
	}
}

func TestRedisFilterHelpers(t *testing.T) {
	hashWhere := redisAtomicWhere("field", "=", "email")
	if !filterRedisHash("email", "alice@example.com", hashWhere) {
		t.Fatal("expected hash filter to match field equality")
	}
	valueWhere := redisAtomicWhere(redisKeyValue, "CONTAINS", "alice")
	if !filterRedisList("alice@example.com", valueWhere) {
		t.Fatal("expected list filter to match value")
	}
	memberWhere := redisAtomicWhere("member", "=", "alice")
	if !filterRedisSet("alice", memberWhere) {
		t.Fatal("expected set filter to treat member as value alias")
	}
	scoreWhere := redisAtomicWhere("score", ">=", "10")
	if !filterRedisZSet("alice", "10", scoreWhere) {
		t.Fatal("expected zset filter to inspect score column")
	}

	// Unsupported compound filters are intentionally ignored instead of hiding data.
	ignoredWhere := &query.WhereCondition{Type: query.WhereConditionTypeAnd}
	if !filterRedisHash("field", redisKeyValue, ignoredWhere) {
		t.Fatal("expected invalid filter expressions to be ignored")
	}
}

func TestRedisMetadataFormattingAndSSLStatus(t *testing.T) {
	plugin := &RedisPlugin{}

	if got := plugin.FormatValue(nil); got != "" {
		t.Fatalf("expected nil values to format as empty strings, got %q", got)
	}
	if got := plugin.FormatValue(42); got != "42" {
		t.Fatalf("expected values to be stringified, got %q", got)
	}

	metadata, ok := sourcecatalog.ResolveSessionMetadata(string(engine.DatabaseType_Redis))
	if !ok || metadata == nil {
		t.Fatalf("expected redis metadata, got %#v", metadata)
	}
	if len(metadata.Operators) == 0 {
		t.Fatal("expected redis operators to be exposed")
	}

	status, err := plugin.GetSSLStatus(&engine.PluginConfig{
		Credentials: &engine.Credentials{
			Type:     string(engine.DatabaseType_Redis),
			Hostname: "cache.internal",
			Advanced: []engine.Record{
				{Key: ssl.KeySSLMode, Value: string(ssl.SSLModeEnabled)},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected redis SSL status lookup to succeed, got %v", err)
	}
	if status == nil || !status.IsEnabled || status.Mode != string(ssl.SSLModeEnabled) {
		t.Fatalf("expected enabled redis SSL status, got %#v", status)
	}
}
