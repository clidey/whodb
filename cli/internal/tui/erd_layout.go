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

	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/pkg/styles"
	"github.com/clidey/whodb/core/src/engine"
)

// erdTableBox holds the rendered box for a single table and its position on the canvas.
type erdTableBox struct {
	name    string
	content string
	width   int
	height  int
	x       int
	y       int
}

// renderTableBox renders a single table as a Unicode box-drawing box.
// If compact is true, only the table name header is shown (no columns).
func renderTableBox(table tableWithColumns, compact bool, focused bool) erdTableBox {
	name := table.StorageUnit.Name

	if compact {
		return renderCompactBox(name, focused)
	}

	return renderExpandedBox(name, table.Columns, focused)
}

// renderCompactBox renders a minimal table box with only the name.
func renderCompactBox(name string, focused bool) erdTableBox {
	// Compact: just header row
	inner := " " + name + " "
	boxWidth := len(inner) + 2 // +2 for left/right border chars

	borderColor := styles.Muted
	if focused {
		borderColor = styles.Primary
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	nameStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)

	top := borderStyle.Render("┌─") + nameStyle.Render(name) + borderStyle.Render(" "+strings.Repeat("─", 1)+"┐")
	bottom := borderStyle.Render("└" + strings.Repeat("─", boxWidth-2) + "┘")

	content := top + "\n" + bottom
	renderedWidth := lipgloss.Width(top)

	return erdTableBox{
		name:    name,
		content: content,
		width:   renderedWidth,
		height:  2,
	}
}

// renderExpandedBox renders a full table box with columns listed inside.
func renderExpandedBox(name string, columns []engine.Column, focused bool) erdTableBox {
	borderColor := styles.Muted
	if focused {
		borderColor = styles.Primary
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	nameStyle := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	colNameStyle := lipgloss.NewStyle().Foreground(styles.Foreground)
	colTypeStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	pkStyle := lipgloss.NewStyle().Foreground(styles.Success)
	fkStyle := lipgloss.NewStyle().Foreground(styles.Info)

	// Build column lines to determine max width
	type colLine struct {
		plain  string // for width calculation (no ANSI)
		styled string
	}

	var lines []colLine
	for _, col := range columns {
		var badges string
		if col.IsPrimary {
			badges += " [PK]"
		}
		if col.IsForeignKey && col.ReferencedTable != nil && col.ReferencedColumn != nil {
			badges += fmt.Sprintf(" [FK]->%s.%s", *col.ReferencedTable, *col.ReferencedColumn)
		} else if col.IsForeignKey {
			badges += " [FK]"
		}

		plain := fmt.Sprintf(" %s: %s%s ", col.Name, col.Type, badges)

		// Build styled version
		styled := " " + colNameStyle.Render(col.Name) + colTypeStyle.Render(": "+col.Type)
		if col.IsPrimary {
			styled += pkStyle.Render(" [PK]")
		}
		if col.IsForeignKey && col.ReferencedTable != nil && col.ReferencedColumn != nil {
			styled += fkStyle.Render(fmt.Sprintf(" [FK]->%s.%s", *col.ReferencedTable, *col.ReferencedColumn))
		} else if col.IsForeignKey {
			styled += fkStyle.Render(" [FK]")
		}
		styled += " "

		lines = append(lines, colLine{plain: plain, styled: styled})
	}

	// Determine box inner width
	minWidth := len(name) + 4 // "─ name ─" needs at least this
	maxLineWidth := minWidth
	for _, l := range lines {
		if len(l.plain) > maxLineWidth {
			maxLineWidth = len(l.plain)
		}
	}
	innerWidth := maxLineWidth

	// Render top border: ┌─ name ──...─┐
	headerPad := innerWidth - len(name) - 2 // -2 for "─ " before name, " " after
	if headerPad < 1 {
		headerPad = 1
	}
	top := borderStyle.Render("┌─") + " " + nameStyle.Render(name) + " " + borderStyle.Render(strings.Repeat("─", headerPad)+"┐")

	// Render column rows
	var rowStrings []string
	rowStrings = append(rowStrings, top)
	for _, l := range lines {
		pad := innerWidth - len(l.plain)
		if pad < 0 {
			pad = 0
		}
		row := borderStyle.Render("│") + l.styled + strings.Repeat(" ", pad) + borderStyle.Render("│")
		rowStrings = append(rowStrings, row)
	}

	// If no columns, show an empty row
	if len(lines) == 0 {
		emptyText := styles.MutedStyle.Render(" (no columns) ")
		pad := innerWidth - len(" (no columns) ")
		if pad < 0 {
			pad = 0
		}
		row := borderStyle.Render("│") + emptyText + strings.Repeat(" ", pad) + borderStyle.Render("│")
		rowStrings = append(rowStrings, row)
	}

	// Bottom border
	bottom := borderStyle.Render("└" + strings.Repeat("─", innerWidth) + "┘")
	rowStrings = append(rowStrings, bottom)

	content := strings.Join(rowStrings, "\n")
	renderedWidth := lipgloss.Width(top)
	renderedHeight := len(rowStrings)

	return erdTableBox{
		name:    name,
		content: content,
		width:   renderedWidth,
		height:  renderedHeight,
	}
}

// layoutERDGrid arranges table boxes in a grid layout that fits within the viewport width.
// Returns the rendered boxes with x/y positions set, and the total canvas dimensions.
func layoutERDGrid(tables []tableWithColumns, viewportWidth int, compact bool, focusedIndex int) ([]erdTableBox, int, int) {
	if len(tables) == 0 {
		return nil, 0, 0
	}

	const hGap = 3 // horizontal gap between boxes
	const vGap = 1 // vertical gap between rows

	// First pass: render all boxes to get their sizes
	boxes := make([]erdTableBox, len(tables))
	for i, table := range tables {
		boxes[i] = renderTableBox(table, compact, i == focusedIndex)
	}

	// Second pass: arrange in rows
	var rows [][]int // each row is a slice of box indices
	var currentRow []int
	rowWidth := 0

	for i, box := range boxes {
		neededWidth := box.width
		if len(currentRow) > 0 {
			neededWidth += hGap
		}

		if rowWidth+neededWidth > viewportWidth && len(currentRow) > 0 {
			rows = append(rows, currentRow)
			currentRow = []int{i}
			rowWidth = box.width
		} else {
			currentRow = append(currentRow, i)
			rowWidth += neededWidth
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	// Third pass: assign positions
	canvasWidth := 0
	canvasHeight := 0
	y := 0

	for _, row := range rows {
		x := 0
		rowHeight := 0

		for j, idx := range row {
			if j > 0 {
				x += hGap
			}
			boxes[idx].x = x
			boxes[idx].y = y
			x += boxes[idx].width
			if boxes[idx].height > rowHeight {
				rowHeight = boxes[idx].height
			}
		}

		if x > canvasWidth {
			canvasWidth = x
		}
		y += rowHeight + vGap
		canvasHeight = y
	}

	return boxes, canvasWidth, canvasHeight
}

// renderERDCanvas composites the table boxes onto a single string canvas.
func renderERDCanvas(boxes []erdTableBox) string {
	if len(boxes) == 0 {
		return ""
	}

	// Find canvas dimensions
	maxX := 0
	maxY := 0
	for _, box := range boxes {
		right := box.x + box.width
		bottom := box.y + box.height
		if right > maxX {
			maxX = right
		}
		if bottom > maxY {
			maxY = bottom
		}
	}

	// Build a 2D grid of strings (one per line)
	// We use a simpler approach: render each box at its y offset,
	// joining lines with appropriate x padding.
	canvas := make([]string, maxY)
	for i := range canvas {
		canvas[i] = ""
	}

	// Place each box line by line
	for _, box := range boxes {
		lines := strings.Split(box.content, "\n")
		for i, line := range lines {
			row := box.y + i
			if row >= len(canvas) {
				break
			}

			// Ensure the canvas row is wide enough
			currentWidth := lipgloss.Width(canvas[row])
			if currentWidth < box.x {
				canvas[row] += strings.Repeat(" ", box.x-currentWidth)
			}
			canvas[row] += line
		}
	}

	return strings.Join(canvas, "\n")
}
