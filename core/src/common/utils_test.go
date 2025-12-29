package common

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

func TestValidateColumnName(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "simple", input: "user_id", expected: true},
		{name: "starts with number", input: "1field", expected: false},
		{name: "contains keyword", input: "drop_table", expected: false},
		{name: "contains dash", input: "first-name", expected: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateColumnName(tc.input); got != tc.expected {
				t.Fatalf("ValidateColumnName(%s) = %v, expected %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestSanitizeConstraintValue(t *testing.T) {
	cases := []struct {
		name         string
		input        string
		expectedOK   bool
		expectedText string
	}{
		{name: "safe value", input: "active", expectedOK: true, expectedText: "active"},
		{name: "contains drop", input: "DROP TABLE users", expectedOK: false},
		{name: "contains comment", input: "value -- comment", expectedOK: false},
		{name: "contains semicolon", input: "abc;", expectedOK: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := SanitizeConstraintValue(tc.input)
			if ok != tc.expectedOK {
				t.Fatalf("SanitizeConstraintValue(%s) ok=%v expected %v", tc.input, ok, tc.expectedOK)
			}
			if ok && got != tc.expectedText {
				t.Fatalf("expected sanitized value %s, got %s", tc.expectedText, got)
			}
		})
	}
}

func TestUtilityHelpers(t *testing.T) {
	if got := EscapeFormula("=SUM(A1)"); got != "'=SUM(A1)" {
		t.Fatalf("expected formula to be escaped, got %s", got)
	}

	if header := FormatCSVHeader("col", "text"); header != "col:text" {
		t.Fatalf("unexpected csv header: %s", header)
	}

	records := []engine.Record{
		{Key: "mode", Value: "readonly"},
	}
	if val := GetRecordValueOrDefault(records, "mode", "rw"); val != "readonly" {
		t.Fatalf("expected existing record value to be returned")
	}
	if val := GetRecordValueOrDefault(records, "missing", "fallback"); val != "fallback" {
		t.Fatalf("expected fallback to be returned when key missing")
	}

	filtered := FilterList([]int{1, 2, 3, 4}, func(v int) bool { return v%2 == 0 })
	if len(filtered) != 2 || filtered[0] != 2 || filtered[1] != 4 {
		t.Fatalf("FilterList did not filter even numbers correctly: %#v", filtered)
	}

	trueStr := "true"
	falseStr := "False"
	if !StrPtrToBool(&trueStr) {
		t.Fatalf("true string should convert to true")
	}
	if StrPtrToBool(&falseStr) {
		t.Fatalf("false string should convert to false")
	}
	if StrPtrToBool(nil) {
		t.Fatalf("nil pointer should convert to false")
	}
}
