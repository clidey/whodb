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
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/pkg/styles"
)

// queryBuffer holds the content of one editor tab.
type queryBuffer struct {
	name string
	text string
}

type EditorView struct {
	parent              *MainModel
	textarea            textarea.Model
	err                 error
	allSuggestions      []suggestion
	filteredSuggestions []suggestion
	showSuggestions     bool
	selectedSuggestion  int
	cursorPos           int
	lastText            string
	width               int
	height              int
	lastWidth           int
	lastHeight          int
	suggestionHeight    int
	// Query execution state for timeout and cancellation support
	queryState  OperationState
	queryCancel context.CancelFunc
	retryPrompt RetryPrompt
	// Debounce autocomplete - sequence ID to detect stale debounce messages
	autocompleteSeqID int
	compact           bool
	// Multi-tab query buffers
	buffers   []queryBuffer
	activeTab int
}

func NewEditorView(parent *MainModel) *EditorView {
	ta := textarea.New()
	ta.Placeholder = "Enter SQL query..."
	ta.Focus()
	ta.SetHeight(10)
	ta.SetWidth(80)
	ta.CharLimit = 0

	return &EditorView{
		parent:              parent,
		textarea:            ta,
		allSuggestions:      []suggestion{},
		filteredSuggestions: []suggestion{},
		selectedSuggestion:  0,
		buffers:             []queryBuffer{{name: "Query 1", text: ""}},
		activeTab:           0,
	}
}

func (v *EditorView) Update(msg tea.Msg) (*EditorView, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case QueryExecutedMsg:
		v.queryState = OperationIdle
		v.queryCancel = nil
		if msg.Err != nil {
			v.err = msg.Err
			conn := v.parent.dbManager.GetCurrentConnection()
			dbName := ""
			if conn != nil {
				dbName = conn.Database
			}
			v.parent.histMgr.Add(msg.Query, false, dbName)
			return v, nil
		}
		conn := v.parent.dbManager.GetCurrentConnection()
		dbName := ""
		if conn != nil {
			dbName = conn.Database
		}
		v.parent.histMgr.Add(msg.Query, true, dbName)
		v.parent.resultsView.SetResults(msg.Result, msg.Query)
		v.parent.PushView(ViewResults)
		v.err = nil
		rowCount := 0
		if msg.Result != nil {
			rowCount = len(msg.Result.Rows)
		}
		return v, v.parent.SetStatus(fmt.Sprintf("Query executed (%d rows)", rowCount))

	case QueryTimeoutMsg:
		v.queryState = OperationIdle
		v.queryCancel = nil
		// Auto-retry with saved preference before showing menu
		preferred := v.parent.config.GetPreferredTimeout()
		if preferred > 0 && !v.retryPrompt.AutoRetried() {
			v.retryPrompt.SetAutoRetried(true)
			return v, v.executeQueryWithTimeout(msg.Query, time.Duration(preferred)*time.Second)
		}
		v.err = fmt.Errorf("query timed out after %s", msg.Timeout)
		v.retryPrompt.Show(msg.Query)
		return v, nil

	case QueryCancelledMsg:
		v.queryState = OperationIdle
		v.queryCancel = nil
		// Don't show error for user-initiated cancel
		return v, nil

	case AutocompleteDebounceMsg:
		// Only process if sequence ID matches (not stale)
		if msg.SeqID == v.autocompleteSeqID {
			v.updateAutocomplete(msg.Text, msg.Pos)
		}
		return v, nil

	case externalEditorResultMsg:
		if msg.err != nil {
			v.err = msg.err
		} else {
			v.textarea.SetValue(strings.TrimRight(msg.content, "\n"))
			v.showSuggestions = false
			v.err = nil
		}
		v.refreshLayout()
		return v, nil

	case tea.WindowSizeMsg:
		v.applyWindowSize(msg.Width, msg.Height)
		return v, nil

	case tea.MouseMsg:
		if v.showSuggestions && len(v.filteredSuggestions) > 0 {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				v.selectedSuggestion--
				if v.selectedSuggestion < 0 {
					v.selectedSuggestion = len(v.filteredSuggestions) - 1
				}
				return v, nil
			case tea.MouseButtonWheelDown:
				v.selectedSuggestion = (v.selectedSuggestion + 1) % len(v.filteredSuggestions)
				return v, nil
			}
		}

	case tea.KeyMsg:
		// Handle retry prompt for timed out queries
		if v.retryPrompt.IsActive() {
			result, handled := v.retryPrompt.HandleKeyMsg(msg.String())
			if handled {
				if result != nil {
					v.err = nil
					if result.Save {
						v.parent.config.SetPreferredTimeout(int(result.Timeout.Seconds()))
						v.parent.config.Save()
					}
					return v, v.executeQueryWithTimeout(v.retryPrompt.TimedOutQuery(), result.Timeout)
				}
				return v, nil
			}
		}

		// IMPORTANT: Check for execute query shortcut FIRST before passing to textarea
		// Alt+Enter (Option+Enter on macOS) - works reliably across all platforms
		if msg.Type == tea.KeyEnter && msg.Alt {
			return v, v.executeQuery()
		}

		// Ctrl+X to run EXPLAIN on current query
		if msg.String() == "ctrl+x" {
			query := v.textarea.Value()
			if query != "" {
				v.parent.explainView.query = query
				v.parent.PushView(ViewExplain)
				return v, v.runExplain(query)
			}
			return v, nil
		}

		// Ctrl+Space to manually trigger autocomplete
		// Ctrl+@ is how Ctrl+Space is typically represented in terminals (ASCII 0)
		if msg.Type == tea.KeyCtrlAt {
			v.triggerAutocomplete()
			return v, nil
		}
		// Also check for null rune
		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 && msg.Runes[0] == 0 {
			v.triggerAutocomplete()
			return v, nil
		}

		// Ctrl+L to clear
		if msg.Type == tea.KeyCtrlL {
			v.textarea.Reset()
			v.err = nil
			v.showSuggestions = false
			v.allSuggestions = []suggestion{}
			v.filteredSuggestions = []suggestion{}
			v.selectedSuggestion = 0
			v.cursorPos = 0
			v.lastText = ""
			v.refreshLayout()
			return v, nil
		}

		// Handle autocomplete navigation when suggestions are shown
		if v.showSuggestions && len(v.filteredSuggestions) > 0 {
			switch msg.Type {
			case tea.KeyTab:
				v.selectedSuggestion = (v.selectedSuggestion + 1) % len(v.filteredSuggestions)
				return v, nil

			case tea.KeyShiftTab:
				v.selectedSuggestion--
				if v.selectedSuggestion < 0 {
					v.selectedSuggestion = len(v.filteredSuggestions) - 1
				}
				return v, nil

			case tea.KeyDown, tea.KeyCtrlN:
				v.selectedSuggestion = (v.selectedSuggestion + 1) % len(v.filteredSuggestions)
				return v, nil

			case tea.KeyUp, tea.KeyCtrlP:
				v.selectedSuggestion--
				if v.selectedSuggestion < 0 {
					v.selectedSuggestion = len(v.filteredSuggestions) - 1
				}
				return v, nil

			case tea.KeyEnter:
				if !msg.Alt {
					v.acceptSuggestion()
					return v, nil
				}
			}
		}

		switch msg.Type {
		case tea.KeyEsc:
			// If a query is running, cancel it
			if v.queryState == OperationRunning && v.queryCancel != nil {
				v.queryCancel()
				return v, nil
			}
			if v.showSuggestions {
				v.showSuggestions = false
				v.selectedSuggestion = 0
				v.refreshLayout()
				return v, nil
			}
			if !v.parent.PopView() {
				v.parent.mode = ViewBrowser
			}
			return v, nil

		case tea.KeyRight:
			// Accept ghost text when cursor is at end of text
			text := v.textarea.Value()
			li := v.textarea.LineInfo()
			atEnd := v.textarea.Line() == strings.Count(text, "\n") &&
				li.ColumnOffset >= len([]rune(strings.Split(text, "\n")[v.textarea.Line()]))
			if atEnd && v.acceptGhostText() {
				v.refreshLayout()
				return v, nil
			}
			// Fall through to let textarea handle normal right-arrow

		case tea.KeyCtrlF:
			// Format/prettify the SQL in the textarea
			text := v.textarea.Value()
			if text != "" {
				formatted := formatSQL(text)
				v.textarea.SetValue(formatted)
				v.showSuggestions = false
				v.refreshLayout()
			}
			return v, nil

		case tea.KeyCtrlO:
			// Open current query in external editor
			return v, v.openExternalEditor()

		case tea.KeyCtrlN:
			// New query tab
			v.addTab()
			v.refreshLayout()
			return v, nil

		case tea.KeyCtrlW:
			// Close current query tab
			v.closeTab()
			v.refreshLayout()
			return v, nil

		case tea.KeyShiftLeft, tea.KeyCtrlPgUp:
			// Switch to previous editor tab
			if len(v.buffers) > 1 && v.activeTab > 0 {
				v.switchToTab(v.activeTab - 1)
				v.refreshLayout()
				return v, nil
			}

		case tea.KeyShiftRight, tea.KeyCtrlPgDown:
			// Switch to next editor tab
			if len(v.buffers) > 1 && v.activeTab < len(v.buffers)-1 {
				v.switchToTab(v.activeTab + 1)
				v.refreshLayout()
				return v, nil
			}
		}
	}

	// Only schedule debounce for actual key events, not spinner ticks etc.
	_, isKeyMsg := msg.(tea.KeyMsg)

	// Pass to textarea
	v.textarea, cmd = v.textarea.Update(msg)

	// Calculate cursor position based on current line and column
	v.updateCursorPosition()

	// Schedule debounced autocomplete only when user types
	if isKeyMsg && v.textarea.Focused() {
		text := v.textarea.Value()
		// Increment sequence ID to invalidate any pending debounce
		v.autocompleteSeqID++
		seqID := v.autocompleteSeqID
		pos := v.cursorPos

		// Create debounced autocomplete command
		debounceCmd := tea.Tick(autocompleteDebounceDelay, func(t time.Time) tea.Msg {
			return AutocompleteDebounceMsg{SeqID: seqID, Text: text, Pos: pos}
		})

		// Combine textarea command with debounce command
		return v, tea.Batch(cmd, debounceCmd)
	}

	return v, cmd
}

func (v *EditorView) View() string {
	var b strings.Builder

	// Render tab bar if multiple buffers exist
	if len(v.buffers) > 1 {
		for i, buf := range v.buffers {
			if i == v.activeTab {
				b.WriteString(styles.ActiveListItemStyle.Render(buf.name))
			} else {
				b.WriteString(styles.RenderMuted(" " + buf.name + " "))
			}
			b.WriteString(" ")
		}
		b.WriteString("\n")
	} else {
		b.WriteString(styles.RenderTitle("SQL Editor"))
	}
	b.WriteString("\n\n")

	// Show loading indicator when query is running
	if v.queryState == OperationRunning {
		b.WriteString(v.parent.SpinnerView() + styles.RenderMuted(" Executing query... Press ESC to cancel"))
		b.WriteString("\n\n")
	}

	b.WriteString(v.textarea.View())

	// Ghost text: show dimmed completion from history
	ghost := v.getGhostText()
	if ghost != "" {
		b.WriteString("  " + styles.MutedStyle.Render(ghost))
	}
	b.WriteString("\n")

	if v.err != nil {
		b.WriteString("\n")
		if v.compact {
			// Inline error in compact/pane mode to avoid overflow
			b.WriteString(styles.RenderError(v.err.Error()))
		} else {
			b.WriteString("\n")
			b.WriteString(styles.RenderErrorBox(v.err.Error()))
		}
	}

	// Show retry prompt for timed out queries
	if v.retryPrompt.IsActive() {
		b.WriteString("\n\n")
		b.WriteString(styles.RenderKey("Retry with longer timeout?"))
		b.WriteString("\n")
		b.WriteString(styles.RenderHelp(
			"[1]", "60s",
			"[2]", "2min",
			"[3]", "5min",
			"[4]", "no limit",
			"esc", "cancel",
		))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	b.WriteString("\n")
	b.WriteString(v.renderSuggestionArea())

	if !v.compact {
		b.WriteString("\n\n")
		bindings := []key.Binding{
			Keys.Editor.Execute,
			Keys.Editor.Explain,
			Keys.Editor.Autocomplete,
			Keys.Editor.Format,
			Keys.Editor.OpenEditor,
			Keys.Editor.Bookmarks,
			Keys.Editor.NewTab,
		}
		if len(v.buffers) > 1 {
			bindings = append(bindings, Keys.Editor.PrevTab, Keys.Editor.NextTab, Keys.Editor.CloseTab)
		}
		bindings = append(bindings,
			Keys.Editor.Clear,
			Keys.Global.NextView,
			Keys.Global.Back,
			Keys.Global.Quit,
		)
		b.WriteString(RenderBindingHelpWidth(v.width, bindings...))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (v *EditorView) executeQuery() tea.Cmd {
	query := v.textarea.Value()
	if query == "" {
		// Return error message immediately
		return func() tea.Msg {
			return QueryExecutedMsg{Err: fmt.Errorf("query is empty"), Query: ""}
		}
	}

	// Prevent executing if already running
	if v.queryState == OperationRunning {
		return nil
	}

	// Reset auto-retry for new queries
	v.retryPrompt.SetAutoRetried(false)

	// Set loading state
	v.queryState = OperationRunning

	// Get timeout from config
	timeout := v.parent.config.GetQueryTimeout()

	// Create context with timeout and cancellation
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	v.queryCancel = cancel

	return func() tea.Msg {
		defer cancel()

		result, err := v.parent.dbManager.ExecuteQueryWithContext(ctx, query)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return QueryTimeoutMsg{Query: query, Timeout: timeout}
			}
			if errors.Is(err, context.Canceled) {
				return QueryCancelledMsg{Query: query}
			}
			return QueryExecutedMsg{Err: err, Query: query}
		}

		return QueryExecutedMsg{Result: result, Query: query}
	}
}

// executeQueryWithTimeout runs a query with a custom timeout duration
func (v *EditorView) executeQueryWithTimeout(query string, timeout time.Duration) tea.Cmd {
	if query == "" {
		return func() tea.Msg {
			return QueryExecutedMsg{Err: fmt.Errorf("query is empty"), Query: ""}
		}
	}

	// Prevent executing if already running
	if v.queryState == OperationRunning {
		return nil
	}

	// Set loading state
	v.queryState = OperationRunning

	// Create context with specified timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	v.queryCancel = cancel

	return func() tea.Msg {
		defer cancel()

		result, err := v.parent.dbManager.ExecuteQueryWithContext(ctx, query)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return QueryTimeoutMsg{Query: query, Timeout: timeout}
			}
			if errors.Is(err, context.Canceled) {
				return QueryCancelledMsg{Query: query}
			}
			return QueryExecutedMsg{Err: err, Query: query}
		}

		return QueryExecutedMsg{Result: result, Query: query}
	}
}

func (v *EditorView) applyWindowSize(width, height int) {
	v.lastWidth = width
	v.lastHeight = height

	v.textarea.SetWidth(width - 8)

	v.suggestionHeight = v.computeSuggestionHeight(height)

	// Title (2) + padding (4) + error/retry/running status
	overhead := 6
	if !v.compact {
		overhead += 4 // help footer
	}
	if v.err != nil {
		overhead += 4
	}
	if v.retryPrompt.IsActive() {
		overhead += 4
	}
	if v.queryState == OperationRunning {
		overhead += 2
	}
	targetHeight := height - overhead - v.suggestionHeight
	if targetHeight < 5 {
		targetHeight = 5
	}
	v.textarea.SetHeight(targetHeight)
}

// getGhostText returns the remaining text from a matching history entry,
// or empty string if no match. Only shows on a single-line query.
func (v *EditorView) getGhostText() string {
	text := v.textarea.Value()
	if text == "" || strings.Contains(text, "\n") {
		return ""
	}
	// Don't show ghost when autocomplete is active
	if v.showSuggestions {
		return ""
	}
	match := v.parent.histMgr.SearchByPrefix(text)
	if match == nil {
		return ""
	}
	// Return only the suffix beyond what's already typed
	return match.Query[len(text):]
}

// acceptGhostText fills in the ghost text from history.
func (v *EditorView) acceptGhostText() bool {
	ghost := v.getGhostText()
	if ghost == "" {
		return false
	}
	match := v.parent.histMgr.SearchByPrefix(v.textarea.Value())
	if match == nil {
		return false
	}
	v.textarea.SetValue(match.Query)
	v.showSuggestions = false
	return true
}

// saveCurrentBuffer stores the textarea content into the active buffer.
func (v *EditorView) saveCurrentBuffer() {
	if v.activeTab >= 0 && v.activeTab < len(v.buffers) {
		v.buffers[v.activeTab].text = v.textarea.Value()
	}
}

// switchToTab saves the current buffer and switches to the given tab index.
func (v *EditorView) switchToTab(idx int) {
	if idx < 0 || idx >= len(v.buffers) {
		return
	}
	v.saveCurrentBuffer()
	v.activeTab = idx
	v.textarea.SetValue(v.buffers[idx].text)
	v.showSuggestions = false
}

// addTab creates a new empty buffer and switches to it.
func (v *EditorView) addTab() {
	v.saveCurrentBuffer()
	name := fmt.Sprintf("Query %d", len(v.buffers)+1)
	v.buffers = append(v.buffers, queryBuffer{name: name, text: ""})
	v.activeTab = len(v.buffers) - 1
	v.textarea.SetValue("")
	v.showSuggestions = false
}

// closeTab removes the active tab. If it's the last tab, creates a new empty one.
func (v *EditorView) closeTab() {
	if len(v.buffers) <= 1 {
		// Reset the only tab instead of removing it
		v.buffers[0].text = ""
		v.textarea.SetValue("")
		v.showSuggestions = false
		return
	}
	v.buffers = append(v.buffers[:v.activeTab], v.buffers[v.activeTab+1:]...)
	if v.activeTab >= len(v.buffers) {
		v.activeTab = len(v.buffers) - 1
	}
	v.textarea.SetValue(v.buffers[v.activeTab].text)
	v.showSuggestions = false
}

func (v *EditorView) refreshLayout() {
	if v.lastWidth > 0 && v.lastHeight > 0 {
		v.applyWindowSize(v.lastWidth, v.lastHeight)
	}
}

// externalEditorResultMsg is sent when the external editor process completes.
type externalEditorResultMsg struct {
	content string
	err     error
}

// openExternalEditor writes the current textarea content to a temp file,
// opens it in $EDITOR/$VISUAL/vi, and reads back the result.
func (v *EditorView) openExternalEditor() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	// Write current content to temp file
	tmpFile, err := os.CreateTemp("", "whodb-query-*.sql")
	if err != nil {
		return func() tea.Msg {
			return externalEditorResultMsg{err: fmt.Errorf("failed to create temp file: %w", err)}
		}
	}

	content := v.textarea.Value()
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return func() tea.Msg {
			return externalEditorResultMsg{err: fmt.Errorf("failed to write temp file: %w", err)}
		}
	}
	tmpFile.Close()
	tmpPath := tmpFile.Name()

	c := exec.Command(editor, tmpPath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.Remove(tmpPath)
		if err != nil {
			return externalEditorResultMsg{err: fmt.Errorf("editor exited with error: %w", err)}
		}
		data, readErr := os.ReadFile(tmpPath)
		if readErr != nil {
			return externalEditorResultMsg{err: fmt.Errorf("failed to read back file: %w", readErr)}
		}
		return externalEditorResultMsg{content: string(data)}
	})
}

// runExplain returns a tea.Cmd that executes EXPLAIN on the given query
// and sends the result as an explainResultMsg.
func (v *EditorView) runExplain(query string) tea.Cmd {
	mgr := v.parent.dbManager
	return func() tea.Msg {
		result, err := mgr.ExecuteExplain(query)
		if err != nil {
			return explainResultMsg{query: query, err: err}
		}

		// Flatten result rows into a single plan string
		var lines []string
		for _, row := range result.Rows {
			lines = append(lines, strings.Join(row, " "))
		}
		plan := strings.Join(lines, "\n")
		return explainResultMsg{query: query, plan: plan}
	}
}
