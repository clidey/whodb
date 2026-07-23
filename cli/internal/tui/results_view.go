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

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	cursor          int
	rowWindowStart  int
	maxColumns      int
	whereCondition  *model.WhereCondition
	visibleColumns  []string
	width           int
	height          int
	editingPageSize bool
	pageSizeInput   textinput.Model
	loading         bool
	goToBottom      bool // Flag to set cursor at bottom after loading
	compact         bool
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
	ti.SetWidth(10)

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
		v.width = msg.Width
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
		if v.results != nil {
			v.updateTable()
		}
		return v, nil

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			v.moveSelectionUp()
			return v, nil
		case tea.MouseWheelDown:
			v.moveSelectionDown()
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
		// Handle initial table load (TableName provided; schema may be empty for SQLite)
		if msg.TableName != "" {
			v.schema = msg.Schema
			v.tableName = msg.TableName
			v.query = ""
			v.columnOffset = 0
			v.currentPage = 0
			v.cursor = 0
			v.rowWindowStart = 0
		}
		if msg.Results != nil {
			v.results = msg.Results
			v.totalRows = int(msg.Results.TotalCount)
			if v.goToBottom {
				if pageRows := v.currentPageRows(); len(pageRows) > 0 {
					v.cursor = len(pageRows) - 1
				} else {
					v.cursor = 0
				}
				v.goToBottom = false
			} else if len(v.currentPageRows()) == 0 || v.currentPage == 0 {
				v.cursor = 0
			}
			v.updateTable()
		}
		return v, nil

	case tea.KeyPressMsg:
		// Handle page size editing mode
		if v.editingPageSize {
			switch msg.String() {
			case "enter":
				if size, err := strconv.Atoi(v.pageSizeInput.Value()); err == nil && size > 0 {
					v.pageSize = size
					v.currentPage = 0
					v.cursor = 0
					v.rowWindowStart = 0
					v.editingPageSize = false
					v.pageSizeInput.Blur()
					v.parent.config.SetPageSize(v.pageSize)
					return v, tea.Batch(v.parent.requestConfigSave(), v.loadPage())
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
				v.cursor = 0
				v.rowWindowStart = 0
				return v, v.loadPage()
			}
			return v, nil

		case key.Matches(msg, Keys.Results.PrevPage):
			if v.hasPreviousPage() {
				v.currentPage--
				v.cursor = 0
				v.rowWindowStart = 0
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
			if v.tableName != "" {
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
			if v.tableName != "" {
				columns, err := v.parent.dbManager.GetColumns(v.schema, v.tableName)
				if err != nil {
					v.parent.err = err
					return v, nil
				}
				v.parent.columnsView.SetTableContext(v.schema, v.tableName, columns)
				v.parent.PushView(ViewColumns)
				return v, nil
			}

		case key.Matches(msg, Keys.Results.AddRow):
			if v.tableName != "" {
				columns, err := v.parent.dbManager.GetColumns(v.schema, v.tableName)
				if err != nil {
					v.parent.err = err
					return v, nil
				}
				v.parent.suspendLayout()
				v.parent.rowWriteView.SetAddContext(v.schema, v.tableName, columns)
				v.parent.PushView(ViewRowWrite)
				return v, nil
			}

		case key.Matches(msg, Keys.Results.EditRow):
			if v.tableName != "" && v.results != nil && !v.results.DisableUpdate {
				pageRows := v.currentPageRows()
				cursor := v.selectedRowIndex(pageRows)
				if cursor >= 0 && cursor < len(pageRows) {
					v.parent.suspendLayout()
					v.parent.rowWriteView.SetEditContext(v.schema, v.tableName, v.results.Columns, v.selectedRowValues(pageRows, cursor))
					v.parent.PushView(ViewRowWrite)
					return v, nil
				}
			}

		case key.Matches(msg, Keys.Results.DeleteRow):
			if v.tableName != "" && v.results != nil {
				pageRows := v.currentPageRows()
				cursor := v.selectedRowIndex(pageRows)
				if cursor >= 0 && cursor < len(pageRows) {
					v.parent.suspendLayout()
					v.parent.rowWriteView.SetDeleteContext(v.schema, v.tableName, v.results.Columns, v.selectedRowValues(pageRows, cursor))
					v.parent.PushView(ViewRowWrite)
					return v, nil
				}
			}

		case key.Matches(msg, Keys.Results.Export):
			if v.tableName != "" {
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
			// Cycle through page sizes; if current size is custom, start from the beginning
			currentIndex := -1
			for i, size := range pageSizes {
				if size == v.pageSize {
					currentIndex = i
					break
				}
			}
			v.pageSize = pageSizes[(currentIndex+1)%len(pageSizes)]
			v.currentPage = 0
			v.cursor = 0
			v.rowWindowStart = 0
			v.parent.config.SetPageSize(v.pageSize)
			return v, tea.Batch(v.parent.requestConfigSave(), v.loadPage())

		case key.Matches(msg, Keys.Results.CustomSize):
			// Enter custom page size mode
			v.editingPageSize = true
			v.pageSizeInput.SetValue("")
			v.pageSizeInput.Focus()
			return v, nil

		case key.Matches(msg, Keys.Results.ViewCell):
			if v.results != nil {
				pageRows := v.currentPageRows()
				cursor := v.selectedRowIndex(pageRows)
				if cursor >= 0 && cursor < len(pageRows) {
					colName, cellValue := v.selectedCell(pageRows, cursor)
					v.parent.jsonViewer.SetContent(colName, cellValue)
					v.parent.PushView(ViewJSON)
					return v, nil
				}
			}

		case key.Matches(msg, Keys.Results.Down):
			// Check if at bottom of current page - auto-paginate to next
			if v.results != nil {
				pageRows := v.currentPageRows()
				if len(pageRows) == 0 {
					return v, nil
				}
				cursor := v.selectedRowIndex(pageRows)
				if cursor >= len(pageRows)-1 {
					// At bottom, try to go to next page
					if v.hasNextPage() {
						v.currentPage++
						v.cursor = 0
						v.rowWindowStart = 0
						return v, v.loadPage()
					}
				} else {
					v.moveSelectionDown()
					return v, nil
				}
			}

		case key.Matches(msg, Keys.Results.Up):
			// Check if at top of current page - auto-paginate to previous
			if v.results != nil {
				pageRows := v.currentPageRows()
				if len(pageRows) == 0 {
					return v, nil
				}
				cursor := v.selectedRowIndex(pageRows)
				if cursor <= 0 && v.currentPage > 0 {
					v.currentPage--
					return v, v.loadPageAndGoToBottom()
				}
				if cursor > 0 {
					v.moveSelectionUp()
					return v, nil
				}
			}
		}
	}

	// Pass to table for navigation (arrows, page up/down, etc.)
	v.table, cmd = v.table.Update(msg)
	v.syncSelectionFromTable()
	return v, cmd
}

func (v *ResultsView) View() string {
	var b strings.Builder
	pageRows := v.currentPageRows()

	if v.query != "" {
		b.WriteString(styles.RenderTitle("Query Results"))
		b.WriteString("\n")
		b.WriteString(styles.RenderMuted(v.query))
		b.WriteString("\n\n")
	} else {
		b.WriteString(styles.RenderTitle("Table Data"))
		b.WriteString("\n\n")
	}

	if v.loading {
		b.WriteString(v.parent.SpinnerView() + styles.RenderMuted(" Loading..."))
	} else if v.results == nil {
		b.WriteString(styles.RenderMuted("No results"))
	} else {
		b.WriteString(v.table.View())
		b.WriteString("\n\n")

		// Show column and row information
		totalCols := len(v.results.Columns)
		if len(v.visibleColumns) > 0 {
			totalCols = len(v.visibleColumns)
		}
		visibleCols := v.maxColumns
		if v.columnOffset+v.maxColumns > totalCols {
			visibleCols = totalCols - v.columnOffset
		}

		b.WriteString(styles.RenderMuted(v.paginationString(totalCols, visibleCols, len(pageRows))))

		// Show page size input if editing
		if v.editingPageSize {
			b.WriteString("\n\n")
			b.WriteString(styles.RenderKey("Page size: "))
			b.WriteString(v.pageSizeInput.View())
			b.WriteString(styles.RenderMuted(" (enter to confirm, esc to cancel)"))
		}
	}

	if !v.compact {
		b.WriteString("\n\n")

		// Show different help based on whether export/where/columns is available
		// Also show appropriate back target (editor for query results, browser for table data)
		backTarget := "browser"
		if v.query != "" {
			backTarget = "editor"
		}

		if v.tableName != "" {
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
			b.WriteString(renderFooterHelpPairsWidth(v.width,
				Keys.Results.Up.Help().Key, Keys.Results.Up.Help().Desc,
				Keys.Results.Down.Help().Key, Keys.Results.Down.Help().Desc,
				Keys.Results.ColLeft.Help().Key, Keys.Results.ColLeft.Help().Desc,
				Keys.Results.ColRight.Help().Key, Keys.Results.ColRight.Help().Desc,
				"scroll", "trackpad/mouse",
				Keys.Results.ViewCell.Help().Key, Keys.Results.ViewCell.Help().Desc,
				Keys.Results.Where.Help().Key, whereLabel,
				Keys.Results.Columns.Help().Key, columnsLabel,
				Keys.Results.AddRow.Help().Key, Keys.Results.AddRow.Help().Desc,
				Keys.Results.Export.Help().Key, Keys.Results.Export.Help().Desc,
				Keys.Global.SchemaDiff.Help().Key, Keys.Global.SchemaDiff.Help().Desc,
				Keys.Global.ERDiagram.Help().Key, Keys.Global.ERDiagram.Help().Desc,
				Keys.Global.MockData.Help().Key, Keys.Global.MockData.Help().Desc,
				Keys.Results.NextPage.Help().Key, Keys.Results.NextPage.Help().Desc,
				Keys.Results.PageSize.Help().Key, Keys.Results.PageSize.Help().Desc,
				Keys.Results.CustomSize.Help().Key, Keys.Results.CustomSize.Help().Desc,
				Keys.Global.Back.Help().Key, backTarget,
			))
			if v.results != nil && !v.results.DisableUpdate {
				b.WriteString("\n")
				b.WriteString(renderFooterHelpPairsWidthNoHelp(v.width,
					Keys.Results.EditRow.Help().Key, Keys.Results.EditRow.Help().Desc,
					Keys.Results.DeleteRow.Help().Key, Keys.Results.DeleteRow.Help().Desc,
				))
			} else {
				b.WriteString("\n")
				b.WriteString(renderFooterHelpPairsWidthNoHelp(v.width,
					Keys.Results.DeleteRow.Help().Key, Keys.Results.DeleteRow.Help().Desc,
				))
			}
		} else {
			b.WriteString(renderFooterHelpPairsWidth(v.width,
				Keys.Results.Up.Help().Key, Keys.Results.Up.Help().Desc,
				Keys.Results.Down.Help().Key, Keys.Results.Down.Help().Desc,
				Keys.Results.ColLeft.Help().Key, Keys.Results.ColLeft.Help().Desc,
				Keys.Results.ColRight.Help().Key, Keys.Results.ColRight.Help().Desc,
				"scroll", "trackpad/mouse",
				Keys.Results.ViewCell.Help().Key, Keys.Results.ViewCell.Help().Desc,
				Keys.Results.Export.Help().Key, Keys.Results.Export.Help().Desc,
				Keys.Global.SchemaDiff.Help().Key, Keys.Global.SchemaDiff.Help().Desc,
				Keys.Global.ERDiagram.Help().Key, Keys.Global.ERDiagram.Help().Desc,
				Keys.Global.MockData.Help().Key, Keys.Global.MockData.Help().Desc,
				Keys.Results.NextPage.Help().Key, Keys.Results.NextPage.Help().Desc,
				Keys.Results.PageSize.Help().Key, Keys.Results.PageSize.Help().Desc,
				Keys.Results.CustomSize.Help().Key, Keys.Results.CustomSize.Help().Desc,
				Keys.Global.Back.Help().Key, backTarget,
			))
		}
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (v *ResultsView) SetResults(results *engine.GetRowsResult, query string) {
	v.results = results
	v.query = query
	v.currentPage = 0
	v.columnOffset = 0
	v.cursor = 0
	v.rowWindowStart = 0
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

func (v *ResultsView) selectedRowIndex(pageRows [][]string) int {
	if len(pageRows) == 0 {
		return -1
	}
	cursor := v.rowWindowStart + v.table.Cursor()
	if cursor < 0 || cursor >= len(pageRows) {
		cursor = v.cursor
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(pageRows) {
		cursor = len(pageRows) - 1
	}
	return cursor
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
		v.cursor = 0
		v.rowWindowStart = 0
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
	visibleRows, windowStart := v.visibleRows(currentRows)
	rows := make([]table.Row, len(visibleRows))
	for i, row := range visibleRows {
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
	// 3. Set the rows
	v.table.SetRows([]table.Row{})
	v.table.SetColumns(columns)
	v.table.SetRows(rows)
	if len(rows) == 0 {
		v.table.SetCursor(0)
		return
	}

	virtualCursor := v.cursor - windowStart
	if virtualCursor < 0 {
		virtualCursor = 0
	}
	if virtualCursor >= len(rows) {
		virtualCursor = len(rows) - 1
	}
	v.table.SetCursor(virtualCursor)
}

func (v *ResultsView) loadPage() tea.Cmd {
	// Only reload if we're viewing a table (not query results)
	if !v.isTableData() {
		// For query results, just update the table locally (pagination is client-side)
		v.cursor = 0
		v.rowWindowStart = 0
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
		if pageRows := v.currentPageRows(); len(pageRows) > 0 {
			v.cursor = len(pageRows) - 1
		} else {
			v.cursor = 0
		}
		v.rowWindowStart = 0
		v.updateTable()
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
	return countAtomicConditions(v.whereCondition)
}

// countAtomicConditions recursively counts the number of leaf (atomic)
// conditions in a WhereCondition tree.
func countAtomicConditions(wc *model.WhereCondition) int {
	if wc == nil {
		return 0
	}
	switch wc.Type {
	case model.WhereConditionTypeAtomic:
		return 1
	case model.WhereConditionTypeAnd:
		if wc.And != nil {
			n := 0
			for _, c := range wc.And.Children {
				n += countAtomicConditions(c)
			}
			return n
		}
	case model.WhereConditionTypeOr:
		if wc.Or != nil {
			n := 0
			for _, c := range wc.Or.Children {
				n += countAtomicConditions(c)
			}
			return n
		}
	}
	return 0
}

// paginationString returns a width-adaptive pagination summary.
func (v *ResultsView) paginationString(totalCols, visibleCols, rowCount int) string {
	totalRows := v.effectiveTotalRows()
	page := v.currentPage + 1
	totalPages := 0
	if totalRows > 0 {
		totalPages = (totalRows + v.pageSize - 1) / v.pageSize
	}

	colStart := v.columnOffset + 1
	colEnd := v.columnOffset + visibleCols

	// Full format
	full := fmt.Sprintf("Columns %d-%d of %d", colStart, colEnd, totalCols)
	if totalPages > 0 {
		full += fmt.Sprintf(" • Showing %d rows (Page %d of %d, size: %d)", rowCount, page, totalPages, v.pageSize)
	} else {
		full += fmt.Sprintf(" • Showing %d rows (Page %d, size: %d)", rowCount, page, v.pageSize)
	}

	avail := v.width - 8
	if avail <= 0 || lipgloss.Width(full) <= avail {
		return full
	}

	// Medium format
	medium := fmt.Sprintf("Cols %d-%d/%d", colStart, colEnd, totalCols)
	if totalPages > 0 {
		medium += fmt.Sprintf(" • %d rows (%d/%d)", rowCount, page, totalPages)
	} else {
		medium += fmt.Sprintf(" • %d rows (pg %d)", rowCount, page)
	}
	if lipgloss.Width(medium) <= avail {
		return medium
	}

	// Narrow format
	if totalPages > 0 {
		return fmt.Sprintf("%d rows (%d/%d)", rowCount, page, totalPages)
	}
	return fmt.Sprintf("%d rows (pg %d)", rowCount, page)
}

// selectedCell returns the column name and cell value for the first visible
// column of the given row. The table cursor selects the row; the first column
// in the visible window is used since the bubbles table does not track a
// horizontal cell cursor.
func (v *ResultsView) selectedCell(pageRows [][]string, cursor int) (string, string) {
	// Build the same column/index mapping used by updateTable
	var columnIndices []int
	var columnsToDisplay []engine.Column
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

	if v.columnOffset >= len(columnsToDisplay) {
		return "", ""
	}

	col := columnsToDisplay[v.columnOffset]
	dataIdx := columnIndices[v.columnOffset]

	row := pageRows[cursor]
	if dataIdx < len(row) {
		return col.Name, row[dataIdx]
	}
	return col.Name, ""
}

func (v *ResultsView) selectedRowValues(pageRows [][]string, cursor int) map[string]string {
	if v.results == nil || cursor < 0 || cursor >= len(pageRows) {
		return nil
	}

	row := pageRows[cursor]
	values := make(map[string]string, len(v.results.Columns))
	for idx, column := range v.results.Columns {
		if idx < len(row) {
			values[column.Name] = row[idx]
		}
	}
	return values
}

func (v *ResultsView) visibleRows(pageRows [][]string) ([][]string, int) {
	total := len(pageRows)
	if total == 0 {
		v.cursor = 0
		v.rowWindowStart = 0
		return nil, 0
	}

	if v.cursor < 0 {
		v.cursor = 0
	}
	if v.cursor >= total {
		v.cursor = total - 1
	}

	windowSize := v.table.Height()
	if windowSize <= 0 || windowSize > total {
		windowSize = total
	}

	maxStart := total - windowSize
	if v.rowWindowStart < 0 {
		v.rowWindowStart = 0
	}
	if v.rowWindowStart > maxStart {
		v.rowWindowStart = maxStart
	}
	if v.cursor < v.rowWindowStart {
		v.rowWindowStart = v.cursor
	}
	if v.cursor >= v.rowWindowStart+windowSize {
		v.rowWindowStart = v.cursor - windowSize + 1
	}

	end := v.rowWindowStart + windowSize
	if end > total {
		end = total
	}
	return pageRows[v.rowWindowStart:end], v.rowWindowStart
}

func (v *ResultsView) moveSelectionUp() {
	pageRows := v.currentPageRows()
	if len(pageRows) == 0 || v.cursor <= 0 {
		return
	}

	v.cursor--
	v.updateTable()
}

func (v *ResultsView) moveSelectionDown() {
	pageRows := v.currentPageRows()
	if len(pageRows) == 0 || v.cursor >= len(pageRows)-1 {
		return
	}

	v.cursor++
	v.updateTable()
}

func (v *ResultsView) syncSelectionFromTable() {
	pageRows := v.currentPageRows()
	if len(pageRows) == 0 {
		v.cursor = 0
		v.rowWindowStart = 0
		return
	}

	cursor := v.rowWindowStart + v.table.Cursor()
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(pageRows) {
		cursor = len(pageRows) - 1
	}
	v.cursor = cursor
}
