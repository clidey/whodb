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
	"github.com/clidey/whodb/core/src/engine"
)

func TestRowWriteView_SetAddContext_ExcludesDatabaseManagedColumns(t *testing.T) {
	setupTestEnv(t)

	parent := NewMainModel()
	if parent.err != nil {
		t.Fatalf("NewMainModel failed: %v", parent.err)
	}

	view := NewRowWriteView(parent)
	view.SetAddContext("public", "users", []engine.Column{
		{Name: "id", Type: "integer", IsAutoIncrement: true},
		{Name: "full_name", Type: "text"},
		{Name: "computed_value", Type: "text", IsComputed: true},
	})

	if view.action != rowWriteActionAdd {
		t.Fatalf("expected add action, got %v", view.action)
	}
	if view.documentMode {
		t.Fatal("expected structured row mode for regular SQL columns")
	}
	if len(view.inputColumns) != 1 {
		t.Fatalf("expected only one writable input column, got %d", len(view.inputColumns))
	}
	if view.inputColumns[0].Name != "full_name" {
		t.Fatalf("expected writable column full_name, got %q", view.inputColumns[0].Name)
	}
	if len(view.inputs) != 1 {
		t.Fatalf("expected one input control, got %d", len(view.inputs))
	}
}

func TestResultsView_AddRowShortcutOpensRowWrite(t *testing.T) {
	m := setupConnectedModelWithTable(t, 100, 30)

	cmd := m.resultsView.LoadTable("", "test_users")
	msg := cmd()
	m.resultsView.Update(msg)
	m.mode = ViewResults

	m.Update(tea.KeyPressMsg{Text: "a", Code: 'a'})

	if m.mode != ViewRowWrite {
		t.Fatalf("expected mode ViewRowWrite after add-row shortcut, got %v", m.mode)
	}
	if m.rowWriteView.action != rowWriteActionAdd {
		t.Fatalf("expected add action, got %v", m.rowWriteView.action)
	}
	if m.rowWriteView.tableName != "test_users" {
		t.Fatalf("expected target table test_users, got %q", m.rowWriteView.tableName)
	}
}

func TestResultsView_DeleteRowShortcutOpensRowWrite(t *testing.T) {
	m := setupConnectedModelWithTable(t, 100, 30)

	cmd := m.resultsView.LoadTable("", "test_users")
	msg := cmd()
	m.resultsView.Update(msg)
	m.mode = ViewResults

	m.Update(tea.KeyPressMsg{Text: "d", Code: 'd'})

	if m.mode != ViewRowWrite {
		t.Fatalf("expected mode ViewRowWrite after delete-row shortcut, got %v", m.mode)
	}
	if m.rowWriteView.action != rowWriteActionDelete {
		t.Fatalf("expected delete action, got %v", m.rowWriteView.action)
	}
	if len(m.rowWriteView.deleteValues) == 0 {
		t.Fatal("expected selected row values to be captured for delete")
	}
}

func TestResultsView_EditRowShortcutOpensRowWrite(t *testing.T) {
	m := setupConnectedModelWithTable(t, 100, 30)

	cmd := m.resultsView.LoadTable("", "test_users")
	msg := cmd()
	m.resultsView.Update(msg)
	m.mode = ViewResults

	m.Update(tea.KeyPressMsg{Text: "u", Code: 'u'})

	if m.mode != ViewRowWrite {
		t.Fatalf("expected mode ViewRowWrite after edit-row shortcut, got %v", m.mode)
	}
	if m.rowWriteView.action != rowWriteActionEdit {
		t.Fatalf("expected edit action, got %v", m.rowWriteView.action)
	}
	if m.rowWriteView.tableName != "test_users" {
		t.Fatalf("expected target table test_users, got %q", m.rowWriteView.tableName)
	}
}

func TestResultsView_EditRowShortcutBlockedWhenUpdatesDisabled(t *testing.T) {
	m := setupConnectedModelWithTable(t, 100, 30)

	cmd := m.resultsView.LoadTable("", "test_users")
	msg := cmd()
	m.resultsView.Update(msg)
	m.mode = ViewResults
	m.resultsView.results.DisableUpdate = true

	m.Update(tea.KeyPressMsg{Text: "u", Code: 'u'})

	if m.mode != ViewResults {
		t.Fatalf("expected mode ViewResults when updates are disabled, got %v", m.mode)
	}
}

func TestRowWriteView_VisibleInputRange_ReachesLastInputs(t *testing.T) {
	setupTestEnv(t)

	parent := NewMainModel()
	if parent.err != nil {
		t.Fatalf("NewMainModel failed: %v", parent.err)
	}
	parent.height = 18
	parent.mode = ViewRowWrite

	view := NewRowWriteView(parent)
	columns := make([]engine.Column, 0, 12)
	for idx := 0; idx < 12; idx++ {
		columns = append(columns, engine.Column{
			Name: "col_" + string(rune('a'+idx)),
			Type: "text",
		})
	}
	view.SetAddContext("public", "data_types", columns)
	view.focusIndex = len(view.inputs) - 1

	start, end := view.visibleInputRange()
	if end != len(view.inputs) {
		t.Fatalf("expected visible range to reach last input, got %d-%d of %d", start, end, len(view.inputs))
	}
	if start >= end {
		t.Fatalf("expected non-empty visible range, got %d-%d", start, end)
	}
}

func TestRowWriteView_AddEditAndDeleteFlow(t *testing.T) {
	m := setupConnectedModelWithTable(t, 100, 30)

	cmd := m.resultsView.LoadTable("", "test_users")
	msg := cmd()
	m.resultsView.Update(msg)
	m.mode = ViewResults

	m.Update(tea.KeyPressMsg{Text: "a", Code: 'a'})
	if m.mode != ViewRowWrite {
		t.Fatalf("expected mode ViewRowWrite after add-row shortcut, got %v", m.mode)
	}

	inputByName := map[string]int{}
	for idx, column := range m.rowWriteView.inputColumns {
		inputByName[column.Name] = idx
	}
	nameIdx, ok := inputByName["name"]
	if !ok {
		t.Fatalf("expected name input in row form, got columns %+v", m.rowWriteView.inputColumns)
	}
	emailIdx, ok := inputByName["email"]
	if !ok {
		t.Fatalf("expected email input in row form, got columns %+v", m.rowWriteView.inputColumns)
	}
	m.rowWriteView.inputs[nameIdx].SetValue("carol")
	m.rowWriteView.inputs[emailIdx].SetValue("c@b.com")

	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModAlt})
	if cmd == nil {
		t.Fatal("expected add-row command")
	}
	_, _ = m.Update(cmd())

	rows, err := m.dbManager.GetRows("", "test_users", nil, 50, 0)
	if err != nil {
		t.Fatalf("GetRows failed after add: %v", err)
	}
	if len(rows.Rows) != 3 {
		t.Fatalf("expected 3 rows after add, got %d", len(rows.Rows))
	}

	cmd = m.resultsView.LoadTable("", "test_users")
	msg = cmd()
	m.resultsView.Update(msg)
	m.mode = ViewResults
	m.resultsView.table.SetCursor(len(m.resultsView.currentPageRows()) - 1)

	m.Update(tea.KeyPressMsg{Text: "u", Code: 'u'})
	if m.mode != ViewRowWrite {
		t.Fatalf("expected mode ViewRowWrite after edit-row shortcut, got %v", m.mode)
	}
	if m.rowWriteView.action != rowWriteActionEdit {
		t.Fatalf("expected edit action, got %v", m.rowWriteView.action)
	}

	inputByName = map[string]int{}
	for idx, column := range m.rowWriteView.inputColumns {
		inputByName[column.Name] = idx
	}
	nameIdx, ok = inputByName["name"]
	if !ok {
		t.Fatalf("expected name input in edit form, got columns %+v", m.rowWriteView.inputColumns)
	}
	m.rowWriteView.inputs[nameIdx].SetValue("carol-updated")

	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModAlt})
	if cmd == nil {
		t.Fatal("expected edit-row command")
	}
	_, _ = m.Update(cmd())

	rows, err = m.dbManager.GetRows("", "test_users", nil, 50, 0)
	if err != nil {
		t.Fatalf("GetRows failed after edit: %v", err)
	}
	if len(rows.Rows) != 3 {
		t.Fatalf("expected 3 rows after edit, got %d", len(rows.Rows))
	}
	lastRow := rows.Rows[len(rows.Rows)-1]
	if len(lastRow) < 2 || lastRow[1] != "carol-updated" {
		t.Fatalf("expected edited row value, got %+v", lastRow)
	}

	cmd = m.resultsView.LoadTable("", "test_users")
	msg = cmd()
	m.resultsView.Update(msg)
	m.mode = ViewResults
	m.resultsView.table.SetCursor(len(m.resultsView.currentPageRows()) - 1)

	m.Update(tea.KeyPressMsg{Text: "d", Code: 'd'})
	if m.mode != ViewRowWrite {
		t.Fatalf("expected mode ViewRowWrite after delete-row shortcut, got %v", m.mode)
	}

	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected delete-row command")
	}
	_, _ = m.Update(cmd())

	rows, err = m.dbManager.GetRows("", "test_users", nil, 50, 0)
	if err != nil {
		t.Fatalf("GetRows failed after delete: %v", err)
	}
	if len(rows.Rows) != 2 {
		t.Fatalf("expected 2 rows after delete, got %d", len(rows.Rows))
	}
}
