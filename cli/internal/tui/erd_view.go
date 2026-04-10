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
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/pkg/styles"
)

// erdDataLoadedMsg is sent when the ERD data (tables + columns) has been loaded.
type erdDataLoadedMsg struct {
	tables []tableWithColumns
	err    error
	schema string
}

// ERDView renders an entity-relationship diagram using Unicode box-drawing characters.
// It shows tables, their columns, and foreign key annotations. Accessible via Ctrl+K.
type ERDView struct {
	parent       *MainModel
	tables       []tableWithColumns
	loading      bool
	err          error
	compact      bool
	focusedIndex int
	viewport     viewport.Model
	width        int
	height       int
	ready        bool
	schema       string
}

// NewERDView creates a new ERDView attached to the given parent model.
func NewERDView(parent *MainModel) *ERDView {
	return &ERDView{
		parent: parent,
		width:  80,
		height: 20,
	}
}

// Update handles input and messages for the ERD view.
func (v *ERDView) Update(msg tea.Msg) (*ERDView, tea.Cmd) {
	switch msg := msg.(type) {
	case erdDataLoadedMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.tables = msg.tables
		v.schema = msg.schema
		v.focusedIndex = 0
		v.rebuildViewport()
		return v, nil

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		if !v.loading && v.err == nil {
			v.rebuildViewport()
		}
		return v, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, Keys.Global.Back):
			if !v.parent.PopView() {
				v.parent.mode = ViewBrowser
			}
			return v, nil

		case key.Matches(msg, Keys.ERD.ToggleZoom):
			v.compact = !v.compact
			v.rebuildViewport()
			return v, nil

		case key.Matches(msg, Keys.ERD.NextTable):
			if len(v.tables) > 0 {
				v.focusedIndex = (v.focusedIndex + 1) % len(v.tables)
				v.rebuildViewport()
			}
			return v, nil

		case key.Matches(msg, Keys.ERD.PrevTable):
			if len(v.tables) > 0 {
				v.focusedIndex--
				if v.focusedIndex < 0 {
					v.focusedIndex = len(v.tables) - 1
				}
				v.rebuildViewport()
			}
			return v, nil
		}
	}

	// Forward remaining messages (arrow keys, etc.) to the viewport for scrolling
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// View renders the ERD view.
func (v *ERDView) View() string {
	var b strings.Builder

	title := "ER Diagram"
	if v.schema != "" {
		title += " — " + v.schema
	}
	b.WriteString(styles.RenderTitle(title))
	b.WriteString("\n\n")

	if v.err != nil {
		b.WriteString(styles.RenderErrorBox(v.err.Error()))
		b.WriteString("\n\n")
		b.WriteString(RenderBindingHelpWidth(v.width, Keys.Global.Back))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	if v.loading {
		b.WriteString(v.parent.SpinnerView() + styles.RenderMuted(" Loading tables..."))
		b.WriteString("\n\n")
		b.WriteString(RenderBindingHelpWidth(v.width, Keys.Global.Back))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	if len(v.tables) == 0 {
		b.WriteString(styles.RenderMuted("No tables found."))
		b.WriteString("\n\n")
		b.WriteString(RenderBindingHelpWidth(v.width, Keys.Global.Back))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	// Table summary line
	focusedName := ""
	if v.focusedIndex >= 0 && v.focusedIndex < len(v.tables) {
		focusedName = v.tables[v.focusedIndex].StorageUnit.Name
	}
	modeLabel := "expanded"
	if v.compact {
		modeLabel = "compact"
	}
	summary := fmt.Sprintf("%d tables (%s) — focused: %s", len(v.tables), modeLabel, focusedName)
	b.WriteString(styles.RenderMuted(summary))
	b.WriteString("\n\n")

	// Viewport with the diagram
	if !v.ready {
		v.rebuildViewport()
	}
	b.WriteString(v.viewport.View())
	b.WriteString("\n\n")

	// Help bar
	b.WriteString(RenderBindingHelpWidth(v.width,
		Keys.ERD.NextTable,
		Keys.ERD.PrevTable,
		Keys.ERD.ToggleZoom,
		Keys.ERD.ScrollUp,
		Keys.ERD.ScrollDown,
		Keys.Global.Back,
	))

	// Scroll percentage
	if v.viewport.TotalLineCount() > v.viewport.VisibleLineCount() {
		pct := v.viewport.ScrollPercent() * 100
		var scrollPct string
		if pct >= 99.5 {
			scrollPct = "bottom"
		} else {
			scrollPct = fmt.Sprintf("%.0f%%", pct)
		}
		b.WriteString("  ")
		b.WriteString(styles.RenderMuted(scrollPct))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

// loadERDData returns a tea.Cmd that fetches all tables and their columns.
func (v *ERDView) loadERDData() tea.Cmd {
	browserSchema := v.parent.browserView.currentSchema

	return func() tea.Msg {
		conn := v.parent.dbManager.GetCurrentConnection()
		if conn == nil {
			return erdDataLoadedMsg{err: fmt.Errorf("no connection")}
		}

		schema := browserSchema
		if schema == "" {
			schemas, err := v.parent.dbManager.GetSchemas()
			if err != nil {
				schemas = []string{}
			}
			if len(schemas) > 0 {
				schema = selectBestSchema(schemas)
			}
		}

		units, err := v.parent.dbManager.GetStorageUnits(schema)
		if err != nil {
			return erdDataLoadedMsg{err: fmt.Errorf("failed to get tables: %w", err)}
		}

		var tables []tableWithColumns
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

		return erdDataLoadedMsg{tables: tables, schema: schema}
	}
}

// rebuildViewport re-renders the diagram and sets it as the viewport content.
func (v *ERDView) rebuildViewport() {
	contentWidth := v.width - 8
	if contentWidth < 20 {
		contentWidth = 20
	}
	contentHeight := v.height - 14
	if contentHeight < 3 {
		contentHeight = 3
	}

	boxes, _, _ := layoutERDGrid(v.tables, contentWidth, v.compact, v.focusedIndex)
	canvas := renderERDCanvas(boxes)

	v.viewport = viewport.New(contentWidth, contentHeight)
	v.viewport.SetContent(canvas)
	v.ready = true
}
