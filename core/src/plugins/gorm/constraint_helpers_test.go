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

package gorm_plugin

import (
	"reflect"
	"testing"
)

func TestEnsureConstraintEntry(t *testing.T) {
	constraints := make(map[string]map[string]any)

	// First call should create the entry
	entry := EnsureConstraintEntry(constraints, "column1")
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	entry["test"] = true

	// Second call should return existing entry
	entry2 := EnsureConstraintEntry(constraints, "column1")
	if entry2["test"] != true {
		t.Fatal("expected to get same entry back")
	}

	// Different column should get different entry
	entry3 := EnsureConstraintEntry(constraints, "column2")
	if entry3["test"] != nil {
		t.Fatal("expected new entry for different column")
	}
}

func TestParseINClauseValues(t *testing.T) {
	tests := []struct {
		name     string
		clause   string
		expected []string
	}{
		{
			name:     "simple IN clause",
			clause:   "status IN ('active', 'inactive', 'pending')",
			expected: []string{"active", "inactive", "pending"},
		},
		{
			name:     "MSSQL bracketed column",
			clause:   "([status] IN ('active', 'inactive', 'pending'))",
			expected: []string{"active", "inactive", "pending"},
		},
		{
			name:     "MSSQL N prefix",
			clause:   "([status] IN (N'active', N'inactive'))",
			expected: []string{"active", "inactive"},
		},
		{
			name:     "double quotes",
			clause:   `status IN ("open", "closed")`,
			expected: []string{"open", "closed"},
		},
		{
			name:     "no spaces after IN",
			clause:   "status IN('a','b','c')",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "MySQL charset prefix",
			clause:   "status IN (_utf8mb4'active', _utf8mb4'inactive')",
			expected: []string{"active", "inactive"},
		},
		{
			name:     "no IN clause",
			clause:   "status >= 0",
			expected: nil,
		},
		{
			name:     "empty IN clause",
			clause:   "status IN ()",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseINClauseValues(tt.clause)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseINClauseValues(%q) = %v, want %v", tt.clause, result, tt.expected)
			}
		})
	}
}

func TestParseValueList(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "single quotes",
			content:  "'a', 'b', 'c'",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "double quotes",
			content:  `"x", "y"`,
			expected: []string{"x", "y"},
		},
		{
			name:     "mixed spacing",
			content:  "'a','b' , 'c'",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty",
			content:  "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseValueList(tt.content)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseValueList(%q) = %v, want %v", tt.content, result, tt.expected)
			}
		})
	}
}

func TestParseMinMaxConstraints(t *testing.T) {
	tests := []struct {
		name           string
		clause         string
		expectedMin    float64
		expectedMax    float64
		expectedHasMin bool
		expectedHasMax bool
	}{
		{
			name:           "greater than or equal",
			clause:         "amount >= 0",
			expectedMin:    0,
			expectedHasMin: true,
		},
		{
			name:           "greater than",
			clause:         "quantity > 0",
			expectedMin:    1,
			expectedHasMin: true,
		},
		{
			name:           "less than or equal",
			clause:         "score <= 100",
			expectedMax:    100,
			expectedHasMax: true,
		},
		{
			name:           "less than",
			clause:         "age < 120",
			expectedMax:    119,
			expectedHasMax: true,
		},
		{
			name:           "between",
			clause:         "rating BETWEEN 1 AND 5",
			expectedMin:    1,
			expectedMax:    5,
			expectedHasMin: true,
			expectedHasMax: true,
		},
		{
			name:           "negative numbers",
			clause:         "temp >= -40",
			expectedMin:    -40,
			expectedHasMin: true,
		},
		{
			name:           "decimal numbers",
			clause:         "price >= 0.01",
			expectedMin:    0.01,
			expectedHasMin: true,
		},
		{
			name:   "no constraints",
			clause: "status IN ('a', 'b')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseMinMaxConstraints(tt.clause)
			if result.HasMin != tt.expectedHasMin {
				t.Errorf("HasMin = %v, want %v", result.HasMin, tt.expectedHasMin)
			}
			if result.HasMax != tt.expectedHasMax {
				t.Errorf("HasMax = %v, want %v", result.HasMax, tt.expectedHasMax)
			}
			if result.HasMin && result.Min != tt.expectedMin {
				t.Errorf("Min = %v, want %v", result.Min, tt.expectedMin)
			}
			if result.HasMax && result.Max != tt.expectedMax {
				t.Errorf("Max = %v, want %v", result.Max, tt.expectedMax)
			}
		})
	}
}

func TestSanitizeConstraintValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "postgres type cast",
			value:    "'active'::text",
			expected: "active",
		},
		{
			name:     "postgres array type cast",
			value:    "'values'::text[]",
			expected: "values",
		},
		{
			name:     "simple quoted",
			value:    "'value'",
			expected: "value",
		},
		{
			name:     "double quoted",
			value:    `"value"`,
			expected: "value",
		},
		{
			name:     "whitespace",
			value:    "  'value'  ",
			expected: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeConstraintValue(tt.value)
			if result != tt.expected {
				t.Errorf("SanitizeConstraintValue(%q) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}

func TestExtractColumnNameFromClause(t *testing.T) {
	tests := []struct {
		name     string
		clause   string
		expected string
	}{
		{
			name:     "simple column",
			clause:   "status IN ('a', 'b')",
			expected: "status",
		},
		{
			name:     "postgresql with type cast",
			clause:   "(status)::text = ANY (ARRAY['a'::text])",
			expected: "status",
		},
		{
			name:     "mysql backticks",
			clause:   "`status` IN ('a', 'b')",
			expected: "status",
		},
		{
			name:     "mssql brackets",
			clause:   "[status] IN ('a', 'b')",
			expected: "status",
		},
		{
			name:     "min constraint",
			clause:   "price >= 0",
			expected: "price",
		},
		{
			name:     "between constraint",
			clause:   "age BETWEEN 18 AND 120",
			expected: "age",
		},
		{
			name:     "underscore column",
			clause:   "order_status IN ('pending')",
			expected: "order_status",
		},
		{
			name:     "empty clause",
			clause:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractColumnNameFromClause(tt.clause)
			if result != tt.expected {
				t.Errorf("ExtractColumnNameFromClause(%q) = %q, want %q", tt.clause, result, tt.expected)
			}
		})
	}
}
