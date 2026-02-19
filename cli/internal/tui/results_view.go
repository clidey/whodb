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

	"github.com/charmbracelet/bubbles/key"
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
	loading         bool
	goToBottom      bool // Flag to set cursor at bottom after loading
}

// Available page sizes for cycling
var pageSizes = []int{10, 25, 50, 100}

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
		pageSize:      parent.config.GetPageSize(),
		columnOffset:  0,
		maxColumns:    10,
		pageSizeInput: ti,
	}
}

func (v *ResultsView) Update(msg tea.Msg) (*ResultsView, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		overhead := 14
		if v.query != "" {
			overhead += 3
		}
		h := msg.Height - overhead
		if h < 5 {
			h = 5
		}
		v.table.SetHeight(h)
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

	case PageLoadedMsg:
		v.loading = false
		if msg.Err != nil {
			v.parent.err = msg.Err
			v.whereCondition = nil
			return v, nil
		}
		// Handle initial table load (Schema/TableName provided)
		if msg.Schema != "" && msg.TableName != "" {
			v.schema = msg.Schema
			v.tableName = msg.TableName
			v.query = ""
			v.columnOffset = 0
			v.currentPage = 0
		}
		if msg.Results != nil {
			v.results = msg.Results
			v.totalRows = int(msg.Results.TotalCount)
			v.updateTable()
			// If goToBottom flag is set, move cursor to bottom
			if v.goToBottom {
				if pageRows := v.currentPageRows(); len(pageRows) > 0 {
					v.table.SetCursor(len(pageRows) - 1)
				}
				v.goToBottom = false
			}
		}
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
					v.parent.config.SetPageSize(v.pageSize)
					v.parent.config.Save()
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

		switch {
		case key.Matches(msg, Keys.Global.Back):
			if !v.parent.PopView() {
				v.parent.mode = ViewBrowser
			}
			return v, nil

		case key.Matches(msg, Keys.Results.NextPage):
			// Check if we can go to next page
			if v.hasNextPage() {
				v.currentPage++
				return v, v.loadPage()
			}
			return v, nil

		case key.Matches(msg, Keys.Results.PrevPage):
			if v.hasPreviousPage() {
				v.currentPage--
				return v, v.loadPage()
			}
			return v, nil

		case key.Matches(msg, Keys.Results.ColLeft):
			if v.columnOffset > 0 {
				v.columnOffset--
				v.updateTable()
			}
			return v, nil

		case key.Matches(msg, Keys.Results.ColRight):
			if v.results != nil && v.columnOffset+v.maxColumns < len(v.results.Columns) {
				v.columnOffset++
				v.updateTable()
			}
			return v, nil

		case key.Matches(msg, Keys.Results.Where):
			// WHERE conditions are only available when viewing table data
			if v.schema != "" && v.tableName != "" {
				columns, err := v.parent.dbManager.GetColumns(v.schema, v.tableName)
				if err != nil {
					v.parent.err = err
					return v, nil
				}
				v.parent.whereView.SetTableContext(v.schema, v.tableName, columns, v.whereCondition)
				v.parent.PushView(ViewWhere)
				return v, nil
			}

		case key.Matches(msg, Keys.Results.Columns):
			// Column selection is only available when viewing table data
			if v.schema != "" && v.tableName != "" {
				columns, err := v.parent.dbManager.GetColumns(v.schema, v.tableName)
				if err != nil {
					v.parent.err = err
					return v, nil
				}
				v.parent.columnsView.SetTableContext(v.schema, v.tableName, columns)
				v.parent.PushView(ViewColumns)
				return v, nil
			}

		case key.Matches(msg, Keys.Results.Export):
			if v.schema != "" && v.tableName != "" {
				// Export table data
				v.parent.exportView.SetExportData(v.schema, v.tableName)
				v.parent.PushView(ViewExport)
				return v, nil
			} else if v.results != nil && v.query != "" {
				// Export query results
				v.parent.exportView.SetExportDataFromQuery(v.results)
				v.parent.PushView(ViewExport)
				return v, nil
			}

		case key.Matches(msg, Keys.Results.PageSize):
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
			v.parent.config.SetPageSize(v.pageSize)
			v.parent.config.Save()
			return v, v.loadPage()

		case key.Matches(msg, Keys.Results.CustomSize):
			// Enter custom page size mode
			v.editingPageSize = true
			v.pageSizeInput.SetValue("")
			v.pageSizeInput.Focus()
			return v, nil

		case key.Matches(msg, Keys.Results.Down):
			// Check if at bottom of current page - auto-paginate to next
			if v.results != nil {
				pageRows := v.currentPageRows()
				if len(pageRows) == 0 {
					return v, nil
				}
				cursor := v.table.Cursor()
				if cursor >= len(pageRows)-1 {
					// At bottom, try to go to next page
					if v.hasNextPage() {
						v.currentPage++
						v.table.SetCursor(0)
						return v, v.loadPage()
					}
				}
			}

		case key.Matches(msg, Keys.Results.Up):
			// Check if at top of current page - auto-paginate to previous
			if v.results != nil {
				pageRows := v.currentPageRows()
				if len(pageRows) == 0 {
					return v, nil
				}
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
	pageRows := v.currentPageRows()

	if v.query != "" {
		b.WriteString(styles.RenderTitle("Query Results"))
		b.WriteString("\n")
		b.WriteString(styles.MutedStyle.Render(v.query))
		b.WriteString("\n\n")
	} else {
		b.WriteString(styles.RenderTitle("Table Data"))
		b.WriteString("\n\n")
	}

	if v.loading {
		b.WriteString(v.parent.SpinnerView() + styles.MutedStyle.Render(" Loading..."))
	} else if v.results == nil {
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
		totalRows := v.effectiveTotalRows()
		if totalRows > 0 {
			totalPages := (totalRows + v.pageSize - 1) / v.pageSize
			rowInfo = fmt.Sprintf("Showing %d rows (Page %d of %d, size: %d)", len(pageRows), v.currentPage+1, totalPages, v.pageSize)
		} else {
			rowInfo = fmt.Sprintf("Showing %d rows (Page %d, size: %d)", len(pageRows), v.currentPage+1, v.pageSize)
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
	// Also show appropriate back target (editor for query results, browser for table data)
	backTarget := "browser"
	if v.query != "" {
		backTarget = "editor"
	}

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

		// Use static bindings for most items, but dynamic labels for where/columns/back
		b.WriteString(styles.RenderHelp(
			Keys.Results.Up.Help().Key, Keys.Results.Up.Help().Desc,
			Keys.Results.Down.Help().Key, Keys.Results.Down.Help().Desc,
			Keys.Results.ColLeft.Help().Key, Keys.Results.ColLeft.Help().Desc,
			Keys.Results.ColRight.Help().Key, Keys.Results.ColRight.Help().Desc,
			"scroll", "trackpad/mouse",
			Keys.Results.Where.Help().Key, whereLabel,
			Keys.Results.Columns.Help().Key, columnsLabel,
			Keys.Results.Export.Help().Key, Keys.Results.Export.Help().Desc,
			Keys.Results.NextPage.Help().Key, Keys.Results.NextPage.Help().Desc,
			Keys.Results.PageSize.Help().Key, Keys.Results.PageSize.Help().Desc,
			Keys.Results.CustomSize.Help().Key, Keys.Results.CustomSize.Help().Desc,
			Keys.Global.Back.Help().Key, backTarget,
		))
	} else {
		b.WriteString(styles.RenderHelp(
			Keys.Results.Up.Help().Key, Keys.Results.Up.Help().Desc,
			Keys.Results.Down.Help().Key, Keys.Results.Down.Help().Desc,
			Keys.Results.ColLeft.Help().Key, Keys.Results.ColLeft.Help().Desc,
			Keys.Results.ColRight.Help().Key, Keys.Results.ColRight.Help().Desc,
			"scroll", "trackpad/mouse",
			Keys.Results.Export.Help().Key, Keys.Results.Export.Help().Desc,
			Keys.Results.NextPage.Help().Key, Keys.Results.NextPage.Help().Desc,
			Keys.Results.PageSize.Help().Key, Keys.Results.PageSize.Help().Desc,
			Keys.Results.CustomSize.Help().Key, Keys.Results.CustomSize.Help().Desc,
			Keys.Global.Back.Help().Key, backTarget,
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
	if v.totalRows == 0 && v.results != nil {
		v.totalRows = len(v.results.Rows)
	}
	v.updateTable()
}

func (v *ResultsView) LoadTable(schema string, tableName string) tea.Cmd {
	conn := v.parent.dbManager.GetCurrentConnection()
	if conn == nil {
		return nil
	}

	// Only reset WHERE condition if we're switching to a different table
	if v.schema != schema || v.tableName != tableName {
		v.whereCondition = nil
		v.visibleColumns = nil
	}

	v.loading = true

	// Capture values for closure
	where := v.whereCondition
	pageSize := v.pageSize

	return func() tea.Msg {
		results, err := v.parent.dbManager.GetRows(schema, tableName, where, pageSize, 0)
		return PageLoadedMsg{Results: results, Err: err, Schema: schema, TableName: tableName}
	}
}

func (v *ResultsView) loadWithWhere() tea.Cmd {
	conn := v.parent.dbManager.GetCurrentConnection()
	if conn == nil {
		return nil
	}

	if v.schema == "" || v.tableName == "" {
		return nil
	}

	v.loading = true

	// Capture values for closure
	schema := v.schema
	tableName := v.tableName
	where := v.whereCondition
	pageSize := v.pageSize

	return func() tea.Msg {
		results, err := v.parent.dbManager.GetRows(schema, tableName, where, pageSize, 0)
		return PageLoadedMsg{Results: results, Err: err}
	}
}

func (v *ResultsView) isTableData() bool {
	return v.schema != "" && v.tableName != ""
}

func (v *ResultsView) effectiveTotalRows() int {
	if v.results == nil {
		return 0
	}
	if v.totalRows > 0 {
		return v.totalRows
	}
	return len(v.results.Rows)
}

func (v *ResultsView) currentPageRows() [][]string {
	if v.results == nil {
		return nil
	}

	// Table browsing already paginates at the database level
	if v.isTableData() || v.pageSize <= 0 {
		return v.results.Rows
	}

	total := len(v.results.Rows)
	if total == 0 {
		return nil
	}

	start := v.currentPage * v.pageSize
	if start >= total {
		start = max(total-v.pageSize, 0)
	}
	end := start + v.pageSize
	if end > total {
		end = total
	}

	return v.results.Rows[start:end]
}

func (v *ResultsView) hasNextPage() bool {
	if v.pageSize <= 0 {
		return false
	}

	total := v.effectiveTotalRows()
	if total > 0 {
		totalPages := (total + v.pageSize - 1) / v.pageSize
		return v.currentPage+1 < totalPages
	}

	return v.results != nil && len(v.results.Rows) == v.pageSize
}

func (v *ResultsView) hasPreviousPage() bool {
	return v.currentPage > 0
}

func (v *ResultsView) updateTable() {
	if v.results == nil {
		v.table.SetRows([]table.Row{})
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
	currentRows := v.currentPageRows()
	rows := make([]table.Row, len(currentRows))
	for i, row := range currentRows {
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
	// Only reload if we're viewing a table (not query results)
	if !v.isTableData() {
		// For query results, just update the table locally (pagination is client-side)
		v.updateTable()
		return nil
	}

	v.loading = true

	// Capture values for closure
	schema := v.schema
	tableName := v.tableName
	where := v.whereCondition
	pageSize := v.pageSize
	offset := v.currentPage * v.pageSize

	return func() tea.Msg {
		results, err := v.parent.dbManager.GetRows(schema, tableName, where, pageSize, offset)
		return PageLoadedMsg{Results: results, Err: err}
	}
}

func (v *ResultsView) loadPageAndGoToBottom() tea.Cmd {
	// Only reload if we're viewing a table (not query results)
	if !v.isTableData() {
		// For query results, just update the table locally and set cursor to bottom
		v.updateTable()
		if pageRows := v.currentPageRows(); len(pageRows) > 0 {
			v.table.SetCursor(len(pageRows) - 1)
		}
		return nil
	}

	v.loading = true
	v.goToBottom = true // Flag to set cursor at bottom when results arrive

	// Capture values for closure
	schema := v.schema
	tableName := v.tableName
	where := v.whereCondition
	pageSize := v.pageSize
	offset := v.currentPage * v.pageSize

	return func() tea.Msg {
		results, err := v.parent.dbManager.GetRows(schema, tableName, where, pageSize, offset)
		return PageLoadedMsg{Results: results, Err: err}
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
