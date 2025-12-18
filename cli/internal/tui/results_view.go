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

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/pkg/styles"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
)

type ResultsView struct {
	parent         *MainModel
	table          table.Model
	results        *engine.GetRowsResult
	query          string
	currentPage    int
	pageSize       int
	totalRows      int
	totalCount     int64 // Total number of rows in the table/query result
	schema         string
	tableName      string
	columnOffset   int
	maxColumns     int
	whereCondition *model.WhereCondition
	visibleColumns []string
}

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

	return &ResultsView{
		parent:       parent,
		table:        t,
		currentPage:  0,
		pageSize:     50,
		columnOffset: 0,
		maxColumns:   10,
		totalCount:   0,
	}
}

// getTotalPages calculates the total number of pages based on totalCount and pageSize
func (v *ResultsView) getTotalPages() int {
	if v.totalCount == 0 || v.pageSize == 0 {
		return 0
	}
	return int((v.totalCount + int64(v.pageSize) - 1) / int64(v.pageSize))
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

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			v.parent.mode = ViewBrowser
			return v, nil

		case "n":
			// Only advance to next page if we're not at the last page
			totalPages := v.getTotalPages()
			if totalPages > 0 && v.currentPage >= totalPages-1 {
				// Already at or past the last page
				return v, nil
			}
			// For query results (no totalCount), check if current page has data
			if totalPages == 0 && len(v.results.Rows) < v.pageSize {
				// Current page is not full, likely the last page
				return v, nil
			}
			v.currentPage++
			return v, v.loadPage()

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

			//case "s":
			//	if v.schema != "" && v.tableName != "" {
			//		v.parent.mode = ViewSchema
			//		return v, v.parent.schemaView.Init()
			//	}
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

		// Build row info with page indicator
		var rowInfo string
		totalPages := v.getTotalPages()
		if totalPages > 0 {
			rowInfo = fmt.Sprintf("Showing %d rows (Page %d of %d)", len(v.results.Rows), v.currentPage+1, totalPages)
		} else {
			rowInfo = fmt.Sprintf("Showing %d rows (Page %d)", len(v.results.Rows), v.currentPage+1)
		}

		b.WriteString(styles.MutedStyle.Render(columnInfo + " • " + rowInfo))
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
			"tab", "next view",
			"esc", "back",
		))
		b.WriteString("\n")
		b.WriteString(styles.MutedStyle.Render("(WHERE and Columns only available for table data, not query results)"))
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
	if results != nil {
		v.totalCount = results.TotalCount
	} else {
		v.totalCount = 0
	}
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
	if results != nil {
		v.totalCount = results.TotalCount
	} else {
		v.totalCount = 0
	}
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
	if results != nil {
		v.totalCount = results.TotalCount
	} else {
		v.totalCount = 0
	}
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
	// 1. If we're changing the number of columns, set columns first (with empty rows)
	// 2. Then set the cursor to 0
	// 3. Finally set the rows
	currentCols := len(v.table.Columns())
	newCols := len(columns)

	if currentCols != newCols {
		// Column count is changing - set columns first with no rows
		v.table.SetRows([]table.Row{})
		v.table.SetColumns(columns)
		v.table.SetCursor(0)
		v.table.SetRows(rows)
	} else {
		// Same column count - just update rows and cursor
		v.table.SetCursor(0)
		v.table.SetRows(rows)
	}

	v.totalRows = len(rows)
}

func (v *ResultsView) loadPage() tea.Cmd {
	return func() tea.Msg {
		// Only reload if we're viewing a table (not query results)
		if v.tableName != "" && v.schema != "" {
			results, err := v.parent.dbManager.GetRows(v.schema, v.tableName, v.whereCondition, v.pageSize, v.currentPage*v.pageSize)
			if err != nil {
				v.parent.err = err
				v.whereCondition = nil
				return nil
			}

			// If no results returned, we've gone past the last page, go back
			if results == nil || len(results.Rows) == 0 {
				if v.currentPage > 0 {
					v.currentPage--
				}
				return nil
			}

			v.results = results
			if results != nil {
				v.totalCount = results.TotalCount
			}
			v.updateTable()
		}
		return nil
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
