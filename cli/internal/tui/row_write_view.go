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
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/clidey/whodb/cli/pkg/styles"
	"github.com/clidey/whodb/core/src/engine"
)

type rowWriteAction int

const (
	rowWriteActionAdd rowWriteAction = iota
	rowWriteActionEdit
	rowWriteActionDelete
)

type rowWriteResultMsg struct {
	action rowWriteAction
	err    error
}

// RowWriteView provides an interactive add-row, edit-row, and delete-row workflow.
type RowWriteView struct {
	parent *MainModel
	width  int
	height int

	action         rowWriteAction
	schema         string
	tableName      string
	columns        []engine.Column
	inputColumns   []engine.Column
	inputs         []textinput.Model
	focusIndex     int
	documentMode   bool
	textarea       textarea.Model
	originalValues map[string]string
	deleteValues   map[string]string
	deletePreview  []string
	working        bool
	err            error
}

// NewRowWriteView creates a new row-write modal view.
func NewRowWriteView(parent *MainModel) *RowWriteView {
	ta := textarea.New()
	ta.Placeholder = "{\n  \"field\": \"value\"\n}"
	ta.CharLimit = 0
	ta.Focus()
	ta.SetWidth(72)
	ta.SetHeight(12)

	return &RowWriteView{
		parent:     parent,
		width:      80,
		height:     24,
		action:     rowWriteActionAdd,
		focusIndex: 0,
		textarea:   ta,
	}
}

// SetAddContext prepares the add-row workflow for the selected storage unit.
func (v *RowWriteView) SetAddContext(schema, tableName string, columns []engine.Column) {
	v.action = rowWriteActionAdd
	v.schema = schema
	v.tableName = tableName
	v.columns = append([]engine.Column(nil), columns...)
	v.inputColumns = writableColumns(columns)
	v.documentMode = len(v.inputColumns) == 1 && strings.EqualFold(v.inputColumns[0].Type, "Document")
	v.inputs = buildRowInputs(v.inputColumns)
	v.focusIndex = 0
	v.originalValues = nil
	v.deleteValues = nil
	v.deletePreview = nil
	v.working = false
	v.err = nil
	if v.documentMode {
		v.textarea.SetValue("{\n  \n}")
	} else {
		v.textarea.SetValue("")
	}
	v.applyDimensions()
	v.syncAddFocus()
}

// SetEditContext prepares the edit-row workflow for the selected row.
func (v *RowWriteView) SetEditContext(schema, tableName string, columns []engine.Column, values map[string]string) {
	v.action = rowWriteActionEdit
	v.schema = schema
	v.tableName = tableName
	v.columns = append([]engine.Column(nil), columns...)
	v.originalValues = copyStringMap(values)
	v.inputColumns = editableColumns(columns)
	v.documentMode = len(v.inputColumns) == 1 && strings.EqualFold(v.inputColumns[0].Type, "Document")
	v.inputs = buildRowInputs(v.inputColumns)
	v.focusIndex = 0
	v.deleteValues = nil
	v.deletePreview = nil
	v.working = false
	v.err = nil

	if v.documentMode {
		v.textarea.SetValue(formatDocumentValue(values["document"]))
	} else {
		v.textarea.SetValue("")
		for idx, column := range v.inputColumns {
			v.inputs[idx].SetValue(values[column.Name])
		}
	}
	v.applyDimensions()
	v.syncAddFocus()
}

// SetDeleteContext prepares the delete-row workflow for the selected row.
func (v *RowWriteView) SetDeleteContext(schema, tableName string, columns []engine.Column, values map[string]string) {
	v.action = rowWriteActionDelete
	v.schema = schema
	v.tableName = tableName
	v.columns = append([]engine.Column(nil), columns...)
	v.inputColumns = nil
	v.inputs = nil
	v.documentMode = false
	v.originalValues = nil
	v.deleteValues = copyStringMap(values)
	v.deletePreview = buildDeletePreview(columns, values)
	v.working = false
	v.err = nil
	v.textarea.Blur()
}

func (v *RowWriteView) Update(msg tea.Msg) (*RowWriteView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.applyDimensions()
		return v, nil

	case rowWriteResultMsg:
		v.working = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}

		v.err = nil
		if !v.parent.PopView() {
			v.parent.mode = ViewResults
		}

		actionLabel := "row added"
		if msg.action == rowWriteActionEdit {
			actionLabel = "row updated"
		}
		if msg.action == rowWriteActionDelete {
			actionLabel = "row deleted"
		}

		cmds := []tea.Cmd{v.parent.SetStatus(actionLabel)}
		if v.parent.resultsView.tableName == v.tableName && v.parent.resultsView.schema == v.schema {
			cmds = append(cmds, v.parent.resultsView.loadPage())
		}
		return v, tea.Batch(cmds...)

	case tea.KeyPressMsg:
		if v.working {
			if msg.String() == "ctrl+c" {
				return v, tea.Quit
			}
			return v, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return v, tea.Quit
		case "esc":
			if !v.parent.PopView() {
				v.parent.mode = ViewResults
			}
			return v, nil
		}

		if v.action == rowWriteActionDelete {
			if msg.String() == "enter" {
				return v, v.startDelete()
			}
			return v, nil
		}

		if key.Matches(msg, rowWriteSubmitBinding(v.action)) {
			if v.action == rowWriteActionEdit {
				return v, v.startEdit()
			}
			return v, v.startAdd()
		}

		if v.documentMode {
			var cmd tea.Cmd
			v.textarea, cmd = v.textarea.Update(msg)
			return v, cmd
		}

		switch msg.String() {
		case "tab", "down", "j", "enter":
			v.moveFocus(1)
			return v, nil
		case "shift+tab", "up", "k":
			v.moveFocus(-1)
			return v, nil
		}

		if len(v.inputs) == 0 {
			return v, nil
		}

		var cmd tea.Cmd
		v.inputs[v.focusIndex], cmd = v.inputs[v.focusIndex].Update(msg)
		return v, cmd
	}

	return v, nil
}

func (v *RowWriteView) applyDimensions() {
	v.textarea.SetWidth(clamp(v.width-10, 28, 100))

	height := v.height - 18
	if height < 8 {
		height = 8
	}
	if height > 18 {
		height = 18
	}
	v.textarea.SetHeight(height)
	for idx := range v.inputs {
		v.inputs[idx].SetWidth(clamp(v.width-12, 24, 96))
	}
}

func (v *RowWriteView) startAdd() tea.Cmd {
	if strings.TrimSpace(v.tableName) == "" {
		v.err = fmt.Errorf("table name is required")
		return nil
	}

	v.err = nil
	v.working = true

	payload, err := v.buildAddPayload()
	if err != nil {
		v.working = false
		v.err = err
		return nil
	}

	schema := v.schema
	tableName := v.tableName
	mgr := v.parent.dbManager

	return func() tea.Msg {
		return rowWriteResultMsg{
			action: rowWriteActionAdd,
			err:    mgr.AddRowFromJSON(schema, tableName, payload),
		}
	}
}

func (v *RowWriteView) startDelete() tea.Cmd {
	if strings.TrimSpace(v.tableName) == "" {
		v.err = fmt.Errorf("table name is required")
		return nil
	}

	v.err = nil
	v.working = true

	schema := v.schema
	tableName := v.tableName
	values := copyStringMap(v.deleteValues)
	mgr := v.parent.dbManager

	return func() tea.Msg {
		return rowWriteResultMsg{
			action: rowWriteActionDelete,
			err:    mgr.DeleteRow(schema, tableName, values),
		}
	}
}

func (v *RowWriteView) View() string {
	var b strings.Builder

	title := "Add Row"
	if v.action == rowWriteActionEdit {
		title = "Edit Row"
	}
	if v.action == rowWriteActionDelete {
		title = "Delete Row"
	}
	b.WriteString(styles.RenderTitle(title))
	b.WriteString("\n\n")

	if v.schema != "" {
		b.WriteString(styles.RenderMuted("  Schema: " + v.schema))
		b.WriteString("\n")
	}
	b.WriteString(styles.RenderMuted("  Target: " + v.tableName))
	b.WriteString("\n\n")

	if v.action == rowWriteActionAdd || v.action == rowWriteActionEdit {
		v.renderAddView(&b)
	} else {
		b.WriteString(styles.RenderErr("  This will delete the selected row."))
		b.WriteString("\n")
		b.WriteString(styles.RenderMuted("  Primary key columns are preferred when the plugin supports them."))
		b.WriteString("\n\n")
		if len(v.deletePreview) == 0 {
			b.WriteString(styles.RenderMuted("  No row values available"))
		} else {
			for _, line := range v.deletePreview {
				b.WriteString("  " + line)
				b.WriteString("\n")
			}
		}
	}

	if v.working {
		b.WriteString("\n\n")
		spinnerLabel := "Writing row..."
		if v.action == rowWriteActionEdit {
			spinnerLabel = "Updating row..."
		}
		if v.action == rowWriteActionDelete {
			spinnerLabel = "Deleting row..."
		}
		b.WriteString(v.parent.SpinnerView() + styles.RenderMuted(" "+spinnerLabel))
	}

	if v.err != nil {
		b.WriteString("\n\n")
		b.WriteString(styles.RenderErrorBox(v.err.Error()))
	}

	b.WriteString("\n\n")
	b.WriteString(renderBindingHelpWidthNoHelp(v.width, rowWriteHelpBindings(v.action)...))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (v *RowWriteView) renderAddView(b *strings.Builder) {
	if v.documentMode {
		label := "  Document JSON:\n\n"
		if v.action == rowWriteActionEdit {
			label = "  Edit document JSON:\n\n"
		}
		b.WriteString(label)
		b.WriteString(v.textarea.View())
		b.WriteString("\n\n")
		b.WriteString(styles.RenderMuted("  Enter a JSON object for the document payload."))
		return
	}

	start, end := v.visibleInputRange()
	v.renderStructuredAddView(b, start, end)
}

func (v *RowWriteView) renderStructuredAddView(b *strings.Builder, start, end int) {
	prompt := "  Fill row values:\n\n"
	if v.action == rowWriteActionEdit {
		prompt = "  Edit row values:\n\n"
	}
	b.WriteString(prompt)
	if len(v.inputs) == 0 {
		b.WriteString(styles.RenderMuted("  No editable columns available"))
		return
	}

	if v.action == rowWriteActionEdit && len(v.columns) > len(v.inputColumns) {
		b.WriteString(styles.RenderMuted("  Primary key and database-managed columns are locked."))
		b.WriteString("\n\n")
	}

	if start > 0 {
		b.WriteString(styles.RenderMuted("  ..."))
		b.WriteString("\n\n")
	}

	for idx := start; idx < end; idx++ {
		column := v.inputColumns[idx]
		label := fmt.Sprintf("  %s", column.Name)
		if column.Type != "" {
			label += " (" + strings.ToLower(column.Type) + ")"
		}
		if idx == v.focusIndex {
			b.WriteString(styles.ActiveListItemStyle.Render(label))
		} else {
			b.WriteString(label)
		}
		b.WriteString("\n")
		b.WriteString("  " + v.inputs[idx].View())
		b.WriteString("\n\n")
	}

	if end < len(v.inputs) {
		b.WriteString(styles.RenderMuted("  ..."))
		b.WriteString("\n\n")
	}

	if v.action == rowWriteActionAdd {
		b.WriteString(styles.RenderMuted("  Leave fields blank to omit them from the insert."))
	}
}

func (v *RowWriteView) visibleInputRange() (int, int) {
	if len(v.inputs) == 0 {
		return 0, 0
	}

	availableHeight := v.parent.ContentHeight()
	if availableHeight <= 0 {
		availableHeight = v.height
	}
	if availableHeight <= 0 {
		availableHeight = 20
	}

	maxVisible := len(v.inputs)
	for maxVisible > 1 {
		start, end := v.computeVisibleInputRange(maxVisible)
		if lipgloss.Height(v.renderStructuredAddPreview(start, end)) <= availableHeight {
			return start, end
		}
		maxVisible--
	}

	return v.computeVisibleInputRange(1)
}

func (v *RowWriteView) computeVisibleInputRange(maxVisible int) (int, int) {
	if maxVisible >= len(v.inputs) {
		return 0, len(v.inputs)
	}

	start := v.focusIndex - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(v.inputs) {
		end = len(v.inputs)
		start = end - maxVisible
	}
	return start, end
}

func (v *RowWriteView) renderStructuredAddPreview(start, end int) string {
	var b strings.Builder

	title := "Add Row"
	if v.action == rowWriteActionEdit {
		title = "Edit Row"
	}
	b.WriteString(styles.RenderTitle(title))
	b.WriteString("\n\n")

	if v.schema != "" {
		b.WriteString(styles.RenderMuted("  Schema: " + v.schema))
		b.WriteString("\n")
	}
	b.WriteString(styles.RenderMuted("  Target: " + v.tableName))
	b.WriteString("\n\n")
	v.renderStructuredAddView(&b, start, end)

	if v.working {
		b.WriteString("\n\n")
		spinnerLabel := "Writing row..."
		if v.action == rowWriteActionEdit {
			spinnerLabel = "Updating row..."
		}
		b.WriteString(v.parent.SpinnerView() + styles.RenderMuted(" "+spinnerLabel))
	}

	if v.err != nil {
		b.WriteString("\n\n")
		b.WriteString(styles.RenderErrorBox(v.err.Error()))
	}

	b.WriteString("\n\n")
	b.WriteString(renderBindingHelpWidthNoHelp(v.width, rowWriteHelpBindings(v.action)...))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (v *RowWriteView) buildAddPayload() (string, error) {
	if v.documentMode {
		payload := strings.TrimSpace(v.textarea.Value())
		if payload == "" {
			return "", fmt.Errorf("fill at least one value")
		}
		return payload, nil
	}

	values := make(map[string]any)
	for idx, column := range v.inputColumns {
		value := strings.TrimSpace(v.inputs[idx].Value())
		if value == "" {
			continue
		}
		values[column.Name] = value
	}

	if len(values) == 0 {
		return "", fmt.Errorf("fill at least one value")
	}

	data, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (v *RowWriteView) buildEditValues() (map[string]string, error) {
	if v.documentMode {
		payload := strings.TrimSpace(v.textarea.Value())
		if payload == "" {
			return nil, fmt.Errorf("fill at least one value")
		}
		return map[string]string{"document": payload}, nil
	}

	values := make(map[string]string, len(v.inputColumns))
	for idx, column := range v.inputColumns {
		values[column.Name] = v.inputs[idx].Value()
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("no editable columns available")
	}
	return values, nil
}

func (v *RowWriteView) startEdit() tea.Cmd {
	if strings.TrimSpace(v.tableName) == "" {
		v.err = fmt.Errorf("table name is required")
		return nil
	}

	v.err = nil
	v.working = true

	values, err := v.buildEditValues()
	if err != nil {
		v.working = false
		v.err = err
		return nil
	}

	schema := v.schema
	tableName := v.tableName
	originalValues := copyStringMap(v.originalValues)
	mgr := v.parent.dbManager

	return func() tea.Msg {
		return rowWriteResultMsg{
			action: rowWriteActionEdit,
			err:    mgr.UpdateRow(schema, tableName, originalValues, values),
		}
	}
}

func (v *RowWriteView) moveFocus(delta int) {
	if len(v.inputs) == 0 {
		return
	}
	v.focusIndex = (v.focusIndex + delta + len(v.inputs)) % len(v.inputs)
	v.syncAddFocus()
}

func (v *RowWriteView) syncAddFocus() {
	v.textarea.Blur()
	for idx := range v.inputs {
		v.inputs[idx].Blur()
	}

	if v.documentMode {
		v.textarea.Focus()
		return
	}

	if len(v.inputs) == 0 {
		return
	}
	v.inputs[v.focusIndex].Focus()
}

func writableColumns(columns []engine.Column) []engine.Column {
	writable := make([]engine.Column, 0, len(columns))
	for _, column := range columns {
		if column.IsAutoIncrement || column.IsComputed {
			continue
		}
		writable = append(writable, column)
	}
	return writable
}

func editableColumns(columns []engine.Column) []engine.Column {
	editable := make([]engine.Column, 0, len(columns))
	for _, column := range columns {
		if column.IsAutoIncrement || column.IsComputed || column.IsPrimary {
			continue
		}
		editable = append(editable, column)
	}
	return editable
}

func buildRowInputs(columns []engine.Column) []textinput.Model {
	inputs := make([]textinput.Model, 0, len(columns))
	for _, column := range columns {
		input := textinput.New()
		input.Placeholder = rowInputPlaceholder(column)
		input.CharLimit = 0
		input.SetWidth(72)
		inputStyles := input.Styles()
		inputStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.Primary)
		inputStyles.Focused.Text = lipgloss.NewStyle().Foreground(styles.Foreground)
		inputStyles.Cursor.Color = styles.Primary
		input.SetStyles(inputStyles)
		inputs = append(inputs, input)
	}
	return inputs
}

func rowInputPlaceholder(column engine.Column) string {
	if column.Type == "" {
		return "Enter value"
	}
	return "Enter " + strings.ToLower(column.Type) + " value"
}

func buildDeletePreview(columns []engine.Column, values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}

	lines := make([]string, 0, min(len(values), 8))
	seen := make(map[string]struct{}, len(values))

	for _, column := range columns {
		value, ok := values[column.Name]
		if !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s", column.Name, truncateRowValue(value)))
		seen[column.Name] = struct{}{}
		if len(lines) == 8 {
			return append(lines, "...")
		}
	}

	for key, value := range values {
		if _, ok := seen[key]; ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s", key, truncateRowValue(value)))
		if len(lines) == 8 {
			return append(lines, "...")
		}
	}

	return lines
}

func truncateRowValue(value string) string {
	if len(value) <= 80 {
		return value
	}
	return value[:77] + "..."
}

func copyStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}

	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func rowWriteSubmitBinding(action rowWriteAction) key.Binding {
	if action == rowWriteActionDelete {
		return key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "delete row"),
		)
	}
	if action == rowWriteActionEdit {
		return key.NewBinding(
			key.WithKeys("alt+enter"),
			key.WithHelp("alt+enter", "update row"),
		)
	}
	return key.NewBinding(
		key.WithKeys("alt+enter"),
		key.WithHelp("alt+enter", "insert row"),
	)
}

func rowWriteHelpBindings(action rowWriteAction) []key.Binding {
	return []key.Binding{
		rowWriteSubmitBinding(action),
		Keys.Global.Back,
		Keys.Global.Quit,
	}
}

func formatDocumentValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "{\n  \n}"
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return value
	}

	pretty, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return value
	}
	return string(pretty)
}
