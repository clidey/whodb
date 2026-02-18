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

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/internal/history"
	"github.com/clidey/whodb/cli/pkg/styles"
)

type historyItem struct {
	entry history.Entry
}

func (i historyItem) Title() string {
	query := i.entry.Query
	if len(query) > 60 {
		query = query[:60] + "..."
	}
	return query
}

func (i historyItem) Description() string {
	status := "✓"
	if !i.entry.Success {
		status = "✗"
	}
	return fmt.Sprintf("%s %s - %s", status, i.entry.Database, i.entry.Timestamp.Format("2006-01-02 15:04:05"))
}

func (i historyItem) FilterValue() string {
	return i.entry.Query
}

type HistoryView struct {
	parent          *MainModel
	list            list.Model
	confirmingClear bool
	executing       bool
	queryCancel     context.CancelFunc
	// Retry prompt state for timed out queries
	retryPrompt   bool
	timedOutQuery string
}

func NewHistoryView(parent *MainModel) *HistoryView {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Query History"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	return &HistoryView{
		parent: parent,
		list:   l,
	}
}

func (v *HistoryView) Update(msg tea.Msg) (*HistoryView, tea.Cmd) {
	switch msg := msg.(type) {
	case HistoryQueryMsg:
		v.executing = false
		v.queryCancel = nil
		if msg.Err != nil {
			// Check for timeout/cancel
			if errors.Is(msg.Err, context.DeadlineExceeded) {
				v.parent.err = fmt.Errorf("query timed out")
				// Enable retry prompt
				v.retryPrompt = true
				v.timedOutQuery = msg.Query
			} else if errors.Is(msg.Err, context.Canceled) {
				// User cancelled, don't show error
				return v, nil
			} else {
				v.parent.err = msg.Err
			}
			return v, nil
		}
		v.parent.resultsView.SetResults(msg.Result, msg.Query)
		v.parent.mode = ViewResults
		return v, nil

	case tea.WindowSizeMsg:
		overhead := 10
		h := msg.Height - overhead
		if h < 5 {
			h = 5
		}
		v.list.SetSize(msg.Width-4, h)
		return v, nil

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			v.list.CursorUp()
			return v, nil
		case tea.MouseButtonWheelDown:
			v.list.CursorDown()
			return v, nil
		}

	case tea.KeyMsg:
		// Handle retry prompt for timed out queries
		if v.retryPrompt {
			switch msg.String() {
			case "1":
				v.retryPrompt = false
				v.parent.err = nil
				return v, v.executeQueryWithTimeout(v.timedOutQuery, 60*time.Second)
			case "2":
				v.retryPrompt = false
				v.parent.err = nil
				return v, v.executeQueryWithTimeout(v.timedOutQuery, 2*time.Minute)
			case "3":
				v.retryPrompt = false
				v.parent.err = nil
				return v, v.executeQueryWithTimeout(v.timedOutQuery, 5*time.Minute)
			case "4":
				v.retryPrompt = false
				v.parent.err = nil
				return v, v.executeQueryWithTimeout(v.timedOutQuery, 24*time.Hour)
			case "esc":
				v.retryPrompt = false
				v.timedOutQuery = ""
				return v, nil
			}
			// Ignore other keys while in retry prompt
			return v, nil
		}

		switch msg.String() {
		case "enter":
			if item, ok := v.list.SelectedItem().(historyItem); ok {
				v.parent.editorView.textarea.SetValue(item.entry.Query)
				v.parent.mode = ViewEditor
				return v, nil
			}

		case "r":
			if v.executing {
				return v, nil // Already executing
			}
			if item, ok := v.list.SelectedItem().(historyItem); ok {
				v.executing = true
				query := item.entry.Query

				// Get timeout from config
				timeout := v.parent.config.GetQueryTimeout()
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				v.queryCancel = cancel

				return v, func() tea.Msg {
					defer cancel()
					result, err := v.parent.dbManager.ExecuteQueryWithContext(ctx, query)
					return HistoryQueryMsg{Result: result, Query: query, Err: err}
				}
			}

		case "D":
			// Show confirmation prompt
			if !v.confirmingClear {
				v.confirmingClear = true
				return v, nil
			}

		case "y", "Y":
			// Confirm clear if in confirmation mode
			if v.confirmingClear {
				v.parent.histMgr.Clear()
				v.refreshList()
				v.confirmingClear = false
				return v, nil
			}

		case "n", "N":
			// Cancel clear confirmation
			if v.confirmingClear {
				v.confirmingClear = false
				return v, nil
			}

		case "esc":
			// Cancel executing query first
			if v.executing && v.queryCancel != nil {
				v.queryCancel()
				return v, nil
			}
			// Cancel confirmation or go back
			if v.confirmingClear {
				v.confirmingClear = false
				return v, nil
			}
			v.parent.mode = ViewBrowser
			return v, nil
		}
	}

	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

func (v *HistoryView) View() string {
	var b strings.Builder

	b.WriteString(styles.RenderTitle("Query History"))
	b.WriteString("\n\n")

	// Show executing state
	if v.executing {
		b.WriteString(styles.MutedStyle.Render("Executing query..."))
		b.WriteString("\n")
		b.WriteString(styles.MutedStyle.Render("Press ESC to cancel"))
		b.WriteString("\n\n")
	}

	// Show retry prompt for timed out queries
	if v.retryPrompt {
		b.WriteString(styles.ErrorStyle.Render("Query timed out"))
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

	// Show confirmation dialog if clearing
	if v.confirmingClear {
		b.WriteString(styles.ErrorStyle.Render("⚠ Clear all history?"))
		b.WriteString("\n\n")
		b.WriteString(styles.MutedStyle.Render("This will delete all query history entries."))
		b.WriteString("\n\n")
		b.WriteString(styles.RenderHelp(
			"[y]", "confirm",
			"[n]/esc", "cancel",
		))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	entries := v.parent.histMgr.GetAll()
	if len(entries) == 0 {
		b.WriteString(styles.MutedStyle.Render("No history entries"))
	} else {
		b.WriteString(v.list.View())
	}

	b.WriteString("\n\n")
	b.WriteString(styles.RenderHelp(
		"↑/k", "up",
		"↓/j", "down",
		"enter", "edit",
		"r", "re-run",
		"shift+d", "clear all",
		"tab", "next view",
		"esc", "back",
		"ctrl+c", "quit",
	))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (v *HistoryView) refreshList() {
	entries := v.parent.histMgr.GetAll()
	items := make([]list.Item, len(entries))
	for i, entry := range entries {
		items[i] = historyItem{entry: entry}
	}
	v.list.SetItems(items)
}

func (v *HistoryView) Init() {
	v.refreshList()
}

func (v *HistoryView) executeQueryWithTimeout(query string, timeout time.Duration) tea.Cmd {
	v.executing = true
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	v.queryCancel = cancel

	return func() tea.Msg {
		defer cancel()
		result, err := v.parent.dbManager.ExecuteQueryWithContext(ctx, query)
		return HistoryQueryMsg{Result: result, Query: query, Err: err}
	}
}
