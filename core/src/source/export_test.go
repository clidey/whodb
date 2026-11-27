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
package source

import (
	"reflect"
	"testing"
)

func TestFreeTextColumnsFollowDeclaredCategories(t *testing.T) {
	metadata := &TypeSessionMetadata{
		AliasMap: map[string]string{"INT4": "INTEGER"},
		TypeDefinitions: []TypeDefinition{
			{ID: "INTEGER", Category: TypeCategoryNumeric},
			{ID: "Int32", Category: TypeCategoryNumeric},
			{ID: "TIMESTAMP", Category: TypeCategoryDatetime},
			{ID: "BOOLEAN", Category: TypeCategoryBoolean},
			{ID: "TEXT", Category: TypeCategoryText},
			{ID: "String", Category: TypeCategoryText},
			{ID: "JSON", Category: TypeCategoryJSON},
		},
	}

	cases := map[string]bool{
		"INTEGER":                          false,
		"int4":                             false,
		"TIMESTAMP(6)":                     false,
		"BOOLEAN":                          false,
		"Nullable(Int32)":                  false,
		"TEXT":                             true,
		"JSON":                             true,
		"LowCardinality(Nullable(String))": true,
		"Map(String, Int32)":               true,
		"SOMETHING_ELSE":                   true,
		"":                                 true,
	}
	for typeName, want := range cases {
		if got := FreeTextColumns(metadata, []Column{{Type: typeName}})[0]; got != want {
			t.Errorf("FreeTextColumns(%q) = %v, want %v", typeName, got, want)
		}
	}

	if got := FreeTextColumns(nil, []Column{{Type: "INTEGER"}}); !reflect.DeepEqual(got, []bool{true}) {
		t.Fatalf("expected every column to be free text without metadata, got %v", got)
	}

	loose := *metadata
	loose.LooseColumnTypes = true
	if got := FreeTextColumns(&loose, []Column{{Type: "INTEGER"}}); !reflect.DeepEqual(got, []bool{true}) {
		t.Fatalf("expected a loosely typed source to trust no declared type, got %v", got)
	}
}

func TestExportSelectionColumnNames(t *testing.T) {
	rows := []map[string]any{{"b": 1, "a": 2}, {"c": 3}}

	declared := &ExportSelection{Columns: []Column{{Name: "c"}, {Name: "a"}}, Rows: rows}
	if got := declared.ColumnNames(); !reflect.DeepEqual(got, []string{"c", "a"}) {
		t.Fatalf("expected declared column order, got %v", got)
	}

	undeclared := &ExportSelection{Rows: rows}
	if got := undeclared.ColumnNames(); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("expected sorted row keys, got %v", got)
	}
}
