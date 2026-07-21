/*
 * Copyright 2026 Clidey, Inc.
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

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/internal/history"
	"github.com/clidey/whodb/cli/internal/tui/layout"
	"github.com/clidey/whodb/cli/pkg/styles"
	"github.com/clidey/whodb/core/graph/model"
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
	ViewMockData
	ViewRowWrite
	ViewBookmarks
	ViewJSON
	ViewCmdLog
	ViewExplain
	ViewDiff
	ViewERD
	ViewAudit
	ViewProfiles
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
	initCommands  []tea.Cmd

	currentProfileName string

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
	mockDataView   *MockDataView
	rowWriteView   *RowWriteView
	bookmarksView  *BookmarksView
	jsonViewer     *JSONViewer
	cmdLogView     *CmdLogView
	explainView    *ExplainView
	diffView       *SchemaDiffView
	erdView        *ERDView
	auditView      *AuditView
	profilesView   *ProfilesView

	// panes maps each ViewMode to its Pane interface for polymorphic layout dispatch.
	panes map[ViewMode]Pane

	// Split-pane layout state (active only when connected).
	activeLayout   layout.LayoutName
	savedLayout    layout.LayoutName // saved when opening a modal, restored on pop
	layoutRoot     *layout.Container
	focusedPaneIdx int

	pendingConfigSave bool
	configSaveToken   uint64
}

type configSaveMsg struct {
	token uint64
}

const configSaveDebounce = 250 * time.Millisecond

func NewMainModel() *MainModel {
	return newMainModel(nil, true)
}

func newMainModel(cfg *config.Config, restoreWorkspace bool) *MainModel {
	var err error
	if cfg == nil {
		cfg, err = config.LoadConfig()
		if err != nil {
			return &MainModel{err: err}
		}
	}

	dbMgr, err := database.NewManagerWithConfig(cfg)
	if err != nil {
		return &MainModel{err: err}
	}

	histMgr, err := history.NewManagerWithConfig(cfg)
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
	m.mockDataView = NewMockDataView(m)
	m.rowWriteView = NewRowWriteView(m)
	m.bookmarksView = NewBookmarksView(m)
	m.jsonViewer = NewJSONViewer(m)
	m.cmdLogView = NewCmdLogView(m)
	m.explainView = NewExplainView(m)
	m.diffView = NewSchemaDiffView(m)
	m.erdView = NewERDView(m)
	m.auditView = NewAuditView(m)
	m.profilesView = NewProfilesView(m)

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
		ViewMockData:   m.mockDataView,
		ViewRowWrite:   m.rowWriteView,
		ViewBookmarks:  m.bookmarksView,
		ViewJSON:       m.jsonViewer,
		ViewCmdLog:     m.cmdLogView,
		ViewExplain:    m.explainView,
		ViewDiff:       m.diffView,
		ViewERD:        m.erdView,
		ViewAudit:      m.auditView,
		ViewProfiles:   m.profilesView,
	}

	if restoreWorkspace {
		m.restoreWorkspace()
	}

	return m
}

func NewMainModelWithConnection(conn *config.Connection) *MainModel {
	m := newMainModel(nil, false)
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

// NewMainModelWithConnectionPrefill creates a model that starts on the
// connection form with the provided connection fields pre-populated.
func NewMainModelWithConnectionPrefill(conn *config.Connection) *MainModel {
	m := newMainModel(nil, false)
	if m.err != nil {
		return m
	}

	if conn != nil {
		m.connectionView.mode = "form"
		m.connectionView.resetForm()
		m.connectionView.prefillFromConnection(*conn)
	}

	return m
}

// NewMainModelWithProfile creates a model that connects using the given
// connection and applies the provided config (which already has profile
// settings like theme, page size, and timeout applied).
func NewMainModelWithProfile(conn *config.Connection, cfg *config.Config, profileName string) *MainModel {
	m := newMainModel(cfg, false)
	if m.err != nil {
		return m
	}

	if err := m.dbManager.Connect(conn); err != nil {
		m.err = err
		return m
	}

	m.mode = ViewBrowser
	m.currentProfileName = profileName
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

	cmds := []tea.Cmd{m.spinner.Tick, tea.RequestBackgroundColor}
	if m.mode == ViewConnection {
		cmds = append(cmds, m.connectionView.Init())
	}
	if len(m.initCommands) > 0 {
		cmds = append(cmds, m.initCommands...)
	} else if m.mode == ViewBrowser && m.dbManager.GetCurrentConnection() != nil {
		cmds = append(cmds, m.browserView.loadTables())
	}
	return tea.Batch(cmds...)
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.err != nil {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
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
				case ViewConnection:
					m.connectionView.refreshList()
					return m, tea.Batch(m.connectionView.pingAllConnections(), m.connectionView.loadDockerConnections())
				}
				return m, nil
			}
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
		}
		return m, nil
	}

	// If showing help, any key dismisses it
	if m.showingHelp {
		if _, ok := msg.(tea.KeyPressMsg); ok {
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

	case configSaveMsg:
		if !m.pendingConfigSave || msg.token != m.configSaveToken {
			return m, nil
		}
		m.pendingConfigSave = false
		_ = m.config.Save()
		return m, nil

	case tea.BackgroundColorMsg:
		styles.SetDarkBackground(msg.IsDark())
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
		m.rowWriteView, _ = m.rowWriteView.Update(msg)
		m.jsonViewer, _ = m.jsonViewer.Update(msg)
		m.cmdLogView, _ = m.cmdLogView.Update(msg)
		m.explainView, _ = m.explainView.Update(msg)
		m.diffView, _ = m.diffView.Update(msg)
		m.erdView, _ = m.erdView.Update(msg)
		m.auditView, _ = m.auditView.Update(msg)
		m.profilesView, _ = m.profilesView.Update(msg)

		// Rebuild layout on resize if connected
		if m.dbManager.GetCurrentConnection() != nil && m.layoutRoot == nil {
			if m.activeLayout == "" {
				m.initLayout()
			} else if m.activeLayout != layout.LayoutSingle {
				m.rebuildLayout()
				m.focusPaneByIndex(m.focusedPaneIdx)
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Batch(m.flushConfigSave(), tea.Quit)

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
				m.suspendLayout()
				m.PushView(ViewHistory)
				return m, nil
			}

		case "ctrl+g":
			// Global shortcut: open Import wizard
			if m.dbManager.GetCurrentConnection() != nil && m.mode != ViewImport {
				m.suspendLayout()
				m.PushView(ViewImport)
				return m, nil
			}

		case "alt+m":
			// Global shortcut: open mock-data wizard
			if m.dbManager.GetCurrentConnection() != nil && m.mode != ViewMockData {
				return m.openMockDataView()
			}

		case "ctrl+b":
			// Global shortcut: open Bookmarks from any view/pane
			if m.dbManager.GetCurrentConnection() != nil && m.mode != ViewBookmarks {
				m.suspendLayout()
				m.bookmarksView.editorQuery = m.editorView.textarea.Value()
				m.PushView(ViewBookmarks)
				return m, nil
			}

		case "ctrl+y":
			// Global shortcut: toggle read-only mode
			if m.dbManager.GetCurrentConnection() != nil {
				return m.toggleReadOnly()
			}

		case "ctrl+d":
			// Global shortcut: toggle command log view
			if m.dbManager.GetCurrentConnection() != nil {
				if m.mode == ViewCmdLog {
					if !m.PopView() {
						m.mode = ViewBrowser
					}
					return m, nil
				}
				m.suspendLayout()
				m.PushView(ViewCmdLog)
				return m, nil
			}

		case "ctrl+v":
			// Global shortcut: open schema diff
			if m.dbManager.GetCurrentConnection() != nil && m.mode != ViewDiff {
				m.suspendLayout()
				m.diffView.prepare()
				m.PushView(ViewDiff)
				return m, nil
			}

		case "ctrl+k":
			// Global shortcut: open ER diagram
			if m.dbManager.GetCurrentConnection() != nil && m.mode != ViewERD {
				m.suspendLayout()
				m.erdView.loading = true
				m.erdView.err = nil
				m.PushView(ViewERD)
				return m, m.erdView.loadERDData()
			}

		case "ctrl+u":
			// Global shortcut: open data quality audit
			if m.dbManager.GetCurrentConnection() != nil && m.mode != ViewAudit {
				m.suspendLayout()
				m.auditView.loading = true
				m.auditView.err = nil
				m.PushView(ViewAudit)
				return m, m.auditView.loadAuditData()
			}

		case "ctrl+p":
			// Global shortcut: open Profiles from any view (including Connection)
			// Skip when in Chat view (ctrl+p is used for message navigation there)
			if m.mode != ViewProfiles && m.mode != ViewChat {
				m.suspendLayout()
				m.PushView(ViewProfiles)
				return m, nil
			}

		case "ctrl+a":
			// Global shortcut: open AI Chat from any view/pane
			if m.dbManager.GetCurrentConnection() != nil && m.mode != ViewChat {
				m.suspendLayout()
				m.PushView(ViewChat)
				return m, m.chatView.Init()
			}

		case "tab", "shift+tab":
			// Let modal views handle Tab themselves (e.g., ERD table cycling)
			if m.mode == ViewERD {
				return m.updateERDView(msg)
			}
			if m.mode == ViewDiff {
				return m.updateDiffView(msg)
			}
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
	case chatResponseMsg, modelsLoadedMsg, chatStreamChunkMsg, chatStreamDoneMsg:
		return m.updateChatView(msg)
	case HistoryQueryMsg:
		return m.updateHistoryView(msg)
	case exportResultMsg:
		return m.updateExportView(msg)
	case schemaLoadedMsg, schemaTableColumnsLoadedMsg:
		return m.updateSchemaView(msg)
	case connectionResultMsg:
		return m.updateConnectionView(msg)
	case importResultMsg, importPreviewMsg:
		return m.updateImportView(msg)
	case mockDataAnalysisMsg, mockDataResultMsg:
		return m.updateMockDataView(msg)
	case rowWriteResultMsg:
		return m.updateRowWriteView(msg)
	case explainResultMsg:
		return m.updateExplainView(msg)
	case schemaDiffResultMsg:
		return m.updateDiffView(msg)
	case erdDataLoadedMsg, erdTableColumnsLoadedMsg:
		return m.updateERDView(msg)
	case auditResultMsg:
		return m.updateAuditView(msg)
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
	case ViewMockData:
		return m.updateMockDataView(msg)
	case ViewRowWrite:
		return m.updateRowWriteView(msg)
	case ViewBookmarks:
		return m.updateBookmarksView(msg)
	case ViewJSON:
		return m.updateJSONViewer(msg)
	case ViewCmdLog:
		return m.updateCmdLogView(msg)
	case ViewExplain:
		return m.updateExplainView(msg)
	case ViewDiff:
		return m.updateDiffView(msg)
	case ViewERD:
		return m.updateERDView(msg)
	case ViewAudit:
		return m.updateAuditView(msg)
	case ViewProfiles:
		return m.updateProfilesView(msg)
	}

	return m, nil
}

func (m *MainModel) updateJSONViewer(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.jsonViewer, cmd = m.jsonViewer.Update(msg)
	return m, cmd
}

func (m *MainModel) updateCmdLogView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.cmdLogView, cmd = m.cmdLogView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateExplainView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.explainView, cmd = m.explainView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateDiffView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.diffView, cmd = m.diffView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateERDView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.erdView, cmd = m.erdView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateAuditView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.auditView, cmd = m.auditView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateBookmarksView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.bookmarksView, cmd = m.bookmarksView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateProfilesView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.profilesView, cmd = m.profilesView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateImportView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.importView, cmd = m.importView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateMockDataView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.mockDataView, cmd = m.mockDataView.Update(msg)
	return m, cmd
}

func (m *MainModel) updateRowWriteView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.rowWriteView, cmd = m.rowWriteView.Update(msg)
	return m, cmd
}

func (m *MainModel) View() tea.View {
	if m.err != nil {
		return m.newView(renderError(m.err.Error()))
	}

	// Show help overlay if active
	if m.showingHelp {
		return m.newView(m.renderHelpOverlay())
	}

	viewIndicator := m.renderViewIndicator()

	var content string

	if m.useMultiPane() && m.layoutRoot != nil {
		// Multi-pane layout rendering with a footer that may wrap to multiple lines.
		helpBar := m.renderGlobalHelpBar()
		helpBarHeight := lipgloss.Height(helpBar)
		if helpBarHeight < 1 {
			helpBarHeight = 1
		}
		contentH := m.layoutContentHeight() - helpBarHeight
		if contentH < MinPaneHeight {
			contentH = MinPaneHeight
		}
		m.layoutRoot.Layout(0, 0, m.width, contentH)
		content = m.layoutRoot.View() + "\n" + helpBar
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
		case ViewMockData:
			content = m.mockDataView.View()
		case ViewRowWrite:
			content = m.rowWriteView.View()
		case ViewBookmarks:
			content = m.bookmarksView.View()
		case ViewJSON:
			content = m.jsonViewer.View()
		case ViewCmdLog:
			content = m.cmdLogView.View()
		case ViewExplain:
			content = m.explainView.View()
		case ViewDiff:
			content = m.diffView.View()
		case ViewERD:
			content = m.erdView.View()
		case ViewAudit:
			content = m.auditView.View()
		case ViewProfiles:
			content = m.profilesView.View()
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

	return m.newView(output)
}

// newView wraps content in a tea.View with alt-screen and mouse support enabled,
// replacing the removed tea.WithAltScreen/tea.WithMouseCellMotion ProgramOptions.
func (m *MainModel) newView(content string) tea.View {
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
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
		m.resultsView.loading ||
		m.mockDataView.analyzing ||
		m.mockDataView.generating ||
		m.rowWriteView.working ||
		m.diffView.loading ||
		m.erdView.loading ||
		m.auditView.loading
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
	case ViewResults, ViewHistory, ViewColumns, ViewSchema, ViewJSON, ViewBookmarks, ViewCmdLog, ViewExplain, ViewERD, ViewAudit:
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
	case ViewProfiles:
		return !m.profilesView.naming
	case ViewDiff:
		return m.diffView.HelpSafe()
	case ViewMockData:
		return false
	case ViewRowWrite:
		return false
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
			Keys.Global.SchemaDiff,
			Keys.Global.ERDiagram,
			Keys.Global.MockData,
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
			Keys.Results.AddRow,
			Keys.Results.EditRow,
			Keys.Results.DeleteRow,
			Keys.Results.Export,
			Keys.Global.SchemaDiff,
			Keys.Global.ERDiagram,
			Keys.Global.MockData,
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

	case ViewMockData:
		b.WriteString(styles.RenderKey("Mock Data\n\n"))
		b.WriteString(RenderBindingHelp(
			key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
			key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev field")),
			key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle overwrite")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "analyze/generate")),
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "re-analyze")),
			Keys.Global.Back,
		))

	case ViewRowWrite:
		b.WriteString(styles.RenderKey("Row Write\n\n"))
		b.WriteString(RenderBindingHelp(rowWriteHelpBindings(m.rowWriteView.action)...))

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

	case ViewDiff:
		b.WriteString(styles.RenderKey("Schema Diff\n\n"))
		if m.diffView.HelpSafe() {
			b.WriteString(RenderBindingHelp(
				Keys.SchemaDiff.Recompare,
				Keys.SchemaDiff.Edit,
				Keys.SchemaDiff.ScrollUp,
				Keys.SchemaDiff.ScrollDown,
				Keys.Global.Back,
			))
		} else {
			b.WriteString(RenderBindingHelp(
				Keys.SchemaDiff.PrevField,
				Keys.SchemaDiff.NextField,
				Keys.SchemaDiff.OptionLeft,
				Keys.SchemaDiff.OptionRight,
				Keys.SchemaDiff.Compare,
				Keys.Global.Back,
			))
		}

	case ViewERD:
		b.WriteString(styles.RenderKey("ER Diagram\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.ERD.NextTable,
			Keys.ERD.PrevTable,
			Keys.ERD.ToggleZoom,
			Keys.ERD.ScrollUp,
			Keys.ERD.ScrollDown,
			Keys.Global.Back,
		))

	case ViewAudit:
		b.WriteString(styles.RenderKey("Data Quality Audit\n\n"))
		b.WriteString(RenderBindingHelp(
			Keys.Audit.Up,
			Keys.Audit.Down,
			Keys.Audit.DrillDown,
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
	tabOrder := []ViewMode{ViewBrowser, ViewEditor, ViewResults, ViewChat}

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
		Keys.Global.Profiles,
		Keys.Global.CmdLog,
		Keys.Global.SchemaDiff,
		Keys.Global.ERDiagram,
		Keys.Global.Audit,
		Keys.Global.Import,
		Keys.Global.MockData,
		Keys.Global.ReadOnly,
		Keys.Global.CycleLayout,
		Keys.Global.CycleTheme,
		Keys.Browser.Disconnect,
		Keys.Global.Quit,
	)
	if m.isHelpSafe() {
		return " " + RenderBindingHelpWidth(m.width, bindings...)
	}
	return " " + renderBindingHelpWidthNoHelp(m.width, bindings...)
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

// suspendLayout saves the current layout and switches to single-pane for a modal view.
func (m *MainModel) suspendLayout() {
	if m.useMultiPane() {
		m.savedLayout = m.activeLayout
		m.activeLayout = layout.LayoutSingle
		m.rebuildLayout()
	}
}

// restoreLayout restores the layout saved by suspendLayout.
func (m *MainModel) restoreLayout() {
	if m.savedLayout != "" {
		m.activeLayout = m.savedLayout
		m.savedLayout = ""
		m.rebuildLayout()
	}
}

func (m *MainModel) openMockDataView() (tea.Model, tea.Cmd) {
	schema, table := m.currentMockDataTarget()
	m.suspendLayout()
	m.mockDataView.SetTarget(schema, table)
	m.PushView(ViewMockData)
	return m, nil
}

func (m *MainModel) currentMockDataTarget() (string, string) {
	if m.resultsView.isTableData() {
		return m.resultsView.schema, m.resultsView.tableName
	}

	if m.browserView.selectedIndex >= 0 && m.browserView.selectedIndex < len(m.browserView.filteredTables) {
		return m.browserView.currentSchema, m.browserView.filteredTables[m.browserView.selectedIndex].Name
	}

	return m.browserView.currentSchema, ""
}

func (m *MainModel) renderViewIndicator() string {
	// Only show main navigable views — contextual actions (History, Export,
	// Where, Columns, etc.) are accessible via shortcuts, not the tab bar.
	views := []struct {
		mode ViewMode
		name string
	}{
		{ViewConnection, "Connection"},
		{ViewBrowser, "Browser"},
		{ViewEditor, "Editor"},
		{ViewResults, "Results"},
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
	// Restore multi-pane layout if we're returning to a layout-compatible view
	m.restoreLayout()
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
		return m, tea.Batch(m.requestConfigSave(), m.SetStatus("Theme: "+next))
	}
	return m, nil
}

// toggleReadOnly flips read-only mode on/off, persists the setting, and shows a status message.
func (m *MainModel) toggleReadOnly() (tea.Model, tea.Cmd) {
	newState := !m.config.GetReadOnly()
	m.config.SetReadOnly(newState)

	label := "OFF"
	if newState {
		label = "ON"
	}
	return m, tea.Batch(m.requestConfigSave(), m.SetStatus("Read-only: "+label))
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

func (m *MainModel) requestConfigSave() tea.Cmd {
	if m == nil || m.config == nil {
		return nil
	}

	m.pendingConfigSave = true
	m.configSaveToken++
	token := m.configSaveToken

	return tea.Tick(configSaveDebounce, func(time.Time) tea.Msg {
		return configSaveMsg{token: token}
	})
}

func (m *MainModel) flushConfigSave() tea.Cmd {
	if m == nil || m.config == nil {
		return nil
	}
	if !m.pendingConfigSave {
		return nil
	}

	m.pendingConfigSave = false
	_ = m.config.Save()
	return nil
}

// PersistWorkspace stores the reconnectable TUI workspace so it can be
// restored on the next plain interactive launch.
func (m *MainModel) PersistWorkspace() error {
	if m == nil || m.config == nil {
		return nil
	}

	if m.dbManager == nil || m.dbManager.GetCurrentConnection() == nil {
		m.config.ClearWorkspace()
		return m.config.Save()
	}

	connectionName := strings.TrimSpace(m.dbManager.GetCurrentConnection().Name)
	if connectionName == "" && strings.TrimSpace(m.currentProfileName) != "" {
		if profile := m.config.GetProfile(m.currentProfileName); profile != nil {
			connectionName = strings.TrimSpace(profile.Connection)
		}
	}
	if connectionName == "" {
		m.config.ClearWorkspace()
		return m.config.Save()
	}

	m.editorView.saveCurrentBuffer()

	layoutName := m.activeLayout
	if layoutName == "" {
		layoutName = layout.LayoutSingle
	}

	workspace := &config.WorkspaceState{
		ConnectionName: connectionName,
		ProfileName:    strings.TrimSpace(m.currentProfileName),
		View:           workspaceViewName(m.workspaceViewMode()),
		Layout:         string(layoutName),
		FocusedPane:    m.focusedPaneIdx,
		Browser: config.WorkspaceBrowserState{
			Schema: m.browserView.currentSchema,
			Table:  m.browserView.selectedTable,
			Filter: m.browserView.filterInput.Value(),
		},
		Editor:  m.workspaceEditorState(),
		Results: m.workspaceResultsState(),
		Diff:    m.diffView.SelectionState(),
		SavedAt: time.Now().UTC().Format(time.RFC3339),
	}

	m.config.SetWorkspace(workspace)
	return m.config.Save()
}

func (m *MainModel) restoreWorkspace() {
	workspace := m.config.GetWorkspace()
	if workspace == nil {
		return
	}

	connectionName := strings.TrimSpace(workspace.ConnectionName)
	if connectionName == "" && strings.TrimSpace(workspace.ProfileName) != "" {
		if profile := m.config.GetProfile(workspace.ProfileName); profile != nil {
			connectionName = strings.TrimSpace(profile.Connection)
		}
	}
	if connectionName == "" {
		return
	}

	conn, _, err := m.dbManager.ResolveConnection(connectionName)
	if err != nil {
		m.clearWorkspaceSnapshot()
		return
	}
	if err := m.dbManager.Connect(conn); err != nil {
		m.clearWorkspaceSnapshot()
		return
	}

	m.currentProfileName = strings.TrimSpace(workspace.ProfileName)
	m.mode = parseWorkspaceView(workspace.View)
	if workspace.Layout != "" {
		m.activeLayout = layout.LayoutName(workspace.Layout)
	}
	if workspace.FocusedPane >= 0 {
		m.focusedPaneIdx = workspace.FocusedPane
	}

	m.applyWorkspaceBrowserState(workspace.Browser)
	m.applyWorkspaceEditorState(workspace.Editor)
	m.applyWorkspaceResultsState(workspace.Results)
	m.diffView.SetSelectionState(workspace.Diff)

	m.initCommands = append(m.initCommands, m.browserView.loadTables())
	if strings.TrimSpace(workspace.Results.Table) != "" {
		m.initCommands = append(m.initCommands, m.resultsView.loadPage())
	}
}

func (m *MainModel) clearWorkspaceSnapshot() {
	m.config.ClearWorkspace()
	_ = m.config.Save()
}

func (m *MainModel) applyWorkspaceBrowserState(state config.WorkspaceBrowserState) {
	m.browserView.currentSchema = strings.TrimSpace(state.Schema)
	m.browserView.selectedTable = strings.TrimSpace(state.Table)
	m.browserView.filtering = false
	m.browserView.filterInput.SetValue(state.Filter)
}

func (m *MainModel) applyWorkspaceEditorState(state config.WorkspaceEditorState) {
	if len(state.Buffers) == 0 {
		m.editorView.buffers = []queryBuffer{{name: "Query 1", text: ""}}
		m.editorView.activeTab = 0
		m.editorView.textarea.SetValue("")
		return
	}

	buffers := make([]queryBuffer, 0, len(state.Buffers))
	for i, buffer := range state.Buffers {
		name := strings.TrimSpace(buffer.Name)
		if name == "" {
			name = fmt.Sprintf("Query %d", i+1)
		}
		buffers = append(buffers, queryBuffer{
			name: name,
			text: buffer.Query,
		})
	}

	activeTab := state.ActiveTab
	if activeTab < 0 || activeTab >= len(buffers) {
		activeTab = 0
	}

	m.editorView.buffers = buffers
	m.editorView.activeTab = activeTab
	m.editorView.textarea.SetValue(buffers[activeTab].text)
	m.editorView.showSuggestions = false
}

func (m *MainModel) applyWorkspaceResultsState(state config.WorkspaceResultsState) {
	if strings.TrimSpace(state.Table) == "" {
		return
	}

	m.resultsView.schema = strings.TrimSpace(state.Schema)
	m.resultsView.tableName = strings.TrimSpace(state.Table)
	m.resultsView.query = ""
	m.resultsView.currentPage = max(state.CurrentPage, 0)
	if state.PageSize > 0 {
		m.resultsView.pageSize = state.PageSize
	}
	m.resultsView.columnOffset = max(state.ColumnOffset, 0)
	m.resultsView.whereCondition = state.Where
	m.resultsView.visibleColumns = append([]string(nil), state.VisibleColumns...)
}

func (m *MainModel) workspaceEditorState() config.WorkspaceEditorState {
	buffers := make([]config.WorkspaceEditorBufferState, len(m.editorView.buffers))
	for i, buffer := range m.editorView.buffers {
		buffers[i] = config.WorkspaceEditorBufferState{
			Name:  buffer.name,
			Query: buffer.text,
		}
	}

	return config.WorkspaceEditorState{
		Buffers:   buffers,
		ActiveTab: m.editorView.activeTab,
	}
}

func (m *MainModel) workspaceResultsState() config.WorkspaceResultsState {
	if strings.TrimSpace(m.resultsView.tableName) == "" {
		return config.WorkspaceResultsState{}
	}

	var where *model.WhereCondition
	if m.resultsView.whereCondition != nil {
		where = cloneWhereCondition(m.resultsView.whereCondition)
	}

	return config.WorkspaceResultsState{
		Schema:         m.resultsView.schema,
		Table:          m.resultsView.tableName,
		CurrentPage:    m.resultsView.currentPage,
		PageSize:       m.resultsView.pageSize,
		ColumnOffset:   m.resultsView.columnOffset,
		VisibleColumns: append([]string(nil), m.resultsView.visibleColumns...),
		Where:          where,
	}
}

func (m *MainModel) workspaceViewMode() ViewMode {
	if isWorkspaceBaseView(m.mode) {
		return m.mode
	}

	for i := len(m.viewHistory) - 1; i >= 0; i-- {
		if isWorkspaceBaseView(m.viewHistory[i]) {
			return m.viewHistory[i]
		}
	}

	if strings.TrimSpace(m.resultsView.tableName) != "" {
		return ViewResults
	}
	return ViewBrowser
}

func isWorkspaceBaseView(mode ViewMode) bool {
	switch mode {
	case ViewBrowser, ViewEditor, ViewResults:
		return true
	default:
		return false
	}
}

func workspaceViewName(mode ViewMode) string {
	switch mode {
	case ViewEditor:
		return "editor"
	case ViewResults:
		return "results"
	default:
		return "browser"
	}
}

func parseWorkspaceView(value string) ViewMode {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "editor":
		return ViewEditor
	case "results":
		return ViewResults
	default:
		return ViewBrowser
	}
}

func cloneWhereCondition(condition *model.WhereCondition) *model.WhereCondition {
	if condition == nil {
		return nil
	}

	clone := &model.WhereCondition{
		Type: condition.Type,
	}

	if condition.Atomic != nil {
		clone.Atomic = &model.AtomicWhereCondition{
			ColumnType: condition.Atomic.ColumnType,
			Key:        condition.Atomic.Key,
			Operator:   condition.Atomic.Operator,
			Value:      condition.Atomic.Value,
		}
	}

	if condition.And != nil {
		clone.And = cloneOperationWhereCondition(condition.And)
	}

	if condition.Or != nil {
		clone.Or = cloneOperationWhereCondition(condition.Or)
	}

	return clone
}

func cloneOperationWhereCondition(condition *model.OperationWhereCondition) *model.OperationWhereCondition {
	if condition == nil {
		return nil
	}

	clone := &model.OperationWhereCondition{}
	if len(condition.Children) > 0 {
		clone.Children = make([]*model.WhereCondition, len(condition.Children))
		for i, child := range condition.Children {
			clone.Children[i] = cloneWhereCondition(child)
		}
	}

	return clone
}
