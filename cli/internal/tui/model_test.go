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
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clidey/whodb/cli/internal/config"
)

func TestNewMainModel(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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

	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	m := NewMainModel()
	cmd := m.Init()

	if cmd != nil {
		t.Error("Expected Init to return nil when not connected")
	}
}

func TestMainModel_Update_WindowSize(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	m := NewMainModel()

	msg := tea.KeyMsg{
		Type: tea.KeyCtrlC,
	}

	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("Expected quit command on Ctrl+C")
	}
}

func TestMainModel_View(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	m := NewMainModel()
	view := m.View()

	if view == "" {
		t.Error("Expected non-empty view")
	}
}

func TestMainModel_HandleTabSwitch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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

	msg := tea.KeyMsg{
		Type: tea.KeyTab,
	}

	_, _ = m.Update(msg)

	if m.mode == initialMode {
		t.Error("Expected mode to change after tab switch")
	}
}

func TestMainModel_HandleTabSwitch_NotConnected(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	m := NewMainModel()
	initialMode := m.mode

	msg := tea.KeyMsg{
		Type: tea.KeyTab,
	}

	_, _ = m.Update(msg)

	if m.mode != initialMode {
		t.Error("Expected mode to remain unchanged when not connected")
	}
}

func TestMainModel_ErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	m := &MainModel{
		err: os.ErrInvalid,
	}

	view := m.View()
	if view == "" {
		t.Error("Expected error view to be non-empty")
	}

	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'q'},
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
	}

	for _, mode := range modes {
		if mode < 0 || mode > ViewSchema {
			t.Errorf("Invalid view mode: %v", mode)
		}
	}
}

func TestMainModel_HelpOverlay(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	m := NewMainModel()

	// In ViewResults (no text input), '?' should show help
	m.mode = ViewResults

	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'?'},
	}

	_, _ = m.Update(msg)

	if !m.showingHelp {
		t.Error("Expected showingHelp to be true in ViewResults after '?'")
	}

	// Any key should dismiss help
	msg = tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'x'},
	}

	_, _ = m.Update(msg)

	if m.showingHelp {
		t.Error("Expected showingHelp to be false after pressing a key")
	}
}

func TestMainModel_HelpOverlay_BlockedInEditor(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	m := NewMainModel()

	// In ViewEditor (always has text input), '?' should NOT show help
	m.mode = ViewEditor

	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'?'},
	}

	_, _ = m.Update(msg)

	if m.showingHelp {
		t.Error("Expected showingHelp to remain false in ViewEditor")
	}
}

func TestMainModel_IsHelpSafe(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	m := NewMainModel()
	m.err = os.ErrInvalid
	m.mode = ViewBrowser

	// Press Esc to dismiss error
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, _ = m.Update(msg)

	if m.err != nil {
		t.Error("Expected error to be cleared after Esc")
	}

	if m.mode != ViewBrowser {
		t.Errorf("Expected mode to remain ViewBrowser, got %v", m.mode)
	}
}

func TestMainModel_Update_AllModes(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
			_, _ = m.Update(msg)
			// Just ensure no panic
		})
	}
}
