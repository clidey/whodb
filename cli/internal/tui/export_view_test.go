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

package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clidey/whodb/core/src/engine"
)

func setupExportViewTest(t *testing.T) (*ExportView, func()) {
	t.Helper()

	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)

	parent := NewMainModel()
	if parent.err != nil {
		t.Fatalf("Failed to create MainModel: %v", parent.err)
	}

	cleanup := func() {
		os.Setenv("HOME", origHome)
	}

	return parent.exportView, cleanup
}

func TestNewExportView(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	if v == nil {
		t.Fatal("NewExportView returned nil")
	}

	if v.selectedFormat != 0 {
		t.Errorf("Expected selectedFormat 0, got %d", v.selectedFormat)
	}

	if v.selectedDelim != 0 {
		t.Errorf("Expected selectedDelim 0, got %d", v.selectedDelim)
	}

	if v.focusIndex != 0 {
		t.Errorf("Expected focusIndex 0, got %d", v.focusIndex)
	}

	if v.overwrite {
		t.Error("Expected overwrite to be false initially")
	}

	if v.confirmOverwrite {
		t.Error("Expected confirmOverwrite to be false initially")
	}
}

func TestExportView_SetExportData(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	v.SetExportData("public", "users")

	if v.schema != "public" {
		t.Errorf("Expected schema 'public', got '%s'", v.schema)
	}

	if v.tableName != "users" {
		t.Errorf("Expected tableName 'users', got '%s'", v.tableName)
	}

	if v.filenameInput.Value() != "users" {
		t.Errorf("Expected filename 'users', got '%s'", v.filenameInput.Value())
	}

	if v.isQueryExport {
		t.Error("Expected isQueryExport to be false")
	}
}

func TestExportView_SetExportDataFromQuery(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	results := &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "id"}, {Name: "name"}},
		Rows:    [][]string{{"1", "test"}},
	}

	v.SetExportDataFromQuery(results)

	if !v.isQueryExport {
		t.Error("Expected isQueryExport to be true")
	}

	if v.queryResults != results {
		t.Error("Expected queryResults to be set")
	}

	if v.filenameInput.Value() != "query_results" {
		t.Errorf("Expected filename 'query_results', got '%s'", v.filenameInput.Value())
	}
}

func TestExportView_Navigation_Tab(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	v.SetExportData("public", "users")
	initialIndex := v.focusIndex

	// Tab moves down
	msg := tea.KeyMsg{Type: tea.KeyTab}
	v, _ = v.Update(msg)

	if v.focusIndex != initialIndex+1 {
		t.Errorf("Expected focusIndex %d after tab, got %d", initialIndex+1, v.focusIndex)
	}
}

func TestExportView_Navigation_UpDown(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	v.SetExportData("public", "users")
	v.focusIndex = 1

	// Down
	msg := tea.KeyMsg{Type: tea.KeyDown}
	v, _ = v.Update(msg)

	if v.focusIndex != 2 {
		t.Errorf("Expected focusIndex 2 after down, got %d", v.focusIndex)
	}

	// Up
	msg = tea.KeyMsg{Type: tea.KeyUp}
	v, _ = v.Update(msg)

	if v.focusIndex != 1 {
		t.Errorf("Expected focusIndex 1 after up, got %d", v.focusIndex)
	}
}

func TestExportView_FormatSelection(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	v.SetExportData("public", "users")
	v.focusIndex = 1 // Format selector
	v.selectedFormat = 0

	// Right changes format
	msg := tea.KeyMsg{Type: tea.KeyRight}
	v, _ = v.Update(msg)

	if v.selectedFormat != 1 {
		t.Errorf("Expected selectedFormat 1 after right, got %d", v.selectedFormat)
	}

	// Wrap around
	v, _ = v.Update(msg)
	if v.selectedFormat != 0 {
		t.Errorf("Expected selectedFormat 0 after wrap, got %d", v.selectedFormat)
	}

	// Left wraps to end
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	v, _ = v.Update(msg)
	if v.selectedFormat != len(exportFormats)-1 {
		t.Errorf("Expected selectedFormat %d after left wrap, got %d", len(exportFormats)-1, v.selectedFormat)
	}
}

func TestExportView_DelimiterSelection(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	v.SetExportData("public", "users")
	v.selectedFormat = 0 // CSV - has delimiter
	v.focusIndex = 2     // Delimiter selector
	v.selectedDelim = 0

	// Right changes delimiter
	msg := tea.KeyMsg{Type: tea.KeyRight}
	v, _ = v.Update(msg)

	if v.selectedDelim != 1 {
		t.Errorf("Expected selectedDelim 1 after right, got %d", v.selectedDelim)
	}
}

func TestExportView_HasDelimiter(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	// CSV has delimiter
	v.selectedFormat = 0
	if !v.hasDelimiter() {
		t.Error("Expected CSV to have delimiter")
	}

	// Excel doesn't have delimiter
	v.selectedFormat = 1
	if v.hasDelimiter() {
		t.Error("Expected Excel to not have delimiter")
	}
}

func TestExportView_OverwriteToggle(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	v.SetExportData("public", "users")
	v.focusIndex = v.overwriteIndex()
	v.overwrite = false

	// Left/right toggles
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	v, _ = v.Update(msg)

	if !v.overwrite {
		t.Error("Expected overwrite to be true after toggle")
	}

	msg = tea.KeyMsg{Type: tea.KeyRight}
	v, _ = v.Update(msg)

	if v.overwrite {
		t.Error("Expected overwrite to be false after second toggle")
	}
}

func TestExportView_Escape(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	v.exporting = false

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.parent.mode != ViewResults {
		t.Errorf("Expected mode ViewResults after Esc, got %v", v.parent.mode)
	}
}

func TestExportView_ConfirmOverwrite_Cancel(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	v.confirmOverwrite = true
	v.confirmIndex = 0

	// Esc cancels
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.confirmOverwrite {
		t.Error("Expected confirmOverwrite to be false after Esc")
	}
}

func TestExportView_ConfirmOverwrite_Navigation(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	v.confirmOverwrite = true
	v.confirmIndex = 0

	// Left/right switches between Yes/No
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	v, _ = v.Update(msg)

	if v.confirmIndex != 1 {
		t.Errorf("Expected confirmIndex 1 after left, got %d", v.confirmIndex)
	}

	msg = tea.KeyMsg{Type: tea.KeyRight}
	v, _ = v.Update(msg)

	if v.confirmIndex != 0 {
		t.Errorf("Expected confirmIndex 0 after right, got %d", v.confirmIndex)
	}
}

func TestExportView_View_Normal(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	v.SetExportData("public", "users")

	view := v.View()

	if !strings.Contains(view, "Export Data") {
		t.Error("Expected title 'Export Data'")
	}

	if !strings.Contains(view, "public.users") {
		t.Error("Expected table name in view")
	}

	if !strings.Contains(view, "Filename") {
		t.Error("Expected 'Filename' label")
	}

	if !strings.Contains(view, "Format") {
		t.Error("Expected 'Format' label")
	}

	if !strings.Contains(view, "Export") {
		t.Error("Expected 'Export' button")
	}

	if !strings.Contains(view, "Cancel") {
		t.Error("Expected 'Cancel' button")
	}
}

func TestExportView_View_QueryExport(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	results := &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "id"}},
		Rows:    [][]string{{"1"}},
	}
	v.SetExportDataFromQuery(results)

	view := v.View()

	if !strings.Contains(view, "query results") {
		t.Error("Expected 'query results' text for query export")
	}
}

func TestExportView_View_Exporting(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	v.exporting = true

	view := v.View()

	if !strings.Contains(view, "Exporting") {
		t.Error("Expected 'Exporting' text")
	}
}

func TestExportView_View_Success(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	v.exportSuccess = true
	v.savedFilePath = "/tmp/test.csv"

	view := v.View()

	if !strings.Contains(view, "Export completed successfully") {
		t.Error("Expected success message")
	}

	if !strings.Contains(view, "/tmp/test.csv") {
		t.Error("Expected file path in success message")
	}
}

func TestExportView_View_Error(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	v.exportError = os.ErrPermission

	view := v.View()

	if !strings.Contains(view, "Export failed") {
		t.Error("Expected error message")
	}
}

func TestExportView_View_ConfirmOverwrite(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	v.confirmOverwrite = true
	v.pendingPath = "/tmp/existing.csv"

	view := v.View()

	if !strings.Contains(view, "Confirm Overwrite") {
		t.Error("Expected 'Confirm Overwrite' title")
	}

	if !strings.Contains(view, "existing.csv") {
		t.Error("Expected file path in confirmation")
	}

	if !strings.Contains(view, "Yes") && !strings.Contains(view, "No") {
		t.Error("Expected Yes/No options")
	}
}

func TestExportView_MouseScroll(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	v.SetExportData("public", "users")
	v.focusIndex = 1

	// Mouse wheel down
	msg := tea.MouseMsg{Button: tea.MouseButtonWheelDown}
	v, _ = v.Update(msg)

	if v.focusIndex != 2 {
		t.Errorf("Expected focusIndex 2 after wheel down, got %d", v.focusIndex)
	}

	// Mouse wheel up
	msg = tea.MouseMsg{Button: tea.MouseButtonWheelUp}
	v, _ = v.Update(msg)

	if v.focusIndex != 1 {
		t.Errorf("Expected focusIndex 1 after wheel up, got %d", v.focusIndex)
	}
}

func TestExportView_IndexHelpers(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	// CSV format - has delimiter field
	v.selectedFormat = 0
	if v.overwriteIndex() != 3 {
		t.Errorf("Expected overwriteIndex 3 for CSV, got %d", v.overwriteIndex())
	}
	if v.exportButtonIndex() != 4 {
		t.Errorf("Expected exportButtonIndex 4 for CSV, got %d", v.exportButtonIndex())
	}
	if v.cancelButtonIndex() != 5 {
		t.Errorf("Expected cancelButtonIndex 5 for CSV, got %d", v.cancelButtonIndex())
	}

	// Excel format - no delimiter field
	v.selectedFormat = 1
	if v.overwriteIndex() != 2 {
		t.Errorf("Expected overwriteIndex 2 for Excel, got %d", v.overwriteIndex())
	}
	if v.exportButtonIndex() != 3 {
		t.Errorf("Expected exportButtonIndex 3 for Excel, got %d", v.exportButtonIndex())
	}
	if v.cancelButtonIndex() != 4 {
		t.Errorf("Expected cancelButtonIndex 4 for Excel, got %d", v.cancelButtonIndex())
	}
}

func TestResolveExportPath(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		input       string
		format      string
		overwrite   bool
		setup       func()
		expectErr   bool
		checkResult func(path string, willOverwrite bool) bool
	}{
		{
			name:      "simple filename adds extension",
			input:     "test",
			format:    "CSV",
			overwrite: false,
			checkResult: func(path string, willOverwrite bool) bool {
				return strings.HasSuffix(path, ".csv") && !willOverwrite
			},
		},
		{
			name:      "excel adds xlsx extension",
			input:     "test",
			format:    "Excel",
			overwrite: false,
			checkResult: func(path string, willOverwrite bool) bool {
				return strings.HasSuffix(path, ".xlsx") && !willOverwrite
			},
		},
		{
			name:      "preserves existing extension",
			input:     "test.csv",
			format:    "CSV",
			overwrite: false,
			checkResult: func(path string, willOverwrite bool) bool {
				return strings.HasSuffix(path, ".csv")
			},
		},
		{
			name:      "empty input returns error",
			input:     "",
			format:    "CSV",
			overwrite: false,
			expectErr: true,
		},
		{
			name:      "whitespace only returns error",
			input:     "   ",
			format:    "CSV",
			overwrite: false,
			expectErr: true,
		},
		{
			name:      "existing file with overwrite true",
			input:     filepath.Join(tempDir, "existing"),
			format:    "CSV",
			overwrite: true,
			setup: func() {
				os.WriteFile(filepath.Join(tempDir, "existing.csv"), []byte("test"), 0644)
			},
			checkResult: func(path string, willOverwrite bool) bool {
				return willOverwrite && strings.HasSuffix(path, "existing.csv")
			},
		},
		{
			name:      "existing file with overwrite false gets suffix",
			input:     filepath.Join(tempDir, "existing2"),
			format:    "CSV",
			overwrite: false,
			setup: func() {
				os.WriteFile(filepath.Join(tempDir, "existing2.csv"), []byte("test"), 0644)
			},
			checkResult: func(path string, willOverwrite bool) bool {
				return !willOverwrite && strings.Contains(path, "existing2_1.csv")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			path, willOverwrite, err := resolveExportPath(tt.input, tt.format, tt.overwrite)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkResult != nil && !tt.checkResult(path, willOverwrite) {
				t.Errorf("Check failed for path='%s', willOverwrite=%v", path, willOverwrite)
			}
		})
	}
}

func TestExportView_ExportResultMsg(t *testing.T) {
	v, cleanup := setupExportViewTest(t)
	defer cleanup()

	v.exporting = true

	// Success message
	msg := exportResultMsg{success: true, err: nil}
	v, _ = v.Update(msg)

	if v.exporting {
		t.Error("Expected exporting to be false after result")
	}

	if !v.exportSuccess {
		t.Error("Expected exportSuccess to be true")
	}

	if v.exportError != nil {
		t.Error("Expected exportError to be nil")
	}

	// Error message
	v.exporting = true
	v.exportSuccess = false
	msg = exportResultMsg{success: false, err: os.ErrPermission}
	v, _ = v.Update(msg)

	if v.exportSuccess {
		t.Error("Expected exportSuccess to be false")
	}

	if v.exportError == nil {
		t.Error("Expected exportError to be set")
	}
}
