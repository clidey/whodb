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
)

func setupConnectionViewTest(t *testing.T) (*ConnectionView, func()) {
	t.Helper()

	setupTestEnv(t)

	clearedEnv := make(map[string]string)
	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		key := parts[0]
		if strings.HasPrefix(key, "WHODB_") && !strings.HasPrefix(key, "WHODB_CLI_") {
			clearedEnv[key] = os.Getenv(key)
			os.Unsetenv(key)
		}
	}

	parent := NewMainModel()
	if parent.err != nil {
		t.Fatalf("Failed to create MainModel: %v", parent.err)
	}

	cleanup := func() {
		for key, value := range clearedEnv {
			os.Setenv(key, value)
		}
	}

	return parent.connectionView, cleanup
}

func TestNewConnectionView_NoConnections(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	if v == nil {
		t.Fatal("NewConnectionView returned nil")
	}

	// With no saved connections, should start in form mode
	if v.mode != "form" {
		t.Errorf("Expected mode 'form' with no connections, got '%s'", v.mode)
	}

	// Database type selector should be focused first
	if v.focusIndex != 7 {
		t.Errorf("Expected focusIndex 7 (db type), got %d", v.focusIndex)
	}
}

func TestNewConnectionView_WithConnections(t *testing.T) {
	setupTestEnv(t)

	// Create config with connections
	cfg, _ := config.LoadConfig()
	cfg.AddConnection(config.Connection{Name: "test", Type: "Postgres", Host: "localhost"})
	cfg.Save()

	parent := NewMainModel()
	if parent.err != nil {
		t.Fatalf("Failed to create MainModel: %v", parent.err)
	}

	v := parent.connectionView

	// With saved connections, should start in list mode
	if v.mode != "list" {
		t.Errorf("Expected mode 'list' with connections, got '%s'", v.mode)
	}
}

func TestConnectionView_ListMode_Navigation_Tab(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "list"
	// Add some items for navigation
	v.parent.config.AddConnection(config.Connection{Name: "conn1"})
	v.parent.config.AddConnection(config.Connection{Name: "conn2"})
	_ = v.parent.config.Save()
	v.refreshList()

	// Tab moves down
	msg := tea.KeyMsg{Type: tea.KeyTab}
	v, _ = v.Update(msg)

	// Just ensure no panic - list handles internal state
}

func TestConnectionView_ListMode_Navigation_ShiftTab(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "list"
	v.parent.config.AddConnection(config.Connection{Name: "conn1"})
	v.parent.config.AddConnection(config.Connection{Name: "conn2"})
	_ = v.parent.config.Save()
	v.refreshList()

	// Shift+Tab moves up
	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	v, _ = v.Update(msg)

	// Just ensure no panic
}

func TestConnectionView_ListMode_NewConnection(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "list"

	// Press 'n' to create new connection
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	v, _ = v.Update(msg)

	if v.mode != "form" {
		t.Errorf("Expected mode 'form' after 'n', got '%s'", v.mode)
	}
}

func TestConnectionView_ListMode_Delete(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "list"
	v.parent.config.AddConnection(config.Connection{Name: "to-delete", Type: "Postgres"})
	_ = v.parent.config.Save()
	v.refreshList()

	initialCount := len(v.parent.config.Connections)

	// Press 'd' to delete
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	v, _ = v.Update(msg)

	if len(v.parent.config.Connections) != initialCount-1 {
		t.Error("Expected connection to be deleted")
	}
}

func TestConnectionView_ListMode_EscConfirmation(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "list"
	v.escPressed = false

	// First ESC - should show confirmation
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, cmd := v.Update(msg)

	if !v.escPressed {
		t.Error("Expected escPressed to be true after first ESC")
	}

	if v.escTimeoutSecs != 3 {
		t.Errorf("Expected escTimeoutSecs 3, got %d", v.escTimeoutSecs)
	}

	if cmd == nil {
		t.Error("Expected tick command after first ESC")
	}
}

func TestConnectionView_ListMode_EscConfirmation_SecondEsc(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "list"
	v.escPressed = true
	v.escTimeoutSecs = 2

	// Second ESC - should quit
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := v.Update(msg)

	// cmd should be tea.Quit
	if cmd == nil {
		t.Error("Expected quit command after second ESC")
	}
}

func TestConnectionView_ListMode_EscTimeoutTick(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "list"
	v.escPressed = true
	v.escTimeoutSecs = 2

	// Simulate tick
	msg := escTimeoutTickMsg{}
	v, _ = v.Update(msg)

	if v.escTimeoutSecs != 1 {
		t.Errorf("Expected escTimeoutSecs 1 after tick, got %d", v.escTimeoutSecs)
	}

	// Another tick to reach 0
	v, _ = v.Update(msg)

	if v.escPressed {
		t.Error("Expected escPressed to be false after timeout")
	}
}

func TestConnectionView_FormMode_NextInput(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.focusIndex = 0 // Name field

	// Tab/Down moves to next input (Postgres: 7→0→1→2→3→4→5→6→8)
	msg := tea.KeyMsg{Type: tea.KeyTab}
	v, _ = v.Update(msg)

	if v.focusIndex != 1 {
		t.Errorf("Expected focusIndex 1 after Tab from 0, got %d", v.focusIndex)
	}
}

func TestConnectionView_FormMode_PrevInput(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.focusIndex = 2

	// Shift+Tab/Up moves to previous input
	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	v, _ = v.Update(msg)

	if v.focusIndex != 1 {
		t.Errorf("Expected focusIndex 1 after Shift+Tab, got %d", v.focusIndex)
	}
}

func TestConnectionView_FormMode_WrapAround(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.focusIndex = 8 // Last position (Connect button)

	// Tab from last wraps to first (db type selector)
	v.nextInput()

	if v.focusIndex != 7 {
		t.Errorf("Expected focusIndex 7 (db type) after wrap, got %d", v.focusIndex)
	}

	// Shift+Tab from db type wraps to last (connect button)
	v.prevInput()

	if v.focusIndex != 8 {
		t.Errorf("Expected focusIndex 8 after backward wrap, got %d", v.focusIndex)
	}
}

func TestConnectionView_FormMode_DbTypeSelection(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.focusIndex = 7 // Database type field
	v.dbTypeIndex = 0

	// Right arrow cycles through types
	msg := tea.KeyMsg{Type: tea.KeyRight}
	v, _ = v.Update(msg)

	if v.dbTypeIndex != 1 {
		t.Errorf("Expected dbTypeIndex 1 after Right, got %d", v.dbTypeIndex)
	}

	// Left arrow goes back
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	v, _ = v.Update(msg)

	if v.dbTypeIndex != 0 {
		t.Errorf("Expected dbTypeIndex 0 after Left, got %d", v.dbTypeIndex)
	}
}

func TestConnectionView_FormMode_DbTypeWrapAround(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.focusIndex = 7
	v.dbTypeIndex = 0

	// Left from first wraps to last
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	v, _ = v.Update(msg)

	if v.dbTypeIndex != len(v.dbTypes)-1 {
		t.Errorf("Expected dbTypeIndex %d (last), got %d", len(v.dbTypes)-1, v.dbTypeIndex)
	}

	// Right from last wraps to first
	msg = tea.KeyMsg{Type: tea.KeyRight}
	v, _ = v.Update(msg)

	if v.dbTypeIndex != 0 {
		t.Errorf("Expected dbTypeIndex 0 after wrap, got %d", v.dbTypeIndex)
	}
}

func TestConnectionView_FormMode_EscapeToList(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	// Add a connection so we can go back to list
	v.parent.config.AddConnection(config.Connection{Name: "test"})
	v.mode = "form"

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.mode != "list" {
		t.Errorf("Expected mode 'list' after Esc, got '%s'", v.mode)
	}
}

func TestConnectionView_FormMode_PasswordPrompt(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.focusIndex = 8         // Connect button
	v.inputs[4].SetValue("") // Empty password

	// Press Enter - should show password prompt
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	v, _ = v.Update(msg)

	if !v.awaitingPassword {
		t.Error("Expected awaitingPassword to be true with empty password")
	}
}

func TestConnectionView_FormMode_PasswordPrompt_Cancel(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.awaitingPassword = true
	v.passwordPrompt.SetValue("secret")

	// Press Esc to cancel
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.awaitingPassword {
		t.Error("Expected awaitingPassword to be false after Esc")
	}

	if v.passwordPrompt.Value() != "" {
		t.Error("Expected password prompt to be cleared")
	}
}

func TestConnectionView_ResetForm(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	// Set some values
	v.inputs[0].SetValue("test connection")
	v.inputs[1].SetValue("myhost")
	v.focusIndex = 5
	v.dbTypeIndex = 3

	v.resetForm()

	// All inputs should be cleared
	for i, input := range v.inputs {
		if input.Value() != "" {
			t.Errorf("Expected input %d to be cleared", i)
		}
	}

	if v.focusIndex != 7 {
		t.Errorf("Expected focusIndex 7 (db type) after reset, got %d", v.focusIndex)
	}

	if v.dbTypeIndex != 0 {
		t.Errorf("Expected dbTypeIndex 0 after reset, got %d", v.dbTypeIndex)
	}
}

func TestConnectionView_GetDefaultPort(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	tests := []struct {
		dbType   string
		expected int
	}{
		{"Postgres", 5432},
		{"MySQL", 3306},
		{"MariaDB", 3306},
		{"MongoDB", 27017},
		{"Redis", 6379},
		{"ClickHouse", 9000},
		{"ElasticSearch", 9200},
		{"SQLite", 0},
		{"Unknown", 5432},
	}

	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			result := v.getDefaultPort(tt.dbType)
			if result != tt.expected {
				t.Errorf("getDefaultPort(%s) = %d, expected %d", tt.dbType, result, tt.expected)
			}
		})
	}
}

func TestConnectionView_UpdatePortPlaceholder(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	// Select MySQL
	v.dbTypeIndex = 1 // MySQL in the list
	v.updatePortPlaceholder()

	if v.inputs[2].Placeholder != "3306" {
		t.Errorf("Expected port placeholder '3306' for MySQL, got '%s'", v.inputs[2].Placeholder)
	}
}

func TestConnectionView_WindowSizeMsg(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "list"

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	v, _ = v.Update(msg)

	// List size should be updated (height - 8)
	// Can't easily verify internal list state, but ensure no panic
}

func TestConnectionView_MouseScroll_List(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "list"
	v.parent.config.AddConnection(config.Connection{Name: "conn1"})
	v.parent.config.AddConnection(config.Connection{Name: "conn2"})
	_ = v.parent.config.Save()
	v.refreshList()

	// Mouse wheel down
	msg := tea.MouseMsg{Button: tea.MouseButtonWheelDown}
	v, _ = v.Update(msg)

	// Mouse wheel up
	msg = tea.MouseMsg{Button: tea.MouseButtonWheelUp}
	v, _ = v.Update(msg)

	// Just ensure no panic
}

func TestConnectionView_MouseScroll_Form(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.focusIndex = 0 // Name field (Postgres)

	// Mouse wheel down moves to next
	msg := tea.MouseMsg{Button: tea.MouseButtonWheelDown}
	v, _ = v.Update(msg)

	if v.focusIndex != 1 {
		t.Errorf("Expected focusIndex 1 after wheel down, got %d", v.focusIndex)
	}

	// Mouse wheel up moves back
	msg = tea.MouseMsg{Button: tea.MouseButtonWheelUp}
	v, _ = v.Update(msg)

	if v.focusIndex != 0 {
		t.Errorf("Expected focusIndex 0 after wheel up, got %d", v.focusIndex)
	}
}

func TestConnectionView_View_List(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "list"
	v.parent.config.AddConnection(config.Connection{Name: "my-db", Type: "Postgres", Host: "localhost"})
	_ = v.parent.config.Save()
	v.refreshList()

	view := v.View()

	if !strings.Contains(view, "Welcome to WhoDB") {
		t.Error("Expected 'Welcome to WhoDB' title")
	}

	if !strings.Contains(view, "new") {
		t.Error("Expected 'new' help text")
	}

	if !strings.Contains(view, "delete") {
		t.Error("Expected 'delete' help text")
	}
}

func TestConnectionView_View_ListWithEscConfirmation(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "list"
	v.escPressed = true
	v.escTimeoutSecs = 2

	view := v.View()

	if !strings.Contains(view, "Press ESC again to quit") {
		t.Error("Expected ESC confirmation message")
	}

	if !strings.Contains(view, "2s") {
		t.Error("Expected countdown in confirmation message")
	}
}

func TestConnectionView_View_Form(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"

	view := v.View()

	if !strings.Contains(view, "New Database Connection") {
		t.Error("Expected 'New Database Connection' title")
	}

	if !strings.Contains(view, "Connection Name") {
		t.Error("Expected 'Connection Name' field")
	}

	if !strings.Contains(view, "Host") {
		t.Error("Expected 'Host' field")
	}

	if !strings.Contains(view, "Port") {
		t.Error("Expected 'Port' field")
	}

	if !strings.Contains(view, "Database Type") {
		t.Error("Expected 'Database Type' field")
	}

	if !strings.Contains(view, "Connect") {
		t.Error("Expected 'Connect' button")
	}
}

func TestConnectionView_View_FormWithError(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.connError = os.ErrInvalid

	view := v.View()

	if !strings.Contains(view, "invalid") {
		t.Error("Expected error message in view")
	}
}

func TestConnectionView_View_PasswordPrompt(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.awaitingPassword = true

	view := v.View()

	if !strings.Contains(view, "Enter Password") {
		t.Error("Expected 'Enter Password' title")
	}

	if !strings.Contains(view, "confirm") {
		t.Error("Expected 'confirm' help text")
	}

	if !strings.Contains(view, "cancel") {
		t.Error("Expected 'cancel' help text")
	}
}

func TestConnectionItem_Methods(t *testing.T) {
	conn := config.Connection{
		Name: "my-connection",
		Type: "Postgres",
		Host: "db.example.com",
	}

	item := connectionItem{conn: conn}

	if item.Title() != "my-connection" {
		t.Errorf("Expected Title 'my-connection', got '%s'", item.Title())
	}

	if item.Description() != "Postgres@db.example.com" {
		t.Errorf("Expected Description 'Postgres@db.example.com', got '%s'", item.Description())
	}

	if item.FilterValue() != "my-connection" {
		t.Errorf("Expected FilterValue 'my-connection', got '%s'", item.FilterValue())
	}
}

func TestConnectionDelegate_Methods(t *testing.T) {
	d := connectionDelegate{}

	if d.Height() != 2 {
		t.Errorf("Expected Height 2, got %d", d.Height())
	}

	if d.Spacing() != 1 {
		t.Errorf("Expected Spacing 1, got %d", d.Spacing())
	}
}

func TestConnectionView_ConnectionResultMsg_Success(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.connecting = true

	// Simulate successful connection result
	// Note: This won't actually change mode since browserView.Init returns nil
	// But we can test error handling
	msg := connectionResultMsg{err: nil}
	v, _ = v.Update(msg)

	// Mode should change to browser
	if v.parent.mode != ViewBrowser {
		t.Errorf("Expected mode ViewBrowser after successful connection, got %v", v.parent.mode)
	}
}

func TestConnectionView_ConnectionResultMsg_Error(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.connecting = true

	msg := connectionResultMsg{err: os.ErrPermission}
	v, _ = v.Update(msg)

	if v.connecting {
		t.Error("Expected connecting to become false after error")
	}

	if v.connError == nil {
		t.Error("Expected connError to be set")
	}

	// Mode should stay in form
	if v.mode != "form" {
		t.Errorf("Expected mode 'form' after error, got '%s'", v.mode)
	}
}

// ============================================================================
// Smart Connection Form (Feature 5) Tests
// ============================================================================

func TestGetVisibleFields_AllTypes(t *testing.T) {
	tests := []struct {
		dbType   string
		expected []int
	}{
		{"Postgres", []int{0, 1, 2, 3, 4, 5, 6}},
		{"MySQL", []int{0, 1, 2, 3, 4, 5, 6}},
		{"MariaDB", []int{0, 1, 2, 3, 4, 5, 6}},
		{"ClickHouse", []int{0, 1, 2, 3, 4, 5, 6}},
		{"SQLite", []int{0, 5}},
		{"MongoDB", []int{0, 1, 2, 3, 4, 5}},
		{"Redis", []int{0, 1, 2, 4, 5}},
		{"ElasticSearch", []int{0, 1, 2, 3, 4}},
	}

	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			result := getVisibleFields(tt.dbType)
			if len(result) != len(tt.expected) {
				t.Errorf("getVisibleFields(%s) returned %d fields, expected %d", tt.dbType, len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("getVisibleFields(%s)[%d] = %d, expected %d", tt.dbType, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestConnectionView_IsFieldVisible(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	// Default type is Postgres - all fields visible
	for i := 0; i < 7; i++ {
		if !v.isFieldVisible(i) {
			t.Errorf("Expected field %d to be visible for Postgres", i)
		}
	}

	// Switch to SQLite
	v.visibleFields = getVisibleFields("SQLite")
	if !v.isFieldVisible(0) {
		t.Error("Expected field 0 (name) to be visible for SQLite")
	}
	if v.isFieldVisible(1) {
		t.Error("Expected field 1 (host) to be hidden for SQLite")
	}
	if v.isFieldVisible(2) {
		t.Error("Expected field 2 (port) to be hidden for SQLite")
	}
	if v.isFieldVisible(3) {
		t.Error("Expected field 3 (username) to be hidden for SQLite")
	}
	if v.isFieldVisible(4) {
		t.Error("Expected field 4 (password) to be hidden for SQLite")
	}
	if !v.isFieldVisible(5) {
		t.Error("Expected field 5 (database) to be visible for SQLite")
	}
	if v.isFieldVisible(6) {
		t.Error("Expected field 6 (schema) to be hidden for SQLite")
	}
}

func TestConnectionView_OnDbTypeChanged_UpdatesVisibility(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"

	// Switch to SQLite (index 2 in the dbTypes list)
	v.dbTypeIndex = 2 // SQLite
	v.onDbTypeChanged()

	if len(v.visibleFields) != 2 {
		t.Errorf("Expected 2 visible fields for SQLite, got %d", len(v.visibleFields))
	}

	// Database placeholder should change
	if v.inputs[5].Placeholder != "/path/to/database.db" {
		t.Errorf("Expected SQLite database placeholder, got '%s'", v.inputs[5].Placeholder)
	}

	// Switch back to Postgres
	v.dbTypeIndex = 0
	v.onDbTypeChanged()

	if len(v.visibleFields) != 7 {
		t.Errorf("Expected 7 visible fields for Postgres, got %d", len(v.visibleFields))
	}

	if v.inputs[5].Placeholder != "mydb" {
		t.Errorf("Expected 'mydb' database placeholder, got '%s'", v.inputs[5].Placeholder)
	}
}

func TestConnectionView_NavigationSkipsHiddenFields_SQLite(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.dbTypeIndex = 2 // SQLite
	v.onDbTypeChanged()

	// Focus order for SQLite: 7 (dbType) → 0 (name) → 5 (database) → 8 (connect) → wrap
	v.focusIndex = 7 // Start at db type

	// Next from dbType(7) goes to name(0)
	v.nextInput()
	if v.focusIndex != 0 {
		t.Errorf("Expected focusIndex 0 (name) after Tab from dbType in SQLite, got %d", v.focusIndex)
	}

	// Next from name(0) should skip host(1), port(2), username(3), password(4) and go to database(5)
	v.nextInput()
	if v.focusIndex != 5 {
		t.Errorf("Expected focusIndex 5 (database) after Tab from name in SQLite, got %d", v.focusIndex)
	}

	// Next from database(5) goes to connect(8)
	v.nextInput()
	if v.focusIndex != 8 {
		t.Errorf("Expected focusIndex 8 (connect) after Tab from database in SQLite, got %d", v.focusIndex)
	}

	// Next from connect(8) wraps to dbType(7)
	v.nextInput()
	if v.focusIndex != 7 {
		t.Errorf("Expected focusIndex 7 (dbType) after wrap, got %d", v.focusIndex)
	}
}

func TestConnectionView_PrevNavigationSkipsHiddenFields_SQLite(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.dbTypeIndex = 2 // SQLite
	v.onDbTypeChanged()

	// Focus order for SQLite: 7 → 0 → 5 → 8
	// Reverse: 8 → 5 → 0 → 7
	v.focusIndex = 5 // Start at database

	// Prev from database(5) should go to name(0)
	v.prevInput()
	if v.focusIndex != 0 {
		t.Errorf("Expected focusIndex 0 (name) after Shift+Tab from database in SQLite, got %d", v.focusIndex)
	}

	// Prev from name(0) goes to dbType(7)
	v.prevInput()
	if v.focusIndex != 7 {
		t.Errorf("Expected focusIndex 7 (dbType) after Shift+Tab from name, got %d", v.focusIndex)
	}

	// Prev from dbType(7) wraps to connect(8)
	v.prevInput()
	if v.focusIndex != 8 {
		t.Errorf("Expected focusIndex 8 (connect) after backward wrap, got %d", v.focusIndex)
	}
}

func TestConnectionView_FormView_SQLiteHidesFields(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.dbTypeIndex = 2 // SQLite
	v.onDbTypeChanged()

	view := v.View()

	// Should show name and database
	if !strings.Contains(view, "Connection Name") {
		t.Error("Expected 'Connection Name' field for SQLite")
	}
	if !strings.Contains(view, "Database") {
		t.Error("Expected 'Database' field for SQLite")
	}

	// Should NOT show host, port, username, password, schema
	if strings.Contains(view, "Host:") {
		t.Error("Expected 'Host' field to be hidden for SQLite")
	}
	if strings.Contains(view, "Port:") {
		t.Error("Expected 'Port' field to be hidden for SQLite")
	}
	if strings.Contains(view, "Username:") {
		t.Error("Expected 'Username' field to be hidden for SQLite")
	}
	if strings.Contains(view, "Password:") {
		t.Error("Expected 'Password' field to be hidden for SQLite")
	}
	if strings.Contains(view, "Schema:") {
		t.Error("Expected 'Schema' field to be hidden for SQLite")
	}
}

func TestConnectionView_FormView_PostgresShowsAllFields(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.dbTypeIndex = 0 // Postgres
	v.onDbTypeChanged()

	view := v.View()

	for _, field := range []string{"Connection Name", "Host:", "Port:", "Username:", "Password:", "Database:", "Schema:"} {
		if !strings.Contains(view, field) {
			t.Errorf("Expected '%s' field to be visible for Postgres", field)
		}
	}
}

func TestConnectionView_PasswordPromptSkipped_SQLite(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.dbTypeIndex = 2 // SQLite
	v.onDbTypeChanged()
	v.focusIndex = 8         // Connect button
	v.inputs[4].SetValue("") // Password empty (but hidden)

	// Press Enter - should NOT show password prompt since password field is hidden
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	v, _ = v.Update(msg)

	if v.awaitingPassword {
		t.Error("Expected awaitingPassword to be false for SQLite (password field hidden)")
	}
}

func TestConnectionView_ResetFormUpdatesVisibility(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.mode = "form"
	v.dbTypeIndex = 2 // SQLite
	v.onDbTypeChanged()

	// Reset should go back to Postgres defaults
	v.resetForm()

	if v.dbTypeIndex != 0 {
		t.Errorf("Expected dbTypeIndex 0 after reset, got %d", v.dbTypeIndex)
	}

	// Focus should start on db type selector
	if v.focusIndex != 7 {
		t.Errorf("Expected focusIndex 7 (db type) after reset, got %d", v.focusIndex)
	}

	// Postgres shows all 7 fields
	if len(v.visibleFields) != 7 {
		t.Errorf("Expected 7 visible fields after reset (Postgres), got %d", len(v.visibleFields))
	}
}

func TestConnectionView_RefreshList(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.parent.config.AddConnection(config.Connection{Name: "new-conn"})
	_ = v.parent.config.Save()

	v.refreshList()

	items := v.list.Items()
	if len(items) != 1 {
		t.Errorf("Expected 1 item after refresh, got %d", len(items))
	}
}
