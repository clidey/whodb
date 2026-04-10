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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/core/src/engine"

	_ "github.com/clidey/whodb/core/src/plugins/sqlite3"
)

func setupConnectedModelForTest(t *testing.T) *MainModel {
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
		Name: "test", Type: "Sqlite3", Host: dbPath, Database: dbPath,
	}
	m := NewMainModelWithConnection(conn)
	if m.err != nil {
		t.Skipf("SQLite not available: %v", m.err)
	}
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	_, err = m.dbManager.ExecuteQuery("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	_, err = m.dbManager.ExecuteQuery("CREATE TABLE orders (id INTEGER PRIMARY KEY, user_id INTEGER, total REAL)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	return m
}

func strPtr(s string) *string { return &s }

// --- ERD Tests ---

func TestERDView_TabCyclesTables(t *testing.T) {
	m := setupConnectedModelForTest(t)

	m.mode = ViewERD
	m.erdView.loading = false
	m.erdView.tables = []tableWithColumns{
		{StorageUnit: engine.StorageUnit{Name: "users"}, Columns: []engine.Column{{Name: "id"}}},
		{StorageUnit: engine.StorageUnit{Name: "orders"}, Columns: []engine.Column{{Name: "id"}}},
	}

	if m.erdView.focusedIndex != 0 {
		t.Fatalf("initial focusedIndex should be 0, got %d", m.erdView.focusedIndex)
	}

	// Tab should cycle to next table, NOT switch views
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.erdView.focusedIndex != 1 {
		t.Errorf("After Tab: focusedIndex should be 1, got %d", m.erdView.focusedIndex)
	}
	if m.mode != ViewERD {
		t.Errorf("After Tab: should still be in ViewERD, got %d", m.mode)
	}

	// Tab again wraps
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.erdView.focusedIndex != 0 {
		t.Errorf("After Tab2: focusedIndex should wrap to 0, got %d", m.erdView.focusedIndex)
	}
}

func TestERDView_ZoomToggle(t *testing.T) {
	m := setupConnectedModelForTest(t)

	m.mode = ViewERD
	m.erdView.loading = false
	m.erdView.tables = []tableWithColumns{
		{StorageUnit: engine.StorageUnit{Name: "users"}, Columns: []engine.Column{{Name: "id", Type: "INTEGER"}}},
	}

	if m.erdView.compact {
		t.Fatal("should start expanded")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if !m.erdView.compact {
		t.Error("should be compact after z")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if m.erdView.compact {
		t.Error("should be expanded after second z")
	}
}

func TestERDView_EscCloses(t *testing.T) {
	m := setupConnectedModelForTest(t)

	m.PushView(ViewERD)
	if m.mode != ViewERD {
		t.Fatal("should be in ViewERD")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode == ViewERD {
		t.Error("Esc should close ERD view")
	}
}

func TestERDView_RenderDoesNotPanic(t *testing.T) {
	m := setupConnectedModelForTest(t)

	m.mode = ViewERD
	m.erdView.loading = false
	m.erdView.tables = []tableWithColumns{
		{
			StorageUnit: engine.StorageUnit{Name: "users"},
			Columns: []engine.Column{
				{Name: "id", Type: "INTEGER", IsPrimary: true},
				{Name: "name", Type: "TEXT"},
			},
		},
		{
			StorageUnit: engine.StorageUnit{Name: "orders"},
			Columns: []engine.Column{
				{Name: "id", Type: "INTEGER", IsPrimary: true},
				{Name: "user_id", Type: "INTEGER", IsForeignKey: true, ReferencedTable: strPtr("users"), ReferencedColumn: strPtr("id")},
			},
		},
	}

	output := m.erdView.View()
	if output == "" {
		t.Error("ERD View() should produce output")
	}
	if !strings.Contains(output, "users") {
		t.Error("output should contain 'users' table")
	}
}

func TestERDLayout_RenderExpandedBox(t *testing.T) {
	table := tableWithColumns{
		StorageUnit: engine.StorageUnit{Name: "test_table"},
		Columns: []engine.Column{
			{Name: "id", Type: "INTEGER", IsPrimary: true},
			{Name: "name", Type: "TEXT"},
		},
	}

	box := renderTableBox(table, false, false)
	if !strings.Contains(box.content, "test_table") {
		t.Error("box should contain table name")
	}
	if !strings.Contains(box.content, "id") {
		t.Error("box should contain column 'id'")
	}
	if !strings.Contains(box.content, "PK") {
		t.Error("box should show PK for primary key")
	}
}

func TestERDLayout_CompactBox(t *testing.T) {
	table := tableWithColumns{
		StorageUnit: engine.StorageUnit{Name: "users"},
		Columns: []engine.Column{
			{Name: "id", Type: "INTEGER"},
		},
	}

	box := renderTableBox(table, true, false)
	if !strings.Contains(box.content, "users") {
		t.Error("compact box should contain table name")
	}
	if strings.Contains(box.content, "INTEGER") {
		t.Error("compact box should not show column types")
	}
}

// --- EXPLAIN Tests ---

func TestExplainView_ReceivesResult(t *testing.T) {
	m := setupConnectedModelForTest(t)
	m.mode = ViewExplain
	m.explainView.query = "SELECT * FROM users"

	msg := explainResultMsg{
		query: "SELECT * FROM users",
		plan:  "SCAN users\n  SEARCH TABLE users",
	}
	m.Update(msg)

	output := m.explainView.View()
	if !strings.Contains(output, "SCAN") || !strings.Contains(output, "users") {
		t.Errorf("Explain view should show the plan, got: %s", output)
	}
}

func TestExplainView_HandlesError(t *testing.T) {
	m := setupConnectedModelForTest(t)
	m.mode = ViewExplain

	msg := explainResultMsg{
		query: "SELECT * FROM nonexistent",
		err:   fmt.Errorf("no such table: nonexistent"),
	}
	m.Update(msg)

	output := m.explainView.View()
	if !strings.Contains(output, "nonexistent") {
		t.Error("should show the error message")
	}
}

func TestExplainView_EscCloses(t *testing.T) {
	m := setupConnectedModelForTest(t)

	m.PushView(ViewExplain)
	if m.mode != ViewExplain {
		t.Fatal("should be in ViewExplain")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode == ViewExplain {
		t.Error("Esc should close Explain view")
	}
}

func TestExplainKeybinding(t *testing.T) {
	help := Keys.Editor.Explain.Help()
	if help.Key != "ctrl+x" {
		t.Errorf("Explain key should be 'ctrl+x', got %q", help.Key)
	}
}

func TestExecuteExplain_SQLite(t *testing.T) {
	m := setupConnectedModelForTest(t)

	result, err := m.dbManager.ExecuteExplain("SELECT * FROM users")
	if err != nil {
		t.Fatalf("ExecuteExplain: %v", err)
	}
	if result == nil || len(result.Rows) == 0 {
		t.Error("EXPLAIN should return rows")
	}
}
