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
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clidey/whodb/core/src/engine"
)

func setupResultsViewTest(t *testing.T) (*ResultsView, func()) {
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

	return parent.resultsView, cleanup
}

func TestNewResultsView(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	if v == nil {
		t.Fatal("NewResultsView returned nil")
	}

	if v.currentPage != 0 {
		t.Errorf("Expected currentPage 0, got %d", v.currentPage)
	}

	if v.pageSize != 50 {
		t.Errorf("Expected default pageSize 50, got %d", v.pageSize)
	}

	if v.columnOffset != 0 {
		t.Errorf("Expected columnOffset 0, got %d", v.columnOffset)
	}

	if v.maxColumns != 10 {
		t.Errorf("Expected maxColumns 10, got %d", v.maxColumns)
	}

	if v.results != nil {
		t.Error("Expected results to be nil initially")
	}
}

func TestResultsView_SetResults(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	results := &engine.GetRowsResult{
		Columns: []engine.Column{
			{Name: "id", Type: "integer"},
			{Name: "name", Type: "text"},
		},
		Rows: [][]string{
			{"1", "Alice"},
			{"2", "Bob"},
		},
		TotalCount: 2,
	}

	v.SetResults(results, "SELECT * FROM users")

	if v.results != results {
		t.Error("Expected results to be set")
	}

	if v.query != "SELECT * FROM users" {
		t.Errorf("Expected query 'SELECT * FROM users', got '%s'", v.query)
	}

	if v.currentPage != 0 {
		t.Errorf("Expected currentPage to reset to 0, got %d", v.currentPage)
	}

	if v.columnOffset != 0 {
		t.Errorf("Expected columnOffset to reset to 0, got %d", v.columnOffset)
	}

	if v.totalRows != 2 {
		t.Errorf("Expected totalRows 2, got %d", v.totalRows)
	}
}

func TestResultsView_SetResults_ClearsTableContext(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	// Set table context first
	v.schema = "public"
	v.tableName = "users"

	results := &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "id", Type: "integer"}},
		Rows:    [][]string{{"1"}},
	}

	v.SetResults(results, "SELECT 1")

	// Schema and tableName should be cleared
	if v.schema != "" {
		t.Errorf("Expected schema to be cleared, got '%s'", v.schema)
	}

	if v.tableName != "" {
		t.Errorf("Expected tableName to be cleared, got '%s'", v.tableName)
	}
}

func TestResultsView_Escape_ToEditor(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	// Set up as query results
	v.query = "SELECT 1"
	v.schema = ""
	v.tableName = ""

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.parent.mode != ViewEditor {
		t.Errorf("Expected mode ViewEditor after Esc from query results, got %v", v.parent.mode)
	}
}

func TestResultsView_Escape_ToBrowser(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	// Set up as table data
	v.query = ""
	v.schema = "public"
	v.tableName = "users"

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.parent.mode != ViewBrowser {
		t.Errorf("Expected mode ViewBrowser after Esc from table data, got %v", v.parent.mode)
	}
}

func TestResultsView_Escape_ToReturnTo(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	// Set explicit returnTo
	v.returnTo = ViewChat
	v.query = "SELECT 1" // Would normally go to editor

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.parent.mode != ViewChat {
		t.Errorf("Expected mode ViewChat (returnTo), got %v", v.parent.mode)
	}

	// returnTo should be reset
	if v.returnTo != 0 {
		t.Error("Expected returnTo to be reset")
	}
}

func TestResultsView_Navigation_LeftRight(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	// Create many columns
	columns := make([]engine.Column, 15)
	for i := 0; i < 15; i++ {
		columns[i] = engine.Column{Name: string(rune('a' + i)), Type: "text"}
	}

	v.results = &engine.GetRowsResult{
		Columns: columns,
		Rows:    [][]string{make([]string, 15)},
	}
	v.maxColumns = 10
	v.columnOffset = 0

	// Move right
	msg := tea.KeyMsg{Type: tea.KeyRight}
	v, _ = v.Update(msg)

	if v.columnOffset != 1 {
		t.Errorf("Expected columnOffset 1 after right, got %d", v.columnOffset)
	}

	// Move left
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	v, _ = v.Update(msg)

	if v.columnOffset != 0 {
		t.Errorf("Expected columnOffset 0 after left, got %d", v.columnOffset)
	}

	// Move left at boundary - should stay at 0
	v, _ = v.Update(msg)
	if v.columnOffset != 0 {
		t.Errorf("Expected columnOffset to stay 0 at left boundary, got %d", v.columnOffset)
	}
}

func TestResultsView_Navigation_VimKeys(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	columns := make([]engine.Column, 15)
	for i := 0; i < 15; i++ {
		columns[i] = engine.Column{Name: string(rune('a' + i)), Type: "text"}
	}

	v.results = &engine.GetRowsResult{
		Columns: columns,
		Rows:    [][]string{make([]string, 15)},
	}
	v.maxColumns = 10
	v.columnOffset = 0

	// 'l' moves right
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	v, _ = v.Update(msg)

	if v.columnOffset != 1 {
		t.Errorf("Expected columnOffset 1 after 'l', got %d", v.columnOffset)
	}

	// 'h' moves left
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	v, _ = v.Update(msg)

	if v.columnOffset != 0 {
		t.Errorf("Expected columnOffset 0 after 'h', got %d", v.columnOffset)
	}
}

func TestResultsView_PageSizeCycle(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	// Start at 50 (default)
	if v.pageSize != 50 {
		t.Fatalf("Expected initial pageSize 50, got %d", v.pageSize)
	}

	// Press 's' to cycle: 50 -> 100
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	v, _ = v.Update(msg)

	if v.pageSize != 100 {
		t.Errorf("Expected pageSize 100 after first 's', got %d", v.pageSize)
	}

	// Press 's' again: 100 -> 10 (wraps around)
	v, _ = v.Update(msg)

	if v.pageSize != 10 {
		t.Errorf("Expected pageSize 10 after second 's' (wrap), got %d", v.pageSize)
	}

	// Press 's': 10 -> 25
	v, _ = v.Update(msg)

	if v.pageSize != 25 {
		t.Errorf("Expected pageSize 25 after third 's', got %d", v.pageSize)
	}
}

func TestResultsView_CustomPageSize_Enter(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	// Press 'S' to enter custom page size mode
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}}
	v, _ = v.Update(msg)

	if !v.editingPageSize {
		t.Error("Expected editingPageSize to be true after 'S'")
	}

	// Type "75"
	for _, r := range "75" {
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		v, _ = v.Update(msg)
	}

	// Press Enter to confirm
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	v, _ = v.Update(msg)

	if v.editingPageSize {
		t.Error("Expected editingPageSize to be false after Enter")
	}

	if v.pageSize != 75 {
		t.Errorf("Expected pageSize 75 after custom input, got %d", v.pageSize)
	}
}

func TestResultsView_CustomPageSize_Escape(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	originalPageSize := v.pageSize

	// Enter custom page size mode
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}}
	v, _ = v.Update(msg)

	// Type something
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}}
	v, _ = v.Update(msg)

	// Press Escape to cancel
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.editingPageSize {
		t.Error("Expected editingPageSize to be false after Esc")
	}

	if v.pageSize != originalPageSize {
		t.Errorf("Expected pageSize to remain %d after cancel, got %d", originalPageSize, v.pageSize)
	}
}

func TestResultsView_CustomPageSize_InvalidInput(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	originalPageSize := v.pageSize

	// Enter custom page size mode
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}}
	v, _ = v.Update(msg)

	// Type invalid input
	for _, r := range "abc" {
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		v, _ = v.Update(msg)
	}

	// Press Enter - should not change page size
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	v, _ = v.Update(msg)

	if v.editingPageSize {
		t.Error("Expected editingPageSize to be false after Enter with invalid input")
	}

	if v.pageSize != originalPageSize {
		t.Errorf("Expected pageSize to remain %d after invalid input, got %d", originalPageSize, v.pageSize)
	}
}

func TestResultsView_NextPage(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	v.results = &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "id", Type: "integer"}},
		Rows:    make([][]string, 100),
	}
	v.totalRows = 100
	v.pageSize = 25
	v.currentPage = 0

	// Press 'n' for next page
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	v, _ = v.Update(msg)

	if v.currentPage != 1 {
		t.Errorf("Expected currentPage 1 after 'n', got %d", v.currentPage)
	}
}

func TestResultsView_PreviousPage(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	v.results = &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "id", Type: "integer"}},
		Rows:    make([][]string, 100),
	}
	v.totalRows = 100
	v.pageSize = 25
	v.currentPage = 2

	// Press 'p' for previous page
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	v, _ = v.Update(msg)

	if v.currentPage != 1 {
		t.Errorf("Expected currentPage 1 after 'p', got %d", v.currentPage)
	}
}

func TestResultsView_PreviousPage_AtStart(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	v.results = &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "id", Type: "integer"}},
		Rows:    make([][]string, 100),
	}
	v.currentPage = 0

	// Press 'p' at first page - should stay
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	v, _ = v.Update(msg)

	if v.currentPage != 0 {
		t.Errorf("Expected currentPage to stay 0, got %d", v.currentPage)
	}
}

func TestResultsView_HasNextPage(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	tests := []struct {
		name        string
		totalRows   int
		pageSize    int
		currentPage int
		expected    bool
	}{
		{"has next page", 100, 25, 0, true},
		{"at last page", 100, 25, 3, false},
		{"exact fit", 100, 50, 1, false},
		{"no results", 0, 25, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v.results = &engine.GetRowsResult{
				Columns: []engine.Column{{Name: "id", Type: "integer"}},
				Rows:    make([][]string, tt.totalRows),
			}
			v.totalRows = tt.totalRows
			v.pageSize = tt.pageSize
			v.currentPage = tt.currentPage

			result := v.hasNextPage()
			if result != tt.expected {
				t.Errorf("hasNextPage() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestResultsView_HasPreviousPage(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	tests := []struct {
		name        string
		currentPage int
		expected    bool
	}{
		{"at first page", 0, false},
		{"at second page", 1, true},
		{"at middle page", 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v.currentPage = tt.currentPage
			result := v.hasPreviousPage()
			if result != tt.expected {
				t.Errorf("hasPreviousPage() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestResultsView_CurrentPageRows(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	// Create 30 rows
	rows := make([][]string, 30)
	for i := 0; i < 30; i++ {
		rows[i] = []string{string(rune('0' + i%10))}
	}

	v.results = &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "id", Type: "integer"}},
		Rows:    rows,
	}
	v.pageSize = 10
	v.currentPage = 1

	pageRows := v.currentPageRows()

	if len(pageRows) != 10 {
		t.Errorf("Expected 10 rows for page 1, got %d", len(pageRows))
	}
}

func TestResultsView_EffectiveTotalRows(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	tests := []struct {
		name      string
		results   *engine.GetRowsResult
		totalRows int
		expected  int
	}{
		{
			name:     "nil results",
			results:  nil,
			expected: 0,
		},
		{
			name: "totalRows set",
			results: &engine.GetRowsResult{
				Rows: make([][]string, 50),
			},
			totalRows: 100,
			expected:  100,
		},
		{
			name: "use row count",
			results: &engine.GetRowsResult{
				Rows: make([][]string, 50),
			},
			totalRows: 0,
			expected:  50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v.results = tt.results
			v.totalRows = tt.totalRows
			result := v.effectiveTotalRows()
			if result != tt.expected {
				t.Errorf("effectiveTotalRows() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestResultsView_IsTableData(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	// Not table data (query results)
	v.schema = ""
	v.tableName = ""

	if v.isTableData() {
		t.Error("Expected isTableData() false for query results")
	}

	// Table data
	v.schema = "public"
	v.tableName = "users"

	if !v.isTableData() {
		t.Error("Expected isTableData() true for table data")
	}
}

func TestResultsView_CountWhereConditions(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	// No conditions
	v.whereCondition = nil
	if count := v.countWhereConditions(); count != 0 {
		t.Errorf("Expected 0 conditions, got %d", count)
	}
}

func TestResultsView_View_QueryResults(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	v.query = "SELECT * FROM users"
	v.results = &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "id", Type: "integer"}},
		Rows:    [][]string{{"1"}},
	}
	v.totalRows = 1

	view := v.View()

	if !strings.Contains(view, "Query Results") {
		t.Error("Expected 'Query Results' title for query results")
	}

	if !strings.Contains(view, "SELECT * FROM users") {
		t.Error("Expected query to be shown")
	}

	if !strings.Contains(view, "editor") {
		t.Error("Expected help text to show 'editor' as back target")
	}
}

func TestResultsView_View_TableData(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	v.query = ""
	v.schema = "public"
	v.tableName = "users"
	v.results = &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "id", Type: "integer"}},
		Rows:    [][]string{{"1"}},
	}
	v.totalRows = 1

	view := v.View()

	if !strings.Contains(view, "Table Data") {
		t.Error("Expected 'Table Data' title for table data")
	}

	if !strings.Contains(view, "browser") {
		t.Error("Expected help text to show 'browser' as back target")
	}

	// Should have where/columns shortcuts for table data
	if !strings.Contains(view, "where") {
		t.Error("Expected 'where' shortcut for table data")
	}

	if !strings.Contains(view, "columns") {
		t.Error("Expected 'columns' shortcut for table data")
	}
}

func TestResultsView_View_NoResults(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	v.results = nil

	view := v.View()

	if !strings.Contains(view, "No results") {
		t.Error("Expected 'No results' when results is nil")
	}
}

func TestResultsView_View_PageSizeEditing(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	v.results = &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "id", Type: "integer"}},
		Rows:    [][]string{{"1"}},
	}
	v.editingPageSize = true

	view := v.View()

	if !strings.Contains(view, "Page size:") {
		t.Error("Expected 'Page size:' prompt when editing")
	}

	if !strings.Contains(view, "enter to confirm") {
		t.Error("Expected confirmation hint when editing page size")
	}
}

func TestResultsView_MouseScroll(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	v.results = &engine.GetRowsResult{
		Columns: []engine.Column{{Name: "id", Type: "integer"}},
		Rows: [][]string{
			{"1"}, {"2"}, {"3"}, {"4"}, {"5"},
			{"6"}, {"7"}, {"8"}, {"9"}, {"10"},
		},
	}
	v.updateTable()

	// Mouse wheel down
	msg := tea.MouseMsg{Type: tea.MouseWheelDown}
	v, _ = v.Update(msg)

	// Mouse wheel up
	msg = tea.MouseMsg{Type: tea.MouseWheelUp}
	v, _ = v.Update(msg)

	// Just ensure no panic - table handles internal cursor state
}

func TestResultsView_WindowSizeMsg(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	v, _ = v.Update(msg)

	// Table dimensions should be adjusted (height - 20, width - 8)
	// We can't easily verify internal table state, but ensure no panic
}

func TestResultsView_PageLoadedMsg(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	// Simulate page loaded message
	msg := pageLoadedMsg{}
	v, cmd := v.Update(msg)

	if cmd != nil {
		t.Error("Expected nil command from pageLoadedMsg")
	}

	// Should just trigger re-render, no state changes
}

func TestResultsView_MaxInt(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{5, 3, 5},
		{3, 5, 5},
		{0, 0, 0},
		{-1, 1, 1},
	}

	for _, tt := range tests {
		result := maxInt(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("maxInt(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestResultsView_ColumnOffsetBoundary(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	columns := make([]engine.Column, 15)
	for i := 0; i < 15; i++ {
		columns[i] = engine.Column{Name: string(rune('a' + i)), Type: "text"}
	}

	v.results = &engine.GetRowsResult{
		Columns: columns,
		Rows:    [][]string{make([]string, 15)},
	}
	v.maxColumns = 10
	v.columnOffset = 0

	// Navigate to the right edge
	msg := tea.KeyMsg{Type: tea.KeyRight}
	for i := 0; i < 10; i++ {
		v, _ = v.Update(msg)
	}

	// Should stop at max offset (15 - 10 = 5)
	if v.columnOffset > 5 {
		t.Errorf("Expected columnOffset <= 5, got %d", v.columnOffset)
	}
}

func TestResultsView_UpdateTable_EmptyResults(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	v.results = nil
	v.updateTable()

	// Should not panic, table should be empty
}

func TestResultsView_UpdateTable_WithVisibleColumns(t *testing.T) {
	v, cleanup := setupResultsViewTest(t)
	defer cleanup()

	v.results = &engine.GetRowsResult{
		Columns: []engine.Column{
			{Name: "id", Type: "integer"},
			{Name: "name", Type: "text"},
			{Name: "email", Type: "text"},
		},
		Rows: [][]string{
			{"1", "Alice", "alice@example.com"},
			{"2", "Bob", "bob@example.com"},
		},
	}
	v.visibleColumns = []string{"id", "email"} // Skip "name"
	v.updateTable()

	// Should not panic, table should only show id and email columns
}
