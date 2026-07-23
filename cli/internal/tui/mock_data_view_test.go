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
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/core/src/engine"
)

func TestMockDataView_SetTarget(t *testing.T) {
	setupTestEnv(t)

	parent := NewMainModel()
	if parent.err != nil {
		t.Fatalf("NewMainModel failed: %v", parent.err)
	}

	view := NewMockDataView(parent)
	view.SetTarget("public", "orders")

	if view.schema != "public" {
		t.Fatalf("expected schema public, got %q", view.schema)
	}
	if view.tableInput.Value() != "orders" {
		t.Fatalf("expected table orders, got %q", view.tableInput.Value())
	}
	if view.rowsInput.Value() != "50" {
		t.Fatalf("expected default rows 50, got %q", view.rowsInput.Value())
	}
	if view.focusIndex != 1 {
		t.Fatalf("expected rows field to be focused when table is prefilled, got focus index %d", view.focusIndex)
	}
}

func TestMainModel_CurrentMockDataTarget_PrefersResultsTable(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	if m.err != nil {
		t.Fatalf("NewMainModel failed: %v", m.err)
	}

	m.browserView.currentSchema = "public"
	m.browserView.filteredTables = []engine.StorageUnit{{Name: "users"}}
	m.browserView.selectedIndex = 0

	m.resultsView.schema = "analytics"
	m.resultsView.tableName = "events"

	schema, table := m.currentMockDataTarget()
	if schema != "analytics" || table != "events" {
		t.Fatalf("expected results view target analytics.events, got %s.%s", schema, table)
	}
}

func TestMainModel_Update_AltMOpensMockData(t *testing.T) {
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

	m.mode = ViewBrowser
	m.browserView.currentSchema = ""
	m.browserView.filteredTables = []engine.StorageUnit{{Name: "users"}}
	m.browserView.selectedIndex = 0

	msg := tea.KeyPressMsg{Code: 'm', Mod: tea.ModAlt}
	_, _ = m.Update(msg)

	if m.mode != ViewMockData {
		t.Fatalf("expected mode ViewMockData after alt+m, got %v", m.mode)
	}
	if m.mockDataView.tableInput.Value() != "users" {
		t.Fatalf("expected selected browser table to prefill mock-data target, got %q", m.mockDataView.tableInput.Value())
	}
}
