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
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/pkg/styles"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
)

type ResultsView struct {
	parent          *MainModel
	table           table.Model
	results         *engine.GetRowsResult
	query           string
	currentPage     int
	pageSize        int
	totalRows       int
	schema          string
	tableName       string
	columnOffset    int
	maxColumns      int
	whereCondition  *model.WhereCondition
	visibleColumns  []string
	editingPageSize bool
	pageSizeInput   textinput.Model
}

// Available page sizes for cycling
var pageSizes = []int{10, 25, 50, 100}

// Message to trigger re-render after page load
type pageLoadedMsg struct{}

func NewResultsView(parent *MainModel) *ResultsView {
	columns := []table.Column{}
	rows := []table.Row{}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Border).
		BorderBottom(true).
		Bold(true).
		Foreground(styles.Primary)
	s.Selected = s.Selected.
		Foreground(styles.Background).
		Background(styles.Primary).
		Bold(false)
	t.SetStyles(s)

	// Page size input
	ti := textinput.New()
	ti.Placeholder = "e.g. 25"
	ti.CharLimit = 5
	ti.Width = 10

	return &ResultsView{
		parent:        parent,
		table:         t,
		currentPage:   0,
		pageSize:      50,
		columnOffset:  0,
		maxColumns:    10,
		pageSizeInput: ti,
	}
}

func (v *ResultsView) Update(msg tea.Msg) (*ResultsView, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.table.SetHeight(msg.Height - 20)
		v.table.SetWidth(msg.Width - 8)
		return v, nil

	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseWheelUp:
			// Move selection up (bubbles table doesn't handle mouse wheel directly)
			v.table.MoveUp(1)
			return v, nil
		case tea.MouseWheelDown:
			// Move selection down (bubbles table doesn't handle mouse wheel directly)
			v.table.MoveDown(1)
			return v, nil
		}
		return v, nil

	case pageLoadedMsg:
		// Page loaded, trigger re-render
		return v, nil

	case tea.KeyMsg:
		// Handle page size editing mode
		if v.editingPageSize {
			switch msg.String() {
			case "enter":
				if size, err := strconv.Atoi(v.pageSizeInput.Value()); err == nil && size > 0 {
					v.pageSize = size
					v.currentPage = 0
					v.editingPageSize = false
					v.pageSizeInput.Blur()
					return v, v.loadPage()
				}
				// Invalid input, just exit edit mode
				v.editingPageSize = false
				v.pageSizeInput.Blur()
				return v, nil
			case "esc":
				v.editingPageSize = false
				v.pageSizeInput.Blur()
				return v, nil
			default:
				v.pageSizeInput, cmd = v.pageSizeInput.Update(msg)
				return v, cmd
			}
		}

		switch msg.String() {
		case "esc":
			v.parent.mode = ViewBrowser
			return v, nil

		case "n":
			// Check if we can go to next page
			if v.totalRows > 0 {
				// For tables with known count, check if we're at the last page
				totalPages := (v.totalRows + v.pageSize - 1) / v.pageSize
				if v.currentPage+1 < totalPages {
					v.currentPage++
					return v, v.loadPage()
				}
			} else if v.results != nil && len(v.results.Rows) == v.pageSize {
				// For query results without total count, allow next page if current page is full
				v.currentPage++
				return v, v.loadPage()
			}
			return v, nil

		case "p":
			if v.currentPage > 0 {
				v.currentPage--
				return v, v.loadPage()
			}
			return v, nil

		case "left", "h":
			if v.columnOffset > 0 {
				v.columnOffset--
				v.updateTable()
			}
			return v, nil

		case "right", "l":
			if v.results != nil && v.columnOffset+v.maxColumns < len(v.results.Columns) {
				v.columnOffset++
				v.updateTable()
			}
			return v, nil

		case "w":
			// WHERE conditions are only available when viewing table data
			if v.schema != "" && v.tableName != "" {
				columns, err := v.parent.dbManager.GetColumns(v.schema, v.tableName)
				if err != nil {
					v.parent.err = err
					return v, nil
				}
				v.parent.whereView.SetTableContext(v.schema, v.tableName, columns, v.whereCondition)
				v.parent.mode = ViewWhere
				return v, nil
			}

		case "c":
			// Column selection is only available when viewing table data
			if v.schema != "" && v.tableName != "" {
				columns, err := v.parent.dbManager.GetColumns(v.schema, v.tableName)
				if err != nil {
					v.parent.err = err
					return v, nil
				}
				v.parent.columnsView.SetTableContext(v.schema, v.tableName, columns)
				v.parent.mode = ViewColumns
				return v, nil
			}

		case "e":
			if v.schema != "" && v.tableName != "" {
				// Export table data
				v.parent.exportView.SetExportData(v.schema, v.tableName)
				v.parent.mode = ViewExport
				return v, nil
			} else if v.results != nil && v.query != "" {
				// Export query results
				v.parent.exportView.SetExportDataFromQuery(v.results)
				v.parent.mode = ViewExport
				return v, nil
			}

		case "s":
			// Cycle through page sizes
			currentIndex := 0
			for i, size := range pageSizes {
				if size == v.pageSize {
					currentIndex = i
					break
				}
			}
			v.pageSize = pageSizes[(currentIndex+1)%len(pageSizes)]
			v.currentPage = 0
			return v, v.loadPage()

		case "S":
			// Enter custom page size mode
			v.editingPageSize = true
			v.pageSizeInput.SetValue("")
			v.pageSizeInput.Focus()
			return v, nil

		case "down", "j":
			// Check if at bottom of current page - auto-paginate to next
			if v.results != nil && len(v.results.Rows) > 0 {
				cursor := v.table.Cursor()
				if cursor >= len(v.results.Rows)-1 {
					// At bottom, try to go to next page
					canGoNext := false
					if v.totalRows > 0 {
						totalPages := (v.totalRows + v.pageSize - 1) / v.pageSize
						canGoNext = v.currentPage+1 < totalPages
					} else {
						canGoNext = len(v.results.Rows) == v.pageSize
					}
					if canGoNext {
						v.currentPage++
						v.table.SetCursor(0)
						return v, v.loadPage()
					}
				}
			}

		case "up", "k":
			// Check if at top of current page - auto-paginate to previous
			if v.results != nil && len(v.results.Rows) > 0 {
				cursor := v.table.Cursor()
				if cursor <= 0 && v.currentPage > 0 {
					v.currentPage--
					return v, v.loadPageAndGoToBottom()
				}
			}
		}
	}

	// Pass to table for navigation (arrows, page up/down, etc.)
	v.table, cmd = v.table.Update(msg)
	return v, cmd
}

func (v *ResultsView) View() string {
	var b strings.Builder

	if v.query != "" {
		b.WriteString(styles.RenderTitle("Query Results"))
		b.WriteString("\n")
		b.WriteString(styles.MutedStyle.Render(v.query))
		b.WriteString("\n\n")
	} else {
		b.WriteString(styles.RenderTitle("Table Data"))
		b.WriteString("\n\n")
	}

	if v.results == nil {
		b.WriteString(styles.MutedStyle.Render("No results"))
	} else {
		b.WriteString(v.table.View())
		b.WriteString("\n\n")

		// Show column and row information
		totalCols := len(v.results.Columns)
		visibleCols := v.maxColumns
		if v.columnOffset+v.maxColumns > totalCols {
			visibleCols = totalCols - v.columnOffset
		}

		columnInfo := fmt.Sprintf("Columns %d-%d of %d", v.columnOffset+1, v.columnOffset+visibleCols, totalCols)

		var rowInfo string
		if v.totalRows > 0 {
			totalPages := (v.totalRows + v.pageSize - 1) / v.pageSize
			rowInfo = fmt.Sprintf("Showing %d rows (Page %d of %d, size: %d)", len(v.results.Rows), v.currentPage+1, totalPages, v.pageSize)
		} else {
			rowInfo = fmt.Sprintf("Showing %d rows (Page %d, size: %d)", len(v.results.Rows), v.currentPage+1, v.pageSize)
		}

		b.WriteString(styles.MutedStyle.Render(columnInfo + " • " + rowInfo))

		// Show page size input if editing
		if v.editingPageSize {
			b.WriteString("\n\n")
			b.WriteString(styles.KeyStyle.Render("Page size: "))
			b.WriteString(v.pageSizeInput.View())
			b.WriteString(styles.MutedStyle.Render(" (enter to confirm, esc to cancel)"))
		}
	}

	b.WriteString("\n\n")

	// Show different help based on whether export/where/columns is available
	if v.schema != "" && v.tableName != "" {
		whereLabel := "where"
		conditionCount := v.countWhereConditions()
		if conditionCount > 0 {
			whereLabel = fmt.Sprintf("where (%d)", conditionCount)
		}

		columnsLabel := "columns"
		if v.results != nil && len(v.results.Columns) > 0 {
			selectedCount := len(v.visibleColumns)
			if selectedCount == 0 {
				selectedCount = len(v.results.Columns)
			}
			columnsLabel = fmt.Sprintf("columns (%d/%d)", selectedCount, len(v.results.Columns))
		}

		b.WriteString(styles.RenderHelp(
			"↑/k", "up",
			"↓/j", "down",
			"←/h", "col left",
			"→/l", "col right",
			"scroll", "trackpad/mouse",
			"[w]", whereLabel,
			"[c]", columnsLabel,
			"[e]", "export",
			"[n/p]", "page",
			"[s/S]", "page size",
			"esc", "back",
		))
	} else {
		b.WriteString(styles.RenderHelp(
			"↑/k", "up",
			"↓/j", "down",
			"←/h", "col left",
			"→/l", "col right",
			"scroll", "trackpad/mouse",
			"[e]", "export",
			"[n/p]", "page",
			"[s/S]", "page size",
			"esc", "back",
		))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (v *ResultsView) SetResults(results *engine.GetRowsResult, query string) {
	v.results = results
	v.query = query
	v.currentPage = 0
	v.columnOffset = 0
	v.schema = ""
	v.tableName = ""
	v.totalRows = int(results.TotalCount)
	v.updateTable()
}

func (v *ResultsView) LoadTable(schema string, tableName string) {
	conn := v.parent.dbManager.GetCurrentConnection()
	if conn == nil {
		return
	}

	// Only reset WHERE condition if we're switching to a different table
	if v.schema != schema || v.tableName != tableName {
		v.whereCondition = nil
		v.visibleColumns = nil
	}

	results, err := v.parent.dbManager.GetRows(schema, tableName, v.whereCondition, v.pageSize, v.currentPage*v.pageSize)
	if err != nil {
		v.parent.err = err
		v.whereCondition = nil
		return
	}

	v.results = results
	v.query = ""
	v.currentPage = 0
	v.columnOffset = 0
	v.schema = schema
	v.tableName = tableName
	v.totalRows = int(results.TotalCount)
	v.updateTable()
}

func (v *ResultsView) loadWithWhere() {
	conn := v.parent.dbManager.GetCurrentConnection()
	if conn == nil {
		return
	}

	if v.schema == "" || v.tableName == "" {
		return
	}

	results, err := v.parent.dbManager.GetRows(v.schema, v.tableName, v.whereCondition, v.pageSize, v.currentPage*v.pageSize)
	if err != nil {
		v.parent.err = err
		v.whereCondition = nil
		return
	}

	v.results = results
	v.query = ""
	v.currentPage = 0
	v.columnOffset = 0
	v.totalRows = int(results.TotalCount)
	v.updateTable()
}

func (v *ResultsView) updateTable() {
	if v.results == nil {
		return
	}

	// Filter columns based on visibleColumns if set
	var columnsToDisplay []engine.Column
	var columnIndices []int
	if len(v.visibleColumns) > 0 {
		for _, visibleCol := range v.visibleColumns {
			for idx, col := range v.results.Columns {
				if col.Name == visibleCol {
					columnsToDisplay = append(columnsToDisplay, col)
					columnIndices = append(columnIndices, idx)
					break
				}
			}
		}
	} else {
		columnsToDisplay = v.results.Columns
		columnIndices = make([]int, len(columnsToDisplay))
		for i := range columnIndices {
			columnIndices[i] = i
		}
	}

	// Calculate visible columns based on offset and max
	totalCols := len(columnsToDisplay)

	// Ensure columnOffset is within bounds
	if v.columnOffset >= totalCols {
		v.columnOffset = 0
	}

	endCol := v.columnOffset + v.maxColumns
	if endCol > totalCols {
		endCol = totalCols
	}

	// Only show visible columns
	visibleCols := columnsToDisplay[v.columnOffset:endCol]
	visibleIndices := columnIndices[v.columnOffset:endCol]
	columns := make([]table.Column, len(visibleCols))
	for i, col := range visibleCols {
		columns[i] = table.Column{
			Title: col.Name,
			Width: 20,
		}
	}

	// Extract visible column data from rows
	rows := make([]table.Row, len(v.results.Rows))
	for i, row := range v.results.Rows {
		visibleRow := make([]string, len(visibleIndices))
		for j, idx := range visibleIndices {
			if idx < len(row) {
				visibleRow[j] = row[idx]
			} else {
				visibleRow[j] = ""
			}
		}
		rows[i] = table.Row(visibleRow)
	}

	// Handle the ordering carefully to avoid index out of range:
	// 1. Clear rows first to prevent index issues
	// 2. Set new columns (headers need to update even if count is same)
	// 3. Reset cursor
	// 4. Set the rows
	v.table.SetRows([]table.Row{})
	v.table.SetColumns(columns)
	v.table.SetCursor(0)
	v.table.SetRows(rows)
}

func (v *ResultsView) loadPage() tea.Cmd {
	return func() tea.Msg {
		// Only reload if we're viewing a table (not query results)
		if v.tableName != "" && v.schema != "" {
			results, err := v.parent.dbManager.GetRows(v.schema, v.tableName, v.whereCondition, v.pageSize, v.currentPage*v.pageSize)
			if err != nil {
				v.parent.err = err
				v.whereCondition = nil
				return pageLoadedMsg{}
			}
			v.results = results
			v.totalRows = int(results.TotalCount)
			v.updateTable()
		}
		return pageLoadedMsg{}
	}
}

func (v *ResultsView) loadPageAndGoToBottom() tea.Cmd {
	return func() tea.Msg {
		// Only reload if we're viewing a table (not query results)
		if v.tableName != "" && v.schema != "" {
			results, err := v.parent.dbManager.GetRows(v.schema, v.tableName, v.whereCondition, v.pageSize, v.currentPage*v.pageSize)
			if err != nil {
				v.parent.err = err
				v.whereCondition = nil
				return pageLoadedMsg{}
			}
			v.results = results
			v.totalRows = int(results.TotalCount)
			v.updateTable()
			// Set cursor to bottom of the new page
			if len(v.results.Rows) > 0 {
				v.table.SetCursor(len(v.results.Rows) - 1)
			}
		}
		return pageLoadedMsg{}
	}
}

func (v *ResultsView) countWhereConditions() int {
	if v.whereCondition == nil {
		return 0
	}
	if v.whereCondition.And != nil && v.whereCondition.And.Children != nil {
		return len(v.whereCondition.And.Children)
	}
	return 0
}
