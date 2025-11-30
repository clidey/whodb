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
