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
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/tui/layout"
)

// setupConnectedModel creates a MainModel connected to a temp SQLite DB.
func setupConnectedModel(t *testing.T, width, height int) *MainModel {
	t.Helper()
	setupTestEnv(t)
	os.Setenv("WHODB_CLI", "true")

	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"
	f, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	f.Close()

	conn := &config.Connection{
		Name:     "test",
		Type:     "Sqlite3",
		Host:     dbPath,
		Database: dbPath,
	}

	m := NewMainModelWithConnection(conn)
	if m.err != nil {
		t.Skipf("SQLite plugin not available: %v", m.err)
	}

	m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return m
}

func TestUseMultiPane_ModalViewsFallback(t *testing.T) {
	m := setupConnectedModel(t, 140, 40)

	// Layout views should use multi-pane
	for _, mode := range []ViewMode{ViewBrowser, ViewEditor, ViewResults} {
		m.mode = mode
		if !m.useMultiPane() {
			t.Errorf("useMultiPane() should be true for mode %d", mode)
		}
	}

	// Modal/overlay views should NOT use multi-pane
	for _, mode := range []ViewMode{ViewExport, ViewWhere, ViewColumns, ViewSchema, ViewHistory, ViewChat} {
		m.mode = mode
		if m.useMultiPane() {
			t.Errorf("useMultiPane() should be false for modal mode %d", mode)
		}
	}
}

func TestUseMultiPane_SingleLayoutReturnsFalse(t *testing.T) {
	m := setupConnectedModel(t, 140, 40)
	m.activeLayout = layout.LayoutSingle
	m.rebuildLayout()

	m.mode = ViewBrowser
	if m.useMultiPane() {
		t.Error("useMultiPane() should be false for Single layout")
	}
}

func TestCompactMode_SetOnRebuild(t *testing.T) {
	m := setupConnectedModel(t, 140, 40)

	if !m.browserView.compact {
		t.Error("BrowserView should be compact in multi-pane")
	}
	if !m.editorView.compact {
		t.Error("EditorView should be compact in multi-pane")
	}
	if !m.resultsView.compact {
		t.Error("ResultsView should be compact in multi-pane")
	}

	// Switch to single — compact should be reset
	m.activeLayout = layout.LayoutSingle
	m.rebuildLayout()
	if m.browserView.compact {
		t.Error("BrowserView should not be compact in single-pane")
	}
}

func TestCompactMode_SuppressesBrowserHelp(t *testing.T) {
	m := setupConnectedModel(t, 80, 30)

	m.browserView.compact = false
	normalOutput := m.browserView.View()

	m.browserView.compact = true
	compactOutput := m.browserView.View()

	if len(compactOutput) >= len(normalOutput) {
		t.Error("Compact mode should produce shorter output (no help text)")
	}
}

func TestLayoutCycling(t *testing.T) {
	m := setupConnectedModel(t, 140, 40)

	expected := []layout.LayoutName{
		layout.LayoutQuery,
		layout.LayoutFull,
		layout.LayoutSingle,
		layout.LayoutExplore,
	}

	for i, exp := range expected {
		m.cycleLayout()
		if m.activeLayout != exp {
			t.Errorf("After cycle %d: got %q, want %q", i+1, m.activeLayout, exp)
		}
	}
}

func TestTabCyclesFocusInMultiPane(t *testing.T) {
	m := setupConnectedModel(t, 140, 40)

	if m.focusedPaneIdx != 0 {
		t.Fatalf("Initial focus should be 0, got %d", m.focusedPaneIdx)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusedPaneIdx != 1 {
		t.Errorf("After Tab: focus should be 1, got %d", m.focusedPaneIdx)
	}
	if m.mode != ViewResults {
		t.Errorf("After Tab: mode should be ViewResults(%d), got %d", ViewResults, m.mode)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusedPaneIdx != 0 {
		t.Errorf("After Tab2: focus should wrap to 0, got %d", m.focusedPaneIdx)
	}
}

func TestBrowserSetDimensions_RecalculatesColumns(t *testing.T) {
	setupTestEnv(t)
	m := NewMainModel()
	if m.err != nil {
		t.Fatalf("NewMainModel failed: %v", m.err)
	}

	m.browserView.SetDimensions(120, 30)
	if m.browserView.columnsPerRow < 2 {
		t.Errorf("At 120 width, columnsPerRow should be >= 2, got %d", m.browserView.columnsPerRow)
	}

	m.browserView.SetDimensions(40, 30)
	if m.browserView.columnsPerRow > 1 {
		t.Errorf("At 40 width, columnsPerRow should be 1, got %d", m.browserView.columnsPerRow)
	}
}

func TestTabBarShowsOnlyMainViews(t *testing.T) {
	setupTestEnv(t)
	m := NewMainModel()
	if m.err != nil {
		t.Fatalf("NewMainModel failed: %v", m.err)
	}
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	indicator := m.renderViewIndicator()

	for _, name := range []string{"Connection", "Browser", "Editor", "Results", "History", "Chat"} {
		if !strings.Contains(indicator, name) {
			t.Errorf("Tab bar should contain %q", name)
		}
	}

	for _, name := range []string{"Export", "Where", "Columns", "Schema"} {
		if strings.Contains(indicator, name) {
			t.Errorf("Tab bar should NOT contain modal view %q", name)
		}
	}
}

func TestGlobalHelpBar_ContainsGlobalShortcuts(t *testing.T) {
	m := setupConnectedModel(t, 140, 40)
	m.mode = ViewBrowser

	helpBar := m.renderGlobalHelpBar()
	if !strings.Contains(helpBar, "history") {
		t.Error("Global help bar should contain 'history'")
	}
	if !strings.Contains(helpBar, "layout") {
		t.Error("Global help bar should contain 'layout'")
	}
}

func TestMultiPaneRender_ProducesOutput(t *testing.T) {
	m := setupConnectedModel(t, 140, 40)

	output := m.View()
	if output == "" {
		t.Fatal("Multi-pane View() should produce output")
	}

	lines := strings.Split(output, "\n")
	if len(lines) < 10 {
		t.Errorf("Expected at least 10 lines, got %d", len(lines))
	}

	found := false
	for _, line := range lines {
		if strings.Contains(line, "Browser") && strings.Contains(line, "─") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Output should contain Browser pane header")
	}
}

func TestPushViewExport_FallsToSinglePane(t *testing.T) {
	m := setupConnectedModel(t, 140, 40)

	// Start in multi-pane Results
	m.mode = ViewResults
	if !m.useMultiPane() {
		t.Fatal("Should be in multi-pane mode")
	}

	// Push Export modal
	m.PushView(ViewExport)
	if m.mode != ViewExport {
		t.Errorf("Mode should be ViewExport, got %d", m.mode)
	}
	if m.useMultiPane() {
		t.Error("useMultiPane should be false for Export modal")
	}

	// Pop back to Results — should resume multi-pane
	m.PopView()
	if m.mode != ViewResults {
		t.Errorf("Mode should be ViewResults after pop, got %d", m.mode)
	}
	if !m.useMultiPane() {
		t.Error("useMultiPane should be true after returning to Results")
	}
}

// setupConnectedModelWithTable creates a model with a SQLite DB containing a test table.
func setupConnectedModelWithTable(t *testing.T, width, height int) *MainModel {
	t.Helper()
	m := setupConnectedModel(t, width, height)

	_, err := m.dbManager.ExecuteQuery("CREATE TABLE IF NOT EXISTS test_users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = m.dbManager.ExecuteQuery("INSERT OR IGNORE INTO test_users VALUES (1, 'alice', 'a@b.com'), (2, 'bob', 'b@b.com')")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	return m
}

func TestPageLoadedMsg_SetsTableNameWithEmptySchema(t *testing.T) {
	m := setupConnectedModelWithTable(t, 100, 30)

	// Load table with empty schema (SQLite)
	cmd := m.resultsView.LoadTable("", "test_users")
	if cmd == nil {
		t.Fatal("LoadTable should return a command")
	}

	// Execute the command and feed the message back
	msg := cmd()
	m.resultsView.Update(msg)

	if m.resultsView.tableName != "test_users" {
		t.Errorf("tableName should be 'test_users', got %q", m.resultsView.tableName)
	}
	if m.resultsView.results == nil {
		t.Error("results should not be nil after LoadTable")
	}
}

func TestResultsView_ExportKeyWorksForTableData(t *testing.T) {
	m := setupConnectedModelWithTable(t, 100, 30)

	// Load table data into results
	cmd := m.resultsView.LoadTable("", "test_users")
	msg := cmd()
	m.resultsView.Update(msg)
	m.mode = ViewResults

	// Press 'e' for export
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if m.mode != ViewExport {
		t.Errorf("After 'e': mode should be ViewExport(%d), got %d", ViewExport, m.mode)
	}
}

func TestResultsView_WhereKeyWorksForTableData(t *testing.T) {
	m := setupConnectedModelWithTable(t, 100, 30)

	cmd := m.resultsView.LoadTable("", "test_users")
	msg := cmd()
	m.resultsView.Update(msg)
	m.mode = ViewResults

	// Press 'w' for where
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if m.mode != ViewWhere {
		t.Errorf("After 'w': mode should be ViewWhere(%d), got %d", ViewWhere, m.mode)
	}
}

func TestResultsView_ColumnsKeyWorksForTableData(t *testing.T) {
	m := setupConnectedModelWithTable(t, 100, 30)

	cmd := m.resultsView.LoadTable("", "test_users")
	msg := cmd()
	m.resultsView.Update(msg)
	m.mode = ViewResults

	// Press 'c' for columns
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if m.mode != ViewColumns {
		t.Errorf("After 'c': mode should be ViewColumns(%d), got %d", ViewColumns, m.mode)
	}
}

func TestResultsView_ExportKeyWorksForQueryResults(t *testing.T) {
	m := setupConnectedModelWithTable(t, 100, 30)

	// Run a query (not table browse) — sets results+query, clears schema/tableName
	result, err := m.dbManager.ExecuteQuery("SELECT * FROM test_users")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	m.resultsView.SetResults(result, "SELECT * FROM test_users")
	m.mode = ViewResults

	// Press 'e' — should work via the query results branch
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if m.mode != ViewExport {
		t.Errorf("After 'e' on query results: mode should be ViewExport(%d), got %d", ViewExport, m.mode)
	}
}

func TestResultsView_WhereDisabledForQueryResults(t *testing.T) {
	m := setupConnectedModelWithTable(t, 100, 30)

	result, err := m.dbManager.ExecuteQuery("SELECT * FROM test_users")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	m.resultsView.SetResults(result, "SELECT * FROM test_users")
	m.mode = ViewResults

	// Press 'w' — should NOT work (Where is table-specific)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if m.mode != ViewResults {
		t.Errorf("After 'w' on query results: mode should stay ViewResults(%d), got %d", ViewResults, m.mode)
	}
}

func TestAsyncMessageRouting_PageLoadedReachesResults(t *testing.T) {
	m := setupConnectedModelWithTable(t, 140, 40)

	// Focus on Browser pane (not Results)
	m.mode = ViewBrowser
	m.focusedPaneIdx = 0

	// Send a PageLoadedMsg — should still reach ResultsView
	m.Update(PageLoadedMsg{
		Results:   nil,
		Schema:    "",
		TableName: "test_users",
	})

	// ResultsView should have received the message and set tableName
	if m.resultsView.tableName != "test_users" {
		t.Errorf("PageLoadedMsg should reach ResultsView even when Browser is focused, tableName=%q", m.resultsView.tableName)
	}
}
