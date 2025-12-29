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

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/pkg/styles"
	"github.com/clidey/whodb/core/src/engine"
)

type tablesLoadedMsg struct {
	tables  []engine.StorageUnit
	schemas []string
	schema  string // The selected schema
	err     error
}

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
	// Retry prompt state for timed out requests
	retryPrompt bool
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
			// Check for timeout - enable retry prompt
			if strings.Contains(msg.err.Error(), "timed out") {
				v.err = msg.err
				v.retryPrompt = true
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
		}

	case tea.KeyMsg:
		// Handle retry prompt for timed out requests
		if v.retryPrompt {
			switch msg.String() {
			case "1":
				v.retryPrompt = false
				v.err = nil
				v.loading = true
				return v, v.loadTablesWithTimeout(60 * time.Second)
			case "2":
				v.retryPrompt = false
				v.err = nil
				v.loading = true
				return v, v.loadTablesWithTimeout(2 * time.Minute)
			case "3":
				v.retryPrompt = false
				v.err = nil
				v.loading = true
				return v, v.loadTablesWithTimeout(5 * time.Minute)
			case "4":
				v.retryPrompt = false
				v.err = nil
				v.loading = true
				return v, v.loadTablesWithTimeout(24 * time.Hour)
			case "esc":
				v.retryPrompt = false
				return v, nil
			}
			// Ignore other keys while in retry prompt
			return v, nil
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
		switch msg.String() {
		case "enter":
			if v.selectedIndex >= 0 && v.selectedIndex < len(v.filteredTables) {
				table := v.filteredTables[v.selectedIndex]
				v.selectedTable = table.Name
				v.parent.resultsView.LoadTable(v.currentSchema, table.Name)
				v.parent.mode = ViewResults
				return v, nil
			}

		case "up", "k":
			if v.selectedIndex >= v.columnsPerRow {
				v.selectedIndex -= v.columnsPerRow
			}
			return v, nil

		case "down", "j":
			if v.selectedIndex+v.columnsPerRow < len(v.filteredTables) {
				v.selectedIndex += v.columnsPerRow
			}
			return v, nil

		case "left", "h":
			if v.selectedIndex > 0 {
				v.selectedIndex--
			}
			return v, nil

		case "right", "l":
			if v.selectedIndex < len(v.filteredTables)-1 {
				v.selectedIndex++
			}
			return v, nil

		case "/", "f":
			// Enter filter mode
			v.filtering = true
			v.filterInput.Focus()
			return v, nil

		case "ctrl+s":
			// Enter schema selection mode
			if len(v.schemas) > 1 {
				v.schemaSelecting = true
			}
			return v, nil

		case "ctrl+r":
			v.loading = true
			v.err = nil
			v.filterInput.SetValue("")
			v.filtering = false
			return v, v.loadTables()

		case "ctrl+e":
			v.parent.mode = ViewEditor
			return v, nil

		case "ctrl+h":
			v.parent.mode = ViewHistory
			return v, nil

		case "ctrl+a":
			v.parent.mode = ViewChat
			return v, v.parent.chatView.Init()

		case "esc":
			if v.parent.dbManager.GetCurrentConnection() != nil {
				v.parent.mode = ViewConnection
				v.parent.dbManager.Disconnect()
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
	if v.retryPrompt {
		b.WriteString(styles.ErrorStyle.Render("Request timed out"))
		b.WriteString("\n\n")
		b.WriteString(styles.MutedStyle.Render("Retry with longer timeout:"))
		b.WriteString("\n")
		b.WriteString(styles.KeyStyle.Render("[1]"))
		b.WriteString(styles.MutedStyle.Render(" 60 seconds  "))
		b.WriteString(styles.KeyStyle.Render("[2]"))
		b.WriteString(styles.MutedStyle.Render(" 2 minutes  "))
		b.WriteString(styles.KeyStyle.Render("[3]"))
		b.WriteString(styles.MutedStyle.Render(" 5 minutes  "))
		b.WriteString(styles.KeyStyle.Render("[4]"))
		b.WriteString(styles.MutedStyle.Render(" No limit"))
		b.WriteString("\n\n")
		b.WriteString(styles.RenderHelp("esc", "cancel"))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	if v.err != nil {
		b.WriteString(styles.RenderErrorBox(v.err.Error()))
		b.WriteString("\n\n")
		b.WriteString(styles.MutedStyle.Render("Press 'r' to retry"))
	} else if v.loading {
		b.WriteString(styles.MutedStyle.Render("Loading tables..."))
	} else if len(v.filteredTables) == 0 {
		b.WriteString(styles.MutedStyle.Render("No tables found in this database."))
		b.WriteString("\n")
		b.WriteString(styles.MutedStyle.Render("Press 'r' to refresh or 'e' to run SQL queries."))
	} else {
		b.WriteString(v.renderTablesGrid())
	}

	b.WriteString("\n\n")

	if v.schemaSelecting {
		b.WriteString(styles.RenderHelp(
			"←/→", "navigate",
			"enter", "select schema",
			"esc", "cancel",
		))
	} else if v.filtering {
		b.WriteString(styles.RenderHelp(
			"esc", "cancel filter",
			"enter", "apply filter",
		))
	} else {
		helpItems := []string{
			"↑/k", "up",
			"↓/j", "down",
			"←/h", "left",
			"→/l", "right",
			"enter", "view data",
			"[/]", "filter",
		}
		if len(v.schemas) > 1 {
			helpItems = append(helpItems, "ctrl+s", "schema")
		}
		helpItems = append(helpItems,
			"ctrl+e", "editor",
			"ctrl+a", "ai chat",
			"ctrl+h", "history",
			"ctrl+r", "refresh",
			"tab", "next view",
			"esc", "disconnect",
			"ctrl+c", "quit",
		)
		b.WriteString(styles.RenderHelp(helpItems...))
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
		if len(tableName) > columnWidth-3 {
			tableName = tableName[:columnWidth-6] + "..."
		}

		// Pad to column width
		padding := columnWidth - len(tableName)
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
