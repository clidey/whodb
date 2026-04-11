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
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/pkg/styles"
)

// costPattern matches Postgres-style cost annotations like "cost=0.00..35.50".
var costPattern = regexp.MustCompile(`cost=[\d.]+\.\.([\d.]+)`)

// ExplainView displays the output of an EXPLAIN (ANALYZE) query in a
// scrollable, color-coded viewport.
type ExplainView struct {
	parent   *MainModel
	viewport viewport.Model
	query    string
	plan     string
	width    int
	height   int
	ready    bool
}

// NewExplainView creates a new ExplainView attached to the given parent model.
func NewExplainView(parent *MainModel) *ExplainView {
	return &ExplainView{
		parent: parent,
	}
}

// Update handles input for the explain view.
func (v *ExplainView) Update(msg tea.Msg) (*ExplainView, tea.Cmd) {
	switch msg := msg.(type) {
	case explainResultMsg:
		if msg.err != nil {
			v.SetPlan(msg.query, "Error: "+msg.err.Error())
		} else {
			v.SetPlan(msg.query, msg.plan)
		}
		return v, nil

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.initViewport()
		return v, nil

	case tea.KeyMsg:
		if key.Matches(msg, Keys.Global.Back) {
			if !v.parent.PopView() {
				v.parent.mode = ViewEditor
			}
			return v, nil
		}
	}

	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// View renders the explain view.
func (v *ExplainView) View() string {
	if !v.ready {
		v.initViewport()
	}

	var b strings.Builder

	b.WriteString(styles.RenderTitle("Query Plan"))
	b.WriteString("\n\n")

	// Show the query being explained
	if v.query != "" {
		truncated := v.query
		maxLen := v.width - 12
		if maxLen < 20 {
			maxLen = 20
		}
		if len(truncated) > maxLen {
			truncated = truncated[:maxLen] + "..."
		}
		b.WriteString(styles.RenderMuted(truncated))
		b.WriteString("\n\n")
	}

	b.WriteString(v.viewport.View())
	b.WriteString("\n\n")

	b.WriteString(RenderBindingHelpWidth(v.width,
		key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("up/dn", "scroll")),
		Keys.Global.Back,
	))

	if v.viewport.TotalLineCount() > v.viewport.VisibleLineCount() {
		pct := v.viewport.ScrollPercent() * 100
		var scrollPct string
		if pct >= 99.5 {
			scrollPct = "bottom"
		} else {
			scrollPct = formatFloat(pct) + "%"
		}
		b.WriteString("  ")
		b.WriteString(styles.RenderMuted(scrollPct))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

// SetPlan stores the explain output and resets the viewport so it is
// rebuilt on the next render pass.
func (v *ExplainView) SetPlan(query, plan string) {
	v.query = query
	v.plan = plan
	v.ready = false
}

// initViewport creates (or resizes) the viewport and fills it with the
// color-coded plan text.
func (v *ExplainView) initViewport() {
	contentWidth := v.width - 8
	if contentWidth < 20 {
		contentWidth = 20
	}
	// Reserve space for title, query line, help footer, and padding
	contentHeight := v.height - 14
	if contentHeight < 3 {
		contentHeight = 3
	}

	v.viewport = viewport.New(contentWidth, contentHeight)
	v.viewport.SetContent(colorizePlan(v.plan))
	v.ready = true
}

// colorizePlan applies color coding to EXPLAIN output. Lines containing a
// Postgres-style cost annotation are colored green/yellow/red based on the
// upper cost bound. Lines without cost information are rendered with the
// default foreground color.
func colorizePlan(plan string) string {
	if strings.TrimSpace(plan) == "" {
		return styles.MutedStyle.Render("(no plan output)")
	}

	lines := strings.Split(plan, "\n")
	var out strings.Builder
	for i, line := range lines {
		if i > 0 {
			out.WriteString("\n")
		}
		out.WriteString(colorizePlanLine(line))
	}
	return out.String()
}

// colorizePlanLine colors a single line of EXPLAIN output. If the line
// contains a cost=...  annotation, the color is determined by the upper cost
// bound: <100 green, 100-1000 yellow, >1000 red.
func colorizePlanLine(line string) string {
	matches := costPattern.FindStringSubmatch(line)
	if len(matches) < 2 {
		return line
	}

	cost, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return line
	}

	var color lipgloss.AdaptiveColor
	switch {
	case cost < 100:
		color = styles.Success
	case cost <= 1000:
		color = styles.Warning
	default:
		color = styles.Error
	}

	return lipgloss.NewStyle().Foreground(color).Render(line)
}
