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
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clidey/whodb/core/src/engine"
)

func setupEditorViewTest(t *testing.T) (*EditorView, func()) {
	t.Helper()

	setupTestEnv(t)

	parent := NewMainModel()
	if parent.err != nil {
		t.Fatalf("Failed to create MainModel: %v", parent.err)
	}

	cleanup := func() {}

	return parent.editorView, cleanup
}

func TestNewEditorView(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	if v == nil {
		t.Fatal("NewEditorView returned nil")
	}

	if len(v.allSuggestions) != 0 {
		t.Error("Expected empty allSuggestions initially")
	}

	if len(v.filteredSuggestions) != 0 {
		t.Error("Expected empty filteredSuggestions initially")
	}

	if v.showSuggestions {
		t.Error("Expected showSuggestions to be false initially")
	}

	if v.selectedSuggestion != 0 {
		t.Errorf("Expected selectedSuggestion 0, got %d", v.selectedSuggestion)
	}
}

func TestEditorView_Escape_HidesSuggestions(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.showSuggestions = true
	v.filteredSuggestions = []suggestion{{label: "test"}}

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.showSuggestions {
		t.Error("Expected showSuggestions to be false after Esc")
	}
}

func TestEditorView_Escape_GoesBack(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.showSuggestions = false

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.parent.mode != ViewBrowser {
		t.Errorf("Expected mode ViewBrowser after Esc, got %v", v.parent.mode)
	}
}

func TestEditorView_CtrlL_Clear(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.textarea.SetValue("SELECT * FROM users")
	v.showSuggestions = true
	v.filteredSuggestions = []suggestion{{label: "test"}}
	v.selectedSuggestion = 5
	v.cursorPos = 10
	v.lastText = "some text"

	msg := tea.KeyMsg{Type: tea.KeyCtrlL}
	v, _ = v.Update(msg)

	if v.textarea.Value() != "" {
		t.Error("Expected textarea to be cleared")
	}

	if v.showSuggestions {
		t.Error("Expected showSuggestions to be false after clear")
	}

	if len(v.allSuggestions) != 0 {
		t.Error("Expected allSuggestions to be cleared")
	}

	if len(v.filteredSuggestions) != 0 {
		t.Error("Expected filteredSuggestions to be cleared")
	}

	if v.selectedSuggestion != 0 {
		t.Errorf("Expected selectedSuggestion 0 after clear, got %d", v.selectedSuggestion)
	}

	if v.cursorPos != 0 {
		t.Errorf("Expected cursorPos 0 after clear, got %d", v.cursorPos)
	}

	if v.lastText != "" {
		t.Error("Expected lastText to be cleared")
	}
}

func TestEditorView_SuggestionNavigation_Tab(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.showSuggestions = true
	v.filteredSuggestions = []suggestion{
		{label: "one"},
		{label: "two"},
		{label: "three"},
	}
	v.selectedSuggestion = 0

	// Tab cycles forward
	msg := tea.KeyMsg{Type: tea.KeyTab}
	v, _ = v.Update(msg)

	if v.selectedSuggestion != 1 {
		t.Errorf("Expected selectedSuggestion 1 after Tab, got %d", v.selectedSuggestion)
	}

	// Tab wraps around
	v.selectedSuggestion = 2
	v, _ = v.Update(msg)

	if v.selectedSuggestion != 0 {
		t.Errorf("Expected selectedSuggestion 0 after wrap, got %d", v.selectedSuggestion)
	}
}

func TestEditorView_SuggestionNavigation_ShiftTab(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.showSuggestions = true
	v.filteredSuggestions = []suggestion{
		{label: "one"},
		{label: "two"},
		{label: "three"},
	}
	v.selectedSuggestion = 1

	// Shift+Tab goes backward
	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	v, _ = v.Update(msg)

	if v.selectedSuggestion != 0 {
		t.Errorf("Expected selectedSuggestion 0 after Shift+Tab, got %d", v.selectedSuggestion)
	}

	// Shift+Tab wraps around from 0
	v, _ = v.Update(msg)

	if v.selectedSuggestion != 2 {
		t.Errorf("Expected selectedSuggestion 2 after wrap, got %d", v.selectedSuggestion)
	}
}

func TestEditorView_SuggestionNavigation_Down(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.showSuggestions = true
	v.filteredSuggestions = []suggestion{
		{label: "one"},
		{label: "two"},
	}
	v.selectedSuggestion = 0

	msg := tea.KeyMsg{Type: tea.KeyDown}
	v, _ = v.Update(msg)

	if v.selectedSuggestion != 1 {
		t.Errorf("Expected selectedSuggestion 1 after Down, got %d", v.selectedSuggestion)
	}
}

func TestEditorView_SuggestionNavigation_Up(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.showSuggestions = true
	v.filteredSuggestions = []suggestion{
		{label: "one"},
		{label: "two"},
	}
	v.selectedSuggestion = 1

	msg := tea.KeyMsg{Type: tea.KeyUp}
	v, _ = v.Update(msg)

	if v.selectedSuggestion != 0 {
		t.Errorf("Expected selectedSuggestion 0 after Up, got %d", v.selectedSuggestion)
	}
}

func TestEditorView_SuggestionNavigation_CtrlN(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.showSuggestions = true
	v.filteredSuggestions = []suggestion{
		{label: "one"},
		{label: "two"},
	}
	v.selectedSuggestion = 0

	msg := tea.KeyMsg{Type: tea.KeyCtrlN}
	v, _ = v.Update(msg)

	if v.selectedSuggestion != 1 {
		t.Errorf("Expected selectedSuggestion 1 after Ctrl+N, got %d", v.selectedSuggestion)
	}
}

func TestEditorView_SuggestionNavigation_CtrlP(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.showSuggestions = true
	v.filteredSuggestions = []suggestion{
		{label: "one"},
		{label: "two"},
	}
	v.selectedSuggestion = 1

	msg := tea.KeyMsg{Type: tea.KeyCtrlP}
	v, _ = v.Update(msg)

	if v.selectedSuggestion != 0 {
		t.Errorf("Expected selectedSuggestion 0 after Ctrl+P, got %d", v.selectedSuggestion)
	}
}

func TestEditorView_MouseScroll_WithSuggestions(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.showSuggestions = true
	v.filteredSuggestions = []suggestion{
		{label: "one"},
		{label: "two"},
		{label: "three"},
	}
	v.selectedSuggestion = 1

	// Mouse wheel down
	msg := tea.MouseMsg{Button: tea.MouseButtonWheelDown}
	v, _ = v.Update(msg)

	if v.selectedSuggestion != 2 {
		t.Errorf("Expected selectedSuggestion 2 after wheel down, got %d", v.selectedSuggestion)
	}

	// Mouse wheel up
	msg = tea.MouseMsg{Button: tea.MouseButtonWheelUp}
	v, _ = v.Update(msg)

	if v.selectedSuggestion != 1 {
		t.Errorf("Expected selectedSuggestion 1 after wheel up, got %d", v.selectedSuggestion)
	}
}

func TestEditorView_MouseScrollUp_WrapAround(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.showSuggestions = true
	v.filteredSuggestions = []suggestion{
		{label: "one"},
		{label: "two"},
	}
	v.selectedSuggestion = 0

	// Mouse wheel up from 0 wraps to end
	msg := tea.MouseMsg{Button: tea.MouseButtonWheelUp}
	v, _ = v.Update(msg)

	if v.selectedSuggestion != 1 {
		t.Errorf("Expected selectedSuggestion 1 after wrap, got %d", v.selectedSuggestion)
	}
}

func TestEditorView_WindowSizeMsg(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	msg := tea.WindowSizeMsg{Width: 120, Height: 50}
	v, _ = v.Update(msg)

	if v.lastWidth != 120 {
		t.Errorf("Expected lastWidth 120, got %d", v.lastWidth)
	}

	if v.lastHeight != 50 {
		t.Errorf("Expected lastHeight 50, got %d", v.lastHeight)
	}
}

func TestEditorView_ApplyWindowSize(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.applyWindowSize(100, 40)

	// Textarea width should be width - 8
	// We can't easily verify internal textarea state, but ensure no panic
}

func TestEditorView_ComputeSuggestionHeight(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	// No suggestions shown
	v.showSuggestions = false
	height := v.computeSuggestionHeight(50)
	if height != 0 {
		t.Errorf("Expected 0 height when suggestions not shown, got %d", height)
	}

	// Empty suggestions
	v.showSuggestions = true
	v.filteredSuggestions = []suggestion{}
	height = v.computeSuggestionHeight(50)
	if height != 0 {
		t.Errorf("Expected 0 height with empty suggestions, got %d", height)
	}

	// With suggestions
	v.filteredSuggestions = []suggestion{{label: "test"}}
	height = v.computeSuggestionHeight(50)
	if height <= 0 {
		t.Error("Expected positive height with suggestions")
	}
}

func TestEditorView_FilterSuggestions(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.allSuggestions = []suggestion{
		{label: "SELECT"},
		{label: "FROM"},
		{label: "users"},
		{label: "users.id"},
	}

	// Empty prefix shows all
	v.filterSuggestions("")
	if len(v.filteredSuggestions) != 4 {
		t.Errorf("Expected 4 suggestions with empty prefix, got %d", len(v.filteredSuggestions))
	}

	// Filter by prefix
	v.filterSuggestions("sel")
	if len(v.filteredSuggestions) != 1 {
		t.Errorf("Expected 1 suggestion for 'sel', got %d", len(v.filteredSuggestions))
	}
	if v.filteredSuggestions[0].label != "SELECT" {
		t.Errorf("Expected 'SELECT', got '%s'", v.filteredSuggestions[0].label)
	}

	// Case insensitive
	v.filterSuggestions("USER")
	if len(v.filteredSuggestions) != 2 {
		t.Errorf("Expected 2 suggestions for 'USER', got %d", len(v.filteredSuggestions))
	}
}

func TestEditorView_FilterSuggestions_QualifiedNames(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.allSuggestions = []suggestion{
		{label: "users.id"},
		{label: "users.name"},
		{label: "orders.id"},
	}

	// Filter by part after dot
	v.filterSuggestions("id")
	if len(v.filteredSuggestions) != 2 {
		t.Errorf("Expected 2 suggestions for 'id', got %d", len(v.filteredSuggestions))
	}
}

func TestFindDiffPosition(t *testing.T) {
	tests := []struct {
		s1, s2   string
		expected int
	}{
		{"hello", "hello", 5}, // identical
		{"hello", "hella", 4}, // differ at position 4
		{"abc", "abcd", 3},    // s2 is longer
		{"abcd", "abc", 3},    // s1 is longer
		{"", "a", 0},          // s1 is empty
		{"a", "", 0},          // s2 is empty
		{"", "", 0},           // both empty
		{"xyz", "abc", 0},     // completely different
	}

	for _, tt := range tests {
		result := findDiffPosition(tt.s1, tt.s2)
		if result != tt.expected {
			t.Errorf("findDiffPosition(%q, %q) = %d, expected %d", tt.s1, tt.s2, result, tt.expected)
		}
	}
}

func TestGetLastWord(t *testing.T) {
	tests := []struct {
		text     string
		expected string
	}{
		{"SELECT * FROM users", "users"},
		{"SELECT * FROM ", "FROM"},
		{"", ""},
		{"word", "word"},
		{"users.id", "id"},
		{"schema.table.column", "column"},
		{"func(arg1, arg2", "arg2"},
		{"`quoted`", "quoted"},
	}

	for _, tt := range tests {
		result := getLastWord(tt.text)
		if result != tt.expected {
			t.Errorf("getLastWord(%q) = %q, expected %q", tt.text, result, tt.expected)
		}
	}
}

func TestEditorView_IsKeyword(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	keywords := []string{"SELECT", "FROM", "WHERE", "AND", "OR", "JOIN", "LEFT", "RIGHT", "ON", "AS"}
	for _, kw := range keywords {
		if !v.isKeyword(kw) {
			t.Errorf("Expected '%s' to be a keyword", kw)
		}
		// Test lowercase
		if !v.isKeyword(strings.ToLower(kw)) {
			t.Errorf("Expected '%s' to be a keyword (case insensitive)", strings.ToLower(kw))
		}
	}

	nonKeywords := []string{"users", "id", "name", "table1"}
	for _, word := range nonKeywords {
		if v.isKeyword(word) {
			t.Errorf("Expected '%s' to NOT be a keyword", word)
		}
	}
}

func TestEditorView_UpdateCursorPosition(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	// Empty text
	v.lastText = ""
	v.textarea.SetValue("")
	v.updateCursorPosition()
	if v.cursorPos != 0 {
		t.Errorf("Expected cursorPos 0 for empty text, got %d", v.cursorPos)
	}

	// First time with text
	v.lastText = ""
	v.textarea.SetValue("hello")
	v.updateCursorPosition()
	if v.cursorPos != 5 {
		t.Errorf("Expected cursorPos 5 for first text, got %d", v.cursorPos)
	}
}

func TestEditorView_View(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	view := v.View()

	if !strings.Contains(view, "SQL Editor") {
		t.Error("Expected 'SQL Editor' title")
	}

	if !strings.Contains(view, "autocomplete") {
		t.Error("Expected 'autocomplete' help text")
	}

	if !strings.Contains(view, "clear") {
		t.Error("Expected 'clear' help text")
	}
}

func TestEditorView_View_WithError(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.err = os.ErrPermission

	view := v.View()

	if !strings.Contains(view, "permission") {
		t.Error("Expected error message in view")
	}
}

func TestEditorView_View_WithSuggestions(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.showSuggestions = true
	v.filteredSuggestions = []suggestion{
		{label: "SELECT", typ: suggestionTypeKeyword, detail: "SQL Keyword"},
		{label: "FROM", typ: suggestionTypeKeyword, detail: "SQL Keyword"},
	}
	v.selectedSuggestion = 0
	v.lastHeight = 50
	v.suggestionHeight = 10

	view := v.View()

	if !strings.Contains(view, "Suggestions") {
		t.Error("Expected 'Suggestions' in view when showing suggestions")
	}

	if !strings.Contains(view, "SELECT") {
		t.Error("Expected 'SELECT' suggestion in view")
	}
}

func TestEditorView_RenderAutocompletePanel(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.filteredSuggestions = []suggestion{
		{label: "users", typ: suggestionTypeTable, detail: "Table"},
		{label: "orders", typ: suggestionTypeTable, detail: "Table"},
	}
	v.selectedSuggestion = 0

	panel := v.renderAutocompletePanel()

	if !strings.Contains(panel, "Suggestions") {
		t.Error("Expected 'Suggestions' header")
	}

	if !strings.Contains(panel, "users") {
		t.Error("Expected 'users' in panel")
	}

	if !strings.Contains(panel, "navigate") {
		t.Error("Expected 'navigate' control hint")
	}

	if !strings.Contains(panel, "accept") {
		t.Error("Expected 'accept' control hint")
	}
}

func TestEditorView_AcceptSuggestion(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.textarea.SetValue("SELECT us")
	v.cursorPos = 9
	v.lastText = "SELECT us"
	v.showSuggestions = true
	v.filteredSuggestions = []suggestion{
		{label: "users", apply: "users"},
	}
	v.selectedSuggestion = 0

	v.acceptSuggestion()

	// Should replace "us" with "users"
	if !strings.Contains(v.textarea.Value(), "users") {
		t.Error("Expected 'users' in textarea after accepting suggestion")
	}

	if v.showSuggestions {
		t.Error("Expected showSuggestions to be false after accept")
	}
}

func TestEditorView_AddKeywordSuggestions(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.addKeywordSuggestions()

	if len(v.allSuggestions) == 0 {
		t.Error("Expected keywords to be added")
	}

	// Check for some common keywords
	hasSelect := false
	hasFrom := false
	for _, sug := range v.allSuggestions {
		if sug.label == "SELECT" {
			hasSelect = true
		}
		if sug.label == "FROM" {
			hasFrom = true
		}
	}

	if !hasSelect {
		t.Error("Expected 'SELECT' in keyword suggestions")
	}

	if !hasFrom {
		t.Error("Expected 'FROM' in keyword suggestions")
	}
}

func TestEditorView_AddFunctionSuggestions(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.addFunctionSuggestions()

	if len(v.allSuggestions) == 0 {
		t.Error("Expected functions to be added")
	}

	// Check for common functions
	hasCount := false
	for _, sug := range v.allSuggestions {
		if sug.label == "COUNT" {
			hasCount = true
			// Function should have () in apply
			if !strings.Contains(sug.apply, "()") {
				t.Error("Expected function apply to contain '()'")
			}
			break
		}
	}

	if !hasCount {
		t.Error("Expected 'COUNT' in function suggestions")
	}
}

func TestEditorView_AddSnippetSuggestions(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.addSnippetSuggestions()

	if len(v.allSuggestions) == 0 {
		t.Error("Expected snippets to be added")
	}

	// Check for snippet types
	hasSnippet := false
	for _, sug := range v.allSuggestions {
		if sug.typ == suggestionTypeSnippet {
			hasSnippet = true
			break
		}
	}

	if !hasSnippet {
		t.Error("Expected at least one snippet suggestion")
	}
}

func TestEditorView_ExtractTablesAndAliases(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{"simple FROM", "SELECT * FROM users", 1},
		{"FROM with alias", "SELECT * FROM users u", 1},
		{"FROM with AS alias", "SELECT * FROM users AS u", 1},
		{"multiple tables", "SELECT * FROM users u JOIN orders o ON u.id = o.user_id", 2},
		{"schema.table", "SELECT * FROM public.users", 1},
		{"no tables", "SELECT 1 + 1", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tables := v.extractTablesAndAliases(tt.text)
			if len(tables) != tt.expected {
				t.Errorf("extractTablesAndAliases(%q) returned %d tables, expected %d", tt.text, len(tables), tt.expected)
			}
		})
	}
}

func TestEditorView_ParseSQLContext(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	// After FROM - expect schema suggestions
	ctx := v.parseSQLContext("SELECT * FROM ", 14)
	if ctx.contextType != suggestionTypeSchema {
		t.Errorf("Expected suggestionTypeSchema after FROM, got %v", ctx.contextType)
	}

	// After JOIN - expect schema suggestions
	ctx = v.parseSQLContext("SELECT * FROM users JOIN ", 25)
	if ctx.contextType != suggestionTypeSchema {
		t.Errorf("Expected suggestionTypeSchema after JOIN, got %v", ctx.contextType)
	}
}

func TestSuggestionType_Constants(t *testing.T) {
	// Verify suggestion type constants are defined
	types := []suggestionType{
		suggestionTypeKeyword,
		suggestionTypeTable,
		suggestionTypeColumn,
		suggestionTypeSchema,
		suggestionTypeMixed,
		suggestionTypeFunction,
		suggestionTypeSnippet,
	}

	for _, typ := range types {
		if typ == "" {
			t.Error("Found empty suggestion type constant")
		}
	}
}

// Tests for query execution message handling

func TestEditorView_QueryExecutedMsg_Success(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	// Set query state to running
	v.queryState = OperationRunning

	// Simulate successful query execution message
	msg := QueryExecutedMsg{
		Result: &engine.GetRowsResult{
			Columns: []engine.Column{{Name: "id", Type: "int"}},
			Rows:    [][]string{{"1"}, {"2"}},
		},
		Query: "SELECT * FROM users",
		Err:   nil,
	}
	v, _ = v.Update(msg)

	// Verify query state was reset
	if v.queryState != OperationIdle {
		t.Errorf("Expected queryState OperationIdle, got %v", v.queryState)
	}

	// Verify cancel function was cleared
	if v.queryCancel != nil {
		t.Error("Expected queryCancel to be nil after completion")
	}
}

func TestEditorView_QueryExecutedMsg_Error(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.queryState = OperationRunning

	// Simulate query execution with error
	msg := QueryExecutedMsg{
		Result: nil,
		Query:  "SELECT * FROM nonexistent",
		Err:    fmt.Errorf("table not found"),
	}
	v, _ = v.Update(msg)

	// Verify query state was reset
	if v.queryState != OperationIdle {
		t.Errorf("Expected queryState OperationIdle, got %v", v.queryState)
	}

	// Verify error was captured in view's err field
	if v.err == nil {
		t.Error("Expected v.err to be set")
	}
}

func TestEditorView_QueryTimeoutMsg(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.queryState = OperationRunning

	// Simulate query timeout message
	msg := QueryTimeoutMsg{
		Query:   "SELECT * FROM slow_table",
		Timeout: 30 * time.Second,
	}
	v, _ = v.Update(msg)

	// Verify query state was reset
	if v.queryState != OperationIdle {
		t.Errorf("Expected queryState OperationIdle after timeout, got %v", v.queryState)
	}

	// Verify error was set in view's err field
	if v.err == nil {
		t.Error("Expected v.err to be set on timeout")
	}
}

func TestEditorView_QueryCancelledMsg(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.queryState = OperationRunning
	v.err = nil // Ensure err is nil before test

	// Simulate query cancelled message
	msg := QueryCancelledMsg{
		Query: "SELECT * FROM users",
	}
	v, _ = v.Update(msg)

	// Verify query state was reset
	if v.queryState != OperationIdle {
		t.Errorf("Expected queryState OperationIdle after cancel, got %v", v.queryState)
	}

	// Cancellation doesn't set an error - the handler just resets state
	// and returns, so v.err remains whatever it was before (nil in this case)
}

func TestEditorView_OperationState_Constants(t *testing.T) {
	// Verify operation state constants
	if OperationIdle != 0 {
		t.Errorf("Expected OperationIdle = 0, got %d", OperationIdle)
	}
	if OperationRunning != 1 {
		t.Errorf("Expected OperationRunning = 1, got %d", OperationRunning)
	}
	if OperationCancelling != 2 {
		t.Errorf("Expected OperationCancelling = 2, got %d", OperationCancelling)
	}
}

func TestEditorView_RetryPrompt_OnTimeout(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	v.queryState = OperationRunning
	testQuery := "SELECT * FROM slow_table"

	// Simulate query timeout message
	msg := QueryTimeoutMsg{
		Query:   testQuery,
		Timeout: 30 * time.Second,
	}
	v, _ = v.Update(msg)

	// Verify retry prompt was enabled
	if !v.retryPrompt {
		t.Error("Expected retryPrompt to be true after timeout")
	}

	// Verify timed out query was stored
	if v.timedOutQuery != testQuery {
		t.Errorf("Expected timedOutQuery '%s', got '%s'", testQuery, v.timedOutQuery)
	}
}

func TestEditorView_RetryPrompt_EscCancels(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	// Set up retry prompt state
	v.retryPrompt = true
	v.timedOutQuery = "SELECT * FROM test"
	v.err = fmt.Errorf("query timed out")

	// Send ESC key
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	// Verify retry prompt was dismissed
	if v.retryPrompt {
		t.Error("Expected retryPrompt to be false after ESC")
	}

	// Verify timed out query was cleared
	if v.timedOutQuery != "" {
		t.Errorf("Expected timedOutQuery to be empty, got '%s'", v.timedOutQuery)
	}
}

func TestEditorView_RetryPrompt_KeyHandling(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		keyType tea.KeyType
	}{
		{"option_1", "1", tea.KeyRunes},
		{"option_2", "2", tea.KeyRunes},
		{"option_3", "3", tea.KeyRunes},
		{"option_4", "4", tea.KeyRunes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, cleanup := setupEditorViewTest(t)
			defer cleanup()

			// Set up retry prompt state
			v.retryPrompt = true
			v.timedOutQuery = "SELECT * FROM test"
			v.err = fmt.Errorf("query timed out")

			// Send number key
			var msg tea.KeyMsg
			if tt.keyType == tea.KeyRunes {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}
			v, cmd := v.Update(msg)

			// Verify retry prompt was dismissed
			if v.retryPrompt {
				t.Error("Expected retryPrompt to be false after selecting retry option")
			}

			// Verify error was cleared
			if v.err != nil {
				t.Error("Expected err to be nil after retry")
			}

			// Verify a command was returned (the query execution)
			if cmd == nil {
				t.Error("Expected a command to be returned for retry")
			}
		})
	}
}

func TestEditorView_RetryPrompt_IgnoresOtherKeys(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	// Set up retry prompt state
	v.retryPrompt = true
	v.timedOutQuery = "SELECT * FROM test"
	v.err = fmt.Errorf("query timed out")

	// Send an unrelated key (like 'a')
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}
	v, _ = v.Update(msg)

	// Verify retry prompt is still active
	if !v.retryPrompt {
		t.Error("Expected retryPrompt to still be true after unrecognized key")
	}

	// Verify query wasn't cleared
	if v.timedOutQuery == "" {
		t.Error("Expected timedOutQuery to still be set")
	}
}

func TestEditorView_RetryPrompt_View(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	// Set up retry prompt state
	v.retryPrompt = true
	v.timedOutQuery = "SELECT * FROM test"
	v.err = fmt.Errorf("query timed out")

	view := v.View()

	// Verify retry prompt is shown
	if !strings.Contains(view, "Retry with longer timeout") {
		t.Error("Expected 'Retry with longer timeout' in view")
	}
	// EditorView uses abbreviated timeout options
	if !strings.Contains(view, "60s") {
		t.Error("Expected '60s' option in view")
	}
	if !strings.Contains(view, "2min") {
		t.Error("Expected '2min' option in view")
	}
	if !strings.Contains(view, "5min") {
		t.Error("Expected '5min' option in view")
	}
	if !strings.Contains(view, "no limit") {
		t.Error("Expected 'no limit' option in view")
	}
}

func TestEditorView_AcceptSuggestion_CursorPositionSync(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	// Setup: "SELECT us" with cursor at position 9 (after "us")
	v.textarea.SetValue("SELECT us")
	v.cursorPos = 9
	v.lastText = "SELECT us"
	v.showSuggestions = true
	v.filteredSuggestions = []suggestion{
		{label: "users", apply: "users"},
	}
	v.selectedSuggestion = 0

	v.acceptSuggestion()

	// After accepting "users", cursor should be at position 12 (SELECT users)
	// startPos = 9 - 2 (length of "us") = 7
	// newCursorPos = 7 + 5 (length of "users") = 12
	expectedPos := 12
	if v.cursorPos != expectedPos {
		t.Errorf("Expected cursorPos %d after accepting suggestion, got %d", expectedPos, v.cursorPos)
	}

	// lastText should be synced
	expectedText := "SELECT users"
	if v.lastText != expectedText {
		t.Errorf("Expected lastText '%s', got '%s'", expectedText, v.lastText)
	}
}

func TestEditorView_AcceptSuggestion_InsertAtCursor(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	// Setup: "SELECT " with cursor at position 7 (after space, no token to replace)
	v.textarea.SetValue("SELECT ")
	v.cursorPos = 7
	v.lastText = "SELECT "
	v.showSuggestions = true
	v.filteredSuggestions = []suggestion{
		{label: "COUNT", apply: "COUNT"},
	}
	v.selectedSuggestion = 0

	v.acceptSuggestion()

	// Should insert "COUNT" at position 7
	expectedText := "SELECT COUNT"
	if v.textarea.Value() != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, v.textarea.Value())
	}

	// Cursor should be at position 12 (7 + 5)
	expectedPos := 12
	if v.cursorPos != expectedPos {
		t.Errorf("Expected cursorPos %d, got %d", expectedPos, v.cursorPos)
	}
}

func TestEditorView_DebounceSequenceID_Increments(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	initialSeqID := v.autocompleteSeqID

	// Simulate typing by calling Update with key messages
	// Each update should increment the sequence ID
	v.textarea.SetValue("S")
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})

	if v.autocompleteSeqID <= initialSeqID {
		t.Error("Expected sequence ID to increment after keystroke")
	}

	seqAfterFirst := v.autocompleteSeqID

	v.textarea.SetValue("SE")
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'E'}})

	if v.autocompleteSeqID <= seqAfterFirst {
		t.Error("Expected sequence ID to increment after second keystroke")
	}
}

func TestEditorView_DebounceMsg_StaleIgnored(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	// Set a specific sequence ID
	v.autocompleteSeqID = 10

	// Process a stale debounce message (seqID doesn't match)
	staleMsg := AutocompleteDebounceMsg{SeqID: 5, Text: "SELECT", Pos: 6}
	v.Update(staleMsg)

	// Since the message was stale, autocomplete should not be triggered
	// The suggestions should remain empty (initial state)
	if v.showSuggestions {
		t.Error("Expected stale debounce message to be ignored")
	}
}

func TestEditorView_DebounceMsg_CurrentProcessed(t *testing.T) {
	v, cleanup := setupEditorViewTest(t)
	defer cleanup()

	// Set a specific sequence ID
	v.autocompleteSeqID = 10

	// Process a current debounce message (seqID matches)
	// Note: This won't show suggestions unless there's context
	// but it should process without panic
	currentMsg := AutocompleteDebounceMsg{SeqID: 10, Text: "SELECT F", Pos: 8}
	v.Update(currentMsg)

	// Should process without error - the autocomplete logic will run
	// We just verify it doesn't panic and returns the view
}
