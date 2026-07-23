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
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/internal/schemadiff"
	"github.com/clidey/whodb/cli/pkg/styles"
)

type schemaDiffField int

const (
	schemaDiffFieldFromConnection schemaDiffField = iota
	schemaDiffFieldToConnection
	schemaDiffFieldFromSchema
	schemaDiffFieldToSchema
)

// SchemaDiffView renders an interactive schema diff workflow in the TUI.
// It lets the user pick two connections, optional schema overrides, and then
// browse the shared schema diff output in a scrollable viewport.
type SchemaDiffView struct {
	parent *MainModel

	width  int
	height int

	connections []database.Connection
	fromIndex   int
	toIndex     int
	field       schemaDiffField

	fromSchemaInput textinput.Model
	toSchemaInput   textinput.Model

	editing  bool
	loading  bool
	err      error
	result   *schemadiff.Result
	viewport viewport.Model
	ready    bool
}

// NewSchemaDiffView creates a new SchemaDiffView attached to the given parent
// model.
func NewSchemaDiffView(parent *MainModel) *SchemaDiffView {
	fromSchemaInput := textinput.New()
	fromSchemaInput.Placeholder = "connection default"
	fromSchemaInput.CharLimit = 128
	fromSchemaInput.SetWidth(28)
	fromSchemaInputStyles := fromSchemaInput.Styles()
	fromSchemaInputStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.Primary)
	fromSchemaInputStyles.Focused.Text = lipgloss.NewStyle().Foreground(styles.Foreground)
	fromSchemaInputStyles.Cursor.Color = styles.Primary
	fromSchemaInput.SetStyles(fromSchemaInputStyles)

	toSchemaInput := textinput.New()
	toSchemaInput.Placeholder = "connection default"
	toSchemaInput.CharLimit = 128
	toSchemaInput.SetWidth(28)
	toSchemaInputStyles := toSchemaInput.Styles()
	toSchemaInputStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.Primary)
	toSchemaInputStyles.Focused.Text = lipgloss.NewStyle().Foreground(styles.Foreground)
	toSchemaInputStyles.Cursor.Color = styles.Primary
	toSchemaInput.SetStyles(toSchemaInputStyles)

	return &SchemaDiffView{
		parent:          parent,
		width:           80,
		height:          24,
		editing:         true,
		fromSchemaInput: fromSchemaInput,
		toSchemaInput:   toSchemaInput,
	}
}

func (v *SchemaDiffView) prepare() {
	fromConnection := v.selectedConnectionName(v.fromIndex)
	toConnection := v.selectedConnectionName(v.toIndex)
	fromSchema := strings.TrimSpace(v.fromSchemaInput.Value())
	toSchema := strings.TrimSpace(v.toSchemaInput.Value())

	v.refreshConnections()
	v.editing = true
	v.loading = false
	v.err = nil
	v.result = nil
	v.ready = false

	defaultSchema := strings.TrimSpace(v.parent.browserView.currentSchema)
	if fromSchema == "" {
		fromSchema = defaultSchema
	}
	if toSchema == "" {
		toSchema = defaultSchema
	}
	v.fromSchemaInput.SetValue(fromSchema)
	v.toSchemaInput.SetValue(toSchema)
	v.selectConnectionByName(fromConnection, true)
	v.selectConnectionByName(toConnection, false)
	v.syncFocus()
}

func (v *SchemaDiffView) helpSafe() bool {
	return !v.editing && !v.loading
}

// HelpSafe reports whether the diff view can safely show the global help
// overlay without stealing focus from an active input field.
func (v *SchemaDiffView) HelpSafe() bool {
	return v.helpSafe()
}

// Update handles input and async messages for the schema diff view.
func (v *SchemaDiffView) Update(msg tea.Msg) (*SchemaDiffView, tea.Cmd) {
	switch msg := msg.(type) {
	case schemaDiffResultMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			v.result = nil
			v.ready = false
			v.editing = true
			v.syncFocus()
			return v, nil
		}

		v.err = nil
		v.result = msg.result
		v.editing = false
		v.rebuildViewport()
		return v, nil

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.fromSchemaInput.SetWidth(clamp(msg.Width-28, 20, 40))
		v.toSchemaInput.SetWidth(clamp(msg.Width-28, 20, 40))
		if v.result != nil && !v.editing {
			v.rebuildViewport()
		}
		return v, nil

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, Keys.Global.Back):
			if !v.parent.PopView() {
				v.parent.mode = ViewBrowser
			}
			return v, nil
		}

		if v.loading {
			return v, nil
		}

		if v.editing {
			return v.updateEditing(msg)
		}

		return v.updateResults(msg)
	}

	if !v.editing {
		var cmd tea.Cmd
		v.viewport, cmd = v.viewport.Update(msg)
		return v, cmd
	}

	return v, nil
}

func (v *SchemaDiffView) updateEditing(msg tea.KeyPressMsg) (*SchemaDiffView, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.SchemaDiff.PrevField):
		v.field--
		if v.field < schemaDiffFieldFromConnection {
			v.field = schemaDiffFieldToSchema
		}
		v.syncFocus()
		return v, nil

	case key.Matches(msg, Keys.SchemaDiff.NextField):
		v.field++
		if v.field > schemaDiffFieldToSchema {
			v.field = schemaDiffFieldFromConnection
		}
		v.syncFocus()
		return v, nil

	case key.Matches(msg, Keys.SchemaDiff.OptionLeft):
		if v.field == schemaDiffFieldFromConnection {
			v.shiftFromConnection(-1)
			return v, nil
		}
		if v.field == schemaDiffFieldToConnection {
			v.shiftToConnection(-1)
			return v, nil
		}

	case key.Matches(msg, Keys.SchemaDiff.OptionRight):
		if v.field == schemaDiffFieldFromConnection {
			v.shiftFromConnection(1)
			return v, nil
		}
		if v.field == schemaDiffFieldToConnection {
			v.shiftToConnection(1)
			return v, nil
		}

	case key.Matches(msg, Keys.SchemaDiff.Compare):
		return v, v.runComparison()
	}

	switch v.field {
	case schemaDiffFieldFromSchema:
		var cmd tea.Cmd
		v.fromSchemaInput, cmd = v.fromSchemaInput.Update(msg)
		return v, cmd
	case schemaDiffFieldToSchema:
		var cmd tea.Cmd
		v.toSchemaInput, cmd = v.toSchemaInput.Update(msg)
		return v, cmd
	default:
		return v, nil
	}
}

func (v *SchemaDiffView) updateResults(msg tea.KeyPressMsg) (*SchemaDiffView, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.SchemaDiff.Edit):
		v.editing = true
		v.syncFocus()
		return v, nil
	case key.Matches(msg, Keys.SchemaDiff.Recompare), key.Matches(msg, Keys.SchemaDiff.Compare):
		return v, v.runComparison()
	}

	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// View renders the schema diff view.
func (v *SchemaDiffView) View() string {
	var b strings.Builder

	b.WriteString(styles.RenderTitle("Schema Diff"))
	b.WriteString("\n\n")

	if len(v.connections) == 0 {
		b.WriteString(styles.RenderMuted("No connections available. Save or configure a connection first."))
		b.WriteString("\n\n")
		b.WriteString(RenderBindingHelpWidth(v.width, Keys.Global.Back))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	b.WriteString(v.renderSelectionForm())
	b.WriteString("\n")

	if v.err != nil {
		b.WriteString("\n")
		b.WriteString(styles.RenderErrorBox(v.err.Error()))
		b.WriteString("\n\n")
		b.WriteString(renderBindingHelpWidthNoHelp(v.width,
			Keys.SchemaDiff.PrevField,
			Keys.SchemaDiff.NextField,
			Keys.SchemaDiff.OptionLeft,
			Keys.SchemaDiff.OptionRight,
			Keys.SchemaDiff.Compare,
			Keys.Global.Back,
		))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	if v.loading {
		b.WriteString("\n")
		b.WriteString(v.parent.SpinnerView() + styles.RenderMuted(" Comparing schema metadata..."))
		b.WriteString("\n\n")
		b.WriteString(renderBindingHelpWidthNoHelp(v.width, Keys.Global.Back))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	if v.editing || v.result == nil {
		b.WriteString("\n")
		b.WriteString(styles.RenderMuted("Pick two connections, adjust schema overrides if needed, then press Enter to compare."))
		b.WriteString("\n\n")
		b.WriteString(renderBindingHelpWidthNoHelp(v.width,
			Keys.SchemaDiff.PrevField,
			Keys.SchemaDiff.NextField,
			Keys.SchemaDiff.OptionLeft,
			Keys.SchemaDiff.OptionRight,
			Keys.SchemaDiff.Compare,
			Keys.Global.Back,
		))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	summary := fmt.Sprintf(
		"Storage units +%d -%d ~%d • Columns +%d -%d ~%d • Relationships +%d -%d ~%d",
		v.result.Summary.AddedStorageUnits,
		v.result.Summary.RemovedStorageUnits,
		v.result.Summary.ChangedStorageUnits,
		v.result.Summary.AddedColumns,
		v.result.Summary.RemovedColumns,
		v.result.Summary.ChangedColumns,
		v.result.Summary.AddedRelationships,
		v.result.Summary.RemovedRelationships,
		v.result.Summary.ChangedRelationships,
	)
	b.WriteString("\n")
	b.WriteString(styles.RenderMuted(summary))
	b.WriteString("\n\n")

	if !v.ready {
		v.rebuildViewport()
	}
	b.WriteString(v.viewport.View())
	b.WriteString("\n\n")
	b.WriteString(RenderBindingHelpWidth(v.width,
		Keys.SchemaDiff.Recompare,
		Keys.SchemaDiff.Edit,
		Keys.SchemaDiff.ScrollUp,
		Keys.SchemaDiff.ScrollDown,
		Keys.Global.Back,
	))

	if v.viewport.TotalLineCount() > v.viewport.VisibleLineCount() {
		pct := v.viewport.ScrollPercent() * 100
		scrollPct := fmt.Sprintf("%.0f%%", pct)
		if pct >= 99.5 {
			scrollPct = "bottom"
		}
		b.WriteString("  ")
		b.WriteString(styles.RenderMuted(scrollPct))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (v *SchemaDiffView) renderSelectionForm() string {
	var b strings.Builder

	b.WriteString(v.renderSelectionRow("From connection", v.currentConnectionLabel(v.fromIndex), v.field == schemaDiffFieldFromConnection))
	b.WriteString("\n")
	b.WriteString(v.renderSelectionRow("To connection", v.currentConnectionLabel(v.toIndex), v.field == schemaDiffFieldToConnection))
	b.WriteString("\n")
	b.WriteString(v.renderSelectionRow("From schema", v.fromSchemaInput.View(), v.field == schemaDiffFieldFromSchema))
	b.WriteString("\n")
	b.WriteString(v.renderSelectionRow("To schema", v.toSchemaInput.View(), v.field == schemaDiffFieldToSchema))

	return b.String()
}

func (v *SchemaDiffView) renderSelectionRow(label, value string, active bool) string {
	labelWidth := 18
	prefix := "  "
	if active {
		prefix = styles.RenderKey("> ")
		value = styles.ActiveListItemStyle.Render(value)
	}
	return fmt.Sprintf("%s%-*s %s", prefix, labelWidth, label+":", value)
}

func (v *SchemaDiffView) syncFocus() {
	v.fromSchemaInput.Blur()
	v.toSchemaInput.Blur()

	switch v.field {
	case schemaDiffFieldFromSchema:
		v.fromSchemaInput.Focus()
	case schemaDiffFieldToSchema:
		v.toSchemaInput.Focus()
	}
}

func (v *SchemaDiffView) refreshConnections() {
	available := v.parent.dbManager.ListAvailableConnections()
	connections := make([]database.Connection, 0, len(available)+1)

	current := v.parent.dbManager.GetCurrentConnection()
	if current != nil {
		connections = append(connections, *current)
	}

	for _, conn := range available {
		if !containsSchemaDiffConnection(connections, conn) {
			connections = append(connections, conn)
		}
	}

	v.connections = connections
	v.fromIndex = 0
	v.toIndex = 0
	if len(v.connections) > 1 {
		v.toIndex = 1
	}
}

func (v *SchemaDiffView) shiftFromConnection(delta int) {
	if len(v.connections) == 0 {
		return
	}
	v.fromIndex = cycleSchemaDiffIndex(v.fromIndex, delta, len(v.connections))
}

func (v *SchemaDiffView) shiftToConnection(delta int) {
	if len(v.connections) == 0 {
		return
	}
	v.toIndex = cycleSchemaDiffIndex(v.toIndex, delta, len(v.connections))
}

func (v *SchemaDiffView) currentConnectionLabel(idx int) string {
	if idx < 0 || idx >= len(v.connections) {
		return ""
	}
	conn := v.connections[idx]
	label := schemaDiffConnectionLabel(conn)

	current := v.parent.dbManager.GetCurrentConnection()
	if current != nil && sameSchemaDiffConnection(conn, *current) {
		label += " (current)"
	}

	return label
}

func (v *SchemaDiffView) selectedConnectionName(idx int) string {
	if idx < 0 || idx >= len(v.connections) {
		return ""
	}
	return strings.TrimSpace(v.connections[idx].Name)
}

func (v *SchemaDiffView) selectConnectionByName(name string, from bool) {
	if strings.TrimSpace(name) == "" {
		return
	}

	for i, conn := range v.connections {
		if strings.TrimSpace(conn.Name) != name {
			continue
		}
		if from {
			v.fromIndex = i
		} else {
			v.toIndex = i
		}
		return
	}
}

// SelectionState returns the current schema diff selection inputs for
// persistence in the CLI workspace snapshot.
func (v *SchemaDiffView) SelectionState() config.WorkspaceDiffState {
	return config.WorkspaceDiffState{
		FromConnection: v.selectedConnectionName(v.fromIndex),
		ToConnection:   v.selectedConnectionName(v.toIndex),
		FromSchema:     strings.TrimSpace(v.fromSchemaInput.Value()),
		ToSchema:       strings.TrimSpace(v.toSchemaInput.Value()),
	}
}

// SetSelectionState restores persisted schema diff inputs into the diff view.
func (v *SchemaDiffView) SetSelectionState(state config.WorkspaceDiffState) {
	v.refreshConnections()
	v.selectConnectionByName(state.FromConnection, true)
	v.selectConnectionByName(state.ToConnection, false)
	v.fromSchemaInput.SetValue(strings.TrimSpace(state.FromSchema))
	v.toSchemaInput.SetValue(strings.TrimSpace(state.ToSchema))
	v.syncFocus()
}

func (v *SchemaDiffView) runComparison() tea.Cmd {
	if len(v.connections) == 0 {
		return nil
	}

	fromConn := v.connections[v.fromIndex]
	toConn := v.connections[v.toIndex]
	if strings.TrimSpace(fromConn.Name) == "" {
		fromConn.Name = schemaDiffConnectionLabel(fromConn)
	}
	if strings.TrimSpace(toConn.Name) == "" {
		toConn.Name = schemaDiffConnectionLabel(toConn)
	}

	fromSchema := strings.TrimSpace(v.fromSchemaInput.Value())
	toSchema := strings.TrimSpace(v.toSchemaInput.Value())

	v.loading = true
	v.err = nil
	v.result = nil
	v.ready = false

	return func() tea.Msg {
		result, err := schemadiff.CompareConnections(&fromConn, &toConn, fromSchema, toSchema)
		return schemaDiffResultMsg{result: result, err: err}
	}
}

func (v *SchemaDiffView) rebuildViewport() {
	contentWidth := v.width - 8
	if contentWidth < 30 {
		contentWidth = 30
	}

	contentHeight := v.height - 18
	if contentHeight < 5 {
		contentHeight = 5
	}

	v.viewport = viewport.New(viewport.WithWidth(contentWidth), viewport.WithHeight(contentHeight))
	v.viewport.SetContent(schemadiff.RenderText(v.result))
	v.ready = true
}

func cycleSchemaDiffIndex(current, delta, length int) int {
	if length <= 0 {
		return 0
	}

	next := current + delta
	for next < 0 {
		next += length
	}
	return next % length
}

func containsSchemaDiffConnection(connections []database.Connection, target database.Connection) bool {
	for _, conn := range connections {
		if sameSchemaDiffConnection(conn, target) {
			return true
		}
	}
	return false
}

func sameSchemaDiffConnection(left, right database.Connection) bool {
	if left.Name != "" && right.Name != "" {
		return left.Name == right.Name
	}

	return left.Type == right.Type &&
		left.Host == right.Host &&
		left.Port == right.Port &&
		left.Username == right.Username &&
		left.Database == right.Database &&
		left.Schema == right.Schema
}

func schemaDiffConnectionLabel(conn database.Connection) string {
	if strings.TrimSpace(conn.Name) != "" {
		return conn.Name
	}

	databaseName := strings.TrimSpace(conn.Database)
	if databaseName == "" {
		databaseName = strings.TrimSpace(conn.Schema)
	}
	if databaseName == "" {
		databaseName = strings.TrimSpace(conn.Host)
	}

	return fmt.Sprintf("%s %s", conn.Type, databaseName)
}
