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
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
)

func setupWhereViewTest(t *testing.T) (*WhereView, func()) {
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

	return parent.whereView, cleanup
}

func TestNewWhereView(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	if v == nil {
		t.Fatal("NewWhereView returned nil")
	}

	if len(v.conditions) != 0 {
		t.Error("Expected empty conditions initially")
	}

	if v.addingNew {
		t.Error("Expected addingNew to be false initially")
	}

	if v.selectedIndex != -1 {
		t.Errorf("Expected selectedIndex -1, got %d", v.selectedIndex)
	}

	// Check default operators
	expectedOps := []string{"=", "!=", ">", "<", ">=", "<=", "LIKE", "IN", "BETWEEN", "IS NULL", "IS NOT NULL"}
	if len(v.operators) != len(expectedOps) {
		t.Errorf("Expected %d operators, got %d", len(expectedOps), len(v.operators))
	}
}

func TestWhereView_SetTableContext(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	columns := []engine.Column{
		{Name: "id", Type: "integer"},
		{Name: "name", Type: "text"},
		{Name: "email", Type: "text"},
	}

	v.SetTableContext("public", "users", columns, nil)

	if v.schema != "public" {
		t.Errorf("Expected schema 'public', got '%s'", v.schema)
	}

	if v.tableName != "users" {
		t.Errorf("Expected tableName 'users', got '%s'", v.tableName)
	}

	if len(v.columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(v.columns))
	}
}

func TestWhereView_SetTableContext_WithExistingConditions(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	columns := []engine.Column{
		{Name: "id", Type: "integer"},
	}

	// Create existing condition
	existing := &model.WhereCondition{
		Type: model.WhereConditionTypeAnd,
		And: &model.OperationWhereCondition{
			Children: []*model.WhereCondition{
				{
					Type: model.WhereConditionTypeAtomic,
					Atomic: &model.AtomicWhereCondition{
						Key:      "id",
						Operator: "=",
						Value:    "123",
					},
				},
			},
		},
	}

	v.SetTableContext("public", "users", columns, existing)

	if len(v.conditions) != 1 {
		t.Fatalf("Expected 1 condition loaded, got %d", len(v.conditions))
	}

	if v.conditions[0].Field != "id" {
		t.Errorf("Expected field 'id', got '%s'", v.conditions[0].Field)
	}

	if v.conditions[0].Operator != "=" {
		t.Errorf("Expected operator '=', got '%s'", v.conditions[0].Operator)
	}

	if v.conditions[0].Value != "123" {
		t.Errorf("Expected value '123', got '%s'", v.conditions[0].Value)
	}
}

func TestWhereView_AddCondition_CtrlA(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	// Press ctrl+a to add new condition
	msg := tea.KeyMsg{Type: tea.KeyCtrlA}
	v, _ = v.Update(msg)

	if !v.addingNew {
		t.Error("Expected addingNew to be true after ctrl+a")
	}

	if v.focusIndex != 0 {
		t.Errorf("Expected focusIndex 0, got %d", v.focusIndex)
	}
}

func TestWhereView_AddCondition_Navigation(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.columns = []engine.Column{
		{Name: "id", Type: "integer"},
		{Name: "name", Type: "text"},
	}
	v.addingNew = true
	v.focusIndex = 0

	// Navigate down through fields
	msg := tea.KeyMsg{Type: tea.KeyDown}
	v, _ = v.Update(msg)
	if v.focusIndex != 1 {
		t.Errorf("Expected focusIndex 1 after down, got %d", v.focusIndex)
	}

	v, _ = v.Update(msg)
	if v.focusIndex != 2 {
		t.Errorf("Expected focusIndex 2 after second down, got %d", v.focusIndex)
	}

	v, _ = v.Update(msg)
	if v.focusIndex != 3 {
		t.Errorf("Expected focusIndex 3 after third down, got %d", v.focusIndex)
	}

	// Wrap around
	v, _ = v.Update(msg)
	if v.focusIndex != 0 {
		t.Errorf("Expected focusIndex 0 after wrap, got %d", v.focusIndex)
	}

	// Navigate up
	msg = tea.KeyMsg{Type: tea.KeyUp}
	v, _ = v.Update(msg)
	if v.focusIndex != 3 {
		t.Errorf("Expected focusIndex 3 after up from 0, got %d", v.focusIndex)
	}
}

func TestWhereView_FieldSelection_LeftRight(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.columns = []engine.Column{
		{Name: "id", Type: "integer"},
		{Name: "name", Type: "text"},
		{Name: "email", Type: "text"},
	}
	v.addingNew = true
	v.focusIndex = 0 // Field selection
	v.currentField = ""

	// Right arrow selects first field
	msg := tea.KeyMsg{Type: tea.KeyRight}
	v, _ = v.Update(msg)
	if v.currentField != "id" {
		t.Errorf("Expected currentField 'id', got '%s'", v.currentField)
	}

	// Another right moves to next
	v, _ = v.Update(msg)
	if v.currentField != "name" {
		t.Errorf("Expected currentField 'name', got '%s'", v.currentField)
	}

	// Left goes back
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	v, _ = v.Update(msg)
	if v.currentField != "id" {
		t.Errorf("Expected currentField 'id' after left, got '%s'", v.currentField)
	}

	// Left at first position stays there
	v, _ = v.Update(msg)
	if v.currentField != "id" {
		t.Errorf("Expected currentField to stay 'id', got '%s'", v.currentField)
	}
}

func TestWhereView_OperatorSelection_LeftRight(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.addingNew = true
	v.focusIndex = 1 // Operator selection
	v.currentOp = ""

	// Right arrow selects first operator
	msg := tea.KeyMsg{Type: tea.KeyRight}
	v, _ = v.Update(msg)
	if v.currentOp != "=" {
		t.Errorf("Expected currentOp '=', got '%s'", v.currentOp)
	}

	// Another right moves to next
	v, _ = v.Update(msg)
	if v.currentOp != "!=" {
		t.Errorf("Expected currentOp '!=', got '%s'", v.currentOp)
	}
}

func TestWhereView_Escape_CancelsAdding(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.addingNew = true
	v.currentField = "id"
	v.currentOp = "="
	v.valueInput.SetValue("123")

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.addingNew {
		t.Error("Expected addingNew to be false after Esc")
	}

	if v.currentField != "" {
		t.Error("Expected currentField to be cleared")
	}

	if v.currentOp != "" {
		t.Error("Expected currentOp to be cleared")
	}

	if v.valueInput.Value() != "" {
		t.Error("Expected valueInput to be cleared")
	}
}

func TestWhereView_Escape_GoesBack(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.addingNew = false

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.parent.mode != ViewResults {
		t.Errorf("Expected mode ViewResults after Esc, got %v", v.parent.mode)
	}
}

func TestWhereView_DeleteCondition_CtrlD(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.conditions = []WhereCondition{
		{Field: "id", Operator: "=", Value: "1"},
		{Field: "name", Operator: "LIKE", Value: "%test%"},
	}
	v.selectedIndex = 0

	msg := tea.KeyMsg{Type: tea.KeyCtrlD}
	v, _ = v.Update(msg)

	if len(v.conditions) != 1 {
		t.Errorf("Expected 1 condition after delete, got %d", len(v.conditions))
	}

	if v.conditions[0].Field != "name" {
		t.Errorf("Expected remaining condition field 'name', got '%s'", v.conditions[0].Field)
	}
}

func TestWhereView_EditCondition_CtrlE(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.conditions = []WhereCondition{
		{Field: "id", Operator: "=", Value: "123"},
	}
	v.selectedIndex = 0

	msg := tea.KeyMsg{Type: tea.KeyCtrlE}
	v, _ = v.Update(msg)

	if !v.addingNew {
		t.Error("Expected addingNew to be true after ctrl+e")
	}

	if v.currentField != "id" {
		t.Errorf("Expected currentField 'id', got '%s'", v.currentField)
	}

	if v.currentOp != "=" {
		t.Errorf("Expected currentOp '=', got '%s'", v.currentOp)
	}

	if v.valueInput.Value() != "123" {
		t.Errorf("Expected valueInput '123', got '%s'", v.valueInput.Value())
	}

	// Original condition should be removed
	if len(v.conditions) != 0 {
		t.Errorf("Expected 0 conditions during edit, got %d", len(v.conditions))
	}
}

func TestWhereView_ConditionSelection_UpDown(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.conditions = []WhereCondition{
		{Field: "id", Operator: "=", Value: "1"},
		{Field: "name", Operator: "=", Value: "test"},
		{Field: "email", Operator: "LIKE", Value: "%@%"},
	}
	v.selectedIndex = -1
	v.addingNew = false

	// Down selects first
	msg := tea.KeyMsg{Type: tea.KeyDown}
	v, _ = v.Update(msg)
	if v.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0, got %d", v.selectedIndex)
	}

	// Down moves to next
	v, _ = v.Update(msg)
	if v.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex 1, got %d", v.selectedIndex)
	}

	// Up moves back
	msg = tea.KeyMsg{Type: tea.KeyUp}
	v, _ = v.Update(msg)
	if v.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0 after up, got %d", v.selectedIndex)
	}
}

func TestWhereView_BuildWhereCondition_Empty(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.conditions = []WhereCondition{}

	result := v.buildWhereCondition()
	if result != nil {
		t.Error("Expected nil for empty conditions")
	}
}

func TestWhereView_BuildWhereCondition_Single(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.conditions = []WhereCondition{
		{Field: "id", Operator: "=", Value: "123"},
	}

	result := v.buildWhereCondition()
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Type != model.WhereConditionTypeAnd {
		t.Errorf("Expected AND type, got %v", result.Type)
	}

	if len(result.And.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(result.And.Children))
	}

	child := result.And.Children[0]
	if child.Atomic.Key != "id" {
		t.Errorf("Expected key 'id', got '%s'", child.Atomic.Key)
	}
}

func TestWhereView_BuildWhereCondition_Multiple(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.conditions = []WhereCondition{
		{Field: "id", Operator: ">", Value: "10"},
		{Field: "status", Operator: "=", Value: "active"},
	}

	result := v.buildWhereCondition()
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.And.Children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(result.And.Children))
	}
}

func TestWhereView_View_Empty(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.schema = "public"
	v.tableName = "users"
	v.conditions = []WhereCondition{}

	view := v.View()

	if !strings.Contains(view, "WHERE Conditions") {
		t.Error("Expected title 'WHERE Conditions'")
	}

	if !strings.Contains(view, "public.users") {
		t.Error("Expected table name in view")
	}

	if !strings.Contains(view, "No conditions added yet") {
		t.Error("Expected 'No conditions added yet' for empty conditions")
	}
}

func TestWhereView_View_WithConditions(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.schema = "public"
	v.tableName = "users"
	v.conditions = []WhereCondition{
		{Field: "id", Operator: "=", Value: "123"},
	}
	v.selectedIndex = 0

	view := v.View()

	if !strings.Contains(view, "id = 123") {
		t.Error("Expected condition 'id = 123' in view")
	}

	if !strings.Contains(view, "Current Conditions") {
		t.Error("Expected 'Current Conditions' header")
	}
}

func TestWhereView_View_AddingMode(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.schema = "public"
	v.tableName = "users"
	v.addingNew = true

	view := v.View()

	if !strings.Contains(view, "Add New Condition") {
		t.Error("Expected 'Add New Condition' header")
	}

	if !strings.Contains(view, "Field") {
		t.Error("Expected 'Field' label")
	}

	if !strings.Contains(view, "Operator") {
		t.Error("Expected 'Operator' label")
	}

	if !strings.Contains(view, "Value") {
		t.Error("Expected 'Value' label")
	}
}

func TestWhereView_HelpText_NormalMode(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.addingNew = false
	view := v.View()

	// Check for ctrl+* shortcuts
	if !strings.Contains(view, "ctrl+a") {
		t.Error("Expected help text to show 'ctrl+a' for add")
	}

	if !strings.Contains(view, "ctrl+e") {
		t.Error("Expected help text to show 'ctrl+e' for edit")
	}

	if !strings.Contains(view, "ctrl+d") {
		t.Error("Expected help text to show 'ctrl+d' for delete")
	}
}

func TestWhereView_MouseScroll(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.conditions = []WhereCondition{
		{Field: "a", Operator: "=", Value: "1"},
		{Field: "b", Operator: "=", Value: "2"},
		{Field: "c", Operator: "=", Value: "3"},
	}
	v.selectedIndex = 0
	v.addingNew = false

	// Mouse wheel down
	msg := tea.MouseMsg{Button: tea.MouseButtonWheelDown}
	v, _ = v.Update(msg)
	if v.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex 1 after wheel down, got %d", v.selectedIndex)
	}

	// Mouse wheel up
	msg = tea.MouseMsg{Button: tea.MouseButtonWheelUp}
	v, _ = v.Update(msg)
	if v.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0 after wheel up, got %d", v.selectedIndex)
	}
}

func TestWhereView_FindColumnIndex(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.columns = []engine.Column{
		{Name: "id"},
		{Name: "name"},
		{Name: "email"},
	}

	tests := []struct {
		name     string
		expected int
	}{
		{"id", 0},
		{"name", 1},
		{"email", 2},
		{"nonexistent", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.findColumnIndex(tt.name)
			if result != tt.expected {
				t.Errorf("findColumnIndex(%s) = %d, expected %d", tt.name, result, tt.expected)
			}
		})
	}
}

func TestWhereView_FindOperatorIndex(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	tests := []struct {
		op       string
		expected int
	}{
		{"=", 0},
		{"!=", 1},
		{"LIKE", 6},
		{"nonexistent", -1},
	}

	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			result := v.findOperatorIndex(tt.op)
			if result != tt.expected {
				t.Errorf("findOperatorIndex(%s) = %d, expected %d", tt.op, result, tt.expected)
			}
		})
	}
}
