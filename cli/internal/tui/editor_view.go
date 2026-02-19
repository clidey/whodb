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

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/pkg/styles"
)

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
	lastWidth           int
	lastHeight          int
	suggestionHeight    int
	// Query execution state for timeout and cancellation support
	queryState  OperationState
	queryCancel context.CancelFunc
	retryPrompt RetryPrompt
	// Debounce autocomplete - sequence ID to detect stale debounce messages
	autocompleteSeqID int
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

	b.WriteString(styles.RenderTitle("SQL Editor"))
	b.WriteString("\n\n")

	// Show loading indicator when query is running
	if v.queryState == OperationRunning {
		b.WriteString(v.parent.SpinnerView() + styles.MutedStyle.Render(" Executing query... Press ESC to cancel"))
		b.WriteString("\n\n")
	}

	b.WriteString(v.textarea.View())
	b.WriteString("\n")

	if v.err != nil {
		b.WriteString("\n\n")
		b.WriteString(styles.RenderErrorBox(v.err.Error()))
	}

	// Show retry prompt for timed out queries
	if v.retryPrompt.IsActive() {
		b.WriteString("\n\n")
		b.WriteString(styles.KeyStyle.Render("Retry with longer timeout?"))
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

	b.WriteString("\n\n")
	b.WriteString(RenderBindingHelp(
		Keys.Editor.Execute,
		Keys.Editor.Autocomplete,
		Keys.Editor.Clear,
		Keys.Global.NextView,
		Keys.Global.Back,
		Keys.Global.Quit,
	))

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

	overhead := 14
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

func (v *EditorView) refreshLayout() {
	if v.lastWidth > 0 && v.lastHeight > 0 {
		v.applyWindowSize(v.lastWidth, v.lastHeight)
	}
}
