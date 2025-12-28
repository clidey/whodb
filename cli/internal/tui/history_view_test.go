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
	"github.com/clidey/whodb/cli/internal/history"
)

func setupHistoryViewTest(t *testing.T) (*HistoryView, func()) {
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

	return parent.historyView, cleanup
}

func TestNewHistoryView(t *testing.T) {
	v, cleanup := setupHistoryViewTest(t)
	defer cleanup()

	if v == nil {
		t.Fatal("NewHistoryView returned nil")
	}

	if v.confirmingClear {
		t.Error("Expected confirmingClear to be false initially")
	}

	if v.list.Title != "Query History" {
		t.Errorf("Expected list title 'Query History', got '%s'", v.list.Title)
	}
}

func TestHistoryView_ClearConfirmation_Enter(t *testing.T) {
	v, cleanup := setupHistoryViewTest(t)
	defer cleanup()

	// Press shift+D to enter confirmation mode
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}}
	v, _ = v.Update(msg)

	if !v.confirmingClear {
		t.Error("Expected confirmingClear to be true after shift+D")
	}
}

func TestHistoryView_ClearConfirmation_Confirm(t *testing.T) {
	v, cleanup := setupHistoryViewTest(t)
	defer cleanup()

	// Add some history
	v.parent.histMgr.Add("SELECT 1", true, "testdb")
	v.parent.histMgr.Add("SELECT 2", true, "testdb")
	v.refreshList()

	// Enter confirmation mode
	v.confirmingClear = true

	// Confirm with 'y'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	v, _ = v.Update(msg)

	if v.confirmingClear {
		t.Error("Expected confirmingClear to be false after confirmation")
	}

	entries := v.parent.histMgr.GetAll()
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", len(entries))
	}
}

func TestHistoryView_ClearConfirmation_ConfirmUppercase(t *testing.T) {
	v, cleanup := setupHistoryViewTest(t)
	defer cleanup()

	v.parent.histMgr.Add("SELECT 1", true, "testdb")
	v.refreshList()

	v.confirmingClear = true

	// Confirm with 'Y' (uppercase)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}}
	v, _ = v.Update(msg)

	if v.confirmingClear {
		t.Error("Expected confirmingClear to be false after Y confirmation")
	}

	entries := v.parent.histMgr.GetAll()
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", len(entries))
	}
}

func TestHistoryView_ClearConfirmation_Cancel_N(t *testing.T) {
	v, cleanup := setupHistoryViewTest(t)
	defer cleanup()

	v.parent.histMgr.Add("SELECT 1", true, "testdb")
	v.refreshList()

	v.confirmingClear = true

	// Cancel with 'n'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	v, _ = v.Update(msg)

	if v.confirmingClear {
		t.Error("Expected confirmingClear to be false after n")
	}

	entries := v.parent.histMgr.GetAll()
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry (not cleared), got %d", len(entries))
	}
}

func TestHistoryView_ClearConfirmation_Cancel_Esc(t *testing.T) {
	v, cleanup := setupHistoryViewTest(t)
	defer cleanup()

	v.parent.histMgr.Add("SELECT 1", true, "testdb")
	v.refreshList()

	v.confirmingClear = true

	// Cancel with Esc
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.confirmingClear {
		t.Error("Expected confirmingClear to be false after Esc")
	}

	entries := v.parent.histMgr.GetAll()
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry (not cleared), got %d", len(entries))
	}
}

func TestHistoryView_Escape_GoesBack(t *testing.T) {
	v, cleanup := setupHistoryViewTest(t)
	defer cleanup()

	v.confirmingClear = false

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.parent.mode != ViewBrowser {
		t.Errorf("Expected mode ViewBrowser after Esc, got %v", v.parent.mode)
	}
}

func TestHistoryView_View_Empty(t *testing.T) {
	v, cleanup := setupHistoryViewTest(t)
	defer cleanup()

	view := v.View()

	if !strings.Contains(view, "Query History") {
		t.Error("Expected view to contain 'Query History'")
	}

	if !strings.Contains(view, "No history entries") {
		t.Error("Expected view to show 'No history entries' when empty")
	}
}

func TestHistoryView_View_WithEntries(t *testing.T) {
	v, cleanup := setupHistoryViewTest(t)
	defer cleanup()

	v.parent.histMgr.Add("SELECT * FROM users", true, "testdb")
	v.refreshList()

	view := v.View()

	if strings.Contains(view, "No history entries") {
		t.Error("Should not show 'No history entries' when entries exist")
	}
}

func TestHistoryView_View_ConfirmationDialog(t *testing.T) {
	v, cleanup := setupHistoryViewTest(t)
	defer cleanup()

	v.confirmingClear = true

	view := v.View()

	if !strings.Contains(view, "Clear all history?") {
		t.Error("Expected confirmation dialog to show 'Clear all history?'")
	}

	if !strings.Contains(view, "confirm") {
		t.Error("Expected confirmation dialog to show confirm option")
	}

	if !strings.Contains(view, "cancel") {
		t.Error("Expected confirmation dialog to show cancel option")
	}
}

func TestHistoryView_MouseScroll(t *testing.T) {
	v, cleanup := setupHistoryViewTest(t)
	defer cleanup()

	// Add multiple entries to enable scrolling
	for i := 0; i < 10; i++ {
		v.parent.histMgr.Add("SELECT "+string(rune('0'+i)), true, "testdb")
	}
	v.refreshList()

	initialIndex := v.list.Index()

	// Mouse wheel down
	msg := tea.MouseMsg{Button: tea.MouseButtonWheelDown}
	v, _ = v.Update(msg)

	if v.list.Index() <= initialIndex {
		// This might not change if already at end, but shouldn't error
	}

	// Mouse wheel up
	msg = tea.MouseMsg{Button: tea.MouseButtonWheelUp}
	v, _ = v.Update(msg)
	// Just ensure no panic
}

func TestHistoryView_RefreshList(t *testing.T) {
	v, cleanup := setupHistoryViewTest(t)
	defer cleanup()

	// Add entries
	v.parent.histMgr.Add("SELECT 1", true, "db1")
	v.parent.histMgr.Add("SELECT 2", false, "db2")

	v.refreshList()

	items := v.list.Items()
	if len(items) != 2 {
		t.Errorf("Expected 2 items after refresh, got %d", len(items))
	}
}

func TestHistoryView_Init(t *testing.T) {
	v, cleanup := setupHistoryViewTest(t)
	defer cleanup()

	v.parent.histMgr.Add("SELECT 1", true, "testdb")

	// List should be empty before Init
	if len(v.list.Items()) != 0 {
		t.Error("Expected empty list before Init")
	}

	v.Init()

	// List should have items after Init
	if len(v.list.Items()) != 1 {
		t.Errorf("Expected 1 item after Init, got %d", len(v.list.Items()))
	}
}

func TestHistoryView_HelpText(t *testing.T) {
	v, cleanup := setupHistoryViewTest(t)
	defer cleanup()

	view := v.View()

	// Check for shift+d notation (not [D])
	if !strings.Contains(view, "shift+d") {
		t.Error("Expected help text to show 'shift+d' for clear all")
	}

	// Check for other shortcuts
	if !strings.Contains(view, "re-run") {
		t.Error("Expected help text to show 're-run'")
	}

	if !strings.Contains(view, "edit") {
		t.Error("Expected help text to show 'edit'")
	}
}

func TestHistoryItem_Title(t *testing.T) {
	entry := historyItem{
		entry: history.Entry{
			Query:    "SELECT * FROM users WHERE id = 1",
			Database: "testdb",
			Success:  true,
		},
	}

	title := entry.Title()
	if title != "SELECT * FROM users WHERE id = 1" {
		t.Errorf("Unexpected title: %s", title)
	}

	// Test truncation
	longQuery := strings.Repeat("SELECT ", 20)
	entry.entry.Query = longQuery
	title = entry.Title()
	if len(title) > 63 { // 60 + "..."
		t.Errorf("Expected truncated title, got length %d", len(title))
	}
	if !strings.HasSuffix(title, "...") {
		t.Error("Expected truncated title to end with ...")
	}
}

func TestHistoryItem_Description(t *testing.T) {
	entry := historyItem{
		entry: history.Entry{
			Query:    "SELECT 1",
			Database: "testdb",
			Success:  true,
		},
	}

	desc := entry.Description()
	if !strings.Contains(desc, "testdb") {
		t.Error("Expected description to contain database name")
	}

	// Success indicator
	entry.entry.Success = true
	desc = entry.Description()
	if !strings.HasPrefix(desc, "✓") {
		t.Error("Expected success indicator for successful query")
	}

	// Failure indicator
	entry.entry.Success = false
	desc = entry.Description()
	if !strings.HasPrefix(desc, "✗") {
		t.Error("Expected failure indicator for failed query")
	}
}

func TestHistoryItem_FilterValue(t *testing.T) {
	entry := historyItem{
		entry: history.Entry{
			Query: "SELECT * FROM users",
		},
	}

	filterVal := entry.FilterValue()
	if filterVal != "SELECT * FROM users" {
		t.Errorf("Expected FilterValue to be query, got: %s", filterVal)
	}
}

func TestHistoryView_RetryPrompt_EscCancels(t *testing.T) {
	v, cleanup := setupHistoryViewTest(t)
	defer cleanup()

	// Set up retry prompt state
	v.retryPrompt = true
	v.timedOutQuery = "SELECT * FROM test"
	v.parent.err = nil

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

func TestHistoryView_RetryPrompt_KeyHandling(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"option_1", "1"},
		{"option_2", "2"},
		{"option_3", "3"},
		{"option_4", "4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, cleanup := setupHistoryViewTest(t)
			defer cleanup()

			// Set up retry prompt state
			v.retryPrompt = true
			v.timedOutQuery = "SELECT * FROM test"
			v.parent.err = nil

			// Send number key
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			v, cmd := v.Update(msg)

			// Verify retry prompt was dismissed
			if v.retryPrompt {
				t.Error("Expected retryPrompt to be false after selecting retry option")
			}

			// Verify error was cleared
			if v.parent.err != nil {
				t.Error("Expected parent.err to be nil after retry")
			}

			// Verify a command was returned (the query execution)
			if cmd == nil {
				t.Error("Expected a command to be returned for retry")
			}
		})
	}
}

func TestHistoryView_RetryPrompt_IgnoresOtherKeys(t *testing.T) {
	v, cleanup := setupHistoryViewTest(t)
	defer cleanup()

	// Set up retry prompt state
	v.retryPrompt = true
	v.timedOutQuery = "SELECT * FROM test"

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

func TestHistoryView_RetryPrompt_View(t *testing.T) {
	v, cleanup := setupHistoryViewTest(t)
	defer cleanup()

	// Set up retry prompt state
	v.retryPrompt = true
	v.timedOutQuery = "SELECT * FROM test"

	view := v.View()

	// Verify retry prompt is shown
	if !strings.Contains(view, "timed out") {
		t.Error("Expected 'timed out' in view")
	}
	if !strings.Contains(view, "60 seconds") {
		t.Error("Expected '60 seconds' option in view")
	}
	if !strings.Contains(view, "2 minutes") {
		t.Error("Expected '2 minutes' option in view")
	}
	if !strings.Contains(view, "5 minutes") {
		t.Error("Expected '5 minutes' option in view")
	}
	if !strings.Contains(view, "No limit") {
		t.Error("Expected 'No limit' option in view")
	}
}
