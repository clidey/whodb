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
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/pkg/styles"
	"github.com/clidey/whodb/core/src/engine"
)

type BrowserView struct {
	parent              *MainModel
	tables              []engine.StorageUnit
	selectedIndex       int
	selectedTable       string
	currentSchema       string
	schemas             []string
	selectedSchemaIndex int
	schemaSelecting     bool
	loading             bool
	err                 error
	width               int
	height              int
	columnsPerRow       int
	filterInput         textinput.Model
	filtering           bool
	filteredTables      []engine.StorageUnit
	retryPrompt         RetryPrompt
}

func NewBrowserView(parent *MainModel) *BrowserView {
	ti := textinput.New()
	ti.Placeholder = "Search tables..."
	ti.CharLimit = 50
	ti.Width = 30
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.Foreground)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	return &BrowserView{
		parent:        parent,
		loading:       true,
		selectedIndex: 0,
		columnsPerRow: 4,
		width:         80,
		height:        20,
		filterInput:   ti,
		filtering:     false,
	}
}

func (v *BrowserView) Update(msg tea.Msg) (*BrowserView, tea.Cmd) {
	switch msg := msg.(type) {
	case tablesLoadedMsg:
		v.loading = false
		if msg.err != nil {
			// Check for timeout - auto-retry with saved preference or show menu
			if strings.Contains(msg.err.Error(), "timed out") {
				preferred := v.parent.config.GetPreferredTimeout()
				if preferred > 0 && !v.retryPrompt.AutoRetried() {
					v.retryPrompt.SetAutoRetried(true)
					v.loading = true
					return v, v.loadTablesWithTimeout(time.Duration(preferred) * time.Second)
				}
				v.err = msg.err
				v.retryPrompt.Show("")
				return v, nil
			}
			v.err = msg.err
			return v, nil
		}
		v.tables = msg.tables
		v.schemas = msg.schemas
		v.currentSchema = msg.schema
		v.applyFilter()
		v.selectedIndex = 0

		// Find the index of currentSchema
		for i, s := range v.schemas {
			if s == v.currentSchema {
				v.selectedSchemaIndex = i
				break
			}
		}
		return v, nil

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		columnWidth := 25
		available := msg.Width - 8
		v.columnsPerRow = clamp(available/columnWidth, 1, 6)
		v.filterInput.Width = clamp(msg.Width-20, 15, 50)
		return v, nil

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if v.selectedIndex >= v.columnsPerRow {
				v.selectedIndex -= v.columnsPerRow
			}
			return v, nil
		case tea.MouseButtonWheelDown:
			if v.selectedIndex+v.columnsPerRow < len(v.filteredTables) {
				v.selectedIndex += v.columnsPerRow
			}
			return v, nil
		case tea.MouseButtonLeft:
			if !v.loading && !v.retryPrompt.IsActive() && len(v.filteredTables) > 0 {
				// Calculate header lines to determine where the grid starts
				headerLines := 4 // title + schema + filter area + blank
				if len(v.schemas) > 0 {
					headerLines++
				}
				if v.filtering || v.filterInput.Value() != "" {
					headerLines++
				}
				gridRow := msg.Y - headerLines
				gridCol := msg.X / 25 // columnWidth = 25
				if gridRow >= 0 && gridCol >= 0 && gridCol < v.columnsPerRow {
					idx := gridRow*v.columnsPerRow + gridCol
					if idx >= 0 && idx < len(v.filteredTables) {
						v.selectedIndex = idx
					}
				}
			}
			return v, nil
		}

	case tea.KeyMsg:
		// Handle retry prompt for timed out requests
		if v.retryPrompt.IsActive() {
			result, handled := v.retryPrompt.HandleKeyMsg(msg.String())
			if handled {
				if result != nil {
					v.err = nil
					v.loading = true
					if result.Save {
						v.parent.config.SetPreferredTimeout(int(result.Timeout.Seconds()))
						v.parent.config.Save()
					}
					return v, v.loadTablesWithTimeout(result.Timeout)
				}
				return v, nil
			}
		}

		// If in filtering mode, handle filter input
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

		// If in schema selecting mode
		if v.schemaSelecting {
			switch msg.String() {
			case "esc":
				v.schemaSelecting = false
				return v, nil
			case "left", "up", "h", "k":
				if v.selectedSchemaIndex > 0 {
					v.selectedSchemaIndex--
				}
				return v, nil
			case "right", "down", "l", "j":
				if v.selectedSchemaIndex < len(v.schemas)-1 {
					v.selectedSchemaIndex++
				}
				return v, nil
			case "enter":
				if v.selectedSchemaIndex >= 0 && v.selectedSchemaIndex < len(v.schemas) {
					v.currentSchema = v.schemas[v.selectedSchemaIndex]
					v.schemaSelecting = false
					v.loading = true
					v.err = nil
					return v, v.loadTables()
				}
				return v, nil
			default:
				// Ignore other keys while in schema selection mode
				// (global shortcuts like ctrl+c are handled at MainModel level)
				return v, nil
			}
		}

		// Normal navigation mode
		switch {
		case key.Matches(msg, Keys.Browser.Select):
			if v.selectedIndex >= 0 && v.selectedIndex < len(v.filteredTables) {
				table := v.filteredTables[v.selectedIndex]
				v.selectedTable = table.Name
				v.parent.resultsView.LoadTable(v.currentSchema, table.Name)
				v.parent.PushView(ViewResults)
				return v, nil
			}

		case key.Matches(msg, Keys.Browser.Up):
			if v.selectedIndex >= v.columnsPerRow {
				v.selectedIndex -= v.columnsPerRow
			}
			return v, nil

		case key.Matches(msg, Keys.Browser.Down):
			if v.selectedIndex+v.columnsPerRow < len(v.filteredTables) {
				v.selectedIndex += v.columnsPerRow
			}
			return v, nil

		case key.Matches(msg, Keys.Browser.Left):
			if v.selectedIndex > 0 {
				v.selectedIndex--
			}
			return v, nil

		case key.Matches(msg, Keys.Browser.Right):
			if v.selectedIndex < len(v.filteredTables)-1 {
				v.selectedIndex++
			}
			return v, nil

		case key.Matches(msg, Keys.Browser.Filter):
			// Enter filter mode
			v.filtering = true
			v.filterInput.Focus()
			return v, nil

		case key.Matches(msg, Keys.Browser.Schema):
			// Enter schema selection mode
			if len(v.schemas) > 1 {
				v.schemaSelecting = true
			}
			return v, nil

		case key.Matches(msg, Keys.Browser.Refresh):
			v.loading = true
			v.err = nil
			v.filterInput.SetValue("")
			v.filtering = false
			return v, v.loadTables()

		case key.Matches(msg, Keys.Browser.Editor):
			v.parent.PushView(ViewEditor)
			return v, nil

		case key.Matches(msg, Keys.Browser.History):
			v.parent.PushView(ViewHistory)
			return v, nil

		case key.Matches(msg, Keys.Browser.AIChat):
			v.parent.PushView(ViewChat)
			return v, v.parent.chatView.Init()

		case key.Matches(msg, Keys.Browser.Disconnect):
			if v.parent.dbManager.GetCurrentConnection() != nil {
				v.parent.mode = ViewConnection
				v.parent.dbManager.Disconnect()
				v.parent.viewHistory = nil
			}
			return v, nil
		}
	}

	return v, nil
}

func (v *BrowserView) View() string {
	conn := v.parent.dbManager.GetCurrentConnection()
	if conn == nil {
		return "No connection"
	}

	var b strings.Builder

	title := fmt.Sprintf("Connected to: %s@%s/%s", conn.Type, conn.Host, conn.Database)
	if conn.Name != "" {
		title = fmt.Sprintf("Connected to: %s (%s@%s/%s)", conn.Name, conn.Type, conn.Host, conn.Database)
	}
	b.WriteString(styles.RenderTitle(title))
	b.WriteString("\n")

	// Show schema selector
	if len(v.schemas) > 0 {
		schemaLabel := "Schema: "
		if v.schemaSelecting {
			schemaLabel = styles.KeyStyle.Render("▶ Schema: ")
		} else {
			schemaLabel = "  Schema: "
		}
		b.WriteString(schemaLabel)

		if v.schemaSelecting {
			// Show all schemas when selecting
			for i, schema := range v.schemas {
				if i == v.selectedSchemaIndex {
					b.WriteString(styles.ActiveListItemStyle.Render(fmt.Sprintf(" %s ", schema)))
				} else {
					b.WriteString(styles.MutedStyle.Render(fmt.Sprintf(" %s ", schema)))
				}
				if i < len(v.schemas)-1 {
					b.WriteString(" ")
				}
			}
		} else {
			// Show only current schema when not selecting
			b.WriteString(styles.ActiveListItemStyle.Render(fmt.Sprintf(" %s ", v.currentSchema)))
			if len(v.schemas) > 1 {
				b.WriteString(styles.MutedStyle.Render(fmt.Sprintf(" +%d more", len(v.schemas)-1)))
			}
		}
		b.WriteString("\n")
	}

	// Show filter input if filtering or if there's a filter applied
	if v.filtering || v.filterInput.Value() != "" {
		filterLabel := "Filter: "
		if v.filtering {
			filterLabel = styles.KeyStyle.Render("Filter: ")
		} else {
			filterLabel = styles.MutedStyle.Render("Filter: ")
		}
		b.WriteString(filterLabel)
		b.WriteString(v.filterInput.View())
		if !v.filtering && v.filterInput.Value() != "" {
			b.WriteString(" ")
			b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("(%d/%d)", len(v.filteredTables), len(v.tables))))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Show retry prompt for timed out requests
	if v.retryPrompt.IsActive() {
		b.WriteString(v.retryPrompt.View())
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	if v.err != nil {
		b.WriteString(styles.RenderErrorBox(v.err.Error()))
		b.WriteString("\n\n")
		b.WriteString(styles.MutedStyle.Render("Press 'r' to retry"))
	} else if v.loading {
		b.WriteString(v.parent.SpinnerView() + styles.MutedStyle.Render(" Loading tables..."))
	} else if len(v.filteredTables) == 0 {
		b.WriteString(styles.MutedStyle.Render("No tables found in this database."))
		b.WriteString("\n")
		b.WriteString(styles.MutedStyle.Render("Press 'r' to refresh or 'e' to run SQL queries."))
	} else {
		b.WriteString(v.renderTablesGrid())
	}

	b.WriteString("\n\n")

	if v.schemaSelecting {
		b.WriteString(RenderBindingHelp(
			Keys.SchemaSelect.NavLeft,
			Keys.SchemaSelect.SelectSchema,
			Keys.Global.Back,
		))
	} else if v.filtering {
		b.WriteString(RenderBindingHelp(
			Keys.Filter.CancelFilter,
			Keys.Filter.ApplyFilter,
		))
	} else {
		bindings := []key.Binding{
			Keys.Browser.Up,
			Keys.Browser.Down,
			Keys.Browser.Left,
			Keys.Browser.Right,
			Keys.Browser.Select,
			Keys.Browser.Filter,
		}
		if len(v.schemas) > 1 {
			bindings = append(bindings, Keys.Browser.Schema)
		}
		bindings = append(bindings,
			Keys.Browser.Editor,
			Keys.Browser.AIChat,
			Keys.Browser.History,
			Keys.Browser.Refresh,
			Keys.Global.NextView,
			Keys.Browser.Disconnect,
			Keys.Global.Quit,
		)
		b.WriteString(RenderBindingHelp(bindings...))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (v *BrowserView) renderTablesGrid() string {
	var b strings.Builder

	columnWidth := 25
	tables := v.filteredTables

	for i := 0; i < len(tables); i++ {
		colIndex := i % v.columnsPerRow

		if colIndex == 0 && i > 0 {
			b.WriteString("\n")
		}

		table := tables[i]
		tableName := table.Name
		if lipgloss.Width(tableName) > columnWidth-3 {
			runes := []rune(tableName)
			for lipgloss.Width(string(runes)) > columnWidth-6 {
				runes = runes[:len(runes)-1]
			}
			tableName = string(runes) + "..."
		}

		// Pad to column width
		padding := columnWidth - lipgloss.Width(tableName)
		if padding < 0 {
			padding = 0
		}
		content := tableName + strings.Repeat(" ", padding)

		if i == v.selectedIndex {
			// Selected item
			b.WriteString(styles.ActiveListItemStyle.Render(content))
		} else {
			// Normal item
			b.WriteString(styles.ListItemStyle.Render(content))
		}
	}

	// Show total count
	b.WriteString("\n\n")
	b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("Total: %d tables", len(tables))))

	return b.String()
}

func (v *BrowserView) loadTables() tea.Cmd {
	return v.loadTablesWithTimeout(v.parent.config.GetQueryTimeout())
}

func (v *BrowserView) loadTablesWithTimeout(timeout time.Duration) tea.Cmd {
	// Capture values needed for closure
	currentSchema := v.currentSchema
	conn := v.parent.dbManager.GetCurrentConnection()

	return func() tea.Msg {
		if conn == nil {
			return tablesLoadedMsg{
				tables:  []engine.StorageUnit{},
				schemas: []string{},
				schema:  "",
				err:     fmt.Errorf("no connection"),
			}
		}

		// Create context with timeout for database operations
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		schemas, err := v.parent.dbManager.GetSchemasWithContext(ctx)
		if err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return tablesLoadedMsg{
					tables:  []engine.StorageUnit{},
					schemas: []string{},
					schema:  currentSchema,
					err:     fmt.Errorf("timed out fetching schemas"),
				}
			}
			return tablesLoadedMsg{
				tables:  []engine.StorageUnit{},
				schemas: []string{},
				schema:  currentSchema,
				err:     fmt.Errorf("failed to get schemas: %w", err),
			}
		}

		if len(schemas) == 0 {
			return tablesLoadedMsg{
				tables:  []engine.StorageUnit{},
				schemas: []string{},
				schema:  "",
				err:     nil,
			}
		}

		// Use currentSchema if already set, otherwise check for preferred schema from connection
		schema := currentSchema
		if schema == "" {
			// Check if connection has a preferred schema
			if conn.Schema != "" {
				// Try to use the preferred schema if it exists
				schemaExists := false
				for _, s := range schemas {
					if s == conn.Schema {
						schemaExists = true
						schema = conn.Schema
						break
					}
				}
				if !schemaExists {
					// Fall back to auto-selection if preferred schema doesn't exist
					schema = selectBestSchema(schemas)
				}
			} else {
				schema = selectBestSchema(schemas)
			}
		}

		// Validate that currentSchema exists in the list
		schemaExists := false
		for _, s := range schemas {
			if s == schema {
				schemaExists = true
				break
			}
		}
		if !schemaExists && len(schemas) > 0 {
			schema = selectBestSchema(schemas)
		}

		units, err := v.parent.dbManager.GetStorageUnitsWithContext(ctx, schema)
		if err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return tablesLoadedMsg{
					tables:  []engine.StorageUnit{},
					schemas: schemas,
					schema:  schema,
					err:     fmt.Errorf("timed out fetching tables"),
				}
			}
			return tablesLoadedMsg{
				tables:  []engine.StorageUnit{},
				schemas: schemas,
				schema:  schema,
				err:     fmt.Errorf("failed to get tables: %w", err),
			}
		}

		return tablesLoadedMsg{
			tables:  units,
			schemas: schemas,
			schema:  schema,
			err:     nil,
		}
	}
}

func (v *BrowserView) Init() tea.Cmd {
	v.retryPrompt.SetAutoRetried(false)
	return v.loadTables()
}

func (v *BrowserView) applyFilter() {
	filterText := strings.ToLower(v.filterInput.Value())

	if filterText == "" {
		v.filteredTables = v.tables
		return
	}

	v.filteredTables = []engine.StorageUnit{}
	for _, table := range v.tables {
		if strings.Contains(strings.ToLower(table.Name), filterText) {
			v.filteredTables = append(v.filteredTables, table)
		}
	}

	// Reset selected index if it's out of bounds
	if v.selectedIndex >= len(v.filteredTables) {
		v.selectedIndex = 0
	}
}

func selectBestSchema(schemas []string) string {
	if len(schemas) == 0 {
		return ""
	}

	systemSchemas := map[string]bool{
		"information_schema": true,
		"pg_catalog":         true,
		"pg_toast":           true,
		"mysql":              true,
		"sys":                true,
		"performance_schema": true,
	}

	userSchemas := []string{}
	for _, s := range schemas {
		if !systemSchemas[s] && !strings.HasPrefix(s, "pg_temp_") && !strings.HasPrefix(s, "pg_toast_temp_") {
			userSchemas = append(userSchemas, s)
		}
	}

	if len(userSchemas) == 0 {
		return schemas[0]
	}

	for _, s := range userSchemas {
		if s == "public" {
			return s
		}
	}

	return userSchemas[0]
}
