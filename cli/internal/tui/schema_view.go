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
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/pkg/styles"
	"github.com/sahilm/fuzzy"
)

type SchemaView struct {
	parent         *MainModel
	tables         []tableWithColumns
	selectedIndex  int
	expandedTables map[string]bool
	loading        bool
	err            error
	width          int
	height         int
	scrollOffset   int
	filterInput    textinput.Model
	filtering      bool
	filteredTables []tableWithColumns
	currentSchema  string
}

func NewSchemaView(parent *MainModel) *SchemaView {
	ti := textinput.New()
	ti.Placeholder = "Search tables..."
	ti.CharLimit = 50
	ti.Width = 30
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.Foreground)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	return &SchemaView{
		parent:         parent,
		loading:        true,
		selectedIndex:  0,
		expandedTables: make(map[string]bool),
		width:          80,
		height:         20,
		filterInput:    ti,
		filtering:      false,
		scrollOffset:   0,
	}
}

func (v *SchemaView) Update(msg tea.Msg) (*SchemaView, tea.Cmd) {
	switch msg := msg.(type) {
	case schemaLoadedMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.tables = msg.tables
		v.currentSchema = msg.schema
		v.applyFilter()
		v.selectedIndex = 0
		return v, nil

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.filterInput.Width = clamp(msg.Width-20, 15, 50)
		return v, nil

	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseWheelUp:
			if v.scrollOffset > 0 {
				v.scrollOffset--
			}
			return v, nil
		case tea.MouseWheelDown:
			// Calculate total items (tables + expanded columns)
			totalItems := 0
			for _, table := range v.filteredTables {
				totalItems++
				if v.expandedTables[table.StorageUnit.Name] {
					totalItems += len(table.Columns)
				}
			}

			headerLines := 6
			if v.filtering || v.filterInput.Value() != "" {
				headerLines = 8
			}
			footerLines := 4
			maxVisibleLines := v.height - headerLines - footerLines
			if maxVisibleLines < 5 {
				maxVisibleLines = 5
			}

			maxScroll := totalItems - maxVisibleLines
			if maxScroll < 0 {
				maxScroll = 0
			}
			if v.scrollOffset < maxScroll {
				v.scrollOffset++
			}
			return v, nil
		}
		return v, nil

	case tea.KeyMsg:
		if v.filtering {
			switch msg.String() {
			case "esc":
				v.filtering = false
				v.filterInput.Blur()
				v.filterInput.SetValue("")
				v.applyFilter()
				return v, nil
			case "enter":
				v.filtering = false
				v.filterInput.Blur()
				v.applyFilter()
				return v, nil
			default:
				var cmd tea.Cmd
				v.filterInput, cmd = v.filterInput.Update(msg)
				v.applyFilter()
				return v, cmd
			}
		}

		switch {
		case key.Matches(msg, Keys.Global.Back):
			if !v.parent.PopView() {
				v.parent.mode = ViewBrowser
			}
			return v, nil

		case key.Matches(msg, Keys.Schema.Up):
			if v.selectedIndex > 0 {
				v.selectedIndex--
				v.ensureSelectedVisible()
			}
			return v, nil

		case key.Matches(msg, Keys.Schema.Down):
			if v.selectedIndex < len(v.filteredTables)-1 {
				v.selectedIndex++
				v.ensureSelectedVisible()
			}
			return v, nil

		case key.Matches(msg, Keys.Schema.Toggle):
			if v.selectedIndex >= 0 && v.selectedIndex < len(v.filteredTables) {
				table := v.filteredTables[v.selectedIndex]
				if v.expandedTables[table.StorageUnit.Name] {
					delete(v.expandedTables, table.StorageUnit.Name)
				} else {
					v.expandedTables[table.StorageUnit.Name] = true
				}
			}
			return v, nil

		case key.Matches(msg, Keys.Schema.ViewData):
			if v.selectedIndex >= 0 && v.selectedIndex < len(v.filteredTables) {
				table := v.filteredTables[v.selectedIndex]
				cmd := v.parent.resultsView.LoadTable(v.currentSchema, table.StorageUnit.Name)
				v.parent.PushView(ViewResults)
				return v, cmd
			}
			return v, nil

		case key.Matches(msg, Keys.Schema.Filter):
			v.filtering = true
			v.filterInput.Focus()
			return v, nil

		case key.Matches(msg, Keys.Schema.Refresh):
			v.loading = true
			v.err = nil
			v.filterInput.SetValue("")
			v.filtering = false
			v.expandedTables = make(map[string]bool)
			return v, v.loadSchema()
		}
	}

	return v, nil
}

func (v *SchemaView) View() string {
	var b strings.Builder

	b.WriteString(styles.RenderTitle("Database Schema"))
	b.WriteString("\n\n")

	if v.filtering || v.filterInput.Value() != "" {
		filterLabel := "Filter: "
		if v.filtering {
			filterLabel = styles.RenderKey("Filter: ")
		} else {
			filterLabel = styles.RenderMuted("Filter: ")
		}
		b.WriteString(filterLabel)
		b.WriteString(v.filterInput.View())
		if !v.filtering && v.filterInput.Value() != "" {
			b.WriteString(" ")
			b.WriteString(styles.RenderMuted(fmt.Sprintf("(%d/%d)", len(v.filteredTables), len(v.tables))))
		}
		b.WriteString("\n\n")
	}

	if v.err != nil {
		b.WriteString(styles.RenderErrorBox(v.err.Error()))
		b.WriteString("\n\n")
		b.WriteString(styles.RenderMuted("Press 'r' to retry"))
	} else if v.loading {
		b.WriteString(v.parent.SpinnerView() + styles.RenderMuted(" Loading schema..."))
	} else if len(v.filteredTables) == 0 {
		b.WriteString(styles.RenderMuted("No tables found."))
	} else {
		b.WriteString(v.renderTables())
	}

	b.WriteString("\n\n")

	if v.filtering {
		b.WriteString(RenderBindingHelp(
			Keys.Filter.CancelFilter,
			Keys.Filter.ApplyFilter,
		))
	} else {
		b.WriteString(RenderBindingHelp(
			Keys.Schema.Up,
			Keys.Schema.Down,
			Keys.Schema.Toggle,
			Keys.Schema.ViewData,
			Keys.Schema.Filter,
			Keys.Schema.Refresh,
			Keys.Global.Back,
		))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (v *SchemaView) renderTables() string {
	var b strings.Builder

	// Calculate available lines for content
	// Header takes: title(2) + filter(2 if shown) + blank line = ~5 lines
	// Footer takes: blank line + help(2 lines) = ~3 lines
	// Total overhead: ~8-10 lines
	headerLines := 6
	if v.filtering || v.filterInput.Value() != "" {
		headerLines = 8
	}
	footerLines := 4
	maxVisibleLines := v.height - headerLines - footerLines
	if maxVisibleLines < 5 {
		maxVisibleLines = 5
	}

	// Build a list of all visible items (tables + their columns if expanded)
	type viewItem struct {
		isTable    bool
		tableIndex int
		columnText string
		isSelected bool
	}

	var allItems []viewItem
	for i, table := range v.filteredTables {
		isSelected := i == v.selectedIndex
		isExpanded := v.expandedTables[table.StorageUnit.Name]

		// Add table item
		allItems = append(allItems, viewItem{
			isTable:    true,
			tableIndex: i,
			isSelected: isSelected,
		})

		// Add column items if expanded
		if isExpanded {
			for _, col := range table.Columns {
				allItems = append(allItems, viewItem{
					isTable:    false,
					columnText: fmt.Sprintf("    %s: %s", col.Name, col.Type),
				})
			}
		}
	}

	// Calculate viewport
	startIdx := v.scrollOffset
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx >= len(allItems) {
		startIdx = len(allItems) - 1
		if startIdx < 0 {
			startIdx = 0
		}
	}

	endIdx := startIdx + maxVisibleLines
	if endIdx > len(allItems) {
		endIdx = len(allItems)
	}

	// Render visible items
	for i := startIdx; i < endIdx; i++ {
		item := allItems[i]

		if item.isTable {
			table := v.filteredTables[item.tableIndex]
			isExpanded := v.expandedTables[table.StorageUnit.Name]

			icon := "▶"
			if isExpanded {
				icon = "▼"
			}

			tableType := ""
			for _, attr := range table.StorageUnit.Attributes {
				if attr.Key == "Type" {
					tableType = attr.Value
					break
				}
			}

			prefix := "  "
			if item.isSelected {
				prefix = styles.RenderKey("▶ ")
			}

			tableLine := fmt.Sprintf("%s %s %s", icon, table.StorageUnit.Name, styles.RenderMuted(fmt.Sprintf("(%s)", tableType)))
			if item.isSelected {
				tableLine = styles.ActiveListItemStyle.Render(tableLine)
			} else {
				tableLine = styles.ListItemStyle.Render(tableLine)
			}

			b.WriteString(prefix)
			b.WriteString(tableLine)
			b.WriteString("\n")
		} else {
			// Column line
			b.WriteString(styles.RenderMuted(item.columnText))
			b.WriteString("\n")
		}
	}

	// Scroll indicators
	if startIdx > 0 || endIdx < len(allItems) {
		totalTables := len(v.filteredTables)
		scrollInfo := fmt.Sprintf("Showing %d-%d of %d items (%d tables)", startIdx+1, endIdx, len(allItems), totalTables)
		if startIdx > 0 {
			scrollInfo += " • ↑ scroll up"
		}
		if endIdx < len(allItems) {
			scrollInfo += " • ↓ scroll down"
		}
		b.WriteString("\n")
		b.WriteString(styles.RenderMuted(scrollInfo))
	}

	return b.String()
}

func (v *SchemaView) loadSchema() tea.Cmd {
	// Capture values before closure to avoid data races
	browserSchema := v.parent.browserView.currentSchema

	return func() tea.Msg {
		conn := v.parent.dbManager.GetCurrentConnection()
		if conn == nil {
			return schemaLoadedMsg{
				tables: []tableWithColumns{},
				err:    fmt.Errorf("no connection"),
			}
		}

		// Use the schema selected in browser view if available
		schema := browserSchema
		if schema == "" {
			schemas, err := v.parent.dbManager.GetSchemas()
			if err != nil {
				return schemaLoadedMsg{
					tables: []tableWithColumns{},
					err:    fmt.Errorf("failed to get schemas: %w", err),
				}
			}

			if len(schemas) == 0 {
				return schemaLoadedMsg{
					tables: []tableWithColumns{},
					err:    nil,
				}
			}

			schema = selectBestSchema(schemas)
		}
		units, err := v.parent.dbManager.GetStorageUnits(schema)
		if err != nil {
			return schemaLoadedMsg{
				tables: []tableWithColumns{},
				err:    fmt.Errorf("failed to get tables: %w", err),
			}
		}

		tables := []tableWithColumns{}
		for _, unit := range units {
			columns, err := v.parent.dbManager.GetColumns(schema, unit.Name)
			if err != nil {
				continue
			}
			tables = append(tables, tableWithColumns{
				StorageUnit: unit,
				Columns:     columns,
			})
		}

		return schemaLoadedMsg{
			tables: tables,
			err:    nil,
			schema: schema,
		}
	}
}

func (v *SchemaView) Init() tea.Cmd {
	return v.loadSchema()
}

func (v *SchemaView) applyFilter() {
	filterText := v.filterInput.Value()

	if filterText == "" {
		v.filteredTables = v.tables
		return
	}

	names := make([]string, len(v.tables))
	for i, t := range v.tables {
		names[i] = t.StorageUnit.Name
	}

	matches := fuzzy.Find(filterText, names)
	v.filteredTables = make([]tableWithColumns, len(matches))
	for i, m := range matches {
		v.filteredTables[i] = v.tables[m.Index]
	}

	if v.selectedIndex >= len(v.filteredTables) {
		v.selectedIndex = 0
	}
}

func (v *SchemaView) ensureSelectedVisible() {
	// Calculate the item index of the selected table
	var selectedItemIndex int
	for i, table := range v.filteredTables {
		if i == v.selectedIndex {
			break
		}
		selectedItemIndex++
		if v.expandedTables[table.StorageUnit.Name] {
			selectedItemIndex += len(table.Columns)
		}
	}

	// Calculate viewport
	headerLines := 6
	if v.filtering || v.filterInput.Value() != "" {
		headerLines = 8
	}
	footerLines := 4
	maxVisibleLines := v.height - headerLines - footerLines
	if maxVisibleLines < 5 {
		maxVisibleLines = 5
	}

	// Adjust scroll offset to keep selected item visible
	if selectedItemIndex < v.scrollOffset {
		v.scrollOffset = selectedItemIndex
	} else if selectedItemIndex >= v.scrollOffset+maxVisibleLines {
		v.scrollOffset = selectedItemIndex - maxVisibleLines + 1
	}
}
