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

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/internal/history"
	"github.com/clidey/whodb/cli/internal/tui/layout"
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
	ViewImport
	ViewBookmarks
	ViewJSON
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

	// Concrete view references — used by existing code and tests.
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
	importView     *ImportView
	bookmarksView  *BookmarksView
	jsonViewer     *JSONViewer

	// panes maps each ViewMode to its Pane interface for polymorphic layout dispatch.
	panes map[ViewMode]Pane

	// Split-pane layout state (active only when connected).
	activeLayout   layout.LayoutName
	layoutRoot     *layout.Container
	focusedPaneIdx int
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
	m.importView = NewImportView(m)
	m.bookmarksView = NewBookmarksView(m)
	m.jsonViewer = NewJSONViewer(m)

	m.panes = map[ViewMode]Pane{
		ViewConnection: m.connectionView,
		ViewBrowser:    m.browserView,
		ViewEditor:     m.editorView,
		ViewResults:    m.resultsView,
		ViewHistory:    m.historyView,
		ViewExport:     m.exportView,
		ViewWhere:      m.whereView,
		ViewColumns:    m.columnsView,
		ViewChat:       m.chatView,
		ViewSchema:     m.schemaView,
		ViewImport:     m.importView,
		ViewBookmarks:  m.bookmarksView,
		ViewJSON:       m.jsonViewer,
	}

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
	// Layout will be initialized on first WindowSizeMsg (width not known yet)
	return m
}

func (m *MainModel) Init() tea.Cmd {
	if m.err != nil {
		return nil
	}

	// Apply saved theme
	themeName := m.config.GetThemeName()
	if t := styles.GetThemeByName(themeName); t != nil {
		styles.SetTheme(t)
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
		m.jsonViewer, _ = m.jsonViewer.Update(msg)

		// Rebuild layout on resize if connected
		if m.dbManager.GetCurrentConnection() != nil && m.layoutRoot == nil {
			m.initLayout()
		}
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

		case "ctrl+t":
			return m.cycleTheme()

		case "ctrl+l":
			if m.dbManager.GetCurrentConnection() != nil {
				return m.cycleLayout()
			}

		case "alt+1":
			if m.useMultiPane() {
				m.focusPaneByIndex(0)
				return m, nil
			}
		case "alt+2":
			if m.useMultiPane() {
				m.focusPaneByIndex(1)
				return m, nil
			}
		case "alt+3":
			if m.useMultiPane() {
				m.focusPaneByIndex(2)
				return m, nil
			}

		case "alt+left":
			if m.useMultiPane() && m.layoutRoot != nil {
				m.layoutRoot.AdjustRatio(-0.05)
				return m, nil
			}
		case "alt+right":
			if m.useMultiPane() && m.layoutRoot != nil {
				m.layoutRoot.AdjustRatio(0.05)
				return m, nil
			}

		case "ctrl+h":
			// Global shortcut: open History from any view/pane
			if m.dbManager.GetCurrentConnection() != nil && m.mode != ViewHistory {
				// In multi-pane, switch to single-pane History
				if m.useMultiPane() {
					m.activeLayout = layout.LayoutSingle
					m.rebuildLayout()
				}
				m.PushView(ViewHistory)
				return m, nil
			}

		case "ctrl+g":
			// Global shortcut: open Import wizard
			if m.dbManager.GetCurrentConnection() != nil && m.mode != ViewImport {
				if m.useMultiPane() {
					m.activeLayout = layout.LayoutSingle
					m.rebuildLayout()
				}
				m.PushView(ViewImport)
				return m, nil
			}

		case "ctrl+b":
			// Global shortcut: open Bookmarks from any view/pane
			if m.dbManager.GetCurrentConnection() != nil && m.mode != ViewBookmarks {
				if m.useMultiPane() {
					m.activeLayout = layout.LayoutSingle
					m.rebuildLayout()
				}
				m.bookmarksView.editorQuery = m.editorView.textarea.Value()
				m.PushView(ViewBookmarks)
				return m, nil
			}

		case "ctrl+y":
			// Global shortcut: toggle read-only mode
			if m.dbManager.GetCurrentConnection() != nil {
				return m.toggleReadOnly()
			}

		case "ctrl+a":
			// Global shortcut: open AI Chat from any view/pane
			if m.dbManager.GetCurrentConnection() != nil && m.mode != ViewChat {
				if m.useMultiPane() {
					m.activeLayout = layout.LayoutSingle
					m.rebuildLayout()
				}
				m.PushView(ViewChat)
				return m, m.chatView.Init()
			}

		case "tab", "shift+tab":
			// Let connection view handle Tab for its own navigation
			if m.mode == ViewConnection {
				return m.updateConnectionView(msg)
			}
			// In multi-pane mode, Tab cycles pane focus
			if m.useMultiPane() {
				if msg.String() == "tab" {
					m.focusNextPane()
				} else {
					m.focusPrevPane()
				}
				return m, nil
			}
			if msg.String() == "tab" {
				return m.handleTabSwitch()
			}
			return m, nil

		}
	}

	// Route async completion messages to their target view regardless of focus.
	// This is critical for multi-pane mode where the focused pane may not be the
	// one that initiated the async operation.
	switch msg.(type) {
	case PageLoadedMsg:
		return m.updateResultsView(msg)
	case QueryExecutedMsg, QueryCancelledMsg, QueryTimeoutMsg, AutocompleteDebounceMsg, externalEditorResultMsg:
		return m.updateEditorView(msg)
	case tablesLoadedMsg, escConfirmTimeoutMsg:
		return m.updateBrowserView(msg)
	case chatResponseMsg, modelsLoadedMsg:
		return m.updateChatView(msg)
	case HistoryQueryMsg:
		return m.updateHistoryView(msg)
	case exportResultMsg:
		return m.updateExportView(msg)
	case schemaLoadedMsg:
		return m.updateSchemaView(msg)
	case connectionResultMsg:
		return m.updateConnectionView(msg)
	case importResultMsg, importPreviewMsg:
		return m.updateImportView(msg)
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
	case ViewImport:
		return m.updateImportView(msg)
	case ViewBookmarks:
		return m.updateBookmarksView(msg)
	case ViewJSON:
		return m.updateJSONViewer(msg)
	}

	return m, nil
}

func (m *MainModel) updateJSONViewer(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.jsonViewer, cmd = m.jsonViewer.Update(msg)
	return m, cmd
}

func (m *MainModel) updateBookmarksView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.bookmarksView, cmd = m.bookmarksView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateImportView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.importView, cmd = m.importView.Update(msg)
	return m, cmd
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

	if m.useMultiPane() && m.layoutRoot != nil {
		// Multi-pane layout rendering (reserve 2 rows for the global help bar)
		helpBarHeight := 2
		contentH := m.layoutContentHeight() - helpBarHeight
		if contentH < MinPaneHeight {
			contentH = MinPaneHeight
		}
		m.layoutRoot.Layout(0, 0, m.width, contentH)
		content = m.layoutRoot.View() + "\n" + m.renderGlobalHelpBar()
	} else {
		// Single-pane rendering (original behavior)
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
		case ViewImport:
			content = m.importView.View()
		case ViewBookmarks:
			content = m.bookmarksView.View()
		case ViewJSON:
			content = m.jsonViewer.View()
		}
	}

	// Add status bar between view indicator and content (only when connected)
	statusBar := m.renderStatusBar()
	var output string
	if statusBar != "" {
		output = viewIndicator + "\n" + statusBar + "\n" + content
	} else {
		output = viewIndicator + "\n" + content
	}

	// Constrain output to terminal height so the top bar never scrolls off screen.
	// Truncate from the bottom — the header stays, content gets clipped.
	if m.height > 0 {
		lines := strings.Split(output, "\n")
		if len(lines) > m.height {
			lines = lines[:m.height]
			output = strings.Join(lines, "\n")
		}
	}

	return output
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

	// Read-only badge (shown first for visibility)
	if m.config.GetReadOnly() {
		parts = append(parts, styles.RenderErr("[READ-ONLY]"))
	}

	// Connection info (always kept)
	connInfo := fmt.Sprintf("%s@%s/%s", conn.Type, conn.Host, conn.Database)
	parts = append(parts, styles.RenderMuted(connInfo))

	// Current schema
	if m.browserView.currentSchema != "" {
		parts = append(parts, styles.RenderMuted("schema:"+m.browserView.currentSchema))
	}

	// Spinner when loading
	if m.isLoading() {
		parts = append(parts, m.spinner.View())
	}

	sep := styles.RenderMuted(" • ")
	result := " " + strings.Join(parts, sep)

	// Truncate to terminal width, dropping right-most parts first
	if m.width > 0 {
		for lipgloss.Width(result) > m.width && len(parts) > 1 {
			parts = parts[:len(parts)-1]
			result = " " + strings.Join(parts, sep)
		}
		if lipgloss.Width(result) > m.width {
			result = ansi.Truncate(result, m.width, "…")
		}
	}

	return result
}

// isHelpSafe returns true if it's safe to show help (no active text input)
func (m *MainModel) isHelpSafe() bool {
	switch m.mode {
	case ViewResults, ViewHistory, ViewColumns, ViewSchema, ViewJSON, ViewBookmarks:
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
		b.WriteString(styles.RenderKey("Browser View\n\n"))
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
		b.WriteString(styles.RenderKey("Results View\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.Results.NextPage,
			Keys.Results.ColLeft,
			Keys.Results.ViewCell,
			Keys.Results.Where,
			Keys.Results.Columns,
			Keys.Results.Export,
			Keys.Results.PageSize,
			Keys.Results.CustomSize,
			Keys.Global.Back,
		))

	case ViewHistory:
		b.WriteString(styles.RenderKey("History View\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.History.Edit,
			Keys.History.Rerun,
			Keys.History.ClearAll,
			Keys.Global.Back,
		))

	case ViewChat:
		b.WriteString(styles.RenderKey("AI Chat View\n\n"))
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
		b.WriteString(styles.RenderKey("Schema View\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.Schema.Toggle,
			Keys.Schema.ViewData,
			Keys.Schema.Filter,
			Keys.Schema.Refresh,
			Keys.Global.Back,
		))

	case ViewColumns:
		b.WriteString(styles.RenderKey("Column Selection\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.Columns.Toggle,
			Keys.Columns.SelectAll,
			Keys.Columns.SelectNone,
			Keys.Columns.Apply,
			Keys.Global.Back,
		))

	case ViewWhere:
		b.WriteString(styles.RenderKey("WHERE Conditions\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.WhereList.Add,
			Keys.WhereList.EditCond,
			Keys.WhereList.Delete,
			Keys.WhereList.Apply,
			Keys.Global.Back,
		))

	case ViewExport:
		b.WriteString(styles.RenderKey("Export View\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.Export.Next,
			Keys.Export.OptionLeft,
			Keys.Export.Export,
			Keys.Global.Back,
		))

	case ViewConnection:
		b.WriteString(styles.RenderKey("Connection View\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.ConnectionList.New,
			Keys.ConnectionList.DeleteConn,
			Keys.ConnectionList.Connect,
			Keys.Global.CycleTheme,
			Keys.ConnectionList.QuitEsc,
		))

	case ViewBookmarks:
		b.WriteString(styles.RenderKey("Bookmarks\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.Bookmarks.Up,
			Keys.Bookmarks.Down,
			Keys.Bookmarks.Load,
			Keys.Bookmarks.Save,
			Keys.Bookmarks.Delete,
			Keys.Global.Back,
		))

	default:
		b.WriteString("No help available for this view\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.RenderMuted("Press any key to close"))

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

// renderGlobalHelpBar renders a context-sensitive help bar at the bottom of
// multi-pane layouts, showing shortcuts for the focused pane + global shortcuts.
func (m *MainModel) renderGlobalHelpBar() string {
	pane := m.ActivePane()
	var bindings []key.Binding
	if pane != nil {
		bindings = append(bindings, pane.HelpBindings()...)
	}
	// Add global multi-pane shortcuts
	bindings = append(bindings,
		Keys.Global.NextView,
		Keys.Browser.History,
		Keys.Browser.AIChat,
		Keys.Global.Import,
		Keys.Global.ReadOnly,
		Keys.Global.CycleLayout,
		Keys.Global.CycleTheme,
		Keys.Browser.Disconnect,
		Keys.Global.Quit,
	)
	return " " + RenderBindingHelpWidth(m.width, bindings...)
}

// initLayout sets up the initial layout based on terminal width.
// Called when a database connection is established.
func (m *MainModel) initLayout() {
	m.activeLayout = layout.AutoLayout(m.width)
	m.focusedPaneIdx = 0
	m.rebuildLayout()
}

// useMultiPane returns true when the layout engine should render connected views.
// Returns false for modal/overlay views (Export, Where, Columns, Schema, History, Chat)
// which render full-screen on top of the layout.
func (m *MainModel) useMultiPane() bool {
	if m.dbManager.GetCurrentConnection() == nil {
		return false
	}
	if m.activeLayout == layout.LayoutSingle {
		return false
	}
	// Only render multi-pane for the views that are IN the layout
	switch m.mode {
	case ViewBrowser, ViewEditor, ViewResults:
		return true
	default:
		return false
	}
}

// rebuildLayout creates a new layout tree for the current activeLayout.
func (m *MainModel) rebuildLayout() {
	// Reset compact mode on all layout views
	m.browserView.SetCompact(false)
	m.editorView.SetCompact(false)
	m.resultsView.SetCompact(false)

	if m.activeLayout == layout.LayoutSingle {
		m.layoutRoot = nil
		return
	}

	// Enable compact mode (suppresses per-pane help text)
	m.browserView.SetCompact(true)
	m.editorView.SetCompact(true)
	m.resultsView.SetCompact(true)

	m.layoutRoot = layout.BuildLayout(m.activeLayout, layout.Panes{
		Browser: m.browserView,
		Editor:  m.editorView,
		Results: m.resultsView,
	})
	m.updateLayoutFocus()
}

// updateLayoutFocus marks the correct leaf as focused in the layout tree.
func (m *MainModel) updateLayoutFocus() {
	if m.layoutRoot == nil {
		return
	}
	leaves := m.layoutRoot.Leaves()
	for i, leaf := range leaves {
		leaf.SetFocused(i == m.focusedPaneIdx)
	}
}

// layoutContentHeight returns the height available for layout content
// (terminal height minus the view indicator and status bar).
func (m *MainModel) layoutContentHeight() int {
	overhead := 1 // view indicator line
	if bar := m.renderStatusBar(); bar != "" {
		overhead += 1 + strings.Count(bar, "\n")
	}
	h := m.height - overhead
	if h < MinPaneHeight {
		h = MinPaneHeight
	}
	return h
}

// MinPaneHeight is imported from the layout package for convenience.
const MinPaneHeight = 4

// cycleLayout switches to the next layout preset and rebuilds.
func (m *MainModel) cycleLayout() (tea.Model, tea.Cmd) {
	m.activeLayout = layout.NextLayout(m.activeLayout)
	m.focusedPaneIdx = 0
	m.rebuildLayout()
	return m, m.SetStatus("Layout: " + string(m.activeLayout))
}

// focusNextPane cycles focus to the next pane in the layout.
func (m *MainModel) focusNextPane() {
	if m.layoutRoot == nil {
		return
	}
	leaves := m.layoutRoot.Leaves()
	if len(leaves) == 0 {
		return
	}
	m.focusedPaneIdx = (m.focusedPaneIdx + 1) % len(leaves)
	m.updateLayoutFocus()

	// Update m.mode to match the focused pane's view type
	m.syncModeFromFocusedPane()
}

// focusPrevPane cycles focus to the previous pane in the layout.
func (m *MainModel) focusPrevPane() {
	if m.layoutRoot == nil {
		return
	}
	leaves := m.layoutRoot.Leaves()
	if len(leaves) == 0 {
		return
	}
	m.focusedPaneIdx--
	if m.focusedPaneIdx < 0 {
		m.focusedPaneIdx = len(leaves) - 1
	}
	m.updateLayoutFocus()
	m.syncModeFromFocusedPane()
}

// focusPaneByIndex focuses a specific pane (0-indexed).
func (m *MainModel) focusPaneByIndex(idx int) {
	if m.layoutRoot == nil {
		return
	}
	leaves := m.layoutRoot.Leaves()
	if idx < 0 || idx >= len(leaves) {
		return
	}
	m.focusedPaneIdx = idx
	m.updateLayoutFocus()
	m.syncModeFromFocusedPane()
}

// syncModeFromFocusedPane updates m.mode to match the focused pane's content.
func (m *MainModel) syncModeFromFocusedPane() {
	if m.layoutRoot == nil {
		return
	}
	leaves := m.layoutRoot.Leaves()
	if m.focusedPaneIdx >= len(leaves) {
		return
	}
	content := leaves[m.focusedPaneIdx].Content()
	switch content {
	case m.browserView:
		m.mode = ViewBrowser
	case m.editorView:
		m.mode = ViewEditor
	case m.resultsView:
		m.mode = ViewResults
	}
}

func (m *MainModel) renderViewIndicator() string {
	// Only show main navigable views — modal views (Export, Where, Columns, Schema)
	// are contextual actions, not top-level tabs.
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
	}

	var parts []string
	for _, view := range views {
		if view.mode == m.mode {
			activeStyle := styles.BaseStyle.
				Foreground(styles.Background).
				Background(styles.Foreground).
				Padding(0, 1)
			parts = append(parts, activeStyle.Render(view.name))
		} else {
			inactiveStyle := styles.BaseStyle.
				Foreground(styles.Foreground).
				Padding(0, 1)
			parts = append(parts, inactiveStyle.Render(view.name))
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

	// Show layout name when in multi-pane mode
	if m.useMultiPane() {
		layoutLabel := styles.MutedStyle.Render("[" + string(m.activeLayout) + "]")
		result += "  " + layoutLabel
	}

	// Append transient status message to the indicator bar (avoids extra line)
	if m.statusMessage != "" {
		result += "  " + styles.RenderOk(m.statusMessage)
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

// ContentHeight returns the number of rows available for view content,
// accounting for the view indicator and status bar lines rendered by MainModel.
func (m *MainModel) ContentHeight() int {
	if m.height <= 0 {
		return 20 // sane default before first WindowSizeMsg
	}
	overhead := lipgloss.Height(m.renderViewIndicator()) + 1 // indicator + newline
	if bar := m.renderStatusBar(); bar != "" {
		overhead += lipgloss.Height(bar) + 1 // status bar + newline
	}
	h := m.height - overhead
	if h < 3 {
		h = 3
	}
	return h
}

// SpinnerView returns the current spinner frame for use in loading indicators.
func (m *MainModel) SpinnerView() string {
	return m.spinner.View()
}

// cycleTheme switches to the next built-in theme and persists the choice.
func (m *MainModel) cycleTheme() (tea.Model, tea.Cmd) {
	themes := styles.ListThemes()
	current := m.config.GetThemeName()

	nextIndex := 0
	for i, name := range themes {
		if name == current {
			nextIndex = (i + 1) % len(themes)
			break
		}
	}

	next := themes[nextIndex]
	if t := styles.GetThemeByName(next); t != nil {
		styles.SetTheme(t)
		m.config.SetThemeName(next)
		m.config.Save()
		return m, m.SetStatus("Theme: " + next)
	}
	return m, nil
}

// toggleReadOnly flips read-only mode on/off, persists the setting, and shows a status message.
func (m *MainModel) toggleReadOnly() (tea.Model, tea.Cmd) {
	newState := !m.config.GetReadOnly()
	m.config.SetReadOnly(newState)
	m.config.Save()

	label := "OFF"
	if newState {
		label = "ON"
	}
	return m, m.SetStatus("Read-only: " + label)
}

// GetPane returns the Pane for the given ViewMode.
func (m *MainModel) GetPane(mode ViewMode) Pane {
	return m.panes[mode]
}

// ActivePane returns the Pane for the currently active view.
func (m *MainModel) ActivePane() Pane {
	return m.panes[m.mode]
}

func renderError(message string) string {
	errorBox := styles.RenderErrorBox(message)
	helpText := styles.RenderHelp("esc", "dismiss", "ctrl+c", "exit")
	return "\n" + errorBox + "\n\n" + helpText + "\n"
}
