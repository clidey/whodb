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
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/clidey/whodb/cli/pkg/styles"
	coremockdata "github.com/clidey/whodb/core/src/mockdata"
)

type mockDataStep int

const (
	mockDataStepConfig mockDataStep = iota
	mockDataStepPlan
	mockDataStepDone
)

type mockDataAnalysisMsg struct {
	analysis *coremockdata.DependencyAnalysis
	err      error
}

type mockDataResultMsg struct {
	result *coremockdata.GenerationResult
	err    error
}

// MockDataView provides an interactive mock-data workflow for the TUI.
type MockDataView struct {
	parent *MainModel
	width  int
	height int

	step         mockDataStep
	tableInput   textinput.Model
	rowsInput    textinput.Model
	densityInput textinput.Model
	focusIndex   int
	schema       string
	overwrite    bool
	analyzing    bool
	generating   bool
	analysis     *coremockdata.DependencyAnalysis
	result       *coremockdata.GenerationResult
	err          error
}

// NewMockDataView creates a new interactive mock-data view.
func NewMockDataView(parent *MainModel) *MockDataView {
	tableInput := newMockDataInput("table_name", 40)
	rowsInput := newMockDataInput("50", 8)
	densityInput := newMockDataInput("default", 10)

	rowsInput.SetValue("50")

	return &MockDataView{
		parent:       parent,
		tableInput:   tableInput,
		rowsInput:    rowsInput,
		densityInput: densityInput,
		width:        80,
		height:       20,
	}
}

func newMockDataInput(placeholder string, width int) textinput.Model {
	input := textinput.New()
	input.Placeholder = placeholder
	input.CharLimit = 128
	input.SetWidth(width)
	inputStyles := input.Styles()
	inputStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.Primary)
	inputStyles.Focused.Text = lipgloss.NewStyle().Foreground(styles.Foreground)
	inputStyles.Cursor.Color = styles.Primary
	input.SetStyles(inputStyles)
	return input
}

// SetTarget resets the form and pre-fills the current schema/table context.
func (v *MockDataView) SetTarget(schema, table string) {
	v.step = mockDataStepConfig
	v.schema = schema
	v.overwrite = false
	v.analyzing = false
	v.generating = false
	v.analysis = nil
	v.result = nil
	v.err = nil

	v.tableInput.SetValue(table)
	v.rowsInput.SetValue("50")
	v.densityInput.SetValue("")

	if strings.TrimSpace(table) == "" {
		v.focusIndex = 0
	} else {
		v.focusIndex = 1
	}
	v.syncFocus()
}

func (v *MockDataView) Update(msg tea.Msg) (*MockDataView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.tableInput.SetWidth(clamp(msg.Width-18, 18, 48))
		v.rowsInput.SetWidth(10)
		v.densityInput.SetWidth(12)
		return v, nil

	case mockDataAnalysisMsg:
		v.analyzing = false
		if msg.err != nil {
			v.err = msg.err
			v.analysis = nil
			return v, nil
		}
		v.err = nil
		v.result = nil
		v.analysis = msg.analysis
		v.step = mockDataStepPlan
		return v, nil

	case mockDataResultMsg:
		v.generating = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}

		v.err = nil
		v.result = msg.result
		v.step = mockDataStepDone

		cmds := []tea.Cmd{
			v.parent.SetStatus(fmt.Sprintf("Generated mock data for %s", strings.TrimSpace(v.tableInput.Value()))),
		}
		if v.parent.resultsView.tableName == strings.TrimSpace(v.tableInput.Value()) &&
			v.parent.resultsView.schema == v.schema {
			cmds = append(cmds, v.parent.resultsView.LoadTable(v.schema, v.parent.resultsView.tableName))
		}
		return v, tea.Batch(cmds...)

	case tea.KeyPressMsg:
		if v.analyzing || v.generating {
			switch msg.String() {
			case "ctrl+c":
				return v, tea.Quit
			}
			return v, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return v, tea.Quit

		case "esc":
			if v.step == mockDataStepPlan {
				v.step = mockDataStepConfig
				return v, nil
			}
			if !v.parent.PopView() {
				v.parent.mode = ViewBrowser
			}
			return v, nil
		}

		switch v.step {
		case mockDataStepConfig:
			return v.updateConfig(msg)
		case mockDataStepPlan:
			return v.updatePlan(msg)
		case mockDataStepDone:
			return v.updateDone(msg)
		}
	}

	return v, nil
}

func (v *MockDataView) updateConfig(msg tea.KeyPressMsg) (*MockDataView, tea.Cmd) {
	switch msg.String() {
	case "tab", "down", "j":
		v.moveFocus(1)
		return v, nil
	case "shift+tab", "up", "k":
		v.moveFocus(-1)
		return v, nil
	case "space":
		if v.focusIndex == 2 {
			v.overwrite = !v.overwrite
			return v, nil
		}
	case "enter":
		switch v.focusIndex {
		case 0, 1, 3:
			v.moveFocus(1)
			return v, nil
		case 2:
			v.overwrite = !v.overwrite
			return v, nil
		case 4:
			return v, v.startAnalysis()
		}
	}

	switch v.focusIndex {
	case 0:
		var cmd tea.Cmd
		v.tableInput, cmd = v.tableInput.Update(msg)
		return v, cmd
	case 1:
		var cmd tea.Cmd
		v.rowsInput, cmd = v.rowsInput.Update(msg)
		return v, cmd
	case 3:
		var cmd tea.Cmd
		v.densityInput, cmd = v.densityInput.Update(msg)
		return v, cmd
	}

	return v, nil
}

func (v *MockDataView) updatePlan(msg tea.KeyPressMsg) (*MockDataView, tea.Cmd) {
	switch msg.String() {
	case "enter":
		return v, v.startGeneration()
	case "a":
		return v, v.startAnalysis()
	}
	return v, nil
}

func (v *MockDataView) updateDone(msg tea.KeyPressMsg) (*MockDataView, tea.Cmd) {
	if msg.String() == "enter" {
		if !v.parent.PopView() {
			v.parent.mode = ViewBrowser
		}
	}
	return v, nil
}

func (v *MockDataView) moveFocus(delta int) {
	v.focusIndex = (v.focusIndex + delta + 5) % 5
	v.syncFocus()
}

func (v *MockDataView) syncFocus() {
	v.tableInput.Blur()
	v.rowsInput.Blur()
	v.densityInput.Blur()

	switch v.focusIndex {
	case 0:
		v.tableInput.Focus()
	case 1:
		v.rowsInput.Focus()
	case 3:
		v.densityInput.Focus()
	}
}

func (v *MockDataView) startAnalysis() tea.Cmd {
	table, rowCount, fkDensityRatio, err := v.parseInputs()
	if err != nil {
		v.err = err
		return nil
	}

	v.err = nil
	v.result = nil
	v.analysis = nil
	v.analyzing = true
	schema := v.schema
	mgr := v.parent.dbManager

	return func() tea.Msg {
		analysis, err := mgr.AnalyzeMockDataDependencies(schema, table, rowCount, fkDensityRatio)
		return mockDataAnalysisMsg{analysis: analysis, err: err}
	}
}

func (v *MockDataView) startGeneration() tea.Cmd {
	table, rowCount, fkDensityRatio, err := v.parseInputs()
	if err != nil {
		v.err = err
		return nil
	}

	v.err = nil
	v.generating = true
	schema := v.schema
	overwrite := v.overwrite
	mgr := v.parent.dbManager

	return func() tea.Msg {
		result, err := mgr.GenerateMockData(schema, table, rowCount, overwrite, fkDensityRatio)
		return mockDataResultMsg{result: result, err: err}
	}
}

func (v *MockDataView) parseInputs() (string, int, int, error) {
	table := strings.TrimSpace(v.tableInput.Value())
	if table == "" {
		return "", 0, 0, fmt.Errorf("table name is required")
	}

	rowText := strings.TrimSpace(v.rowsInput.Value())
	rowCount, err := strconv.Atoi(rowText)
	if err != nil || rowCount <= 0 {
		return "", 0, 0, fmt.Errorf("rows must be a positive integer")
	}

	fkDensityRatio := 0
	densityText := strings.TrimSpace(v.densityInput.Value())
	if densityText != "" {
		fkDensityRatio, err = strconv.Atoi(densityText)
		if err != nil || fkDensityRatio < 0 {
			return "", 0, 0, fmt.Errorf("FK density ratio must be 0 or greater")
		}
	}

	return table, rowCount, fkDensityRatio, nil
}

func (v *MockDataView) View() string {
	var b strings.Builder

	b.WriteString(styles.RenderTitle("Mock Data"))
	b.WriteString("\n\n")

	if v.schema != "" {
		b.WriteString(styles.RenderMuted("  Schema: " + v.schema))
		b.WriteString("\n\n")
	}

	switch v.step {
	case mockDataStepConfig:
		b.WriteString("  Table or collection:\n")
		b.WriteString("  " + v.tableInput.View())
		b.WriteString("\n\n")

		b.WriteString("  Rows to generate:\n")
		b.WriteString("  " + v.rowsInput.View())
		b.WriteString("\n\n")

		overwriteLine := "  [ ] Overwrite existing rows"
		if v.overwrite {
			overwriteLine = "  [x] Overwrite existing rows"
		}
		if v.focusIndex == 2 {
			b.WriteString(styles.ActiveListItemStyle.Render(overwriteLine))
		} else {
			b.WriteString(overwriteLine)
		}
		b.WriteString("\n\n")

		b.WriteString("  FK density ratio (optional):\n")
		b.WriteString("  " + v.densityInput.View())
		b.WriteString("\n")
		b.WriteString(styles.RenderMuted("  Leave blank to use the backend default"))
		b.WriteString("\n\n")

		analyzeButton := "  [Analyze plan]"
		if v.focusIndex == 4 {
			b.WriteString(styles.ActiveListItemStyle.Render(analyzeButton))
		} else {
			b.WriteString(styles.RenderKey(analyzeButton))
		}

		if v.analyzing {
			b.WriteString("\n\n")
			b.WriteString(v.parent.SpinnerView() + styles.RenderMuted(" Analyzing dependencies..."))
		}

	case mockDataStepPlan:
		if v.analysis != nil {
			b.WriteString(styles.RenderMuted(fmt.Sprintf("  Target: %s", strings.TrimSpace(v.tableInput.Value()))))
			b.WriteString("\n")
			b.WriteString(styles.RenderMuted(fmt.Sprintf("  Total rows planned: %d", v.analysis.TotalRows)))
			b.WriteString("\n\n")

			if len(v.analysis.GenerationOrder) > 0 {
				b.WriteString(styles.RenderKey("  Generation order"))
				b.WriteString("\n")
				b.WriteString("  " + strings.Join(v.analysis.GenerationOrder, " -> "))
				b.WriteString("\n\n")
			}

			b.WriteString(styles.RenderKey("  Tables"))
			b.WriteString("\n")
			for _, table := range v.analysis.Tables {
				line := fmt.Sprintf("  - %s: %d rows", table.Table, table.RowCount)
				if table.UsesExistingData {
					line += " (uses existing data)"
				}
				b.WriteString(line)
				b.WriteString("\n")
			}

			if len(v.analysis.Warnings) > 0 {
				b.WriteString("\n")
				b.WriteString(styles.RenderKey("  Warnings"))
				b.WriteString("\n")
				for _, warning := range v.analysis.Warnings {
					b.WriteString("  - " + warning)
					b.WriteString("\n")
				}
			}
		}

		if v.generating {
			b.WriteString("\n")
			b.WriteString(v.parent.SpinnerView() + styles.RenderMuted(" Generating mock data..."))
		} else {
			b.WriteString("\n")
			b.WriteString(styles.RenderKey("  Press Enter to generate"))
			b.WriteString("\n")
			b.WriteString(styles.RenderMuted("  Press A to re-analyze or Esc to edit"))
		}

	case mockDataStepDone:
		if v.result != nil {
			b.WriteString(styles.RenderSuccess(fmt.Sprintf("Generated %d rows", v.result.TotalGenerated)))
			b.WriteString("\n\n")
			for _, detail := range v.result.Details {
				line := fmt.Sprintf("  - %s: %d rows", detail.Table, detail.RowsGenerated)
				if detail.UsedExistingData {
					line += " (used existing data)"
				}
				b.WriteString(line)
				b.WriteString("\n")
			}
			if len(v.result.Warnings) > 0 {
				b.WriteString("\n")
				b.WriteString(styles.RenderKey("  Warnings"))
				b.WriteString("\n")
				for _, warning := range v.result.Warnings {
					b.WriteString("  - " + warning)
					b.WriteString("\n")
				}
			}
			b.WriteString("\n")
			b.WriteString(styles.RenderMuted("  Press Enter or Esc to return"))
		}
	}

	if v.err != nil {
		b.WriteString("\n\n")
		b.WriteString(styles.RenderErrorBox(v.err.Error()))
	}

	b.WriteString("\n\n")
	b.WriteString(renderBindingHelpWidthNoHelp(v.width,
		Keys.Global.Back,
		Keys.Global.Quit,
	))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}
