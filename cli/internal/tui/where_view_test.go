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

	tea "charm.land/bubbletea/v2"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
)

func setupWhereViewTest(t *testing.T) (*WhereView, func()) {
	t.Helper()

	setupTestEnv(t)

	parent := NewMainModel()
	if parent.err != nil {
		t.Fatalf("Failed to create MainModel: %v", parent.err)
	}

	cleanup := func() {}

	return parent.whereView, cleanup
}

func TestNewWhereView(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	if v == nil {
		t.Fatal("NewWhereView returned nil")
	}

	if len(v.groups) != 0 {
		t.Error("Expected empty groups initially")
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

	// Should have one default empty AND group.
	if len(v.groups) != 1 {
		t.Errorf("Expected 1 default group, got %d", len(v.groups))
	}
	if v.groups[0].Logic != "AND" {
		t.Errorf("Expected default group logic 'AND', got '%s'", v.groups[0].Logic)
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

	if len(v.groups) != 1 {
		t.Fatalf("Expected 1 group loaded, got %d", len(v.groups))
	}

	if len(v.groups[0].Conditions) != 1 {
		t.Fatalf("Expected 1 condition in group, got %d", len(v.groups[0].Conditions))
	}

	cond := v.groups[0].Conditions[0]
	if cond.Field != "id" {
		t.Errorf("Expected field 'id', got '%s'", cond.Field)
	}

	if cond.Operator != "=" {
		t.Errorf("Expected operator '=', got '%s'", cond.Operator)
	}

	if cond.Value != "123" {
		t.Errorf("Expected value '123', got '%s'", cond.Value)
	}
}

func TestWhereView_SetTableContext_WithNestedConditions(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	columns := []engine.Column{
		{Name: "name", Type: "text"},
		{Name: "age", Type: "integer"},
		{Name: "role", Type: "text"},
	}

	// Build: (name = 'alice' AND age > 18) OR (role = 'admin')
	existing := &model.WhereCondition{
		Type: model.WhereConditionTypeOr,
		Or: &model.OperationWhereCondition{
			Children: []*model.WhereCondition{
				{
					Type: model.WhereConditionTypeAnd,
					And: &model.OperationWhereCondition{
						Children: []*model.WhereCondition{
							{Type: model.WhereConditionTypeAtomic, Atomic: &model.AtomicWhereCondition{Key: "name", Operator: "=", Value: "alice"}},
							{Type: model.WhereConditionTypeAtomic, Atomic: &model.AtomicWhereCondition{Key: "age", Operator: ">", Value: "18"}},
						},
					},
				},
				{Type: model.WhereConditionTypeAtomic, Atomic: &model.AtomicWhereCondition{Key: "role", Operator: "=", Value: "admin"}},
			},
		},
	}

	v.SetTableContext("public", "users", columns, existing)

	if len(v.groups) != 2 {
		t.Fatalf("Expected 2 groups, got %d", len(v.groups))
	}

	// First group: AND with 2 conditions
	if v.groups[0].Logic != "AND" {
		t.Errorf("Expected first group logic 'AND', got '%s'", v.groups[0].Logic)
	}
	if len(v.groups[0].Conditions) != 2 {
		t.Errorf("Expected 2 conditions in first group, got %d", len(v.groups[0].Conditions))
	}

	// Second group: single condition (wrapped as AND by default)
	if len(v.groups[1].Conditions) != 1 {
		t.Errorf("Expected 1 condition in second group, got %d", len(v.groups[1].Conditions))
	}
	if v.groups[1].Conditions[0].Field != "role" {
		t.Errorf("Expected field 'role', got '%s'", v.groups[1].Conditions[0].Field)
	}
}

func TestWhereView_AddCondition_A(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.SetTableContext("public", "users", []engine.Column{{Name: "id", Type: "integer"}}, nil)

	// Press 'a' to add new condition
	msg := tea.KeyPressMsg{Text: "a", Code: 'a'}
	v, _ = v.Update(msg)

	if !v.addingNew {
		t.Error("Expected addingNew to be true after pressing 'a'")
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
	msg := tea.KeyPressMsg{Code: tea.KeyDown}
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
	msg = tea.KeyPressMsg{Code: tea.KeyUp}
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
	msg := tea.KeyPressMsg{Code: tea.KeyRight}
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
	msg = tea.KeyPressMsg{Code: tea.KeyLeft}
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
	msg := tea.KeyPressMsg{Code: tea.KeyRight}
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

	msg := tea.KeyPressMsg{Code: tea.KeyEsc}
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

	msg := tea.KeyPressMsg{Code: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.parent.mode != ViewResults {
		t.Errorf("Expected mode ViewResults after Esc, got %v", v.parent.mode)
	}
}

func TestWhereView_DeleteCondition_D(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.groups = []conditionGroup{
		{
			Logic: "AND",
			Conditions: []WhereCondition{
				{Field: "id", Operator: "=", Value: "1"},
				{Field: "name", Operator: "LIKE", Value: "%test%"},
			},
		},
	}
	v.rebuildFlatItems()
	// Select the first condition (index 1, after the group header at index 0).
	v.selectedIndex = 1

	msg := tea.KeyPressMsg{Text: "d", Code: 'd'}
	v, _ = v.Update(msg)

	if len(v.groups[0].Conditions) != 1 {
		t.Errorf("Expected 1 condition after delete, got %d", len(v.groups[0].Conditions))
	}

	if v.groups[0].Conditions[0].Field != "name" {
		t.Errorf("Expected remaining condition field 'name', got '%s'", v.groups[0].Conditions[0].Field)
	}
}

func TestWhereView_DeleteGroup(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.groups = []conditionGroup{
		{Logic: "AND", Conditions: []WhereCondition{{Field: "id", Operator: "=", Value: "1"}}},
		{Logic: "OR", Conditions: []WhereCondition{{Field: "name", Operator: "=", Value: "test"}}},
	}
	v.rebuildFlatItems()
	// Select second group header. Items: [group0, cond0, group1, cond1]
	v.selectedIndex = 2

	msg := tea.KeyPressMsg{Text: "d", Code: 'd'}
	v, _ = v.Update(msg)

	if len(v.groups) != 1 {
		t.Errorf("Expected 1 group after delete, got %d", len(v.groups))
	}
	if v.groups[0].Logic != "AND" {
		t.Errorf("Expected remaining group logic 'AND', got '%s'", v.groups[0].Logic)
	}
}

func TestWhereView_EditCondition_CtrlE(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.groups = []conditionGroup{
		{
			Logic: "AND",
			Conditions: []WhereCondition{
				{Field: "id", Operator: "=", Value: "123"},
			},
		},
	}
	v.rebuildFlatItems()
	// Select the condition (index 1, after group header at index 0).
	v.selectedIndex = 1

	msg := tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl}
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
	if len(v.groups[0].Conditions) != 0 {
		t.Errorf("Expected 0 conditions during edit, got %d", len(v.groups[0].Conditions))
	}
}

func TestWhereView_ConditionSelection_UpDown(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.groups = []conditionGroup{
		{
			Logic: "AND",
			Conditions: []WhereCondition{
				{Field: "id", Operator: "=", Value: "1"},
				{Field: "name", Operator: "=", Value: "test"},
				{Field: "email", Operator: "LIKE", Value: "%@%"},
			},
		},
	}
	v.rebuildFlatItems()
	v.selectedIndex = -1
	v.addingNew = false

	// Down selects first (group header)
	msg := tea.KeyPressMsg{Code: tea.KeyDown}
	v, _ = v.Update(msg)
	if v.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0, got %d", v.selectedIndex)
	}

	// Down moves to first condition
	v, _ = v.Update(msg)
	if v.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex 1, got %d", v.selectedIndex)
	}

	// Up moves back to group header
	msg = tea.KeyPressMsg{Code: tea.KeyUp}
	v, _ = v.Update(msg)
	if v.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex 0 after up, got %d", v.selectedIndex)
	}
}

func TestWhereView_NewGroup_G(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.groups = []conditionGroup{
		{Logic: "AND", Conditions: []WhereCondition{{Field: "id", Operator: "=", Value: "1"}}},
	}
	v.rebuildFlatItems()
	v.addingNew = false

	msg := tea.KeyPressMsg{Text: "g", Code: 'g'}
	v, _ = v.Update(msg)

	if len(v.groups) != 2 {
		t.Errorf("Expected 2 groups after pressing 'g', got %d", len(v.groups))
	}

	if v.groups[1].Logic != "AND" {
		t.Errorf("Expected new group logic 'AND', got '%s'", v.groups[1].Logic)
	}
}

func TestWhereView_ToggleLogic_T(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.groups = []conditionGroup{
		{Logic: "AND", Conditions: []WhereCondition{{Field: "id", Operator: "=", Value: "1"}}},
	}
	v.rebuildFlatItems()
	v.selectedIndex = 0 // group header
	v.addingNew = false

	msg := tea.KeyPressMsg{Text: "t", Code: 't'}
	v, _ = v.Update(msg)

	if v.groups[0].Logic != "OR" {
		t.Errorf("Expected group logic toggled to 'OR', got '%s'", v.groups[0].Logic)
	}

	// Toggle again
	v, _ = v.Update(msg)
	if v.groups[0].Logic != "AND" {
		t.Errorf("Expected group logic toggled back to 'AND', got '%s'", v.groups[0].Logic)
	}
}

func TestWhereView_ToggleLogic_FromConditionItem(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.groups = []conditionGroup{
		{Logic: "AND", Conditions: []WhereCondition{{Field: "id", Operator: "=", Value: "1"}}},
	}
	v.rebuildFlatItems()
	// Select the condition item (not the group header).
	v.selectedIndex = 1
	v.addingNew = false

	msg := tea.KeyPressMsg{Text: "t", Code: 't'}
	v, _ = v.Update(msg)

	// Should still toggle the parent group's logic.
	if v.groups[0].Logic != "OR" {
		t.Errorf("Expected group logic toggled to 'OR', got '%s'", v.groups[0].Logic)
	}
}

func TestWhereView_BuildWhereCondition_Empty(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.groups = []conditionGroup{}

	result := v.buildWhereCondition()
	if result != nil {
		t.Error("Expected nil for empty groups")
	}
}

func TestWhereView_BuildWhereCondition_SingleGroup(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.groups = []conditionGroup{
		{
			Logic: "AND",
			Conditions: []WhereCondition{
				{Field: "id", Operator: "=", Value: "123"},
			},
		},
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

func TestWhereView_BuildWhereCondition_MultipleInGroup(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.groups = []conditionGroup{
		{
			Logic: "AND",
			Conditions: []WhereCondition{
				{Field: "id", Operator: ">", Value: "10"},
				{Field: "status", Operator: "=", Value: "active"},
			},
		},
	}

	result := v.buildWhereCondition()
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Single group with 2 conditions -> AND at top with AND child holding 2 atomics
	if result.Type != model.WhereConditionTypeAnd {
		t.Errorf("Expected AND type, got %v", result.Type)
	}
	if result.And == nil {
		t.Fatal("Expected And to be non-nil")
	}
	// The single AND group is wrapped in a top-level AND.
	children := result.And.Children
	if len(children) != 1 {
		t.Fatalf("Expected 1 child (the AND group), got %d", len(children))
	}
	innerAnd := children[0]
	if innerAnd.Type != model.WhereConditionTypeAnd {
		t.Errorf("Expected inner AND type, got %v", innerAnd.Type)
	}
	if len(innerAnd.And.Children) != 2 {
		t.Errorf("Expected 2 atomic children, got %d", len(innerAnd.And.Children))
	}
}

func TestWhereView_BuildWhereCondition_MultipleGroups(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.columns = []engine.Column{
		{Name: "name", Type: "text"},
		{Name: "age", Type: "integer"},
		{Name: "role", Type: "text"},
	}

	v.groups = []conditionGroup{
		{
			Logic: "AND",
			Conditions: []WhereCondition{
				{Field: "name", Operator: "=", Value: "alice"},
				{Field: "age", Operator: ">", Value: "18"},
			},
		},
		{
			Logic: "AND",
			Conditions: []WhereCondition{
				{Field: "role", Operator: "=", Value: "admin"},
			},
		},
	}

	result := v.buildWhereCondition()
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Multiple groups -> top-level OR
	if result.Type != model.WhereConditionTypeOr {
		t.Errorf("Expected top-level OR, got %v", result.Type)
	}
	if result.Or == nil {
		t.Fatal("Expected Or to be non-nil")
	}
	if len(result.Or.Children) != 2 {
		t.Fatalf("Expected 2 children in OR, got %d", len(result.Or.Children))
	}

	// First child should be AND with 2 atomics
	first := result.Or.Children[0]
	if first.Type != model.WhereConditionTypeAnd {
		t.Errorf("Expected first child AND, got %v", first.Type)
	}
	if len(first.And.Children) != 2 {
		t.Errorf("Expected 2 atomics in first group, got %d", len(first.And.Children))
	}

	// Second child should be a single atomic (role = admin)
	second := result.Or.Children[1]
	if second.Type != model.WhereConditionTypeAtomic {
		t.Errorf("Expected second child Atomic, got %v", second.Type)
	}
	if second.Atomic.Key != "role" {
		t.Errorf("Expected key 'role', got '%s'", second.Atomic.Key)
	}
}

func TestWhereView_BuildWhereCondition_EmptyGroupSkipped(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.groups = []conditionGroup{
		{Logic: "AND", Conditions: []WhereCondition{}},
		{Logic: "OR", Conditions: []WhereCondition{{Field: "x", Operator: "=", Value: "1"}}},
	}

	result := v.buildWhereCondition()
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Only one non-empty group, so result should be a simple AND wrapper.
	if result.Type != model.WhereConditionTypeAnd {
		t.Errorf("Expected AND type (single non-empty group), got %v", result.Type)
	}
}

func TestWhereView_RenderTree_SingleGroup(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.groups = []conditionGroup{
		{
			Logic: "AND",
			Conditions: []WhereCondition{
				{Field: "name", Operator: "=", Value: "alice"},
				{Field: "age", Operator: ">", Value: "18"},
			},
		},
	}
	v.rebuildFlatItems()

	tree := v.renderTree()

	if !strings.Contains(tree, "AND") {
		t.Error("Expected 'AND' in tree rendering")
	}
	if !strings.Contains(tree, "name = alice") {
		t.Error("Expected 'name = alice' in tree rendering")
	}
	if !strings.Contains(tree, "age > 18") {
		t.Error("Expected 'age > 18' in tree rendering")
	}
}

func TestWhereView_RenderTree_MultipleGroups(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.groups = []conditionGroup{
		{
			Logic: "AND",
			Conditions: []WhereCondition{
				{Field: "name", Operator: "=", Value: "alice"},
				{Field: "age", Operator: ">", Value: "18"},
			},
		},
		{
			Logic: "OR",
			Conditions: []WhereCondition{
				{Field: "role", Operator: "=", Value: "admin"},
			},
		},
	}
	v.rebuildFlatItems()

	tree := v.renderTree()

	if !strings.Contains(tree, "AND") {
		t.Error("Expected 'AND' in tree rendering")
	}
	if !strings.Contains(tree, "OR") {
		t.Error("Expected 'OR' in tree rendering")
	}
	if !strings.Contains(tree, "role = admin") {
		t.Error("Expected 'role = admin' in tree rendering")
	}
}

func TestWhereView_FlatItems_Count(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.groups = []conditionGroup{
		{
			Logic: "AND",
			Conditions: []WhereCondition{
				{Field: "a", Operator: "=", Value: "1"},
				{Field: "b", Operator: "=", Value: "2"},
			},
		},
		{
			Logic: "OR",
			Conditions: []WhereCondition{
				{Field: "c", Operator: "=", Value: "3"},
			},
		},
	}
	v.rebuildFlatItems()

	// 2 group headers + 3 conditions = 5 items
	if len(v.flatItems) != 5 {
		t.Errorf("Expected 5 flat items, got %d", len(v.flatItems))
	}

	// Verify structure: [group0, cond, cond, group1, cond]
	if !v.flatItems[0].IsGroup || v.flatItems[0].GroupIndex != 0 {
		t.Error("Expected first item to be group 0 header")
	}
	if v.flatItems[1].IsGroup || v.flatItems[1].ConditionIndex != 0 {
		t.Error("Expected second item to be condition 0 of group 0")
	}
	if v.flatItems[2].IsGroup || v.flatItems[2].ConditionIndex != 1 {
		t.Error("Expected third item to be condition 1 of group 0")
	}
	if !v.flatItems[3].IsGroup || v.flatItems[3].GroupIndex != 1 {
		t.Error("Expected fourth item to be group 1 header")
	}
	if v.flatItems[4].IsGroup || v.flatItems[4].ConditionIndex != 0 {
		t.Error("Expected fifth item to be condition 0 of group 1")
	}
}

func TestWhereView_Navigation_AcrossGroups(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.groups = []conditionGroup{
		{Logic: "AND", Conditions: []WhereCondition{{Field: "a", Operator: "=", Value: "1"}}},
		{Logic: "OR", Conditions: []WhereCondition{{Field: "b", Operator: "=", Value: "2"}}},
	}
	v.rebuildFlatItems()
	v.selectedIndex = -1
	v.addingNew = false

	down := tea.KeyPressMsg{Code: tea.KeyDown}

	// Move down through all items
	v, _ = v.Update(down) // -> group 0 header (index 0)
	if v.selectedIndex != 0 {
		t.Errorf("Expected 0, got %d", v.selectedIndex)
	}

	v, _ = v.Update(down) // -> condition a (index 1)
	if v.selectedIndex != 1 {
		t.Errorf("Expected 1, got %d", v.selectedIndex)
	}

	v, _ = v.Update(down) // -> group 1 header (index 2)
	if v.selectedIndex != 2 {
		t.Errorf("Expected 2, got %d", v.selectedIndex)
	}

	v, _ = v.Update(down) // -> condition b (index 3)
	if v.selectedIndex != 3 {
		t.Errorf("Expected 3, got %d", v.selectedIndex)
	}

	// At end, should not go further
	v, _ = v.Update(down)
	if v.selectedIndex != 3 {
		t.Errorf("Expected to stay at 3, got %d", v.selectedIndex)
	}
}

func TestWhereView_View_Empty(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.schema = "public"
	v.tableName = "users"
	v.groups = []conditionGroup{}

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
	v.groups = []conditionGroup{
		{Logic: "AND", Conditions: []WhereCondition{{Field: "id", Operator: "=", Value: "123"}}},
	}
	v.rebuildFlatItems()
	v.selectedIndex = 1 // the condition

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

	// Check for shortcut keys in help text
	if !strings.Contains(view, "[a]") {
		t.Error("Expected help text to show '[a]' for add")
	}

	if !strings.Contains(view, "ctrl+e") {
		t.Error("Expected help text to show 'ctrl+e' for edit")
	}

	if !strings.Contains(view, "[d]") {
		t.Error("Expected help text to show '[d]' for delete")
	}

	if !strings.Contains(view, "[g]") {
		t.Error("Expected help text to show '[g]' for new group")
	}

	if !strings.Contains(view, "[t]") {
		t.Error("Expected help text to show '[t]' for toggle")
	}
}

func TestWhereView_MouseScroll(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.groups = []conditionGroup{
		{
			Logic: "AND",
			Conditions: []WhereCondition{
				{Field: "a", Operator: "=", Value: "1"},
				{Field: "b", Operator: "=", Value: "2"},
				{Field: "c", Operator: "=", Value: "3"},
			},
		},
	}
	v.rebuildFlatItems()
	v.selectedIndex = 0 // group header
	v.addingNew = false

	// Mouse wheel down
	msg := tea.MouseWheelMsg{Button: tea.MouseWheelDown}
	v, _ = v.Update(msg)
	if v.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex 1 after wheel down, got %d", v.selectedIndex)
	}

	// Mouse wheel up
	msg = tea.MouseWheelMsg{Button: tea.MouseWheelUp}
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

func TestWhereView_TotalConditionCount(t *testing.T) {
	v, cleanup := setupWhereViewTest(t)
	defer cleanup()

	v.groups = []conditionGroup{
		{Logic: "AND", Conditions: []WhereCondition{{Field: "a", Operator: "=", Value: "1"}, {Field: "b", Operator: "=", Value: "2"}}},
		{Logic: "OR", Conditions: []WhereCondition{{Field: "c", Operator: "=", Value: "3"}}},
	}

	if count := v.totalConditionCount(); count != 3 {
		t.Errorf("Expected totalConditionCount 3, got %d", count)
	}
}

func TestWhereView_GroupsFromWhereCondition_Nil(t *testing.T) {
	result := groupsFromWhereCondition(nil)
	if result != nil {
		t.Error("Expected nil for nil input")
	}
}

func TestWhereView_GroupsFromWhereCondition_Atomic(t *testing.T) {
	wc := &model.WhereCondition{
		Type:   model.WhereConditionTypeAtomic,
		Atomic: &model.AtomicWhereCondition{Key: "x", Operator: "=", Value: "1"},
	}

	result := groupsFromWhereCondition(wc)
	if len(result) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(result))
	}
	if len(result[0].Conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(result[0].Conditions))
	}
	if result[0].Conditions[0].Field != "x" {
		t.Errorf("Expected field 'x', got '%s'", result[0].Conditions[0].Field)
	}
}

func TestWhereView_GroupsFromWhereCondition_FlatAnd(t *testing.T) {
	wc := &model.WhereCondition{
		Type: model.WhereConditionTypeAnd,
		And: &model.OperationWhereCondition{
			Children: []*model.WhereCondition{
				{Type: model.WhereConditionTypeAtomic, Atomic: &model.AtomicWhereCondition{Key: "a", Operator: "=", Value: "1"}},
				{Type: model.WhereConditionTypeAtomic, Atomic: &model.AtomicWhereCondition{Key: "b", Operator: "=", Value: "2"}},
			},
		},
	}

	result := groupsFromWhereCondition(wc)
	if len(result) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(result))
	}
	if result[0].Logic != "AND" {
		t.Errorf("Expected logic 'AND', got '%s'", result[0].Logic)
	}
	if len(result[0].Conditions) != 2 {
		t.Errorf("Expected 2 conditions, got %d", len(result[0].Conditions))
	}
}

func TestWhereView_GroupsFromWhereCondition_NestedOrAnd(t *testing.T) {
	// (a = 1 AND b = 2) OR (c = 3)
	wc := &model.WhereCondition{
		Type: model.WhereConditionTypeOr,
		Or: &model.OperationWhereCondition{
			Children: []*model.WhereCondition{
				{
					Type: model.WhereConditionTypeAnd,
					And: &model.OperationWhereCondition{
						Children: []*model.WhereCondition{
							{Type: model.WhereConditionTypeAtomic, Atomic: &model.AtomicWhereCondition{Key: "a", Operator: "=", Value: "1"}},
							{Type: model.WhereConditionTypeAtomic, Atomic: &model.AtomicWhereCondition{Key: "b", Operator: "=", Value: "2"}},
						},
					},
				},
				{Type: model.WhereConditionTypeAtomic, Atomic: &model.AtomicWhereCondition{Key: "c", Operator: "=", Value: "3"}},
			},
		},
	}

	result := groupsFromWhereCondition(wc)
	if len(result) != 2 {
		t.Fatalf("Expected 2 groups, got %d", len(result))
	}
	if result[0].Logic != "AND" {
		t.Errorf("Expected first group logic 'AND', got '%s'", result[0].Logic)
	}
	if len(result[0].Conditions) != 2 {
		t.Errorf("Expected 2 conditions in first group, got %d", len(result[0].Conditions))
	}
	if len(result[1].Conditions) != 1 {
		t.Errorf("Expected 1 condition in second group, got %d", len(result[1].Conditions))
	}
}

func TestCountAtomicConditions(t *testing.T) {
	tests := []struct {
		name     string
		wc       *model.WhereCondition
		expected int
	}{
		{"nil", nil, 0},
		{"single atomic", &model.WhereCondition{
			Type:   model.WhereConditionTypeAtomic,
			Atomic: &model.AtomicWhereCondition{Key: "x", Operator: "=", Value: "1"},
		}, 1},
		{"and with 2 atomics", &model.WhereCondition{
			Type: model.WhereConditionTypeAnd,
			And: &model.OperationWhereCondition{
				Children: []*model.WhereCondition{
					{Type: model.WhereConditionTypeAtomic, Atomic: &model.AtomicWhereCondition{Key: "a", Operator: "=", Value: "1"}},
					{Type: model.WhereConditionTypeAtomic, Atomic: &model.AtomicWhereCondition{Key: "b", Operator: "=", Value: "2"}},
				},
			},
		}, 2},
		{"nested or(and(a, b), c)", &model.WhereCondition{
			Type: model.WhereConditionTypeOr,
			Or: &model.OperationWhereCondition{
				Children: []*model.WhereCondition{
					{Type: model.WhereConditionTypeAnd, And: &model.OperationWhereCondition{
						Children: []*model.WhereCondition{
							{Type: model.WhereConditionTypeAtomic, Atomic: &model.AtomicWhereCondition{Key: "a", Operator: "=", Value: "1"}},
							{Type: model.WhereConditionTypeAtomic, Atomic: &model.AtomicWhereCondition{Key: "b", Operator: "=", Value: "2"}},
						},
					}},
					{Type: model.WhereConditionTypeAtomic, Atomic: &model.AtomicWhereCondition{Key: "c", Operator: "=", Value: "3"}},
				},
			},
		}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countAtomicConditions(tt.wc)
			if result != tt.expected {
				t.Errorf("countAtomicConditions() = %d, expected %d", result, tt.expected)
			}
		})
	}
}
