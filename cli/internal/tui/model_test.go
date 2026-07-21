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

package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/tui/layout"
	graphmodel "github.com/clidey/whodb/core/graph/model"
)

func TestNewMainModel(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	if m == nil {
		t.Fatal("NewMainModel returned nil")
	}

	if m.err != nil {
		t.Fatalf("NewMainModel failed with error: %v", m.err)
	}

	if m.mode != ViewConnection {
		t.Errorf("Expected initial mode ViewConnection, got %v", m.mode)
	}

	if m.dbManager == nil {
		t.Error("Expected dbManager to be non-nil")
	}

	if m.histMgr == nil {
		t.Error("Expected histMgr to be non-nil")
	}

	if m.config == nil {
		t.Error("Expected config to be non-nil")
	}

	if m.connectionView == nil {
		t.Error("Expected connectionView to be non-nil")
	}

	if m.browserView == nil {
		t.Error("Expected browserView to be non-nil")
	}

	if m.editorView == nil {
		t.Error("Expected editorView to be non-nil")
	}

	if m.resultsView == nil {
		t.Error("Expected resultsView to be non-nil")
	}

	if m.historyView == nil {
		t.Error("Expected historyView to be non-nil")
	}
}

func TestNewMainModelWithConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	setupTestEnv(t)
	tempDir := t.TempDir()

	dbPath := tempDir + "/test.db"
	conn := &config.Connection{
		Name:     "test-sqlite",
		Type:     "Sqlite",
		Host:     dbPath,
		Database: dbPath,
	}

	m := NewMainModelWithConnection(conn)
	if m == nil {
		t.Fatal("NewMainModelWithConnection returned nil")
	}

	if m.err != nil {
		t.Skipf("Skipping test - database plugin not available: %v", m.err)
	}

	if m.mode != ViewBrowser {
		t.Errorf("Expected initial mode ViewBrowser, got %v", m.mode)
	}

	if m.dbManager.GetCurrentConnection() == nil {
		t.Error("Expected current connection to be set")
	}
}

func TestMainModel_Init(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	cmd := m.Init()

	// Init always returns at least the spinner tick command
	if cmd == nil {
		t.Error("Expected Init to return spinner tick command")
	}
}

func TestMainModel_Update_WindowSize(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()

	msg := tea.WindowSizeMsg{
		Width:  100,
		Height: 50,
	}

	_, _ = m.Update(msg)

	if m.width != 100 {
		t.Errorf("Expected width 100, got %d", m.width)
	}

	if m.height != 50 {
		t.Errorf("Expected height 50, got %d", m.height)
	}
}

func TestMainModel_Update_CtrlC(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()

	msg := tea.KeyPressMsg{
		Code: 'c', Mod: tea.ModCtrl,
	}

	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("Expected quit command on Ctrl+C")
	}
}

func TestMainModel_View(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	view := m.View()

	if view.Content == "" {
		t.Error("Expected non-empty view")
	}
}

func TestMainModel_PersistAndRestoreWorkspace(t *testing.T) {
	setupTestEnv(t)

	dbPath := filepath.Join(t.TempDir(), "workspace-restore.db")
	conn := config.Connection{
		Name:     "workspace-sqlite",
		Type:     "Sqlite3",
		Host:     dbPath,
		Database: dbPath,
	}

	cfg := config.DefaultConfig()
	cfg.AddConnection(conn)
	cfg.AddProfile(config.Profile{Name: "daily", Connection: conn.Name})
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	modelWithConn := NewMainModelWithProfile(&conn, cfg, "daily")
	if modelWithConn.err != nil {
		t.Skipf("Skipping workspace restore test - database plugin not available: %v", modelWithConn.err)
	}
	defer modelWithConn.dbManager.Disconnect()

	schema, err := modelWithConn.dbManager.ResolveSnapshotSchema(&conn, "")
	if err != nil {
		t.Fatalf("ResolveSnapshotSchema failed: %v", err)
	}

	if _, err := modelWithConn.dbManager.ExecuteQuery("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)"); err != nil {
		t.Fatalf("Failed to seed SQLite table: %v", err)
	}

	modelWithConn.mode = ViewResults
	modelWithConn.activeLayout = layout.LayoutFull
	modelWithConn.focusedPaneIdx = 2
	modelWithConn.browserView.currentSchema = schema
	modelWithConn.browserView.selectedTable = "users"
	modelWithConn.browserView.filterInput.SetValue("us")
	modelWithConn.editorView.buffers = []queryBuffer{
		{name: "Users", text: "SELECT * FROM users"},
		{name: "Count", text: "SELECT COUNT(*) FROM users"},
	}
	modelWithConn.editorView.activeTab = 1
	modelWithConn.editorView.textarea.SetValue(modelWithConn.editorView.buffers[1].text)
	modelWithConn.resultsView.schema = schema
	modelWithConn.resultsView.tableName = "users"
	modelWithConn.resultsView.currentPage = 2
	modelWithConn.resultsView.pageSize = 25
	modelWithConn.resultsView.columnOffset = 1
	modelWithConn.resultsView.visibleColumns = []string{"id", "name"}
	modelWithConn.resultsView.whereCondition = &graphmodel.WhereCondition{
		Type: graphmodel.WhereConditionTypeAtomic,
		Atomic: &graphmodel.AtomicWhereCondition{
			ColumnType: "TEXT",
			Key:        "name",
			Operator:   "LIKE",
			Value:      "A%",
		},
	}
	modelWithConn.diffView.SetSelectionState(config.WorkspaceDiffState{
		FromConnection: conn.Name,
		ToConnection:   conn.Name,
		FromSchema:     schema,
		ToSchema:       schema,
	})

	if err := modelWithConn.PersistWorkspace(); err != nil {
		t.Fatalf("PersistWorkspace failed: %v", err)
	}

	restored := newMainModel(nil, true)
	if restored.err != nil {
		t.Fatalf("Restored model failed: %v", restored.err)
	}
	defer restored.dbManager.Disconnect()

	if restored.dbManager.GetCurrentConnection() == nil || restored.dbManager.GetCurrentConnection().Name != conn.Name {
		t.Fatalf("expected restored connection %q, got %#v", conn.Name, restored.dbManager.GetCurrentConnection())
	}
	if restored.currentProfileName != "daily" {
		t.Fatalf("expected restored profile daily, got %q", restored.currentProfileName)
	}
	if restored.mode != ViewResults {
		t.Fatalf("expected restored mode ViewResults, got %v", restored.mode)
	}
	if restored.activeLayout != layout.LayoutFull {
		t.Fatalf("expected restored layout %s, got %s", layout.LayoutFull, restored.activeLayout)
	}
	if restored.focusedPaneIdx != 2 {
		t.Fatalf("expected focused pane 2, got %d", restored.focusedPaneIdx)
	}
	if restored.browserView.currentSchema != schema || restored.browserView.selectedTable != "users" {
		t.Fatalf("unexpected restored browser state: schema=%q table=%q", restored.browserView.currentSchema, restored.browserView.selectedTable)
	}
	if restored.browserView.filterInput.Value() != "us" {
		t.Fatalf("expected restored browser filter, got %q", restored.browserView.filterInput.Value())
	}
	if len(restored.editorView.buffers) != 2 || restored.editorView.activeTab != 1 {
		t.Fatalf("unexpected restored editor state: buffers=%d active=%d", len(restored.editorView.buffers), restored.editorView.activeTab)
	}
	if restored.editorView.buffers[1].text != "SELECT COUNT(*) FROM users" {
		t.Fatalf("unexpected restored editor buffer: %#v", restored.editorView.buffers[1])
	}
	if restored.resultsView.tableName != "users" || restored.resultsView.pageSize != 25 || restored.resultsView.currentPage != 2 {
		t.Fatalf("unexpected restored results state: table=%q pageSize=%d page=%d", restored.resultsView.tableName, restored.resultsView.pageSize, restored.resultsView.currentPage)
	}
	if restored.resultsView.whereCondition == nil || restored.resultsView.whereCondition.Atomic == nil || restored.resultsView.whereCondition.Atomic.Value != "A%" {
		t.Fatalf("unexpected restored where condition: %#v", restored.resultsView.whereCondition)
	}

	diffState := restored.diffView.SelectionState()
	if diffState.FromConnection != conn.Name || diffState.ToConnection != conn.Name {
		t.Fatalf("unexpected restored diff state: %#v", diffState)
	}
	if len(restored.initCommands) == 0 {
		t.Fatal("expected restored model to schedule workspace init commands")
	}
}

func TestMainModel_HandleTabSwitch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	setupTestEnv(t)
	tempDir := t.TempDir()

	dbPath := tempDir + "/test.db"
	conn := &config.Connection{
		Name:     "test-sqlite",
		Type:     "Sqlite",
		Host:     dbPath,
		Database: dbPath,
	}

	m := NewMainModelWithConnection(conn)
	if m.err != nil {
		t.Skipf("Skipping test - database plugin not available: %v", m.err)
	}

	initialMode := m.mode

	msg := tea.KeyPressMsg{
		Code: tea.KeyTab,
	}

	_, _ = m.Update(msg)

	if m.mode == initialMode {
		t.Error("Expected mode to change after tab switch")
	}
}

func TestMainModel_HandleTabSwitch_NotConnected(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	initialMode := m.mode

	msg := tea.KeyPressMsg{
		Code: tea.KeyTab,
	}

	_, _ = m.Update(msg)

	if m.mode != initialMode {
		t.Error("Expected mode to remain unchanged when not connected")
	}
}

func TestMainModel_ErrorHandling(t *testing.T) {
	setupTestEnv(t)

	m := &MainModel{
		err: os.ErrInvalid,
	}

	view := m.View()
	if view.Content == "" {
		t.Error("Expected error view to be non-empty")
	}

	msg := tea.KeyPressMsg{
		Text: "q", Code: 'q',
	}

	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("Expected quit command on 'q' in error state")
	}
}

func TestViewMode_String(t *testing.T) {
	modes := []ViewMode{
		ViewConnection,
		ViewBrowser,
		ViewEditor,
		ViewResults,
		ViewHistory,
		ViewExport,
		ViewWhere,
		ViewColumns,
		ViewChat,
		ViewSchema,
		ViewMockData,
		ViewRowWrite,
	}

	for _, mode := range modes {
		if mode < 0 || mode > ViewProfiles {
			t.Errorf("Invalid view mode: %v", mode)
		}
	}
}

func TestMainModel_HelpOverlay(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()

	// In ViewResults (no text input), '?' should show help
	m.mode = ViewResults

	msg := tea.KeyPressMsg{
		Text: "?", Code: '?',
	}

	_, _ = m.Update(msg)

	if !m.showingHelp {
		t.Error("Expected showingHelp to be true in ViewResults after '?'")
	}

	// Any key should dismiss help
	msg = tea.KeyPressMsg{
		Text: "x", Code: 'x',
	}

	_, _ = m.Update(msg)

	if m.showingHelp {
		t.Error("Expected showingHelp to be false after pressing a key")
	}
}

func TestMainModel_HelpOverlay_BlockedInEditor(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()

	// In ViewEditor (always has text input), '?' should NOT show help
	m.mode = ViewEditor

	msg := tea.KeyPressMsg{
		Text: "?", Code: '?',
	}

	_, _ = m.Update(msg)

	if m.showingHelp {
		t.Error("Expected showingHelp to remain false in ViewEditor")
	}
}

func TestMainModel_IsHelpSafe(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()

	tests := []struct {
		mode     ViewMode
		setup    func()
		expected bool
		name     string
	}{
		{ViewResults, nil, true, "ViewResults always safe"},
		{ViewHistory, nil, true, "ViewHistory always safe"},
		{ViewColumns, nil, true, "ViewColumns always safe"},
		{ViewSchema, nil, true, "ViewSchema always safe"},
		{ViewEditor, nil, false, "ViewEditor always has text input"},
		{ViewBrowser, func() { m.browserView.filtering = false }, true, "ViewBrowser safe when not filtering"},
		{ViewBrowser, func() { m.browserView.filtering = true }, false, "ViewBrowser unsafe when filtering"},
		{ViewConnection, func() { m.connectionView.mode = "list" }, true, "ViewConnection safe in list mode"},
		{ViewConnection, func() { m.connectionView.mode = "form" }, false, "ViewConnection unsafe in form mode"},
		{ViewWhere, func() { m.whereView.addingNew = false }, true, "ViewWhere safe when not adding"},
		{ViewWhere, func() { m.whereView.addingNew = true }, false, "ViewWhere unsafe when adding"},
		{ViewMockData, nil, false, "ViewMockData always has text input"},
		{ViewRowWrite, func() { m.rowWriteView.action = rowWriteActionAdd }, false, "ViewRowWrite add flow has text input"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.mode = tt.mode
			if tt.setup != nil {
				tt.setup()
			}

			result := m.isHelpSafe()
			if result != tt.expected {
				t.Errorf("isHelpSafe() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestMainModel_RenderHelpOverlay(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()

	modes := []ViewMode{
		ViewBrowser,
		ViewResults,
		ViewHistory,
		ViewChat,
		ViewSchema,
		ViewColumns,
		ViewWhere,
		ViewExport,
		ViewMockData,
		ViewRowWrite,
		ViewConnection,
	}

	for _, mode := range modes {
		t.Run(mode.String(), func(t *testing.T) {
			m.mode = mode
			output := m.renderHelpOverlay()

			if output == "" {
				t.Error("Expected non-empty help overlay")
			}

			// Should contain "Keyboard Shortcuts" title
			if !contains(output, "Keyboard Shortcuts") {
				t.Error("Expected help overlay to contain 'Keyboard Shortcuts'")
			}

			// Should contain dismiss instruction
			if !contains(output, "Press any key to close") {
				t.Error("Expected help overlay to contain dismiss instruction")
			}
		})
	}
}

// Helper for ViewMode.String() - add if not exists
func (v ViewMode) String() string {
	names := []string{
		"ViewConnection",
		"ViewBrowser",
		"ViewEditor",
		"ViewResults",
		"ViewHistory",
		"ViewExport",
		"ViewWhere",
		"ViewColumns",
		"ViewChat",
		"ViewSchema",
		"ViewImport",
		"ViewMockData",
		"ViewRowWrite",
		"ViewBookmarks",
		"ViewJSON",
		"ViewCmdLog",
		"ViewExplain",
		"ViewDiff",
		"ViewERD",
		"ViewAudit",
		"ViewProfiles",
	}
	if int(v) < len(names) {
		return names[v]
	}
	return "Unknown"
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestMainModel_ErrorDismiss(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	m.err = os.ErrInvalid
	m.mode = ViewBrowser

	// Press Esc to dismiss error
	msg := tea.KeyPressMsg{Code: tea.KeyEsc}
	_, _ = m.Update(msg)

	if m.err != nil {
		t.Error("Expected error to be cleared after Esc")
	}

	if m.mode != ViewBrowser {
		t.Errorf("Expected mode to remain ViewBrowser, got %v", m.mode)
	}
}

func TestMainModel_Update_AllModes(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()

	// Test that Update doesn't panic for any mode
	modes := []ViewMode{
		ViewConnection,
		ViewBrowser,
		ViewEditor,
		ViewResults,
		ViewHistory,
		ViewExport,
		ViewWhere,
		ViewColumns,
		ViewChat,
		ViewSchema,
	}

	for _, mode := range modes {
		t.Run(mode.String(), func(t *testing.T) {
			m.mode = mode
			m.err = nil

			// Send a simple key message
			msg := tea.KeyPressMsg{Text: "x", Code: 'x'}
			_, _ = m.Update(msg)
			// Just ensure no panic
		})
	}
}

func TestMainModel_PushView(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	m.mode = ViewBrowser

	m.PushView(ViewResults)

	if m.mode != ViewResults {
		t.Errorf("Expected mode ViewResults, got %v", m.mode)
	}
	if len(m.viewHistory) != 1 {
		t.Fatalf("Expected 1 entry in view history, got %d", len(m.viewHistory))
	}
	if m.viewHistory[0] != ViewBrowser {
		t.Errorf("Expected ViewBrowser on stack, got %v", m.viewHistory[0])
	}
}

func TestMainModel_PopView(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	m.mode = ViewBrowser

	// Pop from empty stack
	if m.PopView() {
		t.Error("Expected PopView to return false on empty stack")
	}
	if m.mode != ViewBrowser {
		t.Error("Expected mode unchanged after empty pop")
	}

	// Push then pop
	m.PushView(ViewResults)
	m.PushView(ViewExport)

	if m.mode != ViewExport {
		t.Errorf("Expected mode ViewExport, got %v", m.mode)
	}

	if !m.PopView() {
		t.Error("Expected PopView to return true")
	}
	if m.mode != ViewResults {
		t.Errorf("Expected mode ViewResults after first pop, got %v", m.mode)
	}

	if !m.PopView() {
		t.Error("Expected PopView to return true")
	}
	if m.mode != ViewBrowser {
		t.Errorf("Expected mode ViewBrowser after second pop, got %v", m.mode)
	}

	if m.PopView() {
		t.Error("Expected PopView to return false on empty stack")
	}
}

func TestMainModel_PushView_DeepNavigation(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	m.mode = ViewBrowser

	// Simulate: Browser → Results → Where → (pop back through)
	m.PushView(ViewResults)
	m.PushView(ViewWhere)

	if len(m.viewHistory) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(m.viewHistory))
	}

	m.PopView()
	if m.mode != ViewResults {
		t.Errorf("Expected ViewResults, got %v", m.mode)
	}

	m.PopView()
	if m.mode != ViewBrowser {
		t.Errorf("Expected ViewBrowser, got %v", m.mode)
	}
}

func TestMainModel_TabSwitch_ClearsStack(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	m.mode = ViewBrowser

	// Build up a navigation stack
	m.PushView(ViewResults)
	m.PushView(ViewWhere)

	if len(m.viewHistory) != 2 {
		t.Fatalf("Expected 2 entries before tab switch, got %d", len(m.viewHistory))
	}

	// Simulate tab switch
	m.viewHistory = nil
	m.mode = ViewEditor

	if len(m.viewHistory) != 0 {
		t.Errorf("Expected empty stack after tab switch, got %d", len(m.viewHistory))
	}
}

func TestMainModel_SetStatus(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()

	cmd := m.SetStatus("Query executed (5 rows)")
	if m.statusMessage != "Query executed (5 rows)" {
		t.Errorf("Expected status message to be set, got %q", m.statusMessage)
	}
	if cmd == nil {
		t.Error("Expected SetStatus to return a tick command for auto-dismiss")
	}
}

func TestMainModel_StatusMessageTimeout(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	m.statusMessage = "Test message"

	// Process the timeout message
	result, _ := m.Update(statusMessageTimeoutMsg{})
	model := result.(*MainModel)

	if model.statusMessage != "" {
		t.Errorf("Expected status message to be cleared after timeout, got %q", model.statusMessage)
	}
}

func TestMainModel_RenderStatusBar_NotConnected(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	m.mode = ViewBrowser

	bar := m.renderStatusBar()
	if bar != "" {
		t.Errorf("Expected empty status bar when not connected, got %q", bar)
	}
}

func TestMainModel_RenderStatusBar_ConnectionView(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	m.mode = ViewConnection

	bar := m.renderStatusBar()
	if bar != "" {
		t.Errorf("Expected empty status bar on connection view, got %q", bar)
	}
}

func TestMainModel_IsLoading(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()

	// Clear all loading states (some views start with loading=true)
	m.browserView.loading = false
	m.schemaView.loading = false
	m.editorView.queryState = OperationIdle
	m.chatView.sending = false
	m.chatView.loadingModels = false
	m.exportView.exporting = false
	m.historyView.executing = false
	m.connectionView.connecting = false

	if m.isLoading() {
		t.Error("Expected isLoading=false when all views are idle")
	}

	m.browserView.loading = true
	if !m.isLoading() {
		t.Error("Expected isLoading=true when browser is loading")
	}
	m.browserView.loading = false

	m.editorView.queryState = OperationRunning
	if !m.isLoading() {
		t.Error("Expected isLoading=true when editor query is running")
	}
	m.editorView.queryState = OperationIdle

	m.chatView.sending = true
	if !m.isLoading() {
		t.Error("Expected isLoading=true when chat is sending")
	}
	m.chatView.sending = false
}

func TestMainModel_SpinnerView(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	view := m.SpinnerView()

	// Spinner should return some non-empty string (the dot character)
	if view == "" {
		t.Error("Expected SpinnerView to return non-empty string")
	}
}

func TestMainModel_RenderStatusBar_Truncation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	setupTestEnv(t)
	tempDir := t.TempDir()

	dbPath := tempDir + "/test.db"
	conn := &config.Connection{
		Name:     "test-sqlite",
		Type:     "Sqlite",
		Host:     dbPath,
		Database: dbPath,
	}

	m := NewMainModelWithConnection(conn)
	if m.err != nil {
		t.Skipf("Skipping test - database plugin not available: %v", m.err)
	}

	m.browserView.currentSchema = "very_long_schema_name_that_takes_space"
	m.statusMessage = "Some status message here"

	// Very narrow width should still produce output without panic
	m.width = 20
	bar := m.renderStatusBar()
	if bar == "" {
		t.Error("Expected non-empty status bar even at narrow width")
	}

	// Wide width should include all parts
	m.width = 200
	bar = m.renderStatusBar()
	if bar == "" {
		t.Error("Expected non-empty status bar at wide width")
	}
}
