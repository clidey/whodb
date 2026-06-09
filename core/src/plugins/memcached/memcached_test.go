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

package memcached

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/query"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

func TestConvertWhereCondition(t *testing.T) {
	condition, err := convertWhereCondition(nil)
	if err != nil {
		t.Fatalf("expected nil where condition to succeed, got %v", err)
	}
	if condition != nil {
		t.Fatalf("expected nil condition result, got %#v", condition)
	}

	condition, err = convertWhereCondition(&query.WhereCondition{
		Type: query.WhereConditionTypeAtomic,
		Atomic: &query.AtomicWhereCondition{
			Key:      "Value",
			Operator: "contains",
			Value:    "foo",
		},
	})
	if err != nil {
		t.Fatalf("expected atomic condition to succeed, got %v", err)
	}
	want := map[string]memcachedFilter{
		"Value": {Operator: "CONTAINS", Value: "foo"},
	}
	if !reflect.DeepEqual(condition, want) {
		t.Fatalf("unexpected converted condition: got %#v want %#v", condition, want)
	}

	if _, err := convertWhereCondition(&query.WhereCondition{Type: query.WhereConditionTypeAnd}); err == nil {
		t.Fatal("expected compound Memcached conditions to be rejected")
	}
}

func TestEvaluateCondition(t *testing.T) {
	testCases := []struct {
		name     string
		value    string
		operator string
		target   string
		want     bool
	}{
		{name: "equals", value: "foo", operator: "=", target: "foo", want: true},
		{name: "not equals", value: "foo", operator: "!=", target: "bar", want: true},
		{name: "greater than", value: "9", operator: ">", target: "2", want: true},
		{name: "contains", value: "hello world", operator: "CONTAINS", target: "world", want: true},
		{name: "starts with", value: "prefix-value", operator: "STARTS WITH", target: "prefix", want: true},
		{name: "ends with", value: "prefix-value", operator: "ENDS WITH", target: "value", want: true},
		{name: "in", value: "beta", operator: "IN", target: "alpha, beta, gamma", want: true},
		{name: "not in", value: "beta", operator: "NOT IN", target: "alpha, gamma", want: true},
		{name: "unknown operator", value: "foo", operator: "??", target: "foo", want: false},
	}

	for _, tc := range testCases {
		if got := evaluateCondition(tc.value, tc.operator, tc.target); got != tc.want {
			t.Fatalf("%s: expected %t, got %t", tc.name, tc.want, got)
		}
	}
}

func TestFilterMemcachedRows(t *testing.T) {
	rows := [][]string{
		{"hello world", "1"},
		{"goodbye", "2"},
	}

	filtered := filterMemcachedRows(rows, &query.WhereCondition{
		Type: query.WhereConditionTypeAtomic,
		Atomic: &query.AtomicWhereCondition{
			Key:      "Value",
			Operator: "contains",
			Value:    "world",
		},
	})
	if len(filtered) != 1 || filtered[0][0] != "hello world" {
		t.Fatalf("expected one matching row, got %#v", filtered)
	}

	filtered = filterMemcachedRows(rows, &query.WhereCondition{
		Type: query.WhereConditionTypeAtomic,
		Atomic: &query.AtomicWhereCondition{
			Key:      "Flags",
			Operator: "IN",
			Value:    "2, 3",
		},
	})
	if len(filtered) != 1 || filtered[0][1] != "2" {
		t.Fatalf("expected flags filter to keep second row, got %#v", filtered)
	}

	// Unsupported compound filters are ignored and the original rows are returned.
	filtered = filterMemcachedRows(rows, &query.WhereCondition{Type: query.WhereConditionTypeAnd})
	if !reflect.DeepEqual(filtered, rows) {
		t.Fatalf("expected invalid filter to leave rows unchanged, got %#v", filtered)
	}
}

func TestMemcachedMetadataAndHelpers(t *testing.T) {
	plugin := &MemcachedPlugin{}
	if got := plugin.FormatValue(nil); got != "" {
		t.Fatalf("expected nil to format as empty string, got %q", got)
	}
	if got := plugin.FormatValue(42); got != "42" {
		t.Fatalf("expected non-nil value to use fmt string, got %q", got)
	}

	metadata, ok := sourcecatalog.ResolveSessionMetadata(string(engine.DatabaseType_Memcached))
	if !ok || metadata == nil {
		t.Fatalf("expected memcached metadata, got %#v", metadata)
	}
	if !sort.StringsAreSorted(metadata.Operators) {
		t.Fatalf("expected operators to be sorted, got %#v", metadata.Operators)
	}

	pluginDef := NewMemcachedPlugin()
	if pluginDef.Type != engine.DatabaseType_Memcached {
		t.Fatalf("expected Memcached plugin type, got %q", pluginDef.Type)
	}
	if pluginDef.PluginFunctions == nil {
		t.Fatal("expected plugin functions to be configured")
	}
}

func TestValidateMemcachedKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"simple key", "order:1", false},
		{"utf8 key", "cafe:été", false},
		{"empty key", "", true},
		{"space", "order 1", true},
		{"tab", "order\t1", true},
		{"newline", "order\nflush_all", true},
		{"carriage return", "order\r\nflush_all", true},
		{"delete control", "order" + string(rune(0x7f)), true},
		{"too long", strings.Repeat("a", maxMemcachedKeyLength+1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMemcachedKey(tt.key)
			if tt.wantErr && err == nil {
				t.Fatal("expected key validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected key to be valid, got %v", err)
			}
		})
	}
}
