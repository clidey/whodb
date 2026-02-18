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
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/pkg/styles"
	"github.com/clidey/whodb/core/src/engine"
)

type ExportView struct {
	parent           *MainModel
	filenameInput    textinput.Model
	selectedFormat   int
	selectedDelim    int
	focusIndex       int
	schema           string
	tableName        string
	exporting        bool
	exportSuccess    bool
	exportError      error
	queryResults     *engine.GetRowsResult
	isQueryExport    bool
	savedFilePath    string // Store the full path of the saved file
	overwrite        bool   // Whether to overwrite existing file
	confirmOverwrite bool   // Whether we're confirming overwrite
	confirmIndex     int    // 0: Yes, 1: No
	pendingPath      string // Resolved path chosen before export
}

var exportFormats = []string{"CSV", "Excel"}
var exportDelimiters = []string{",", ";", "|"}
var exportDelimiterLabels = []string{"Comma (,)", "Semicolon (;)", "Pipe (|)"}

func NewExportView(parent *MainModel) *ExportView {
	ti := textinput.New()
	ti.Placeholder = "export"
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.Foreground)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	return &ExportView{
		parent:           parent,
		filenameInput:    ti,
		selectedFormat:   0,
		selectedDelim:    0,
		focusIndex:       0,
		overwrite:        false,
		confirmOverwrite: false,
		confirmIndex:     0,
		pendingPath:      "",
	}
}

func (v *ExportView) SetExportData(schema, tableName string) {
	v.schema = schema
	v.tableName = tableName
	v.filenameInput.SetValue(tableName)
	v.exporting = false
	v.exportSuccess = false
	v.exportError = nil
	v.queryResults = nil
	v.isQueryExport = false
	v.savedFilePath = ""
	v.overwrite = false
	v.confirmOverwrite = false
	v.confirmIndex = 0
	v.pendingPath = ""
}

func (v *ExportView) SetExportDataFromQuery(results *engine.GetRowsResult) {
	v.queryResults = results
	v.isQueryExport = true
	v.schema = ""
	v.tableName = ""
	v.filenameInput.SetValue("query_results")
	v.exporting = false
	v.exportSuccess = false
	v.exportError = nil
	v.savedFilePath = ""
	v.overwrite = false
	v.confirmOverwrite = false
	v.confirmIndex = 0
	v.pendingPath = ""
}

func (v *ExportView) Update(msg tea.Msg) (*ExportView, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.filenameInput.Width = clamp(msg.Width-12, 20, 60)
		return v, nil

	case exportResultMsg:
		v.exporting = false
		if msg.success {
			v.exportSuccess = true
			v.exportError = nil
		} else {
			v.exportSuccess = false
			v.exportError = msg.err
		}
		return v, nil

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			v.focusIndex--
			if v.focusIndex < 0 {
				v.focusIndex = v.maxIndex()
			}
			if v.focusIndex == 0 {
				v.filenameInput.Focus()
			} else {
				v.filenameInput.Blur()
			}
			return v, nil
		case tea.MouseButtonWheelDown:
			v.focusIndex++
			if v.focusIndex > v.maxIndex() {
				v.focusIndex = 0
			}
			if v.focusIndex == 0 {
				v.filenameInput.Focus()
			} else {
				v.filenameInput.Blur()
			}
			return v, nil
		}

	case tea.KeyMsg:
		// Handle overwrite confirmation overlay
		if v.confirmOverwrite {
			switch msg.String() {
			case "esc":
				v.confirmOverwrite = false
				v.confirmIndex = 0
				return v, nil
			case "left", "h", "right", "l", "tab", "down", "up", "j", "k":
				if v.confirmIndex == 0 {
					v.confirmIndex = 1
				} else {
					v.confirmIndex = 0
				}
				return v, nil
			case "enter":
				if v.confirmIndex == 0 {
					// Proceed
					return v, v.performExport()
				}
				v.confirmOverwrite = false
				v.confirmIndex = 0
				return v, nil
			}
			return v, nil
		}
		switch msg.String() {
		case "esc":
			if !v.exporting {
				v.parent.mode = ViewResults
				return v, nil
			}

		case "enter":
			expIdx := v.exportButtonIndex()
			cancelIdx := v.cancelButtonIndex()
			owIdx := v.overwriteIndex()
			if v.focusIndex == expIdx {
				// Resolve and check overwrite
				input := v.filenameInput.Value()
				format := exportFormats[v.selectedFormat]
				resolved, willOverwrite, err := resolveExportPath(input, format, v.overwrite)
				if err != nil {
					v.exportError = err
					return v, nil
				}
				v.pendingPath = resolved
				if willOverwrite {
					v.confirmOverwrite = true
					v.confirmIndex = 0
					return v, nil
				}
				return v, v.performExport()
			} else if v.focusIndex == cancelIdx {
				// Cancel button
				v.parent.mode = ViewResults
				return v, nil
			} else if v.focusIndex == owIdx {
				v.overwrite = !v.overwrite
				return v, nil
			}

		case "tab", "down", "j":
			v.focusIndex++
			if v.focusIndex > v.maxIndex() {
				v.focusIndex = 0
			}
			if v.focusIndex == 0 {
				v.filenameInput.Focus()
			} else {
				v.filenameInput.Blur()
			}
			return v, nil

		case "shift+tab", "up", "k":
			v.focusIndex--
			if v.focusIndex < 0 {
				v.focusIndex = v.maxIndex()
			}
			if v.focusIndex == 0 {
				v.filenameInput.Focus()
			} else {
				v.filenameInput.Blur()
			}
			return v, nil

		case "left":
			// Only handle left arrow for options, not filename input
			if v.focusIndex == 1 {
				v.selectedFormat--
				if v.selectedFormat < 0 {
					v.selectedFormat = len(exportFormats) - 1
				}
				return v, nil
			} else if v.hasDelimiter() && v.focusIndex == 2 {
				v.selectedDelim--
				if v.selectedDelim < 0 {
					v.selectedDelim = len(exportDelimiters) - 1
				}
				return v, nil
			} else if v.focusIndex == v.overwriteIndex() {
				v.overwrite = !v.overwrite
				return v, nil
			}
			// For filename input (focusIndex == 0), let it fall through to be handled by textinput

		case "right":
			// Only handle right arrow for options, not filename input
			if v.focusIndex == 1 {
				v.selectedFormat++
				if v.selectedFormat >= len(exportFormats) {
					v.selectedFormat = 0
				}
				return v, nil
			} else if v.hasDelimiter() && v.focusIndex == 2 {
				v.selectedDelim++
				if v.selectedDelim >= len(exportDelimiters) {
					v.selectedDelim = 0
				}
				return v, nil
			} else if v.focusIndex == v.overwriteIndex() {
				v.overwrite = !v.overwrite
				return v, nil
			}
			// For filename input (focusIndex == 0), let it fall through to be handled by textinput

		}
	}

	if v.focusIndex == 0 {
		v.filenameInput, cmd = v.filenameInput.Update(msg)
	}

	return v, cmd
}

func (v *ExportView) View() string {
	var b strings.Builder

	b.WriteString(styles.RenderTitle("Export Data"))
	b.WriteString("\n\n")

	if v.exporting {
		b.WriteString(styles.MutedStyle.Render("Exporting..."))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	if v.exportSuccess {
		b.WriteString(styles.SuccessStyle.Render("Export completed successfully!"))
		b.WriteString("\n\n")
		if v.savedFilePath != "" {
			b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("File saved to: %s", v.savedFilePath)))
			b.WriteString("\n\n")
		}
		b.WriteString(styles.MutedStyle.Render("Press ESC to go back"))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	if v.exportError != nil {
		b.WriteString(styles.RenderErrorBox(fmt.Sprintf("Export failed: %s", v.exportError.Error())))
		b.WriteString("\n\n")
		b.WriteString(styles.MutedStyle.Render("Press ESC to go back"))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	// Overwrite confirmation overlay
	if v.confirmOverwrite {
		b.WriteString(styles.RenderTitle("Confirm Overwrite"))
		b.WriteString("\n\n")
		path := v.pendingPath
		if path == "" {
			// Fallback compute for display
			input := v.filenameInput.Value()
			format := exportFormats[v.selectedFormat]
			if p, _, err := resolveExportPath(input, format, true); err == nil {
				path = p
			}
		}
		b.WriteString(styles.MutedStyle.Render("File exists:"))
		b.WriteString("\n  ")
		b.WriteString(styles.KeyStyle.Render(path))
		b.WriteString("\n\n")
		b.WriteString(styles.MutedStyle.Render("Overwrite this file?"))
		b.WriteString("\n\n  ")
		if v.confirmIndex == 0 {
			b.WriteString(styles.ActiveListItemStyle.Render(" Yes "))
			b.WriteString("  ")
			b.WriteString(styles.MutedStyle.Render("[No]"))
		} else {
			b.WriteString(styles.KeyStyle.Render("[Yes]"))
			b.WriteString("  ")
			b.WriteString(styles.ActiveListItemStyle.Render(" No "))
		}
		b.WriteString("\n\n")
		b.WriteString(styles.RenderHelp(
			"←/→", "select",
			"enter", "confirm",
			"esc", "cancel",
		))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	// Info text
	if v.isQueryExport {
		b.WriteString(styles.MutedStyle.Render("Exporting query results"))
	} else {
		b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("Exporting table: %s.%s", v.schema, v.tableName)))
	}
	b.WriteString("\n\n")

	// Filename input
	filenameLabel := "Filename:"
	if v.focusIndex == 0 {
		filenameLabel = styles.KeyStyle.Render("▶ " + filenameLabel)
	} else {
		filenameLabel = "  " + filenameLabel
	}
	b.WriteString(filenameLabel)
	b.WriteString("\n")
	b.WriteString("  " + v.filenameInput.View())
	b.WriteString("\n")

	// Show full path where file will be saved
	filename := v.filenameInput.Value()
	if filename == "" {
		filename = v.filenameInput.Placeholder
	}
	format := "CSV"
	if v.selectedFormat >= 0 && v.selectedFormat < len(exportFormats) {
		format = exportFormats[v.selectedFormat]
	}
	// Compute resolved candidate and detect if original target exists
	if candidate, willOverwrite, err := resolveExportPath(filename, format, v.overwrite); err == nil {
		// Also compute the original absolute path (no suffix), and whether it exists
		if _, origExists, _ := resolveExportPath(filename, format, true); true {
			b.WriteString("  ")
			if willOverwrite {
				b.WriteString(styles.KeyStyle.Render(fmt.Sprintf("Will overwrite: %s", candidate)))
			} else {
				b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("Will save to: %s", candidate)))
				// If overwrite disabled and original exists, hint about auto-suffix
				if !v.overwrite && origExists {
					b.WriteString("\n  ")
					b.WriteString(styles.MutedStyle.Render("Existing file detected; using next available name."))
				}
			}
		}
	}
	b.WriteString("\n\n")

	// Format selector
	formatLabel := "Format:"
	if v.focusIndex == 1 {
		formatLabel = styles.KeyStyle.Render("▶ " + formatLabel)
	} else {
		formatLabel = "  " + formatLabel
	}
	b.WriteString(formatLabel)
	b.WriteString("\n")
	b.WriteString("  ")
	for i, format := range exportFormats {
		if i == v.selectedFormat {
			if v.focusIndex == 1 {
				b.WriteString(styles.ActiveListItemStyle.Render(fmt.Sprintf(" %s ", format)))
			} else {
				b.WriteString(styles.KeyStyle.Render(fmt.Sprintf("[%s]", format)))
			}
		} else {
			b.WriteString(styles.MutedStyle.Render(fmt.Sprintf(" %s ", format)))
		}
		if i < len(exportFormats)-1 {
			b.WriteString("  ")
		}
	}
	b.WriteString("\n\n")

	// Delimiter selector (only for CSV)
	if v.hasDelimiter() {
		delimLabel := "Delimiter:"
		if v.focusIndex == 2 {
			delimLabel = styles.KeyStyle.Render("▶ " + delimLabel)
		} else {
			delimLabel = "  " + delimLabel
		}
		b.WriteString(delimLabel)
		b.WriteString("\n")
		b.WriteString("  ")
		for i, label := range exportDelimiterLabels {
			if i == v.selectedDelim {
				if v.focusIndex == 2 {
					b.WriteString(styles.ActiveListItemStyle.Render(fmt.Sprintf(" %s ", label)))
				} else {
					b.WriteString(styles.KeyStyle.Render(fmt.Sprintf("[%s]", label)))
				}
			} else {
				b.WriteString(styles.MutedStyle.Render(fmt.Sprintf(" %s ", label)))
			}
			if i < len(exportDelimiterLabels)-1 {
				b.WriteString("  ")
			}
		}
		b.WriteString("\n\n")
	} else {
		b.WriteString("\n")
	}

	// Overwrite toggle
	owLabel := "Overwrite existing:"
	if v.focusIndex == v.overwriteIndex() {
		owLabel = styles.KeyStyle.Render("▶ " + owLabel)
	} else {
		owLabel = "  " + owLabel
	}
	b.WriteString(owLabel)
	b.WriteString("\n  ")
	if v.overwrite {
		b.WriteString(styles.ActiveListItemStyle.Render(" On "))
		b.WriteString("  ")
		b.WriteString(styles.MutedStyle.Render("[Off]"))
	} else {
		b.WriteString(styles.KeyStyle.Render("[On]"))
		b.WriteString("  ")
		b.WriteString(styles.ActiveListItemStyle.Render(" Off "))
	}
	b.WriteString("\n\n")

	// Buttons
	b.WriteString("  ")
	if v.focusIndex == v.exportButtonIndex() {
		b.WriteString(styles.ActiveListItemStyle.Render(" Export "))
	} else {
		b.WriteString(styles.KeyStyle.Render("[Export]"))
	}
	b.WriteString("  ")
	if v.focusIndex == v.cancelButtonIndex() {
		b.WriteString(styles.ActiveListItemStyle.Render(" Cancel "))
	} else {
		b.WriteString(styles.MutedStyle.Render("[Cancel]"))
	}
	b.WriteString("\n\n")

	// Help text
	b.WriteString(styles.RenderHelp(
		"↑/k", "prev",
		"↓/j", "next",
		"←", "prev option",
		"→", "next option",
		"enter", "select",
		"esc", "cancel",
	))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

type exportResultMsg struct {
	success bool
	err     error
}

func (v *ExportView) performExport() tea.Cmd {
	return func() tea.Msg {
		v.exporting = true

		filename := v.filenameInput.Value()
		format := exportFormats[v.selectedFormat]
		delimiter := exportDelimiters[v.selectedDelim]

		// Resolve to an absolute path with appropriate extension
		var fullFilename string
		var err error
		if v.pendingPath != "" {
			fullFilename = v.pendingPath
		} else {
			fullFilename, _, err = resolveExportPath(filename, format, v.overwrite)
			if err != nil {
				v.exporting = false
				return exportResultMsg{success: false, err: err}
			}
		}

		// Create directory if it doesn't exist
		dir := filepath.Dir(fullFilename)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0700); err != nil {
				v.exporting = false
				return exportResultMsg{success: false, err: fmt.Errorf("failed to create directory: %w", err)}
			}
		}

		// Record absolute path for display
		v.savedFilePath = fullFilename

		// Perform the export
		if v.isQueryExport {
			// Export query results from memory
			if format == "CSV" {
				err = v.parent.dbManager.ExportResultsToCSV(v.queryResults, fullFilename, delimiter)
			} else {
				err = v.parent.dbManager.ExportResultsToExcel(v.queryResults, fullFilename)
			}
		} else {
			// Export table data (fetch from database)
			if format == "CSV" {
				err = v.parent.dbManager.ExportToCSV(v.schema, v.tableName, fullFilename, delimiter)
			} else {
				err = v.parent.dbManager.ExportToExcel(v.schema, v.tableName, fullFilename)
			}
		}

		v.exporting = false

		if err != nil {
			v.exportError = err
			return exportResultMsg{success: false, err: err}
		}

		v.exportSuccess = true
		v.confirmOverwrite = false
		v.confirmIndex = 0
		v.pendingPath = ""
		return exportResultMsg{success: true, err: nil}
	}
}

// resolveExportPath expands ~, resolves relative paths against CWD,
// cleans the path, and appends the proper extension. Returns an absolute path
// and whether this would overwrite an existing file (when overwrite is true).
func resolveExportPath(input, format string, overwrite bool) (string, bool, error) {
	// Reject empty after trimming spaces and remove null bytes
	s := strings.ReplaceAll(strings.TrimSpace(input), "\x00", "")
	if s == "" {
		return "", false, fmt.Errorf("invalid path or filename")
	}

	// Expand leading ~ to the user's home directory
	if strings.HasPrefix(s, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			if s == "~" {
				s = home
			} else if strings.HasPrefix(s, "~/") || strings.HasPrefix(s, "~\\") {
				s = filepath.Join(home, s[2:])
			}
		}
	}

	// Clean the path
	p := filepath.Clean(s)

	// Resolve relative paths against the current working directory
	if !filepath.IsAbs(p) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", false, fmt.Errorf("failed to get working directory: %w", err)
		}
		p = filepath.Join(cwd, p)
	}

	// Ensure correct extension: if user provided an extension, respect it.
	// Otherwise, append the selected format's default extension.
	wantExt := ".csv"
	if strings.EqualFold(format, "Excel") {
		wantExt = ".xlsx"
	}
	curExt := strings.ToLower(filepath.Ext(p))
	if curExt == "" {
		p = p + wantExt
	}

	// Return absolute path and ensure we don't overwrite existing files
	abs, err := filepath.Abs(p)
	if err != nil {
		// Fallback to given path if Abs fails
		abs = p
	}

	// If the file already exists, decide based on overwrite flag
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		if overwrite {
			return abs, true, nil
		}
		// Find next free suffix
		dir := filepath.Dir(abs)
		base := filepath.Base(abs)
		ext := filepath.Ext(base)
		name := strings.TrimSuffix(base, ext)
		for i := 1; ; i++ {
			candidate := filepath.Join(dir, fmt.Sprintf("%s_%d%s", name, i, ext))
			if _, statErr := os.Stat(candidate); os.IsNotExist(statErr) {
				return candidate, false, nil
			}
		}
	}

	return abs, false, nil
}

// Helpers for focus indices and options
func (v *ExportView) hasDelimiter() bool {
	return v.selectedFormat >= 0 && v.selectedFormat < len(exportFormats) && exportFormats[v.selectedFormat] == "CSV"
}

func (v *ExportView) overwriteIndex() int {
	if v.hasDelimiter() {
		return 3
	}
	return 2
}

func (v *ExportView) exportButtonIndex() int {
	return v.overwriteIndex() + 1
}

func (v *ExportView) cancelButtonIndex() int {
	return v.overwriteIndex() + 2
}

func (v *ExportView) maxIndex() int {
	return v.cancelButtonIndex()
}
