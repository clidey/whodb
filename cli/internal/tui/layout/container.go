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

// Package layout implements a binary-tree container system for split-pane
// terminal layouts. Leaf nodes hold a Renderable (any TUI view), branch
// nodes split space horizontally or vertically between two children.
package layout

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/pkg/styles"
)

// Renderable is the interface for anything that can be placed in a layout slot.
// The tui.Pane interface satisfies this automatically.
type Renderable interface {
	View() string
	SetDimensions(width, height int)
}

// Direction of a container split.
type Direction int

const (
	Horizontal Direction = iota // children laid out left to right
	Vertical                    // children laid out top to bottom
)

// Container is a node in the layout tree.
type Container struct {
	direction Direction
	ratio     float64 // 0.0–1.0, portion of space given to the first child
	children  [2]*Container
	content   Renderable
	label     string // display name shown in the pane header
	focused   bool

	// Computed geometry from the most recent Layout pass.
	x, y, w, h int
}

// MinPaneWidth is the minimum width a pane can be sized to.
const MinPaneWidth = 20

// MinPaneHeight is the minimum height a pane can be sized to.
const MinPaneHeight = 4

// borderOverhead is the number of rows consumed by the pane header separator.
const borderOverhead = 1

// NewLeaf creates a leaf container holding a Renderable.
func NewLeaf(label string, content Renderable) *Container {
	return &Container{
		content: content,
		label:   label,
		ratio:   0.5,
	}
}

// NewSplit creates a branch container splitting space between two children.
// ratio is the fraction (0.0–1.0) of space given to the first child.
func NewSplit(dir Direction, ratio float64, first, second *Container) *Container {
	if ratio < 0.1 {
		ratio = 0.1
	}
	if ratio > 0.9 {
		ratio = 0.9
	}
	return &Container{
		direction: dir,
		ratio:     ratio,
		children:  [2]*Container{first, second},
	}
}

// IsLeaf returns true if this container holds content (no children).
func (c *Container) IsLeaf() bool {
	return c.content != nil
}

// Label returns the display label for this container.
func (c *Container) Label() string {
	return c.label
}

// SetFocused marks this leaf as the focused pane.
func (c *Container) SetFocused(focused bool) {
	c.focused = focused
}

// IsFocused returns whether this leaf is the focused pane.
func (c *Container) IsFocused() bool {
	return c.focused
}

// Content returns the Renderable for a leaf, or nil for a branch.
func (c *Container) Content() Renderable {
	return c.content
}

// Children returns the two child containers for a branch, or nil for a leaf.
func (c *Container) Children() [2]*Container {
	return c.children
}

// Geometry returns the computed x, y, width, height from the last Layout pass.
func (c *Container) Geometry() (x, y, w, h int) {
	return c.x, c.y, c.w, c.h
}

// Ratio returns the current split ratio.
func (c *Container) Ratio() float64 {
	return c.ratio
}

// SetRatio updates the split ratio, clamped to [0.1, 0.9].
func (c *Container) SetRatio(ratio float64) {
	if ratio < 0.1 {
		ratio = 0.1
	}
	if ratio > 0.9 {
		ratio = 0.9
	}
	c.ratio = ratio
}

// AdjustRatio shifts the split ratio by delta (positive = grow first child).
func (c *Container) AdjustRatio(delta float64) {
	c.SetRatio(c.ratio + delta)
}

// Direction returns the split direction.
func (c *Container) Direction() Direction {
	return c.direction
}

// Layout computes geometry for the entire tree starting at (x, y) with size (w, h).
// It recursively assigns dimensions and calls SetDimensions on each leaf's content.
func (c *Container) Layout(x, y, w, h int) {
	c.x, c.y, c.w, c.h = x, y, w, h

	if c.IsLeaf() {
		// Reserve 1 row for the pane header in multi-pane layouts.
		// Single-pane layouts skip the header (handled by the caller).
		contentH := h
		if contentH < 1 {
			contentH = 1
		}
		c.content.SetDimensions(w, contentH)
		return
	}

	first, second := c.children[0], c.children[1]
	if first == nil || second == nil {
		return
	}

	switch c.direction {
	case Horizontal:
		// Split width: 1 column for the divider
		divider := 1
		available := w - divider
		if available < MinPaneWidth*2 {
			// Not enough room — give all to first child
			first.Layout(x, y, w, h)
			second.Layout(x, y, 0, 0)
			return
		}
		firstW := int(float64(available) * c.ratio)
		if firstW < MinPaneWidth {
			firstW = MinPaneWidth
		}
		secondW := available - firstW
		if secondW < MinPaneWidth {
			secondW = MinPaneWidth
			firstW = available - secondW
		}
		first.Layout(x, y, firstW, h)
		second.Layout(x+firstW+divider, y, secondW, h)

	case Vertical:
		// Split height: 1 row for the divider
		divider := 1
		available := h - divider
		if available < MinPaneHeight*2 {
			first.Layout(x, y, w, h)
			second.Layout(x, y, 0, 0)
			return
		}
		firstH := int(float64(available) * c.ratio)
		if firstH < MinPaneHeight {
			firstH = MinPaneHeight
		}
		secondH := available - firstH
		if secondH < MinPaneHeight {
			secondH = MinPaneHeight
			firstH = available - secondH
		}
		first.Layout(x, y, w, firstH)
		second.Layout(x, y+firstH+divider, w, secondH)
	}
}

// View renders the entire layout tree into a single string.
func (c *Container) View() string {
	if c.w <= 0 || c.h <= 0 {
		return ""
	}

	if c.IsLeaf() {
		return c.renderLeaf()
	}

	first, second := c.children[0], c.children[1]
	if first == nil || second == nil {
		return ""
	}

	firstView := first.View()
	secondView := second.View()

	// If one child has no space, show only the other
	if first.w <= 0 || first.h <= 0 {
		return secondView
	}
	if second.w <= 0 || second.h <= 0 {
		return firstView
	}

	switch c.direction {
	case Horizontal:
		divider := renderVerticalDivider(c.h)
		return lipgloss.JoinHorizontal(lipgloss.Top, firstView, divider, secondView)
	case Vertical:
		divider := renderHorizontalDivider(c.w)
		return lipgloss.JoinVertical(lipgloss.Left, firstView, divider, secondView)
	}

	return ""
}

// renderLeaf renders a single pane with a header label and its content.
func (c *Container) renderLeaf() string {
	headerH := borderOverhead
	contentH := c.h - headerH
	if contentH < 1 {
		contentH = 1
	}

	// Tell the content its available space (below the header)
	c.content.SetDimensions(c.w, contentH)
	raw := c.content.View()

	// Truncate/pad content to exact dimensions
	content := fitToBox(raw, c.w, contentH)

	header := renderPaneHeader(c.label, c.w, c.focused)
	return header + "\n" + content
}

// ViewSingle renders a single leaf WITHOUT a pane header (for single-pane layout).
func (c *Container) ViewSingle() string {
	if !c.IsLeaf() {
		return c.View()
	}
	c.content.SetDimensions(c.w, c.h)
	return c.content.View()
}

// Leaves returns all leaf containers in the tree (left to right, top to bottom).
func (c *Container) Leaves() []*Container {
	if c.IsLeaf() {
		return []*Container{c}
	}
	var leaves []*Container
	if c.children[0] != nil {
		leaves = append(leaves, c.children[0].Leaves()...)
	}
	if c.children[1] != nil {
		leaves = append(leaves, c.children[1].Leaves()...)
	}
	return leaves
}

// renderPaneHeader renders a thin separator line with the pane label.
func renderPaneHeader(label string, width int, focused bool) string {
	if width <= 0 {
		return ""
	}

	color := styles.Muted
	if focused {
		color = styles.Primary
	}

	lineChar := "─"
	style := lipgloss.NewStyle().Foreground(color)

	if label == "" {
		return style.Render(strings.Repeat(lineChar, width))
	}

	labelStr := " " + label + " "
	labelWidth := lipgloss.Width(labelStr)
	remaining := width - labelWidth
	if remaining < 2 {
		return style.Render(strings.Repeat(lineChar, width))
	}

	leftLine := strings.Repeat(lineChar, 1)
	rightLine := strings.Repeat(lineChar, remaining-1)
	return style.Render(leftLine + labelStr + rightLine)
}

// renderVerticalDivider renders a single-column vertical divider.
func renderVerticalDivider(height int) string {
	if height <= 0 {
		return ""
	}
	style := lipgloss.NewStyle().Foreground(styles.Border)
	lines := make([]string, height)
	for i := range lines {
		lines[i] = style.Render("│")
	}
	return strings.Join(lines, "\n")
}

// renderHorizontalDivider renders a single-row horizontal divider.
func renderHorizontalDivider(width int) string {
	if width <= 0 {
		return ""
	}
	style := lipgloss.NewStyle().Foreground(styles.Border)
	return style.Render(strings.Repeat("─", width))
}

// fitToBox truncates or pads content to exactly width×height.
func fitToBox(content string, width, height int) string {
	lines := strings.Split(content, "\n")

	// Truncate to height
	if len(lines) > height {
		lines = lines[:height]
	}

	// Pad/truncate each line to width
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w > width {
			lines[i] = truncateLine(line, width)
		} else if w < width {
			lines[i] = line + strings.Repeat(" ", width-w)
		}
	}

	// Pad with empty lines if not enough
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return strings.Join(lines, "\n")
}

// truncateLine truncates a string (with ANSI codes) to the given visible width.
func truncateLine(s string, width int) string {
	if width <= 0 {
		return ""
	}
	// Use lipgloss/ansi truncation
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}
