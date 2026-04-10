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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/pkg/styles"
)

// importStep tracks the wizard progress.
type importStep int

const (
	importStepFile    importStep = iota // enter file path
	importStepTable                     // enter target table name
	importStepOptions                   // header/delimiter/create-table toggles
	importStepPreview                   // preview data
	importStepDone                      // import complete or error
)

// importResultMsg is sent when the background import completes.
type importResultMsg struct {
	result *database.ImportResult
	err    error
}

// importPreviewMsg is sent when the file preview is loaded.
type importPreviewMsg struct {
	headers []string
	rows    [][]string
	err     error
}

// ImportView provides an interactive import wizard.
type ImportView struct {
	parent *MainModel
	width  int
	height int

	step         importStep
	fileInput    textinput.Model
	tableInput   textinput.Model
	focusIndex   int // which field is focused in options step
	hasHeader    bool
	createTable  bool
	importing    bool
	importErr    error
	importResult *database.ImportResult

	// Preview data
	previewHeaders []string
	previewRows    [][]string
	previewLoading bool
}

// NewImportView creates a new import wizard.
func NewImportView(parent *MainModel) *ImportView {
	fi := textinput.New()
	fi.Placeholder = "path/to/data.csv"
	fi.CharLimit = 256
	fi.Width = 50
	fi.PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	fi.TextStyle = lipgloss.NewStyle().Foreground(styles.Foreground)
	fi.Cursor.Style = lipgloss.NewStyle().Foreground(styles.Primary)
	fi.Focus()

	ti := textinput.New()
	ti.Placeholder = "table_name"
	ti.CharLimit = 100
	ti.Width = 30
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.Foreground)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	return &ImportView{
		parent:      parent,
		hasHeader:   true,
		createTable: true,
		fileInput:   fi,
		tableInput:  ti,
	}
}

func (v *ImportView) Update(msg tea.Msg) (*ImportView, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case importPreviewMsg:
		v.previewLoading = false
		if msg.err != nil {
			v.importErr = msg.err
			return v, nil
		}
		v.previewHeaders = msg.headers
		v.previewRows = msg.rows
		v.importErr = nil
		v.step = importStepPreview
		return v, nil

	case importResultMsg:
		v.importing = false
		if msg.err != nil {
			v.importErr = msg.err
			v.step = importStepDone
			return v, nil
		}
		v.importResult = msg.result
		v.importErr = nil
		v.step = importStepDone
		return v, nil

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if v.step == importStepDone {
				// Reset and go back
				v.reset()
			}
			if !v.parent.PopView() {
				v.parent.mode = ViewBrowser
			}
			return v, nil

		case "ctrl+c":
			return v, tea.Quit
		}

		switch v.step {
		case importStepFile:
			return v.updateFileStep(msg)
		case importStepTable:
			return v.updateTableStep(msg)
		case importStepOptions:
			return v.updateOptionsStep(msg)
		case importStepPreview:
			return v.updatePreviewStep(msg)
		}
	}

	return v, cmd
}

func (v *ImportView) updateFileStep(msg tea.KeyMsg) (*ImportView, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if v.fileInput.Value() != "" {
			v.step = importStepTable
			v.fileInput.Blur()
			v.tableInput.Focus()
			// Auto-suggest table name from file name
			if v.tableInput.Value() == "" {
				base := filepath.Base(v.fileInput.Value())
				name := strings.TrimSuffix(base, filepath.Ext(base))
				name = strings.Map(func(r rune) rune {
					if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
						return r
					}
					return '_'
				}, name)
				v.tableInput.SetValue(name)
			}
		}
		return v, nil
	}
	v.fileInput, _ = v.fileInput.Update(msg)
	return v, nil
}

func (v *ImportView) updateTableStep(msg tea.KeyMsg) (*ImportView, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if v.tableInput.Value() != "" {
			v.step = importStepOptions
			v.tableInput.Blur()
			v.focusIndex = 0
		}
		return v, nil
	}
	v.tableInput, _ = v.tableInput.Update(msg)
	return v, nil
}

func (v *ImportView) updateOptionsStep(msg tea.KeyMsg) (*ImportView, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if v.focusIndex > 0 {
			v.focusIndex--
		}
	case "down", "j":
		if v.focusIndex < 2 { // 3 options: header, create-table, [Import]
			v.focusIndex++
		}
	case " ", "enter":
		switch v.focusIndex {
		case 0:
			v.hasHeader = !v.hasHeader
		case 1:
			v.createTable = !v.createTable
		case 2:
			// Start preview
			return v, v.loadPreview()
		}
	}
	return v, nil
}

func (v *ImportView) updatePreviewStep(msg tea.KeyMsg) (*ImportView, tea.Cmd) {
	switch msg.String() {
	case "enter", "i":
		// Start import
		v.importing = true
		return v, v.startImport()
	}
	return v, nil
}

func (v *ImportView) loadPreview() tea.Cmd {
	filePath := v.fileInput.Value()
	hasHeader := v.hasHeader
	v.previewLoading = true

	return func() tea.Msg {
		opts := database.ImportOptions{HasHeader: hasHeader}
		headers, rows, err := database.PreviewImport(filePath, opts, 5)
		return importPreviewMsg{headers: headers, rows: rows, err: err}
	}
}

func (v *ImportView) startImport() tea.Cmd {
	filePath := v.fileInput.Value()
	tableName := v.tableInput.Value()
	schema := v.parent.browserView.currentSchema
	hasHeader := v.hasHeader
	createTable := v.createTable
	mgr := v.parent.dbManager

	return func() tea.Msg {
		opts := database.ImportOptions{
			HasHeader:   hasHeader,
			CreateTable: createTable,
			BatchSize:   500,
		}

		headers, rows, err := database.ReadFileForImport(filePath, opts)
		if err != nil {
			return importResultMsg{err: err}
		}

		result, err := mgr.ImportData(schema, tableName, headers, rows, opts)
		return importResultMsg{result: result, err: err}
	}
}

func (v *ImportView) reset() {
	v.step = importStepFile
	v.fileInput.SetValue("")
	v.tableInput.SetValue("")
	v.fileInput.Focus()
	v.hasHeader = true
	v.createTable = true
	v.importing = false
	v.importErr = nil
	v.importResult = nil
	v.previewHeaders = nil
	v.previewRows = nil
	v.focusIndex = 0
}

func (v *ImportView) View() string {
	var b strings.Builder

	b.WriteString(styles.RenderTitle("Import Data"))
	b.WriteString("\n\n")

	switch v.step {
	case importStepFile:
		b.WriteString("  File path:\n")
		b.WriteString("  " + v.fileInput.View())
		b.WriteString("\n\n")
		b.WriteString(styles.RenderMuted("  Supports CSV (.csv) and Excel (.xlsx) files"))

	case importStepTable:
		b.WriteString(styles.RenderMuted("  File: " + v.fileInput.Value()))
		b.WriteString("\n\n")
		b.WriteString("  Table name:\n")
		b.WriteString("  " + v.tableInput.View())

	case importStepOptions:
		b.WriteString(styles.RenderMuted("  File: " + v.fileInput.Value()))
		b.WriteString("\n")
		b.WriteString(styles.RenderMuted("  Table: " + v.tableInput.Value()))
		b.WriteString("\n\n")

		options := []struct {
			label   string
			checked bool
		}{
			{"First row is header", v.hasHeader},
			{"Create table if not exists", v.createTable},
		}
		for i, opt := range options {
			check := "[ ]"
			if opt.checked {
				check = "[✓]"
			}
			line := fmt.Sprintf("  %s %s", check, opt.label)
			if i == v.focusIndex {
				b.WriteString(styles.ActiveListItemStyle.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}

		b.WriteString("\n")
		importBtn := "  [Import]"
		if v.focusIndex == 2 {
			b.WriteString(styles.ActiveListItemStyle.Render(importBtn))
		} else {
			b.WriteString(styles.RenderKey(importBtn))
		}

	case importStepPreview:
		if v.previewLoading {
			b.WriteString(v.parent.SpinnerView() + styles.RenderMuted(" Reading file..."))
		} else if v.importErr != nil {
			b.WriteString(styles.RenderError(v.importErr.Error()))
		} else {
			b.WriteString(styles.RenderMuted("  Preview (first 5 rows):"))
			b.WriteString("\n\n")
			b.WriteString(v.renderPreviewTable())
			b.WriteString("\n\n")
			if v.importing {
				b.WriteString(v.parent.SpinnerView() + styles.RenderMuted(" Importing..."))
			} else {
				b.WriteString(styles.RenderKey("  Press Enter to import"))
			}
		}

	case importStepDone:
		if v.importErr != nil {
			b.WriteString(styles.RenderError(v.importErr.Error()))
		} else if v.importResult != nil {
			b.WriteString(styles.RenderSuccess(fmt.Sprintf("Imported %d rows into %s",
				v.importResult.RowsImported, v.tableInput.Value())))
			if v.importResult.TableCreated {
				b.WriteString("\n")
				b.WriteString(styles.RenderSuccess("Table created"))
			}
		}
	}

	b.WriteString("\n\n")
	b.WriteString(RenderBindingHelpWidth(v.width,
		Keys.Global.Back,
		Keys.Global.Quit,
	))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (v *ImportView) renderPreviewTable() string {
	if len(v.previewHeaders) == 0 {
		return styles.RenderMuted("  No data")
	}

	var b strings.Builder

	// Header
	b.WriteString("  ")
	for i, h := range v.previewHeaders {
		if i > 0 {
			b.WriteString("  ")
		}
		w := 15
		if len(h) > w {
			h = h[:w-1] + "…"
		}
		b.WriteString(styles.RenderKey(fmt.Sprintf("%-*s", w, h)))
	}
	b.WriteString("\n")

	// Separator
	b.WriteString("  ")
	for i := range v.previewHeaders {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(styles.RenderMuted(strings.Repeat("─", 15)))
	}
	b.WriteString("\n")

	// Rows
	for _, row := range v.previewRows {
		b.WriteString("  ")
		for i := range v.previewHeaders {
			if i > 0 {
				b.WriteString("  ")
			}
			val := ""
			if i < len(row) {
				val = row[i]
			}
			if len(val) > 15 {
				val = val[:14] + "…"
			}
			b.WriteString(fmt.Sprintf("%-15s", val))
		}
		b.WriteString("\n")
	}

	return b.String()
}
