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
	tea "github.com/charmbracelet/bubbletea"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/internal/history"
	"github.com/clidey/whodb/cli/pkg/styles"
)

type ViewMode int

const (
	ViewConnection ViewMode = iota
	ViewBrowser
	ViewEditor
	ViewResults
	ViewHistory
	ViewExport
	ViewWhere
	ViewColumns
	ViewChat
	ViewSchema
)

type MainModel struct {
	mode      ViewMode
	width     int
	height    int
	dbManager *database.Manager
	histMgr   *history.Manager
	config    *config.Config
	err       error

	connectionView *ConnectionView
	browserView    *BrowserView
	editorView     *EditorView
	resultsView    *ResultsView
	historyView    *HistoryView
	exportView     *ExportView
	whereView      *WhereView
	columnsView    *ColumnsView
	chatView       *ChatView
	schemaView     *SchemaView
}

func NewMainModel() *MainModel {
	dbMgr, err := database.NewManager()
	if err != nil {
		return &MainModel{err: err}
	}

	histMgr, err := history.NewManager()
	if err != nil {
		return &MainModel{err: err}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return &MainModel{err: err}
	}

	m := &MainModel{
		mode:      ViewConnection,
		dbManager: dbMgr,
		histMgr:   histMgr,
		config:    cfg,
	}

	m.connectionView = NewConnectionView(m)
	m.browserView = NewBrowserView(m)
	m.editorView = NewEditorView(m)
	m.resultsView = NewResultsView(m)
	m.historyView = NewHistoryView(m)
	m.exportView = NewExportView(m)
	m.whereView = NewWhereView(m)
	m.columnsView = NewColumnsView(m)
	m.chatView = NewChatView(m)
	m.schemaView = NewSchemaView(m)

	return m
}

func NewMainModelWithConnection(conn *config.Connection) *MainModel {
	m := NewMainModel()
	if m.err != nil {
		return m
	}

	if err := m.dbManager.Connect(conn); err != nil {
		m.err = err
		return m
	}

	m.mode = ViewBrowser
	return m
}

func (m *MainModel) Init() tea.Cmd {
	if m.mode == ViewBrowser && m.dbManager.GetCurrentConnection() != nil {
		return m.browserView.loadTables()
	}
	return nil
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.err != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "ctrl+c" || msg.String() == "q" {
				return m, tea.Quit
			}
			if msg.String() == "esc" {
				m.err = nil
				switch m.mode {
				case ViewWhere:
					m.mode = ViewResults
				case ViewConnection:
					m.mode = ViewConnection
				case ViewBrowser:
					m.mode = ViewBrowser
				case ViewEditor:
					m.mode = ViewEditor
				case ViewResults:
					m.mode = ViewResults
				case ViewHistory:
					m.mode = ViewHistory
				default:
					// todo: need a better default
					panic("oops. how did this happen? please open a ticket at https://github.com/clidey/whodb/issues and tell us how you got here :)")
				}
				return m, nil
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "tab":
			return m.handleTabSwitch()

		case "?":
			return m, nil
		}
	}

	switch m.mode {
	case ViewConnection:
		return m.updateConnectionView(msg)
	case ViewBrowser:
		return m.updateBrowserView(msg)
	case ViewEditor:
		return m.updateEditorView(msg)
	case ViewResults:
		return m.updateResultsView(msg)
	case ViewHistory:
		return m.updateHistoryView(msg)
	case ViewExport:
		return m.updateExportView(msg)
	case ViewWhere:
		return m.updateWhereView(msg)
	case ViewColumns:
		return m.updateColumnsView(msg)
	case ViewChat:
		return m.updateChatView(msg)
	case ViewSchema:
		return m.updateSchemaView(msg)
	}

	return m, nil
}

func (m *MainModel) View() string {
	if m.err != nil {
		return renderError(m.err.Error())
	}

	viewIndicator := m.renderViewIndicator()

	var content string
	switch m.mode {
	case ViewConnection:
		content = m.connectionView.View()
	case ViewBrowser:
		content = m.browserView.View()
	case ViewEditor:
		content = m.editorView.View()
	case ViewResults:
		content = m.resultsView.View()
	case ViewHistory:
		content = m.historyView.View()
	case ViewExport:
		content = m.exportView.View()
	case ViewWhere:
		content = m.whereView.View()
	case ViewColumns:
		content = m.columnsView.View()
	case ViewChat:
		content = m.chatView.View()
	case ViewSchema:
		content = m.schemaView.View()
	}

	return viewIndicator + "\n" + content
}

func (m *MainModel) handleTabSwitch() (tea.Model, tea.Cmd) {
	if m.dbManager.GetCurrentConnection() == nil {
		return m, nil
	}

	nextMode := (m.mode + 1) % 10

	if nextMode == ViewConnection {
		nextMode = ViewBrowser
	}

	// Skip export, where, columns, chat, and schema views in tab switching
	if nextMode == ViewExport || nextMode == ViewWhere || nextMode == ViewColumns || nextMode == ViewChat || nextMode == ViewSchema {
		nextMode = ViewBrowser
	}

	m.mode = nextMode
	return m, nil
}

func (m *MainModel) updateConnectionView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.connectionView, cmd = m.connectionView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateBrowserView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.browserView, cmd = m.browserView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateEditorView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.editorView, cmd = m.editorView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateResultsView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.resultsView, cmd = m.resultsView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateHistoryView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.historyView, cmd = m.historyView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateExportView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.exportView, cmd = m.exportView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateWhereView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.whereView, cmd = m.whereView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateColumnsView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.columnsView, cmd = m.columnsView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateChatView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.chatView, cmd = m.chatView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateSchemaView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.schemaView, cmd = m.schemaView.Update(msg)
	return m, cmd
}

func (m *MainModel) renderViewIndicator() string {
	views := []struct {
		mode ViewMode
		name string
	}{
		{ViewConnection, "Connection"},
		{ViewBrowser, "Browser"},
		{ViewEditor, "Editor"},
		{ViewResults, "Results"},
		{ViewHistory, "History"},
		{ViewExport, "Export"},
		{ViewWhere, "Where"},
		{ViewColumns, "Columns"},
		{ViewChat, "Chat"},
		{ViewSchema, "Schema"},
	}

	// Define which views are tab-accessible
	tabbableViews := map[ViewMode]bool{
		ViewBrowser: true,
		ViewEditor:  true,
		ViewResults: true,
		ViewHistory: true,
	}

	var parts []string
	for _, view := range views {
		if view.mode == m.mode {
			// Current view: white background with black text
			activeStyle := styles.BaseStyle.
				Foreground(styles.Background).
				Background(styles.Foreground).
				Padding(0, 1)
			parts = append(parts, activeStyle.Render(view.name))
		} else if tabbableViews[view.mode] {
			// Tab-accessible views: normal white text
			inactiveStyle := styles.BaseStyle.
				Foreground(styles.Foreground).
				Padding(0, 1)
			parts = append(parts, inactiveStyle.Render(view.name))
		} else {
			// Non-tabbable views: dimmed gray text
			dimmedStyle := styles.BaseStyle.
				Foreground(styles.Muted).
				Padding(0, 1)
			parts = append(parts, dimmedStyle.Render(view.name))
		}
	}

	// Join all parts with a separator
	separator := " "
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += separator
		}
		result += part
	}

	return result
}

func renderError(message string) string {
	errorBox := styles.RenderErrorBox(message)
	helpText := styles.RenderHelp("esc", "dismiss", "ctrl+c", "exit")
	return "\n" + errorBox + "\n\n" + helpText + "\n"
}
