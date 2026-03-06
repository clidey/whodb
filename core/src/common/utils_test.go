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

package common

import (
	"strings"
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

	if header := FormatCSVHeader("col", "text"); header != "col" {
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

func TestResolveLocalURL(t *testing.T) {
	// Non-localhost URLs should always pass through unchanged
	nonLocal := []string{
		"https://api.openai.com/v1",
		"http://my-server:8080/api",
		"http://192.168.1.100:1234/v1",
		"https://example.com",
	}
	for _, u := range nonLocal {
		if got := ResolveLocalURL(u); got != u {
			t.Fatalf("ResolveLocalURL(%q) = %q, want unchanged", u, got)
		}
	}

	// Invalid URLs should pass through unchanged
	if got := ResolveLocalURL("://bad"); got != "://bad" {
		t.Fatalf("ResolveLocalURL with invalid URL should return unchanged, got %q", got)
	}

	// Localhost URLs: verify that the host is rewritten in Docker/WSL2,
	// or unchanged when running natively.
	localhostCases := []struct {
		name  string
		input string
	}{
		{"with port and path", "http://localhost:1234/v1"},
		{"without port", "http://localhost/api"},
		{"127.0.0.1 with port", "http://127.0.0.1:8080/v1/chat"},
		{"127.0.0.1 without port", "http://127.0.0.1/test"},
	}

	inDocker := IsRunningInsideDocker()
	inWSL2 := IsRunningInsideWSL2()

	for _, tc := range localhostCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveLocalURL(tc.input)
			if inDocker {
				if !strings.Contains(got, "host.docker.internal") {
					t.Fatalf("expected Docker rewrite, got %q", got)
				}
			} else if inWSL2 {
				if wslHost := GetWSL2WindowsHost(); wslHost != "" {
					if !strings.Contains(got, wslHost) {
						t.Fatalf("expected WSL2 rewrite to %s, got %q", wslHost, got)
					}
				}
			} else {
				if got != tc.input {
					t.Fatalf("outside Docker/WSL2, expected unchanged, got %q", got)
				}
			}
		})
	}

	// Port preservation: the path and port must survive the rewrite
	got := ResolveLocalURL("http://localhost:1234/v1/chat/completions")
	if !strings.Contains(got, ":1234/v1/chat/completions") {
		t.Fatalf("port and path not preserved: %q", got)
	}
}
