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
	"time"

	"github.com/charmbracelet/bubbles/spinner"
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
	mode          ViewMode
	width         int
	height        int
	dbManager     *database.Manager
	histMgr       *history.Manager
	config        *config.Config
	err           error
	showingHelp   bool
	spinner       spinner.Model
	statusMessage string
	viewHistory   []ViewMode

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

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.MutedStyle

	m := &MainModel{
		mode:      ViewConnection,
		dbManager: dbMgr,
		histMgr:   histMgr,
		config:    cfg,
		spinner:   s,
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
	if m.err != nil {
		return nil
	}
	cmds := []tea.Cmd{m.spinner.Tick}
	if m.mode == ViewBrowser && m.dbManager.GetCurrentConnection() != nil {
		cmds = append(cmds, m.browserView.loadTables())
	}
	return tea.Batch(cmds...)
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
				case ViewExport, ViewColumns, ViewChat, ViewSchema:
					m.mode = ViewBrowser
				}
				return m, nil
			}
		}
		return m, nil
	}

	// If showing help, any key dismisses it
	if m.showingHelp {
		if _, ok := msg.(tea.KeyMsg); ok {
			m.showingHelp = false
			return m, nil
		}
	}

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case statusMessageTimeoutMsg:
		m.statusMessage = ""
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Forward to all views so inactive views have correct dimensions when switched to
		m.connectionView, _ = m.connectionView.Update(msg)
		m.browserView, _ = m.browserView.Update(msg)
		m.editorView, _ = m.editorView.Update(msg)
		m.resultsView, _ = m.resultsView.Update(msg)
		m.historyView, _ = m.historyView.Update(msg)
		m.columnsView, _ = m.columnsView.Update(msg)
		m.chatView, _ = m.chatView.Update(msg)
		m.schemaView, _ = m.schemaView.Update(msg)
		m.exportView, _ = m.exportView.Update(msg)
		m.whereView, _ = m.whereView.Update(msg)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "?":
			// Show help for views without active text input
			if m.isHelpSafe() {
				m.showingHelp = true
				return m, nil
			}
			// Otherwise fall through to let view handle it

		case "tab", "shift+tab":
			// Let connection view handle Tab for its own navigation
			if m.mode == ViewConnection {
				return m.updateConnectionView(msg)
			}
			if msg.String() == "tab" {
				return m.handleTabSwitch()
			}
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

	// Show help overlay if active
	if m.showingHelp {
		return m.renderHelpOverlay()
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

	// Add status bar between view indicator and content (only when connected)
	statusBar := m.renderStatusBar()
	if statusBar != "" {
		return viewIndicator + "\n" + statusBar + "\n" + content
	}

	return viewIndicator + "\n" + content
}

// SetStatus sets a transient status message that auto-dismisses after 3 seconds.
func (m *MainModel) SetStatus(msg string) tea.Cmd {
	m.statusMessage = msg
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return statusMessageTimeoutMsg{}
	})
}

// isLoading returns true if any view is currently performing an async operation.
func (m *MainModel) isLoading() bool {
	return m.browserView.loading ||
		m.editorView.queryState == OperationRunning ||
		m.chatView.sending ||
		m.chatView.loadingModels ||
		m.exportView.exporting ||
		m.schemaView.loading ||
		m.historyView.executing ||
		m.connectionView.connecting ||
		m.resultsView.loading
}

// renderStatusBar renders the persistent status bar shown when connected.
func (m *MainModel) renderStatusBar() string {
	if m.mode == ViewConnection {
		return ""
	}

	conn := m.dbManager.GetCurrentConnection()
	if conn == nil {
		return ""
	}

	var parts []string

	// Connection info
	connInfo := fmt.Sprintf("%s@%s/%s", conn.Type, conn.Host, conn.Database)
	parts = append(parts, styles.MutedStyle.Render(connInfo))

	// Current schema
	if m.browserView.currentSchema != "" {
		parts = append(parts, styles.MutedStyle.Render("schema:"+m.browserView.currentSchema))
	}

	// Spinner when loading
	if m.isLoading() {
		parts = append(parts, m.spinner.View())
	}

	// Transient status message
	if m.statusMessage != "" {
		parts = append(parts, styles.SuccessStyle.Render(m.statusMessage))
	}

	return " " + strings.Join(parts, styles.MutedStyle.Render(" • "))
}

// isHelpSafe returns true if it's safe to show help (no active text input)
func (m *MainModel) isHelpSafe() bool {
	switch m.mode {
	case ViewResults, ViewHistory, ViewColumns, ViewSchema:
		// These views don't have text input
		return true
	case ViewBrowser:
		// Browser is safe when not filtering
		return !m.browserView.filtering
	case ViewChat:
		// Chat is safe when not focused on message input
		return m.chatView.focusField != focusFieldMessage
	case ViewWhere:
		// Where is safe when not adding/editing
		return !m.whereView.addingNew
	case ViewExport:
		// Export is safe when not on filename field
		return m.exportView.focusIndex != 0
	case ViewConnection:
		// Connection is safe in list mode
		return m.connectionView.mode == "list"
	case ViewEditor:
		// Editor always has text input
		return false
	}
	return false
}

// renderHelpOverlay renders a help overlay for the current view
func (m *MainModel) renderHelpOverlay() string {
	var b strings.Builder

	b.WriteString(styles.RenderTitle("Keyboard Shortcuts"))
	b.WriteString("\n\n")

	switch m.mode {
	case ViewBrowser:
		b.WriteString(styles.KeyStyle.Render("Browser View\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.Browser.Schema,
			Keys.Browser.Refresh,
			Keys.Browser.Editor,
			Keys.Browser.AIChat,
			Keys.Browser.History,
			Keys.Browser.Filter,
			Keys.Browser.Select,
			Keys.Browser.Disconnect,
		))

	case ViewResults:
		b.WriteString(styles.KeyStyle.Render("Results View\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.Results.NextPage,
			Keys.Results.ColLeft,
			Keys.Results.Where,
			Keys.Results.Columns,
			Keys.Results.Export,
			Keys.Results.PageSize,
			Keys.Results.CustomSize,
			Keys.Global.Back,
		))

	case ViewHistory:
		b.WriteString(styles.KeyStyle.Render("History View\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.History.Edit,
			Keys.History.Rerun,
			Keys.History.ClearAll,
			Keys.Global.Back,
		))

	case ViewChat:
		b.WriteString(styles.KeyStyle.Render("AI Chat View\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.Chat.CycleFieldUp,
			Keys.Chat.ChangeLeft,
			Keys.Chat.Send,
			Keys.Chat.SelectPrevMsg,
			Keys.Chat.FocusInput,
			Keys.Chat.RevokeConsent,
			Keys.Global.Back,
		))

	case ViewSchema:
		b.WriteString(styles.KeyStyle.Render("Schema View\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.Schema.Toggle,
			Keys.Schema.ViewData,
			Keys.Schema.Filter,
			Keys.Schema.Refresh,
			Keys.Global.Back,
		))

	case ViewColumns:
		b.WriteString(styles.KeyStyle.Render("Column Selection\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.Columns.Toggle,
			Keys.Columns.SelectAll,
			Keys.Columns.SelectNone,
			Keys.Columns.Apply,
			Keys.Global.Back,
		))

	case ViewWhere:
		b.WriteString(styles.KeyStyle.Render("WHERE Conditions\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.WhereList.Add,
			Keys.WhereList.EditCond,
			Keys.WhereList.Delete,
			Keys.WhereList.Apply,
			Keys.Global.Back,
		))

	case ViewExport:
		b.WriteString(styles.KeyStyle.Render("Export View\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.Export.Next,
			Keys.Export.OptionLeft,
			Keys.Export.Export,
			Keys.Global.Back,
		))

	case ViewConnection:
		b.WriteString(styles.KeyStyle.Render("Connection View\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.ConnectionList.New,
			Keys.ConnectionList.DeleteConn,
			Keys.ConnectionList.Connect,
			Keys.ConnectionList.QuitEsc,
		))

	default:
		b.WriteString("No help available for this view\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.MutedStyle.Render("Press any key to close"))

	return styles.BaseStyle.Padding(1, 2).Render(b.String())
}

func (m *MainModel) handleTabSwitch() (tea.Model, tea.Cmd) {
	if m.dbManager.GetCurrentConnection() == nil {
		return m, nil
	}

	// Define explicit tab order for tabbable views
	tabOrder := []ViewMode{ViewBrowser, ViewEditor, ViewResults, ViewHistory, ViewChat}

	// Find current position in tab order
	currentIndex := -1
	for i, mode := range tabOrder {
		if mode == m.mode {
			currentIndex = i
			break
		}
	}

	// Tab switching clears the navigation stack
	m.viewHistory = nil

	// Move to next view in tab order (or start at beginning if not in tabbable view)
	if currentIndex == -1 {
		m.mode = tabOrder[0]
	} else {
		m.mode = tabOrder[(currentIndex+1)%len(tabOrder)]
	}

	m.onViewEnter(m.mode)

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
		{ViewChat, "Chat"},
		{ViewExport, "Export"},
		{ViewWhere, "Where"},
		{ViewColumns, "Columns"},
		{ViewSchema, "Schema"},
	}

	// Define which views are tab-accessible
	tabbableViews := map[ViewMode]bool{
		ViewBrowser: true,
		ViewEditor:  true,
		ViewResults: true,
		ViewHistory: true,
		ViewChat:    true,
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

// PushView saves the current mode on the view history stack and switches to newView.
func (m *MainModel) PushView(newView ViewMode) {
	m.viewHistory = append(m.viewHistory, m.mode)
	m.mode = newView
	m.onViewEnter(newView)
}

// PopView restores the previous view from the history stack.
// Returns false if the stack is empty.
func (m *MainModel) PopView() bool {
	if len(m.viewHistory) == 0 {
		return false
	}
	m.mode = m.viewHistory[len(m.viewHistory)-1]
	m.viewHistory = m.viewHistory[:len(m.viewHistory)-1]
	return true
}

// onViewEnter is called when a view becomes active, to refresh view-specific state.
func (m *MainModel) onViewEnter(mode ViewMode) {
	switch mode {
	case ViewHistory:
		m.historyView.refreshList()
	}
}

// SpinnerView returns the current spinner frame for use in loading indicators.
func (m *MainModel) SpinnerView() string {
	return m.spinner.View()
}

func renderError(message string) string {
	errorBox := styles.RenderErrorBox(message)
	helpText := styles.RenderHelp("esc", "dismiss", "ctrl+c", "exit")
	return "\n" + errorBox + "\n\n" + helpText + "\n"
}
