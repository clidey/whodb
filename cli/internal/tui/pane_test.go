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
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPaneRegistryPopulated(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	if m.err != nil {
		t.Fatalf("NewMainModel failed: %v", m.err)
	}

	expectedModes := []ViewMode{
		ViewConnection, ViewBrowser, ViewEditor, ViewResults,
		ViewHistory, ViewExport, ViewWhere, ViewColumns,
		ViewChat, ViewSchema, ViewImport,
	}

	if len(m.panes) != len(expectedModes) {
		t.Errorf("Expected %d panes, got %d", len(expectedModes), len(m.panes))
	}

	for _, mode := range expectedModes {
		pane := m.GetPane(mode)
		if pane == nil {
			t.Errorf("Pane for mode %d is nil", mode)
		}
	}
}

func TestActivePaneReturnsCurrentView(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	if m.err != nil {
		t.Fatalf("NewMainModel failed: %v", m.err)
	}

	// Default mode is ViewConnection
	pane := m.ActivePane()
	if pane == nil {
		t.Fatal("ActivePane returned nil")
	}

	// Should be the same instance as the concrete connectionView
	if pane != Pane(m.connectionView) {
		t.Error("ActivePane should return the connectionView pane")
	}
}

func TestPaneSetDimensions(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	if m.err != nil {
		t.Fatalf("NewMainModel failed: %v", m.err)
	}

	// Test SetDimensions on each pane
	for mode, pane := range m.panes {
		pane.SetDimensions(120, 40)

		// Verify the view reports non-empty output after dimensions are set
		output := pane.View()
		if output == "" {
			t.Errorf("Pane for mode %d returned empty View() after SetDimensions", mode)
		}
	}
}

func TestPaneFocusable(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	if m.err != nil {
		t.Fatalf("NewMainModel failed: %v", m.err)
	}

	for mode, pane := range m.panes {
		if !pane.Focusable() {
			t.Errorf("Pane for mode %d should be focusable", mode)
		}
	}
}

func TestPaneUpdatePaneDoesNotPanic(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	if m.err != nil {
		t.Fatalf("NewMainModel failed: %v", m.err)
	}

	// Send a WindowSizeMsg through UpdatePane to verify it doesn't panic
	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	for mode, pane := range m.panes {
		cmd := pane.UpdatePane(msg)
		// cmd may be nil, that's fine — just verify no panic
		_ = cmd
		_ = mode
	}
}

func TestPaneOnFocusOnBlurDoNotPanic(t *testing.T) {
	setupTestEnv(t)

	m := NewMainModel()
	if m.err != nil {
		t.Fatalf("NewMainModel failed: %v", m.err)
	}

	for _, pane := range m.panes {
		pane.OnFocus()
		pane.OnBlur()
	}
}
