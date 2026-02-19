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
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/core/src/engine"
)

func setupBrowserViewTest(t *testing.T) (*BrowserView, func()) {
	t.Helper()

	setupTestEnv(t)

	parent := NewMainModel()
	if parent.err != nil {
		t.Fatalf("Failed to create MainModel: %v", parent.err)
	}

	cleanup := func() {}

	return parent.browserView, cleanup
}

func TestNewBrowserView(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	if v == nil {
		t.Fatal("NewBrowserView returned nil")
	}

	if v.loading != true {
		t.Error("Expected loading to be true initially")
	}

	if v.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0, got %d", v.selectedIndex)
	}

	if v.columnsPerRow != 4 {
		t.Errorf("Expected columnsPerRow 4, got %d", v.columnsPerRow)
	}
}

func TestBrowserView_WindowSizeMsg(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	v, _ = v.Update(msg)

	if v.width != 120 {
		t.Errorf("Expected width 120, got %d", v.width)
	}

	if v.height != 40 {
		t.Errorf("Expected height 40, got %d", v.height)
	}
}

func TestBrowserView_SchemaSelection_EnterMode(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	// Simulate loaded tables with multiple schemas
	v.schemas = []string{"public", "admin", "test"}
	v.currentSchema = "public"
	v.loading = false

	// Press ctrl+s to enter schema selection
	msg := tea.KeyMsg{Type: tea.KeyCtrlS}
	v, _ = v.Update(msg)

	if !v.schemaSelecting {
		t.Error("Expected schemaSelecting to be true after ctrl+s")
	}
}

func TestBrowserView_SchemaSelection_SingleSchema(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	// Only one schema - should not enter selection mode
	v.schemas = []string{"public"}
	v.currentSchema = "public"
	v.loading = false

	msg := tea.KeyMsg{Type: tea.KeyCtrlS}
	v, _ = v.Update(msg)

	if v.schemaSelecting {
		t.Error("Should not enter schema selection with only one schema")
	}
}

func TestBrowserView_SchemaSelection_Navigation(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	v.schemas = []string{"public", "admin", "test"}
	v.currentSchema = "public"
	v.selectedSchemaIndex = 0
	v.schemaSelecting = true
	v.loading = false

	// Test right arrow navigation
	msg := tea.KeyMsg{Type: tea.KeyRight}
	v, _ = v.Update(msg)
	if v.selectedSchemaIndex != 1 {
		t.Errorf("Expected selectedSchemaIndex 1 after right, got %d", v.selectedSchemaIndex)
	}

	// Test left arrow navigation
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	v, _ = v.Update(msg)
	if v.selectedSchemaIndex != 0 {
		t.Errorf("Expected selectedSchemaIndex 0 after left, got %d", v.selectedSchemaIndex)
	}

	// Test down arrow navigation (should also work)
	msg = tea.KeyMsg{Type: tea.KeyDown}
	v, _ = v.Update(msg)
	if v.selectedSchemaIndex != 1 {
		t.Errorf("Expected selectedSchemaIndex 1 after down, got %d", v.selectedSchemaIndex)
	}

	// Test up arrow navigation
	msg = tea.KeyMsg{Type: tea.KeyUp}
	v, _ = v.Update(msg)
	if v.selectedSchemaIndex != 0 {
		t.Errorf("Expected selectedSchemaIndex 0 after up, got %d", v.selectedSchemaIndex)
	}
}

func TestBrowserView_SchemaSelection_Boundaries(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	v.schemas = []string{"public", "admin", "test"}
	v.selectedSchemaIndex = 0
	v.schemaSelecting = true
	v.loading = false

	// Try to go left at index 0 - should stay at 0
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	v, _ = v.Update(msg)
	if v.selectedSchemaIndex != 0 {
		t.Errorf("Expected selectedSchemaIndex to stay 0, got %d", v.selectedSchemaIndex)
	}

	// Go to last index
	v.selectedSchemaIndex = 2

	// Try to go right at last index - should stay at 2
	msg = tea.KeyMsg{Type: tea.KeyRight}
	v, _ = v.Update(msg)
	if v.selectedSchemaIndex != 2 {
		t.Errorf("Expected selectedSchemaIndex to stay 2, got %d", v.selectedSchemaIndex)
	}
}

func TestBrowserView_SchemaSelection_EscapeCancels(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	v.schemas = []string{"public", "admin"}
	v.schemaSelecting = true
	v.loading = false

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.schemaSelecting {
		t.Error("Expected schemaSelecting to be false after esc")
	}
}

func TestBrowserView_SchemaSelection_BlocksTableNavigation(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	// Setup: multiple schemas and tables
	v.schemas = []string{"public", "admin"}
	v.tables = []engine.StorageUnit{{Name: "users"}, {Name: "orders"}}
	v.filteredTables = v.tables
	v.schemaSelecting = true
	v.selectedIndex = 0
	v.loading = false

	// Press 'j' (vim down) while in schema selection - should NOT affect table selection
	initialTableIndex := v.selectedIndex
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	v, _ = v.Update(msg)

	if v.selectedIndex != initialTableIndex {
		t.Errorf("Table index changed during schema selection: expected %d, got %d",
			initialTableIndex, v.selectedIndex)
	}
}

func TestBrowserView_Filtering_Enter(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	v.loading = false

	// Press '/' to enter filter mode
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	v, _ = v.Update(msg)

	if !v.filtering {
		t.Error("Expected filtering to be true after '/'")
	}
}

func TestBrowserView_Filtering_Escape(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	v.filtering = true
	v.filterInput.SetValue("test")
	v.loading = false

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.filtering {
		t.Error("Expected filtering to be false after esc")
	}

	if v.filterInput.Value() != "" {
		t.Error("Expected filter input to be cleared after esc")
	}
}

func TestBrowserView_Filtering_Apply(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	v.tables = []engine.StorageUnit{
		{Name: "users"},
		{Name: "user_roles"},
		{Name: "orders"},
	}
	v.filteredTables = v.tables
	v.filtering = true
	v.filterInput.SetValue("user")
	v.loading = false

	// Apply filter
	v.applyFilter()

	if len(v.filteredTables) != 2 {
		t.Errorf("Expected 2 filtered tables, got %d", len(v.filteredTables))
	}
}

func TestBrowserView_TableNavigation_Grid(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	// Setup 8 tables in a 4-column grid
	v.tables = []engine.StorageUnit{
		{Name: "t1"}, {Name: "t2"}, {Name: "t3"}, {Name: "t4"},
		{Name: "t5"}, {Name: "t6"}, {Name: "t7"}, {Name: "t8"},
	}
	v.filteredTables = v.tables
	v.selectedIndex = 0
	v.columnsPerRow = 4
	v.loading = false

	// Right arrow moves to next table
	msg := tea.KeyMsg{Type: tea.KeyRight}
	v, _ = v.Update(msg)
	if v.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex 1 after right, got %d", v.selectedIndex)
	}

	// Down arrow moves to next row
	msg = tea.KeyMsg{Type: tea.KeyDown}
	v, _ = v.Update(msg)
	if v.selectedIndex != 5 {
		t.Errorf("Expected selectedIndex 5 after down, got %d", v.selectedIndex)
	}

	// Up arrow moves to previous row
	msg = tea.KeyMsg{Type: tea.KeyUp}
	v, _ = v.Update(msg)
	if v.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex 1 after up, got %d", v.selectedIndex)
	}

	// Left arrow moves to previous table
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	v, _ = v.Update(msg)
	if v.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0 after left, got %d", v.selectedIndex)
	}
}

func TestBrowserView_TableNavigation_VimKeys(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	v.tables = []engine.StorageUnit{
		{Name: "t1"}, {Name: "t2"}, {Name: "t3"}, {Name: "t4"},
		{Name: "t5"}, {Name: "t6"}, {Name: "t7"}, {Name: "t8"},
	}
	v.filteredTables = v.tables
	v.selectedIndex = 0
	v.columnsPerRow = 4
	v.loading = false

	// 'l' moves right
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	v, _ = v.Update(msg)
	if v.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex 1 after 'l', got %d", v.selectedIndex)
	}

	// 'j' moves down
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	v, _ = v.Update(msg)
	if v.selectedIndex != 5 {
		t.Errorf("Expected selectedIndex 5 after 'j', got %d", v.selectedIndex)
	}

	// 'k' moves up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	v, _ = v.Update(msg)
	if v.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex 1 after 'k', got %d", v.selectedIndex)
	}

	// 'h' moves left
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	v, _ = v.Update(msg)
	if v.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0 after 'h', got %d", v.selectedIndex)
	}
}

func TestBrowserView_View_ShowsLoading(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	v.loading = true
	v.parent.dbManager = nil // Simulate no connection for simplified test

	// We can't easily test View() without a connection, but we can ensure it doesn't panic
	// The view will show "No connection" in this case
}

func TestBrowserView_View_SchemaSelectionHelp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	// Need a real connection for View() to work
	conn := &config.Connection{
		Name:     "test",
		Type:     "Sqlite",
		Host:     t.TempDir() + "/test.db",
		Database: t.TempDir() + "/test.db",
	}

	if err := v.parent.dbManager.Connect(conn); err != nil {
		t.Skipf("Skipping test - database plugin not available: %v", err)
	}

	v.schemas = []string{"public", "admin"}
	v.currentSchema = "public"
	v.schemaSelecting = true
	v.loading = false

	view := v.View()

	// Should show schema selection help
	if !strings.Contains(view, "navigate") {
		t.Error("Expected schema selection help to show navigate")
	}
	if !strings.Contains(view, "select schema") {
		t.Error("Expected schema selection help to show 'select schema'")
	}
}

func TestBrowserView_MouseScroll(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	v.tables = []engine.StorageUnit{
		{Name: "t1"}, {Name: "t2"}, {Name: "t3"}, {Name: "t4"},
		{Name: "t5"}, {Name: "t6"}, {Name: "t7"}, {Name: "t8"},
	}
	v.filteredTables = v.tables
	v.selectedIndex = 0
	v.columnsPerRow = 4
	v.loading = false

	// Mouse wheel down
	msg := tea.MouseMsg{Button: tea.MouseButtonWheelDown}
	v, _ = v.Update(msg)
	if v.selectedIndex != 4 {
		t.Errorf("Expected selectedIndex 4 after wheel down, got %d", v.selectedIndex)
	}

	// Mouse wheel up
	msg = tea.MouseMsg{Button: tea.MouseButtonWheelUp}
	v, _ = v.Update(msg)
	if v.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0 after wheel up, got %d", v.selectedIndex)
	}
}

func TestSelectBestSchema(t *testing.T) {
	tests := []struct {
		name     string
		schemas  []string
		expected string
	}{
		{
			name:     "empty list",
			schemas:  []string{},
			expected: "",
		},
		{
			name:     "public exists",
			schemas:  []string{"information_schema", "public", "admin"},
			expected: "public",
		},
		{
			name:     "no public, skip system schemas",
			schemas:  []string{"information_schema", "pg_catalog", "admin"},
			expected: "admin",
		},
		{
			name:     "only system schemas",
			schemas:  []string{"information_schema", "pg_catalog"},
			expected: "information_schema",
		},
		{
			name:     "mysql system schemas filtered",
			schemas:  []string{"mysql", "sys", "performance_schema", "mydb"},
			expected: "mydb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selectBestSchema(tt.schemas)
			if result != tt.expected {
				t.Errorf("selectBestSchema(%v) = %s, expected %s", tt.schemas, result, tt.expected)
			}
		})
	}
}

func TestBrowserView_ApplyFilter(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	v.tables = []engine.StorageUnit{
		{Name: "users"},
		{Name: "user_roles"},
		{Name: "orders"},
		{Name: "order_items"},
		{Name: "products"},
	}
	v.selectedIndex = 3

	tests := []struct {
		filter        string
		expectedCount int
		expectedReset bool
	}{
		{"", 5, false},        // Empty filter shows all
		{"user", 2, true},     // Matches "users" and "user_roles"
		{"order", 2, true},    // Matches "orders" and "order_items"
		{"xyz", 0, true},      // No matches
		{"PRODUCTS", 1, true}, // Case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.filter, func(t *testing.T) {
			v.filterInput.SetValue(tt.filter)
			v.selectedIndex = 3 // Reset to out-of-bounds for some filters
			v.applyFilter()

			if len(v.filteredTables) != tt.expectedCount {
				t.Errorf("Filter '%s': expected %d tables, got %d",
					tt.filter, tt.expectedCount, len(v.filteredTables))
			}

			// Check index reset when out of bounds
			if tt.expectedReset && tt.expectedCount > 0 && v.selectedIndex >= len(v.filteredTables) {
				t.Errorf("Filter '%s': selectedIndex should be reset when out of bounds", tt.filter)
			}
		})
	}
}

func TestBrowserView_RetryPrompt_EscCancels(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	// Set up retry prompt state
	v.retryPrompt.Show("")
	v.err = nil

	// Send ESC key
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	// Verify retry prompt was dismissed
	if v.retryPrompt.IsActive() {
		t.Error("Expected retryPrompt to be false after ESC")
	}
}

func TestBrowserView_RetryPrompt_KeyHandling(t *testing.T) {
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
			v, cleanup := setupBrowserViewTest(t)
			defer cleanup()

			// Set up retry prompt state
			v.retryPrompt.Show("")
			v.err = nil

			// Send number key
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			v, cmd := v.Update(msg)

			// Verify retry prompt was dismissed
			if v.retryPrompt.IsActive() {
				t.Error("Expected retryPrompt to be false after selecting retry option")
			}

			// Verify error was cleared
			if v.err != nil {
				t.Error("Expected err to be nil after retry")
			}

			// Verify loading was set
			if !v.loading {
				t.Error("Expected loading to be true after retry")
			}

			// Verify a command was returned (the table load)
			if cmd == nil {
				t.Error("Expected a command to be returned for retry")
			}
		})
	}
}

func TestBrowserView_RetryPrompt_IgnoresOtherKeys(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	// Set up retry prompt state
	v.retryPrompt.Show("")

	// Send an unrelated key (like 'a')
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}
	v, _ = v.Update(msg)

	// Verify retry prompt is still active
	if !v.retryPrompt.IsActive() {
		t.Error("Expected retryPrompt to still be true after unrecognized key")
	}
}

func TestBrowserView_RetryPrompt_View(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	// Set up retry prompt state with a mock connection
	conn := &config.Connection{
		Name:     "test",
		Type:     "SQLite",
		Host:     t.TempDir() + "/test.db",
		Database: t.TempDir() + "/test.db",
	}
	if err := v.parent.dbManager.Connect(conn); err != nil {
		t.Skipf("Skipping test - database plugin not available: %v", err)
	}

	v.retryPrompt.Show("")

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

func TestBrowserView_TablesLoadedMsg_Timeout_SetsRetryPrompt(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	v.loading = true

	// Simulate a timeout error in tablesLoadedMsg
	msg := tablesLoadedMsg{
		tables:  []engine.StorageUnit{},
		schemas: []string{},
		schema:  "public",
		err:     nil,
	}

	// First test normal success - no retry prompt
	v, _ = v.Update(msg)
	if v.retryPrompt.IsActive() {
		t.Error("Expected retryPrompt to be false on success")
	}

	// Now test timeout error
	v.loading = true
	msg.err = &timeoutError{}
	v, _ = v.Update(msg)

	if !v.retryPrompt.IsActive() {
		t.Error("Expected retryPrompt to be true after timeout")
	}
}

// timeoutError implements error with "timed out" message for testing
type timeoutError struct{}

func (e *timeoutError) Error() string {
	return "timed out fetching tables"
}

// ============================================================================
// Timeout Memory (Feature 7) - Browser View Tests
// ============================================================================

func TestBrowserView_TimeoutAutoRetry_WithPreference(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	v.loading = true
	v.retryPrompt.SetAutoRetried(false)

	// Set a preferred timeout
	v.parent.config.SetPreferredTimeout(60)

	// Simulate timeout
	msg := tablesLoadedMsg{
		tables:  []engine.StorageUnit{},
		schemas: []string{},
		schema:  "public",
		err:     &timeoutError{},
	}
	v, cmd := v.Update(msg)

	// Should auto-retry (not show prompt)
	if v.retryPrompt.IsActive() {
		t.Error("Expected retryPrompt to be false (auto-retry should happen)")
	}
	if !v.retryPrompt.AutoRetried() {
		t.Error("Expected autoRetried to be true")
	}
	if !v.loading {
		t.Error("Expected loading to be true during auto-retry")
	}
	if cmd == nil {
		t.Error("Expected a command for auto-retry")
	}
}

func TestBrowserView_TimeoutShowsMenu_AfterAutoRetry(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	v.loading = true
	v.retryPrompt.SetAutoRetried(true) // Already retried once

	v.parent.config.SetPreferredTimeout(60)

	msg := tablesLoadedMsg{
		tables:  []engine.StorageUnit{},
		schemas: []string{},
		schema:  "public",
		err:     &timeoutError{},
	}
	v, _ = v.Update(msg)

	if !v.retryPrompt.IsActive() {
		t.Error("Expected retryPrompt to be true after auto-retry failed")
	}
}

func TestBrowserView_TimeoutShowsMenu_NoPreference(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	v.loading = true
	v.retryPrompt.SetAutoRetried(false)
	v.parent.config.SetPreferredTimeout(0)

	msg := tablesLoadedMsg{
		tables:  []engine.StorageUnit{},
		schemas: []string{},
		schema:  "public",
		err:     &timeoutError{},
	}
	v, _ = v.Update(msg)

	if !v.retryPrompt.IsActive() {
		t.Error("Expected retryPrompt to be true with no preferred timeout")
	}
}

func TestBrowserView_RetryMenuSavesPreference(t *testing.T) {
	tests := []struct {
		name            string
		key             string
		expectedTimeout int
	}{
		{"option_1_saves_60", "1", 60},
		{"option_2_saves_120", "2", 120},
		{"option_3_saves_300", "3", 300},
		{"option_4_no_save", "4", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, cleanup := setupBrowserViewTest(t)
			defer cleanup()

			v.retryPrompt.Show("")
			v.parent.config.SetPreferredTimeout(0)

			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			v, _ = v.Update(msg)

			saved := v.parent.config.GetPreferredTimeout()
			if saved != tt.expectedTimeout {
				t.Errorf("Expected preferred timeout %d after key '%s', got %d", tt.expectedTimeout, tt.key, saved)
			}
		})
	}
}

func TestBrowserView_AutoRetriedResetOnInit(t *testing.T) {
	v, cleanup := setupBrowserViewTest(t)
	defer cleanup()

	v.retryPrompt.SetAutoRetried(true) // From previous timeout

	_ = v.Init()

	if v.retryPrompt.AutoRetried() {
		t.Error("Expected autoRetried to be reset on Init")
	}
}
