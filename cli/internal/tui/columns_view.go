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
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/pkg/styles"
	"github.com/clidey/whodb/core/src/engine"
)

type ColumnsView struct {
	parent        *MainModel
	columns       []engine.Column
	selected      map[string]bool
	selectedIndex int
	schema        string
	tableName     string
	scrollOffset  int
	height        int
}

func NewColumnsView(parent *MainModel) *ColumnsView {
	return &ColumnsView{
		parent:        parent,
		columns:       []engine.Column{},
		selected:      make(map[string]bool),
		selectedIndex: 0,
	}
}

func (v *ColumnsView) SetTableContext(schema, tableName string, columns []engine.Column) {
	// If switching to a different table, reset selections to all columns
	if v.schema != schema || v.tableName != tableName {
		v.selected = make(map[string]bool)
		for _, col := range columns {
			v.selected[col.Name] = true
		}
		v.selectedIndex = 0
	}

	v.schema = schema
	v.tableName = tableName
	v.columns = columns
}

func (v *ColumnsView) Update(msg tea.Msg) (*ColumnsView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.height = msg.Height
		return v, nil

	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseWheelUp:
			if v.scrollOffset > 0 {
				v.scrollOffset--
				// If selected item is now below viewport, move selection up
				maxVisible := v.height - 12
				if maxVisible < 1 {
					maxVisible = 10
				}
				if v.selectedIndex >= v.scrollOffset+maxVisible {
					v.selectedIndex = v.scrollOffset + maxVisible - 1
					if v.selectedIndex >= len(v.columns) {
						v.selectedIndex = len(v.columns) - 1
					}
				}
			}
			return v, nil
		case tea.MouseWheelDown:
			maxVisible := v.height - 12
			if maxVisible < 1 {
				maxVisible = 10
			}
			maxScroll := len(v.columns) - maxVisible
			if maxScroll < 0 {
				maxScroll = 0
			}
			if v.scrollOffset < maxScroll {
				v.scrollOffset++
				// If selected item is now above viewport, move selection down
				if v.selectedIndex < v.scrollOffset {
					v.selectedIndex = v.scrollOffset
				}
			}
			return v, nil
		}
		return v, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, Keys.Global.Back):
			if !v.parent.PopView() {
				v.parent.mode = ViewResults
			}
			return v, nil

		case key.Matches(msg, Keys.Columns.Apply):
			// Apply column selection and return to results
			v.parent.resultsView.visibleColumns = v.getSelectedColumns()
			cmd := v.parent.resultsView.loadWithWhere()
			if !v.parent.PopView() {
				v.parent.mode = ViewResults
			}
			return v, cmd

		case key.Matches(msg, Keys.Columns.SelectAll):
			// Select all
			for i := range v.columns {
				v.selected[v.columns[i].Name] = true
			}
			return v, nil

		case key.Matches(msg, Keys.Columns.SelectNone):
			// Select none
			for i := range v.columns {
				v.selected[v.columns[i].Name] = false
			}
			return v, nil

		case key.Matches(msg, Keys.Columns.Toggle):
			// Toggle current selection
			if v.selectedIndex >= 0 && v.selectedIndex < len(v.columns) {
				col := v.columns[v.selectedIndex].Name
				v.selected[col] = !v.selected[col]
			}
			return v, nil

		case key.Matches(msg, Keys.Columns.Up):
			if v.selectedIndex > 0 {
				v.selectedIndex--
				// Only scroll if selection goes above visible area
				if v.selectedIndex < v.scrollOffset {
					v.scrollOffset = v.selectedIndex
				}
			}
			return v, nil

		case key.Matches(msg, Keys.Columns.Down):
			if v.selectedIndex < len(v.columns)-1 {
				v.selectedIndex++
				// Only scroll if selection goes below visible area
				maxVisible := v.height - 12
				if maxVisible < 1 {
					maxVisible = 10
				}
				if v.selectedIndex >= v.scrollOffset+maxVisible {
					v.scrollOffset = v.selectedIndex - maxVisible + 1
				}
			}
			return v, nil
		}
	}

	return v, nil
}

func (v *ColumnsView) View() string {
	var b strings.Builder

	// Fixed header
	b.WriteString(styles.RenderTitle("Select Columns"))
	b.WriteString("\n")
	b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("Table: %s.%s", v.schema, v.tableName)))
	b.WriteString("\n\n")

	selectedCount := 0
	for _, isSelected := range v.selected {
		if isSelected {
			selectedCount++
		}
	}
	b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("%d of %d columns selected", selectedCount, len(v.columns))))
	b.WriteString("\n\n")

	// Scrollable content area
	maxVisible := v.height - 12
	if maxVisible < 1 {
		maxVisible = 10
	}

	startIdx := v.scrollOffset
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx >= len(v.columns) {
		startIdx = len(v.columns) - 1
	}

	endIdx := startIdx + maxVisible
	if endIdx > len(v.columns) {
		endIdx = len(v.columns)
	}

	for i := startIdx; i < endIdx; i++ {
		col := v.columns[i]
		prefix := "  "
		checkbox := "[ ]"
		if v.selected[col.Name] {
			checkbox = "[✓]"
		}

		colDisplay := fmt.Sprintf("%s %s (%s)", checkbox, col.Name, col.Type)

		if i == v.selectedIndex {
			prefix = styles.KeyStyle.Render("▶ ")
			b.WriteString(prefix + styles.ActiveListItemStyle.Render(colDisplay))
		} else {
			b.WriteString(prefix + styles.ListItemStyle.Render(colDisplay))
		}
		b.WriteString("\n")
	}

	// Scroll indicators
	if v.scrollOffset > 0 || endIdx < len(v.columns) {
		scrollInfo := fmt.Sprintf("Columns %d-%d of %d", startIdx+1, endIdx, len(v.columns))
		if v.scrollOffset > 0 {
			scrollInfo += " • ↑ scroll up"
		}
		if endIdx < len(v.columns) {
			scrollInfo += " • ↓ scroll down"
		}
		b.WriteString("\n")
		b.WriteString(styles.MutedStyle.Render(scrollInfo))
	}

	// Fixed footer
	b.WriteString("\n\n")
	b.WriteString(RenderBindingHelp(
		Keys.Columns.Up,
		Keys.Columns.Down,
		Keys.Columns.Toggle,
		Keys.Columns.SelectAll,
		Keys.Columns.SelectNone,
		Keys.Columns.Apply,
		Keys.Global.Back,
	))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (v *ColumnsView) getSelectedColumns() []string {
	var result []string
	for _, col := range v.columns {
		if v.selected[col.Name] {
			result = append(result, col.Name)
		}
	}
	return result
}
