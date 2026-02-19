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
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clidey/whodb/core/src/engine"
)

func setupColumnsViewTest(t *testing.T) (*ColumnsView, func()) {
	t.Helper()

	setupTestEnv(t)

	parent := NewMainModel()
	if parent.err != nil {
		t.Fatalf("Failed to create MainModel: %v", parent.err)
	}

	cleanup := func() {}

	return parent.columnsView, cleanup
}

func TestNewColumnsView(t *testing.T) {
	v, cleanup := setupColumnsViewTest(t)
	defer cleanup()

	if v == nil {
		t.Fatal("NewColumnsView returned nil")
	}

	if len(v.columns) != 0 {
		t.Error("Expected empty columns initially")
	}

	if v.selected == nil {
		t.Error("Expected selected map to be initialized")
	}

	if v.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0, got %d", v.selectedIndex)
	}
}

func TestColumnsView_SetTableContext(t *testing.T) {
	v, cleanup := setupColumnsViewTest(t)
	defer cleanup()

	columns := []engine.Column{
		{Name: "id", Type: "integer"},
		{Name: "name", Type: "text"},
		{Name: "email", Type: "text"},
	}

	v.SetTableContext("public", "users", columns)

	if v.schema != "public" {
		t.Errorf("Expected schema 'public', got '%s'", v.schema)
	}

	if v.tableName != "users" {
		t.Errorf("Expected tableName 'users', got '%s'", v.tableName)
	}

	if len(v.columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(v.columns))
	}

	// All columns should be selected by default
	for _, col := range columns {
		if !v.selected[col.Name] {
			t.Errorf("Expected column '%s' to be selected by default", col.Name)
		}
	}
}

func TestColumnsView_SetTableContext_DifferentTable(t *testing.T) {
	v, cleanup := setupColumnsViewTest(t)
	defer cleanup()

	// Set first table
	columns1 := []engine.Column{{Name: "id", Type: "integer"}}
	v.SetTableContext("public", "users", columns1)
	v.selected["id"] = false // Deselect

	// Set different table - should reset
	columns2 := []engine.Column{{Name: "order_id", Type: "integer"}}
	v.SetTableContext("public", "orders", columns2)

	// New column should be selected
	if !v.selected["order_id"] {
		t.Error("Expected new column to be selected when switching tables")
	}
}

func TestColumnsView_ToggleSelection_Space(t *testing.T) {
	v, cleanup := setupColumnsViewTest(t)
	defer cleanup()

	v.columns = []engine.Column{{Name: "id", Type: "integer"}}
	v.selected["id"] = true
	v.selectedIndex = 0

	// Toggle with space
	msg := tea.KeyMsg{Type: tea.KeySpace}
	v, _ = v.Update(msg)

	if v.selected["id"] {
		t.Error("Expected column to be deselected after space")
	}

	// Toggle again
	v, _ = v.Update(msg)

	if !v.selected["id"] {
		t.Error("Expected column to be selected after second space")
	}
}

func TestColumnsView_ToggleSelection_X(t *testing.T) {
	v, cleanup := setupColumnsViewTest(t)
	defer cleanup()

	v.columns = []engine.Column{{Name: "id", Type: "integer"}}
	v.selected["id"] = true
	v.selectedIndex = 0

	// Toggle with 'x'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	v, _ = v.Update(msg)

	if v.selected["id"] {
		t.Error("Expected column to be deselected after 'x'")
	}
}

func TestColumnsView_SelectAll(t *testing.T) {
	v, cleanup := setupColumnsViewTest(t)
	defer cleanup()

	v.columns = []engine.Column{
		{Name: "id", Type: "integer"},
		{Name: "name", Type: "text"},
	}
	v.selected["id"] = false
	v.selected["name"] = false

	// Press 'a' to select all
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	v, _ = v.Update(msg)

	if !v.selected["id"] || !v.selected["name"] {
		t.Error("Expected all columns to be selected after 'a'")
	}
}

func TestColumnsView_SelectNone(t *testing.T) {
	v, cleanup := setupColumnsViewTest(t)
	defer cleanup()

	v.columns = []engine.Column{
		{Name: "id", Type: "integer"},
		{Name: "name", Type: "text"},
	}
	v.selected["id"] = true
	v.selected["name"] = true

	// Press 'n' to select none
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	v, _ = v.Update(msg)

	if v.selected["id"] || v.selected["name"] {
		t.Error("Expected no columns to be selected after 'n'")
	}
}

func TestColumnsView_Navigation_UpDown(t *testing.T) {
	v, cleanup := setupColumnsViewTest(t)
	defer cleanup()

	v.columns = []engine.Column{
		{Name: "id", Type: "integer"},
		{Name: "name", Type: "text"},
		{Name: "email", Type: "text"},
	}
	v.selectedIndex = 0
	v.height = 50 // Large enough to not scroll

	// Down
	msg := tea.KeyMsg{Type: tea.KeyDown}
	v, _ = v.Update(msg)
	if v.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex 1 after down, got %d", v.selectedIndex)
	}

	// Down again
	v, _ = v.Update(msg)
	if v.selectedIndex != 2 {
		t.Errorf("Expected selectedIndex 2 after second down, got %d", v.selectedIndex)
	}

	// Down at end - should stay
	v, _ = v.Update(msg)
	if v.selectedIndex != 2 {
		t.Errorf("Expected selectedIndex to stay 2, got %d", v.selectedIndex)
	}

	// Up
	msg = tea.KeyMsg{Type: tea.KeyUp}
	v, _ = v.Update(msg)
	if v.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex 1 after up, got %d", v.selectedIndex)
	}
}

func TestColumnsView_Navigation_VimKeys(t *testing.T) {
	v, cleanup := setupColumnsViewTest(t)
	defer cleanup()

	v.columns = []engine.Column{
		{Name: "id", Type: "integer"},
		{Name: "name", Type: "text"},
	}
	v.selectedIndex = 0
	v.height = 50

	// 'j' goes down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	v, _ = v.Update(msg)
	if v.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex 1 after 'j', got %d", v.selectedIndex)
	}

	// 'k' goes up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	v, _ = v.Update(msg)
	if v.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0 after 'k', got %d", v.selectedIndex)
	}
}

func TestColumnsView_Escape(t *testing.T) {
	v, cleanup := setupColumnsViewTest(t)
	defer cleanup()

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.parent.mode != ViewResults {
		t.Errorf("Expected mode ViewResults after Esc, got %v", v.parent.mode)
	}
}

func TestColumnsView_WindowSizeMsg(t *testing.T) {
	v, cleanup := setupColumnsViewTest(t)
	defer cleanup()

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	v, _ = v.Update(msg)

	if v.height != 50 {
		t.Errorf("Expected height 50, got %d", v.height)
	}
}

func TestColumnsView_GetSelectedColumns(t *testing.T) {
	v, cleanup := setupColumnsViewTest(t)
	defer cleanup()

	v.columns = []engine.Column{
		{Name: "id", Type: "integer"},
		{Name: "name", Type: "text"},
		{Name: "email", Type: "text"},
	}
	v.selected["id"] = true
	v.selected["name"] = false
	v.selected["email"] = true

	result := v.getSelectedColumns()

	if len(result) != 2 {
		t.Errorf("Expected 2 selected columns, got %d", len(result))
	}

	// Order should match column order
	hasId := false
	hasEmail := false
	for _, col := range result {
		if col == "id" {
			hasId = true
		}
		if col == "email" {
			hasEmail = true
		}
	}

	if !hasId || !hasEmail {
		t.Error("Expected 'id' and 'email' in selected columns")
	}
}

func TestColumnsView_View(t *testing.T) {
	v, cleanup := setupColumnsViewTest(t)
	defer cleanup()

	v.schema = "public"
	v.tableName = "users"
	v.columns = []engine.Column{
		{Name: "id", Type: "integer"},
		{Name: "name", Type: "text"},
	}
	v.selected["id"] = true
	v.selected["name"] = false
	v.height = 50

	view := v.View()

	if !strings.Contains(view, "Select Columns") {
		t.Error("Expected title 'Select Columns'")
	}

	if !strings.Contains(view, "public.users") {
		t.Error("Expected table name in view")
	}

	if !strings.Contains(view, "1 of 2 columns selected") {
		t.Error("Expected selection count in view")
	}

	// Check for column display with checkbox
	if !strings.Contains(view, "[✓]") {
		t.Error("Expected checked checkbox for selected column")
	}

	if !strings.Contains(view, "[ ]") {
		t.Error("Expected unchecked checkbox for unselected column")
	}
}

func TestColumnsView_MouseScroll(t *testing.T) {
	v, cleanup := setupColumnsViewTest(t)
	defer cleanup()

	// Create many columns to enable scrolling
	v.columns = make([]engine.Column, 30)
	for i := 0; i < 30; i++ {
		v.columns[i] = engine.Column{Name: string(rune('a' + i)), Type: "text"}
	}
	v.height = 20 // Small height to require scrolling
	v.scrollOffset = 0

	// Mouse wheel down
	msg := tea.MouseMsg{Type: tea.MouseWheelDown}
	v, _ = v.Update(msg)

	if v.scrollOffset <= 0 {
		t.Error("Expected scrollOffset to increase after wheel down")
	}

	// Mouse wheel up
	initialOffset := v.scrollOffset
	msg = tea.MouseMsg{Type: tea.MouseWheelUp}
	v, _ = v.Update(msg)

	if v.scrollOffset >= initialOffset {
		t.Error("Expected scrollOffset to decrease after wheel up")
	}
}

func TestColumnsView_ScrollWithNavigation(t *testing.T) {
	v, cleanup := setupColumnsViewTest(t)
	defer cleanup()

	// Create columns that require scrolling
	v.columns = make([]engine.Column, 20)
	for i := 0; i < 20; i++ {
		v.columns[i] = engine.Column{Name: string(rune('a' + i)), Type: "text"}
	}
	v.height = 15 // Only fits ~3 items
	v.selectedIndex = 0
	v.scrollOffset = 0

	// Navigate down many times
	msg := tea.KeyMsg{Type: tea.KeyDown}
	for i := 0; i < 10; i++ {
		v, _ = v.Update(msg)
	}

	// Should have auto-scrolled
	if v.scrollOffset == 0 {
		t.Error("Expected scrollOffset to increase when navigating past visible area")
	}

	// Navigate back up
	msg = tea.KeyMsg{Type: tea.KeyUp}
	for i := 0; i < 15; i++ {
		v, _ = v.Update(msg)
	}

	// Should scroll back to top
	if v.scrollOffset > v.selectedIndex {
		t.Error("Expected scrollOffset to adjust when navigating above visible area")
	}
}

func TestColumnsView_View_EmptyColumns(t *testing.T) {
	v, cleanup := setupColumnsViewTest(t)
	defer cleanup()

	// Set empty columns - this previously caused a panic (startIdx = -1)
	v.columns = []engine.Column{}
	v.height = 30

	view := v.View()

	if !strings.Contains(view, "No columns") {
		t.Error("Expected 'No columns' message for empty columns")
	}
}
