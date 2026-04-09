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

import tea "github.com/charmbracelet/bubbletea"

// Compile-time interface compliance checks.
var (
	_ Pane = (*BrowserView)(nil)
	_ Pane = (*ConnectionView)(nil)
	_ Pane = (*EditorView)(nil)
	_ Pane = (*ResultsView)(nil)
	_ Pane = (*HistoryView)(nil)
	_ Pane = (*ExportView)(nil)
	_ Pane = (*WhereView)(nil)
	_ Pane = (*ColumnsView)(nil)
	_ Pane = (*ChatView)(nil)
	_ Pane = (*SchemaView)(nil)
)

// ---------------------------------------------------------------------------
// BrowserView
// ---------------------------------------------------------------------------

func (v *BrowserView) UpdatePane(msg tea.Msg) tea.Cmd {
	_, cmd := v.Update(msg)
	return cmd
}

func (v *BrowserView) SetDimensions(width, height int) {
	v.width = width
	v.height = height
}

func (v *BrowserView) Focusable() bool { return true }
func (v *BrowserView) OnFocus()        {}
func (v *BrowserView) OnBlur()         {}

// ---------------------------------------------------------------------------
// ConnectionView
// ---------------------------------------------------------------------------

func (v *ConnectionView) UpdatePane(msg tea.Msg) tea.Cmd {
	_, cmd := v.Update(msg)
	return cmd
}

func (v *ConnectionView) SetDimensions(width, height int) {
	v.width = width
	v.height = height
}

func (v *ConnectionView) Focusable() bool { return true }
func (v *ConnectionView) OnFocus()        {}
func (v *ConnectionView) OnBlur()         {}

// ---------------------------------------------------------------------------
// EditorView
// ---------------------------------------------------------------------------

func (v *EditorView) UpdatePane(msg tea.Msg) tea.Cmd {
	_, cmd := v.Update(msg)
	return cmd
}

func (v *EditorView) SetDimensions(width, height int) {
	v.width = width
	v.height = height
}

func (v *EditorView) Focusable() bool { return true }
func (v *EditorView) OnFocus()        { v.textarea.Focus() }
func (v *EditorView) OnBlur()         { v.textarea.Blur() }

// ---------------------------------------------------------------------------
// ResultsView
// ---------------------------------------------------------------------------

func (v *ResultsView) UpdatePane(msg tea.Msg) tea.Cmd {
	_, cmd := v.Update(msg)
	return cmd
}

func (v *ResultsView) SetDimensions(width, height int) {
	v.width = width
	v.height = height
}

func (v *ResultsView) Focusable() bool { return true }
func (v *ResultsView) OnFocus()        {}
func (v *ResultsView) OnBlur()         {}

// ---------------------------------------------------------------------------
// HistoryView
// ---------------------------------------------------------------------------

func (v *HistoryView) UpdatePane(msg tea.Msg) tea.Cmd {
	_, cmd := v.Update(msg)
	return cmd
}

func (v *HistoryView) SetDimensions(width, height int) {
	v.width = width
	v.height = height
}

func (v *HistoryView) Focusable() bool { return true }
func (v *HistoryView) OnFocus()        { v.refreshList() }
func (v *HistoryView) OnBlur()         {}

// ---------------------------------------------------------------------------
// ExportView
// ---------------------------------------------------------------------------

func (v *ExportView) UpdatePane(msg tea.Msg) tea.Cmd {
	_, cmd := v.Update(msg)
	return cmd
}

func (v *ExportView) SetDimensions(width, height int) {
	v.width = width
	v.height = height
}

func (v *ExportView) Focusable() bool { return true }
func (v *ExportView) OnFocus()        {}
func (v *ExportView) OnBlur()         {}

// ---------------------------------------------------------------------------
// WhereView
// ---------------------------------------------------------------------------

func (v *WhereView) UpdatePane(msg tea.Msg) tea.Cmd {
	_, cmd := v.Update(msg)
	return cmd
}

func (v *WhereView) SetDimensions(width, height int) {
	v.width = width
	v.height = height
}

func (v *WhereView) Focusable() bool { return true }
func (v *WhereView) OnFocus()        {}
func (v *WhereView) OnBlur()         {}

// ---------------------------------------------------------------------------
// ColumnsView
// ---------------------------------------------------------------------------

func (v *ColumnsView) UpdatePane(msg tea.Msg) tea.Cmd {
	_, cmd := v.Update(msg)
	return cmd
}

func (v *ColumnsView) SetDimensions(width, height int) {
	v.width = width
	v.height = height
}

func (v *ColumnsView) Focusable() bool { return true }
func (v *ColumnsView) OnFocus()        {}
func (v *ColumnsView) OnBlur()         {}

// ---------------------------------------------------------------------------
// ChatView
// ---------------------------------------------------------------------------

func (v *ChatView) UpdatePane(msg tea.Msg) tea.Cmd {
	_, cmd := v.Update(msg)
	return cmd
}

func (v *ChatView) SetDimensions(width, height int) {
	v.width = width
	v.height = height
}

func (v *ChatView) Focusable() bool { return true }
func (v *ChatView) OnFocus()        {}
func (v *ChatView) OnBlur()         {}

// ---------------------------------------------------------------------------
// SchemaView
// ---------------------------------------------------------------------------

func (v *SchemaView) UpdatePane(msg tea.Msg) tea.Cmd {
	_, cmd := v.Update(msg)
	return cmd
}

func (v *SchemaView) SetDimensions(width, height int) {
	v.width = width
	v.height = height
}

func (v *SchemaView) Focusable() bool { return true }
func (v *SchemaView) OnFocus()        {}
func (v *SchemaView) OnBlur()         {}
