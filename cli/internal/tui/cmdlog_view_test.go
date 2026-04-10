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
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clidey/whodb/cli/internal/database"
)

func TestCmdLogViewRender(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	if m.err != nil {
		t.Fatalf("NewMainModel failed: %v", m.err)
	}

	m.cmdLogView.width = 120
	m.cmdLogView.height = 40

	output := m.cmdLogView.View()
	if output == "" {
		t.Error("CmdLogView.View() returned empty string")
	}
	if !strings.Contains(output, "Command Log") {
		t.Error("expected 'Command Log' title in output")
	}
	if !strings.Contains(output, "No queries executed yet") {
		t.Error("expected empty state message when no queries logged")
	}
}

func TestCmdLogViewPaneInterface(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	if m.err != nil {
		t.Fatalf("NewMainModel failed: %v", m.err)
	}

	pane := m.GetPane(ViewCmdLog)
	if pane == nil {
		t.Fatal("Pane for ViewCmdLog is nil")
	}

	if !pane.Focusable() {
		t.Error("CmdLogView should be focusable")
	}

	pane.SetDimensions(100, 40)
	output := pane.View()
	if output == "" {
		t.Error("Pane View() returned empty after SetDimensions")
	}
}

func TestCmdLogViewEscCloses(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	if m.err != nil {
		t.Fatalf("NewMainModel failed: %v", m.err)
	}

	m.mode = ViewCmdLog
	m.viewHistory = []ViewMode{ViewBrowser}

	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	m.cmdLogView, _ = m.cmdLogView.Update(escMsg)

	if m.mode != ViewBrowser {
		t.Errorf("expected mode ViewBrowser after Esc, got %d", m.mode)
	}
}

func TestCmdLogViewWindowResize(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	if m.err != nil {
		t.Fatalf("NewMainModel failed: %v", m.err)
	}

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	m.cmdLogView, _ = m.cmdLogView.Update(msg)

	if m.cmdLogView.width != 80 {
		t.Errorf("expected width 80, got %d", m.cmdLogView.width)
	}
	if m.cmdLogView.height != 24 {
		t.Errorf("expected height 24, got %d", m.cmdLogView.height)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{500 * time.Microsecond, "500us"},
		{12 * time.Millisecond, "12ms"},
		{150 * time.Millisecond, "150ms"},
		{1500 * time.Millisecond, "1.5s"},
		{3 * time.Second, "3.0s"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestRenderLogEntry(t *testing.T) {
	entry := database.QueryLogEntry{
		Query:     "SELECT * FROM users WHERE id = 1",
		Timestamp: time.Date(2026, 1, 15, 14, 23, 5, 0, time.UTC),
		Duration:  12 * time.Millisecond,
		Success:   true,
		RowCount:  5,
	}

	result := renderLogEntry(entry, 80)
	if !strings.Contains(result, "14:23:05") {
		t.Error("expected timestamp in output")
	}
	if !strings.Contains(result, "SELECT * FROM users WHERE id = 1") {
		t.Error("expected query in output")
	}
	if !strings.Contains(result, "12ms") {
		t.Error("expected duration in output")
	}
	if !strings.Contains(result, "5 rows") {
		t.Error("expected row count in output")
	}
}

func TestRenderLogEntryError(t *testing.T) {
	entry := database.QueryLogEntry{
		Query:     "INSERT INTO logs VALUES (...)",
		Timestamp: time.Date(2026, 1, 15, 14, 23, 2, 0, time.UTC),
		Duration:  3 * time.Millisecond,
		Success:   false,
		Error:     "constraint violation",
	}

	result := renderLogEntry(entry, 80)
	if !strings.Contains(result, "ERR") {
		t.Error("expected 'ERR' in error output")
	}
	if !strings.Contains(result, "constraint violation") {
		t.Error("expected error message in output")
	}
}

func TestRenderLogEntryTruncation(t *testing.T) {
	entry := database.QueryLogEntry{
		Query:     strings.Repeat("SELECT ", 20),
		Timestamp: time.Now(),
		Duration:  1 * time.Millisecond,
		Success:   true,
	}

	result := renderLogEntry(entry, 30)
	if !strings.Contains(result, "...") {
		t.Error("expected truncation indicator for long query")
	}
}
