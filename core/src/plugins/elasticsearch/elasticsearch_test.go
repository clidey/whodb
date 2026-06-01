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
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v8/esapi"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/query"
)

func TestConvertAtomicConditionToES(t *testing.T) {
	tests := []struct {
		name   string
		atomic *query.AtomicWhereCondition
		want   map[string]any
	}{
		{
			name: "id equality uses ids query",
			atomic: &query.AtomicWhereCondition{
				Key: "_id", Operator: "=", Value: "doc-1",
			},
			want: map[string]any{
				"ids": map[string]any{
					"values": []any{"doc-1"},
				},
			},
		},
		{
			name: "contains uses wildcard",
			atomic: &query.AtomicWhereCondition{
				Key: "email", Operator: "CONTAINS", Value: "example.com",
			},
			want: map[string]any{
				"wildcard": map[string]any{
					"email": map[string]any{"value": "*example.com*"},
				},
			},
		},
		{
			name: "terms parses csv values",
			atomic: &query.AtomicWhereCondition{
				Key: "status", Operator: "TERMS", Value: "paid, pending",
			},
			want: map[string]any{
				"terms": map[string]any{
					"status": []any{"paid", "pending"},
				},
			},
		},
		{
			name: "range accepts open upper bound",
			atomic: &query.AtomicWhereCondition{
				Key: "price", Operator: "RANGE", Value: "10,",
			},
			want: map[string]any{
				"range": map[string]any{
					"price": map[string]any{"gte": "10"},
				},
			},
		},
		{
			name: "unknown operators fall back to match",
			atomic: &query.AtomicWhereCondition{
				Key: "notes", Operator: "UNSUPPORTED", Value: "needle",
			},
			want: map[string]any{
				"match": map[string]any{
					"notes": "needle",
				},
			},
		},
	}

	for _, tt := range tests {
		got, err := convertAtomicConditionToES(tt.atomic)
		if err != nil {
			t.Fatalf("%s: expected conversion to succeed, got %v", tt.name, err)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("%s: unexpected clause\n got: %#v\nwant: %#v", tt.name, got, tt.want)
		}
	}
}

func TestConvertWhereConditionToES(t *testing.T) {
	where := &query.WhereCondition{
		Type: query.WhereConditionTypeAnd,
		And: &query.OperationWhereCondition{
			Children: []*query.WhereCondition{
				{
					Type: query.WhereConditionTypeAtomic,
					Atomic: &query.AtomicWhereCondition{
						Key:      "status",
						Operator: "=",
						Value:    "paid",
					},
				},
				{
					Type: query.WhereConditionTypeOr,
					Or: &query.OperationWhereCondition{
						Children: []*query.WhereCondition{
							{
								Type: query.WhereConditionTypeAtomic,
								Atomic: &query.AtomicWhereCondition{
									Key:      "priority",
									Operator: "=",
									Value:    "high",
								},
							},
							{
								Type: query.WhereConditionTypeAtomic,
								Atomic: &query.AtomicWhereCondition{
									Key:      "priority",
									Operator: "=",
									Value:    "urgent",
								},
							},
						},
					},
				},
			},
		},
	}

	got, err := convertWhereConditionToES(where)
	if err != nil {
		t.Fatalf("expected nested ES condition conversion to succeed, got %v", err)
	}
	mustClauses, ok := got["must"].([]map[string]any)
	if !ok || len(mustClauses) != 2 {
		t.Fatalf("expected two must clauses, got %#v", got)
	}
	nestedBool, ok := mustClauses[1]["bool"].(map[string]any)
	if !ok {
		t.Fatalf("expected OR child to be wrapped in bool query, got %#v", mustClauses[1])
	}
	if nestedBool["minimum_should_match"] != 1 {
		t.Fatalf("expected minimum_should_match=1, got %#v", nestedBool)
	}

	if _, err := convertWhereConditionToES(&query.WhereCondition{Type: query.WhereConditionTypeAtomic}); err == nil {
		t.Fatal("expected invalid atomic condition to fail")
	}
}

func TestElasticSearchHelpers(t *testing.T) {
	if got := parseCSVToSlice("paid, pending"); !reflect.DeepEqual(got, []any{"paid", "pending"}) {
		t.Fatalf("expected CSV values to be trimmed, got %#v", got)
	}
	min, max := parseRangeBounds("10, 20")
	if min != "10" || max != "20" {
		t.Fatalf("expected parsed range bounds, got %q %q", min, max)
	}

	if got := inferElasticSearchType(map[string]any{"id": 1}); got != "object" {
		t.Fatalf("expected object type inference, got %q", got)
	}
	if got := inferElasticSearchType([]any{"a"}); got != "array" {
		t.Fatalf("expected array type inference, got %q", got)
	}
	if got := mergeElasticTypes("text", "keyword"); got != "mixed" {
		t.Fatalf("expected mixed type merge, got %q", got)
	}

	mappings := buildElasticMappings([]engine.Record{
		{Key: "title", Value: "text"},
		{Key: "price", Value: "decimal"},
		{Key: "", Value: "text"},
	})
	if len(mappings) != 2 {
		t.Fatalf("expected empty field names to be filtered, got %#v", mappings)
	}
	if mappings["title"].(map[string]any)["type"] != "text" || mappings["price"].(map[string]any)["type"] != "double" {
		t.Fatalf("unexpected mappings: %#v", mappings)
	}

	res := &esapi.Response{
		StatusCode: 400,
		Body:       io.NopCloser(strings.NewReader(`{"error":"bad query"}`)),
	}
	if got := formatElasticError(res); got != `{"error":"bad query"}` {
		t.Fatalf("expected formatted body error, got %q", got)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("expected response body to remain readable, got %v", err)
	}
	if string(body) != `{"error":"bad query"}` {
		t.Fatalf("expected response body to be restored, got %q", string(body))
	}
}

func TestOpenSearchProductHeaderInterceptor(t *testing.T) {
	roundTrip := opensearchProductHeaderInterceptor(func(req *http.Request) (*http.Response, error) {
		return &http.Response{Header: http.Header{}, Body: io.NopCloser(strings.NewReader("{}"))}, nil
	})

	res, err := roundTrip(&http.Request{})
	if err != nil {
		t.Fatalf("expected interceptor round trip to succeed, got %v", err)
	}
	defer res.Body.Close()
	if got := res.Header.Get("X-Elastic-Product"); got != "Elasticsearch" {
		t.Fatalf("expected product header to be set for OpenSearch compatibility, got %q", got)
	}
}
