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

import "testing"

func TestRecordsToObjectDefinitionNormalizesConstraintKeys(t *testing.T) {
	definition := RecordsToObjectDefinition("users", []Record{{
		Key:   "id",
		Value: "INTEGER",
		Extra: map[string]string{
			"Primary":           "true",
			"Nullable":          "false",
			"is_primary":        "true",
			"auto_increment":    "true",
			"default_value":     "1",
			"referenced_table":  "accounts",
			"referenced_column": "id",
			"check_values":      "1,2",
			"check_min":         "1",
			"check_max":         "2",
		},
	}})

	if definition.Name != "users" || len(definition.Columns) != 1 {
		t.Fatalf("unexpected definition: %#v", definition)
	}
	column := definition.Columns[0]
	if !column.Primary || !column.Identity || column.Nullable == nil || *column.Nullable {
		t.Fatalf("expected primary identity not-null column, got %#v", column)
	}
	if column.DefaultValue == nil || *column.DefaultValue != "1" {
		t.Fatalf("expected default value to be normalized, got %#v", column.DefaultValue)
	}
	if column.ForeignKey == nil || column.ForeignKey.Table != "accounts" || column.ForeignKey.Column != "id" {
		t.Fatalf("expected foreign key to be normalized, got %#v", column.ForeignKey)
	}
	if len(column.CheckValues) != 2 || column.CheckValues[0] != "1" || column.CheckValues[1] != "2" {
		t.Fatalf("expected check values to be split, got %#v", column.CheckValues)
	}
	if column.CheckMin == nil || *column.CheckMin != 1 || column.CheckMax == nil || *column.CheckMax != 2 {
		t.Fatalf("expected min/max checks to be parsed, got %#v", column)
	}
}
