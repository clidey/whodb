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

	// First input should be focused
	if v.focusIndex != 0 {
		t.Errorf("Expected focusIndex 0, got %d", v.focusIndex)
	}
}

func TestNewConnectionView_WithConnections(t *testing.T) {
	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

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
	v.focusIndex = 0

	// Tab/Down moves to next input
	msg := tea.KeyMsg{Type: tea.KeyTab}
	v, _ = v.Update(msg)

	if v.focusIndex != 1 {
		t.Errorf("Expected focusIndex 1 after Tab, got %d", v.focusIndex)
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

	// Tab from last wraps to first
	v.nextInput()

	if v.focusIndex != 0 {
		t.Errorf("Expected focusIndex 0 after wrap, got %d", v.focusIndex)
	}

	// Tab from first backwards wraps to last
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
	v.focusIndex = 8 // Connect button
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

	if v.focusIndex != 0 {
		t.Errorf("Expected focusIndex 0 after reset, got %d", v.focusIndex)
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
	v.focusIndex = 0

	// Mouse wheel down
	msg := tea.MouseMsg{Button: tea.MouseButtonWheelDown}
	v, _ = v.Update(msg)

	if v.focusIndex != 1 {
		t.Errorf("Expected focusIndex 1 after wheel down, got %d", v.focusIndex)
	}

	// Mouse wheel up
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

func TestConnectionView_RefreshList(t *testing.T) {
	v, cleanup := setupConnectionViewTest(t)
	defer cleanup()

	v.parent.config.AddConnection(config.Connection{Name: "new-conn"})

	v.refreshList()

	items := v.list.Items()
	if len(items) != 1 {
		t.Errorf("Expected 1 item after refresh, got %d", len(items))
	}
}
