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

package testharness

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// AssertContains checks if output contains substring.
func AssertContains(t *testing.T, output, substring string) {
	t.Helper()

	if !strings.Contains(output, substring) {
		t.Errorf("Expected output to contain %q, but it doesn't.\nOutput: %s", substring, output)
	}
}

// AssertNotContains checks if output does NOT contain substring.
func AssertNotContains(t *testing.T, output, substring string) {
	t.Helper()

	if strings.Contains(output, substring) {
		t.Errorf("Expected output to NOT contain %q, but it does.\nOutput: %s", substring, output)
	}
}

// AssertJSONContains verifies JSON output contains expected key.
func AssertJSONContains(t *testing.T, output, key string) {
	t.Helper()

	var data any
	if err := json.Unmarshal([]byte(output), &data); err != nil {
		t.Fatalf("Failed to parse JSON: %v\nOutput: %s", err, output)
	}

	// Check if it's an array and any item contains the key
	if arr, ok := data.([]any); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				if _, exists := m[key]; exists {
					return
				}
			}
		}
		t.Errorf("Expected JSON array to contain objects with key %q, but none found.\nOutput: %s", key, output)
		return
	}

	// Check if it's an object with the key
	if m, ok := data.(map[string]any); ok {
		if _, exists := m[key]; !exists {
			t.Errorf("Expected JSON object to contain key %q, but it doesn't.\nOutput: %s", key, output)
		}
		return
	}

	t.Errorf("Expected JSON to be an array or object, got something else.\nOutput: %s", output)
}

// AssertJSONArrayLength parses JSON array and checks length.
func AssertJSONArrayLength(t *testing.T, output string, expected int) {
	t.Helper()

	var arr []any
	if err := json.Unmarshal([]byte(output), &arr); err != nil {
		t.Fatalf("Failed to parse JSON array: %v\nOutput: %s", err, output)
	}

	if len(arr) != expected {
		t.Errorf("Expected JSON array length %d, got %d.\nOutput: %s", expected, len(arr), output)
	}
}

// AssertJSONArrayMinLength parses JSON array and checks minimum length.
func AssertJSONArrayMinLength(t *testing.T, output string, minLength int) {
	t.Helper()

	var arr []any
	if err := json.Unmarshal([]byte(output), &arr); err != nil {
		t.Fatalf("Failed to parse JSON array: %v\nOutput: %s", err, output)
	}

	if len(arr) < minLength {
		t.Errorf("Expected JSON array length >= %d, got %d.\nOutput: %s", minLength, len(arr), output)
	}
}

// AssertJSONArrayContainsValue checks if any element in the JSON array has the given key-value pair.
func AssertJSONArrayContainsValue(t *testing.T, output, key, value string) {
	t.Helper()

	var arr []map[string]any
	if err := json.Unmarshal([]byte(output), &arr); err != nil {
		t.Fatalf("Failed to parse JSON array: %v\nOutput: %s", err, output)
	}

	for _, item := range arr {
		if v, ok := item[key]; ok {
			// Handle string values
			if str, ok := v.(string); ok && str == value {
				return
			}
			// Handle numeric values that might be represented as float64
			if str, ok := v.(float64); ok {
				if strings.Contains(value, ".") {
					// Float comparison
					if v == str {
						return
					}
				}
			}
		}
	}

	t.Errorf("Expected JSON array to contain an object where %q = %q, but none found.\nOutput: %s", key, value, output)
}

// AssertFileExists checks if a file exists at the given path.
func AssertFileExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Expected file to exist at %s, but it doesn't", path)
	}
}

// AssertFileNotEmpty checks if a file exists and is not empty.
func AssertFileNotEmpty(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Errorf("Expected file to exist at %s, but it doesn't", path)
		return
	}
	if err != nil {
		t.Fatalf("Failed to stat file %s: %v", path, err)
	}
	if info.Size() == 0 {
		t.Errorf("Expected file at %s to be non-empty, but it's empty", path)
	}
}

// AssertFileContains checks if a file contains the given substring.
func AssertFileContains(t *testing.T, path, substring string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	if !strings.Contains(string(content), substring) {
		t.Errorf("Expected file %s to contain %q, but it doesn't.\nContent: %s", path, substring, string(content))
	}
}
