/*
 * Copyright 2025 Clidey, Inc.
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

package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// --- ParseFormat Tests ---

func TestParseFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Format
		wantErr bool
	}{
		{name: "auto", input: "auto", want: FormatAuto},
		{name: "empty_is_auto", input: "", want: FormatAuto},
		{name: "table", input: "table", want: FormatTable},
		{name: "plain", input: "plain", want: FormatPlain},
		{name: "json", input: "json", want: FormatJSON},
		{name: "csv", input: "csv", want: FormatCSV},
		{name: "uppercase_JSON", input: "JSON", want: FormatJSON},
		{name: "mixed_case_Table", input: "TaBlE", want: FormatTable},
		{name: "invalid", input: "xml", wantErr: true},
		{name: "invalid_number", input: "123", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFormat(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseFormat(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseFormat(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseFormat_ErrorMessage(t *testing.T) {
	_, err := ParseFormat("invalid")
	if err == nil {
		t.Fatal("Expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "unknown format") {
		t.Errorf("Error should mention 'unknown format', got: %v", err)
	}
	if !strings.Contains(err.Error(), "valid:") {
		t.Errorf("Error should list valid options, got: %v", err)
	}
}

// --- Writer Creation Tests ---

func TestNew_Defaults(t *testing.T) {
	w := New()
	if w.format != FormatAuto {
		t.Errorf("Default format = %v, want %v", w.format, FormatAuto)
	}
	if w.quiet {
		t.Error("Default quiet should be false")
	}
}

func TestNew_WithOptions(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	w := New(
		WithFormat(FormatJSON),
		WithQuiet(true),
		WithOutput(&out),
		WithErrorOutput(&errOut),
	)

	if w.format != FormatJSON {
		t.Errorf("Format = %v, want %v", w.format, FormatJSON)
	}
	if !w.quiet {
		t.Error("quiet should be true")
	}
}

func TestWriter_IsTTY_NonTTY(t *testing.T) {
	var buf bytes.Buffer
	w := New(WithOutput(&buf))
	if w.IsTTY() {
		t.Error("bytes.Buffer should not be detected as TTY")
	}
}

func TestWriter_ColorEnabled_NonTTY(t *testing.T) {
	var buf bytes.Buffer
	w := New(WithOutput(&buf))
	// Non-TTY outputs should not have color enabled
	if w.ColorEnabled() {
		t.Error("ColorEnabled should be false for non-TTY")
	}
}

// --- Format Resolution Tests ---

func TestWriter_ResolveFormat_Explicit(t *testing.T) {
	var buf bytes.Buffer
	w := New(WithOutput(&buf), WithFormat(FormatJSON))
	if w.resolveFormat() != FormatJSON {
		t.Error("Explicit format should be returned as-is")
	}
}

func TestWriter_ResolveFormat_Auto_NonTTY(t *testing.T) {
	var buf bytes.Buffer
	w := New(WithOutput(&buf), WithFormat(FormatAuto))
	// Non-TTY with auto should resolve to plain
	if w.resolveFormat() != FormatPlain {
		t.Errorf("Auto format on non-TTY = %v, want %v", w.resolveFormat(), FormatPlain)
	}
}

// --- JSON Output Tests ---

func TestWriter_WriteJSON(t *testing.T) {
	var buf bytes.Buffer
	w := New(WithOutput(&buf), WithFormat(FormatJSON))

	result := &QueryResult{
		Columns: []Column{{Name: "id"}, {Name: "name"}},
		Rows:    [][]any{{1, "Alice"}, {2, "Bob"}},
	}

	err := w.WriteQueryResult(result)
	if err != nil {
		t.Fatalf("WriteQueryResult error: %v", err)
	}

	// Parse the output
	var output []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	if len(output) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(output))
	}

	if output[0]["name"] != "Alice" {
		t.Errorf("First row name = %v, want Alice", output[0]["name"])
	}
}

func TestWriter_WriteJSON_EmptyResult(t *testing.T) {
	var buf bytes.Buffer
	w := New(WithOutput(&buf), WithFormat(FormatJSON))

	result := &QueryResult{
		Columns: []Column{{Name: "id"}},
		Rows:    [][]any{},
	}

	err := w.WriteQueryResult(result)
	if err != nil {
		t.Fatalf("WriteQueryResult error: %v", err)
	}

	var output []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	if len(output) != 0 {
		t.Errorf("Expected empty array, got %d rows", len(output))
	}
}

func TestWriter_WriteJSON_ColumnMismatch(t *testing.T) {
	var buf bytes.Buffer
	w := New(WithOutput(&buf), WithFormat(FormatJSON))

	// Row has more values than columns
	result := &QueryResult{
		Columns: []Column{{Name: "id"}},
		Rows:    [][]any{{1, "extra", "values"}},
	}

	err := w.WriteQueryResult(result)
	if err != nil {
		t.Fatalf("WriteQueryResult error: %v", err)
	}

	// Should only include the first column
	var output []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	if _, exists := output[0]["extra"]; exists {
		t.Error("Extra values should not be included")
	}
}

// --- CSV Output Tests ---

func TestWriter_WriteCSV(t *testing.T) {
	var buf bytes.Buffer
	w := New(WithOutput(&buf), WithFormat(FormatCSV))

	result := &QueryResult{
		Columns: []Column{{Name: "id"}, {Name: "name"}},
		Rows:    [][]any{{1, "Alice"}, {2, "Bob"}},
	}

	err := w.WriteQueryResult(result)
	if err != nil {
		t.Fatalf("WriteQueryResult error: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines (header + 2 rows), got %d", len(lines))
	}

	if lines[0] != "id,name" {
		t.Errorf("Header = %q, want %q", lines[0], "id,name")
	}

	if lines[1] != "1,Alice" {
		t.Errorf("Row 1 = %q, want %q", lines[1], "1,Alice")
	}
}

func TestWriter_WriteCSV_WithCommas(t *testing.T) {
	var buf bytes.Buffer
	w := New(WithOutput(&buf), WithFormat(FormatCSV))

	result := &QueryResult{
		Columns: []Column{{Name: "data"}},
		Rows:    [][]any{{"value,with,commas"}},
	}

	err := w.WriteQueryResult(result)
	if err != nil {
		t.Fatalf("WriteQueryResult error: %v", err)
	}

	output := buf.String()
	// CSV should quote fields with commas
	if !strings.Contains(output, `"value,with,commas"`) {
		t.Errorf("CSV should quote commas, got: %s", output)
	}
}

func TestWriter_WriteCSV_EmptyResult(t *testing.T) {
	var buf bytes.Buffer
	w := New(WithOutput(&buf), WithFormat(FormatCSV))

	result := &QueryResult{
		Columns: []Column{{Name: "id"}, {Name: "name"}},
		Rows:    [][]any{},
	}

	err := w.WriteQueryResult(result)
	if err != nil {
		t.Fatalf("WriteQueryResult error: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have header only
	if len(lines) != 1 {
		t.Errorf("Expected 1 line (header only), got %d", len(lines))
	}
}

// --- Plain Output Tests ---

func TestWriter_WritePlain(t *testing.T) {
	var buf bytes.Buffer
	w := New(WithOutput(&buf), WithFormat(FormatPlain))

	result := &QueryResult{
		Columns: []Column{{Name: "id"}, {Name: "name"}},
		Rows:    [][]any{{1, "Alice"}, {2, "Bob"}},
	}

	err := w.WriteQueryResult(result)
	if err != nil {
		t.Fatalf("WriteQueryResult error: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	// Tab-separated
	if !strings.Contains(lines[0], "\t") {
		t.Error("Plain format should use tabs")
	}

	if !strings.Contains(lines[0], "id") || !strings.Contains(lines[0], "name") {
		t.Errorf("Header should contain column names, got: %s", lines[0])
	}
}

// --- Table Output Tests ---

func TestWriter_WriteTable(t *testing.T) {
	var buf bytes.Buffer
	w := New(WithOutput(&buf), WithFormat(FormatTable))

	result := &QueryResult{
		Columns: []Column{{Name: "id"}, {Name: "name"}},
		Rows:    [][]any{{1, "Alice"}, {2, "Bob"}},
	}

	err := w.WriteQueryResult(result)
	if err != nil {
		t.Fatalf("WriteQueryResult error: %v", err)
	}

	output := buf.String()

	// Should contain column names
	if !strings.Contains(output, "id") || !strings.Contains(output, "name") {
		t.Error("Table should contain column names")
	}

	// Should contain separator line
	if !strings.Contains(output, "───") {
		t.Error("Table should contain separator")
	}

	// Should contain data
	if !strings.Contains(output, "Alice") || !strings.Contains(output, "Bob") {
		t.Error("Table should contain data values")
	}
}

func TestWriter_WriteTable_EmptyResult(t *testing.T) {
	var buf bytes.Buffer
	w := New(WithOutput(&buf), WithFormat(FormatTable))

	result := &QueryResult{
		Columns: []Column{{Name: "id"}},
		Rows:    [][]any{},
	}

	err := w.WriteQueryResult(result)
	if err != nil {
		t.Fatalf("WriteQueryResult error: %v", err)
	}

	output := buf.String()
	// Should still have header and separator
	if !strings.Contains(output, "id") {
		t.Error("Empty table should still show header")
	}
}

// --- Message Output Tests ---

func TestWriter_Info(t *testing.T) {
	var errBuf bytes.Buffer
	w := New(WithErrorOutput(&errBuf))

	w.Info("test message %d", 42)

	output := errBuf.String()
	if !strings.Contains(output, "test message 42") {
		t.Errorf("Info output = %q, want to contain 'test message 42'", output)
	}
}

func TestWriter_Info_Quiet(t *testing.T) {
	var errBuf bytes.Buffer
	w := New(WithErrorOutput(&errBuf), WithQuiet(true))

	w.Info("should not appear")

	if errBuf.Len() != 0 {
		t.Errorf("Info with quiet=true should not output, got: %s", errBuf.String())
	}
}

func TestWriter_Error(t *testing.T) {
	var errBuf bytes.Buffer
	w := New(WithErrorOutput(&errBuf))

	w.Error("something went wrong: %s", "details")

	output := errBuf.String()
	if !strings.Contains(output, "Error:") {
		t.Error("Error output should contain 'Error:' prefix")
	}
	if !strings.Contains(output, "something went wrong: details") {
		t.Errorf("Error output = %q", output)
	}
}

func TestWriter_Success(t *testing.T) {
	var errBuf bytes.Buffer
	w := New(WithErrorOutput(&errBuf))

	w.Success("operation completed")

	output := errBuf.String()
	if !strings.Contains(output, "✓") {
		t.Error("Success output should contain checkmark")
	}
	if !strings.Contains(output, "operation completed") {
		t.Errorf("Success output = %q", output)
	}
}

func TestWriter_Success_Quiet(t *testing.T) {
	var errBuf bytes.Buffer
	w := New(WithErrorOutput(&errBuf), WithQuiet(true))

	w.Success("should not appear")

	if errBuf.Len() != 0 {
		t.Errorf("Success with quiet=true should not output, got: %s", errBuf.String())
	}
}

// --- Format Constant Tests ---

func TestFormatConstants(t *testing.T) {
	// Ensure format constants have expected string values
	if FormatAuto != "auto" {
		t.Errorf("FormatAuto = %q, want 'auto'", FormatAuto)
	}
	if FormatTable != "table" {
		t.Errorf("FormatTable = %q, want 'table'", FormatTable)
	}
	if FormatPlain != "plain" {
		t.Errorf("FormatPlain = %q, want 'plain'", FormatPlain)
	}
	if FormatJSON != "json" {
		t.Errorf("FormatJSON = %q, want 'json'", FormatJSON)
	}
	if FormatCSV != "csv" {
		t.Errorf("FormatCSV = %q, want 'csv'", FormatCSV)
	}
}

// --- QueryResult and Column Type Tests ---

func TestQueryResult_Structure(t *testing.T) {
	result := QueryResult{
		Columns: []Column{
			{Name: "id", Type: "integer"},
			{Name: "name", Type: "text"},
		},
		Rows: [][]any{{1, "test"}},
	}

	if len(result.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(result.Columns))
	}
	if result.Columns[0].Name != "id" {
		t.Errorf("Column 0 name = %q, want 'id'", result.Columns[0].Name)
	}
	if result.Columns[0].Type != "integer" {
		t.Errorf("Column 0 type = %q, want 'integer'", result.Columns[0].Type)
	}
}

func TestColumn_TypeOptional(t *testing.T) {
	col := Column{Name: "data"}
	if col.Type != "" {
		t.Error("Type should be empty when not specified")
	}
}

// --- Edge Cases ---

func TestWriter_WriteQueryResult_NilValues(t *testing.T) {
	var buf bytes.Buffer
	w := New(WithOutput(&buf), WithFormat(FormatJSON))

	result := &QueryResult{
		Columns: []Column{{Name: "value"}},
		Rows:    [][]any{{nil}},
	}

	err := w.WriteQueryResult(result)
	if err != nil {
		t.Fatalf("WriteQueryResult error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "null") {
		t.Errorf("nil should serialize as null in JSON, got: %s", output)
	}
}

func TestWriter_WriteQueryResult_SpecialCharacters(t *testing.T) {
	var buf bytes.Buffer
	w := New(WithOutput(&buf), WithFormat(FormatJSON))

	result := &QueryResult{
		Columns: []Column{{Name: "text"}},
		Rows:    [][]any{{"line1\nline2\ttab\"quote"}},
	}

	err := w.WriteQueryResult(result)
	if err != nil {
		t.Fatalf("WriteQueryResult error: %v", err)
	}

	// Should be valid JSON
	var output []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Output should be valid JSON: %v", err)
	}
}

func TestWriter_WriteQueryResult_UnicodeCharacters(t *testing.T) {
	var buf bytes.Buffer
	w := New(WithOutput(&buf), WithFormat(FormatPlain))

	result := &QueryResult{
		Columns: []Column{{Name: "text"}},
		Rows:    [][]any{{"日本語"}, {"émoji 🎉"}},
	}

	err := w.WriteQueryResult(result)
	if err != nil {
		t.Fatalf("WriteQueryResult error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "日本語") {
		t.Error("Should preserve Japanese characters")
	}
	if !strings.Contains(output, "🎉") {
		t.Error("Should preserve emoji")
	}
}

func TestWriter_WriteQueryResult_LargeNumbers(t *testing.T) {
	var buf bytes.Buffer
	w := New(WithOutput(&buf), WithFormat(FormatJSON))

	result := &QueryResult{
		Columns: []Column{{Name: "big"}},
		Rows:    [][]any{{int64(9223372036854775807)}},
	}

	err := w.WriteQueryResult(result)
	if err != nil {
		t.Fatalf("WriteQueryResult error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "9223372036854775807") {
		t.Errorf("Large number not preserved: %s", output)
	}
}
