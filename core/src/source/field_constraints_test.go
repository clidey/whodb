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

func TestNormalizeFieldConstraintMapsLegacyKeys(t *testing.T) {
	constraints := map[string]any{
		"type":              "varchar",
		"nullable":          "false",
		"is_primary":        true,
		"unique":            true,
		"auto_increment":    "true",
		"default":           "draft",
		"check_values":      []string{"draft", "published"},
		"check_min":         "1.5",
		"check_max":         float64(9),
		"length":            int64(255),
		"precision":         "10",
		"scale":             2,
		"referenced_table":  "accounts",
		"referenced_column": "id",
	}

	field := NormalizeFieldConstraint("status", constraints)

	if field.Name != "status" || field.Type != "varchar" {
		t.Fatalf("expected field identity to be normalized, got %#v", field)
	}
	if field.Nullable == nil || *field.Nullable {
		t.Fatalf("expected nullable=false, got %#v", field.Nullable)
	}
	if !field.Primary || !field.Unique || !field.Identity {
		t.Fatalf("expected primary/unique/identity flags, got %#v", field)
	}
	if field.DefaultValue == nil || *field.DefaultValue != "draft" {
		t.Fatalf("expected default value, got %#v", field.DefaultValue)
	}
	if len(field.AllowedValues) != 2 || field.AllowedValues[0] != "draft" || field.AllowedValues[1] != "published" {
		t.Fatalf("expected allowed values, got %#v", field.AllowedValues)
	}
	if field.CheckMin == nil || *field.CheckMin != 1.5 {
		t.Fatalf("expected check min, got %#v", field.CheckMin)
	}
	if field.CheckMax == nil || *field.CheckMax != 9 {
		t.Fatalf("expected check max, got %#v", field.CheckMax)
	}
	if field.Length == nil || *field.Length != 255 {
		t.Fatalf("expected length, got %#v", field.Length)
	}
	if field.Precision == nil || *field.Precision != 10 {
		t.Fatalf("expected precision, got %#v", field.Precision)
	}
	if field.Scale == nil || *field.Scale != 2 {
		t.Fatalf("expected scale, got %#v", field.Scale)
	}
	if field.ForeignKey == nil || field.ForeignKey.Table != "accounts" || field.ForeignKey.Column != "id" {
		t.Fatalf("expected foreign key, got %#v", field.ForeignKey)
	}
}

func TestNormalizeFieldConstraintsSortsFields(t *testing.T) {
	fields := NormalizeFieldConstraints(map[string]map[string]any{
		"b": {"type": "text"},
		"a": {"type": "int"},
	})

	if len(fields) != 2 || fields[0].Name != "a" || fields[1].Name != "b" {
		t.Fatalf("expected stable field order, got %#v", fields)
	}
}

func TestMergeFieldConstraintsWithColumnsFillsTypeAndMissingFields(t *testing.T) {
	fields := []FieldConstraints{
		{Name: "id", Primary: true},
	}
	columns := []Column{
		{Name: "id", Type: "integer", IsAutoIncrement: true},
		{Name: "name", Type: "varchar"},
	}

	merged := MergeFieldConstraintsWithColumns(fields, columns)

	if len(merged) != 2 {
		t.Fatalf("expected constraints for both columns, got %#v", merged)
	}
	if merged[0].Name != "id" || merged[0].Type != "integer" || !merged[0].Primary || !merged[0].Identity {
		t.Fatalf("expected id constraints to be enriched from columns, got %#v", merged[0])
	}
	if merged[1].Name != "name" || merged[1].Type != "varchar" {
		t.Fatalf("expected missing column to be added, got %#v", merged[1])
	}
}
