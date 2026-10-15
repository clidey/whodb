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

package platform

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeWhereConditionPassesPlatformConditionsThrough(t *testing.T) {
	raw := map[string]any{"Type": "Atomic", "Atomic": map[string]any{"Key": "status", "Operator": "=", "Value": "open", "ColumnType": "String"}}
	got, err := NormalizeWhereCondition(raw)
	if err != nil || got["Type"] != "Atomic" {
		t.Fatalf("passthrough = %v %#v", err, got)
	}
	if got, err := NormalizeWhereCondition(nil); err != nil || got != nil {
		t.Fatalf("empty = %v %#v", err, got)
	}
}

func TestNormalizeWhereConditionTranslatesSimpleGrammar(t *testing.T) {
	got, err := NormalizeWhereCondition(map[string]any{
		"status": "open",
		"amount": map[string]any{"gte": float64(100), "lt": float64(500)},
		"_or":    []any{map[string]any{"region": "EU"}, map[string]any{"region": map[string]any{"icontains": "uk"}}},
		"note":   map[string]any{"isnull": false},
		"tier":   map[string]any{"in": []any{"gold", "silver"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var buffer strings.Builder
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(got); err != nil {
		t.Fatal(err)
	}
	text := buffer.String()
	for _, want := range []string{
		`"Type":"And"`,
		`"Key":"status","Operator":"=","Value":"open"`,
		`"Key":"amount","Operator":">=","Value":"100"`,
		`"Key":"amount","Operator":"<","Value":"500"`,
		`"Type":"Or"`,
		`"Key":"region","Operator":"ILIKE","Value":"%uk%"`,
		`"Key":"note","Operator":"IS NOT NULL"`,
		`"Key":"tier","Operator":"IN","Value":"gold,silver"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("translated condition missing %s:\n%s", want, text)
		}
	}
}

func TestNormalizeWhereConditionRejectsUnknownOperator(t *testing.T) {
	_, err := NormalizeWhereCondition(map[string]any{"amount": map[string]any{"approx": 1}})
	if err == nil || !strings.Contains(err.Error(), "unsupported where operator") {
		t.Fatalf("err = %v", err)
	}
	_, err = NormalizeWhereCondition(map[string]any{"amount": map[string]any{"between": []any{1}}})
	if err == nil || !strings.Contains(err.Error(), "exactly two") {
		t.Fatalf("between err = %v", err)
	}
}
