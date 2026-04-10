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
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/clidey/whodb/cli/pkg/styles"
)

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
	_ Pane = (*ImportView)(nil)
	_ Pane = (*BookmarksView)(nil)
	_ Pane = (*JSONViewer)(nil)
	_ Pane = (*CmdLogView)(nil)
	_ Pane = (*ExplainView)(nil)
	_ Pane = (*ERDView)(nil)
)

// ---------------------------------------------------------------------------
// BrowserView
// ---------------------------------------------------------------------------

func (v *BrowserView) UpdatePane(msg tea.Msg) tea.Cmd { _, cmd := v.Update(msg); return cmd }
func (v *BrowserView) SetDimensions(width, height int) {
	v.width = width
	v.height = height
	columnWidth := 25
	available := width - 8
	v.columnsPerRow = clamp(available/columnWidth, 1, 6)
	v.filterInput.Width = clamp(width-20, 15, 50)
}
func (v *BrowserView) Focusable() bool   { return true }
func (v *BrowserView) OnFocus()          {}
func (v *BrowserView) OnBlur()           {}
func (v *BrowserView) SetCompact(c bool) { v.compact = c }
func (v *BrowserView) HelpBindings() []key.Binding {
	bindings := []key.Binding{
		Keys.Browser.Up, Keys.Browser.Down, Keys.Browser.Left, Keys.Browser.Right,
		Keys.Browser.Select, Keys.Browser.Filter,
	}
	if len(v.schemas) > 1 {
		bindings = append(bindings, Keys.Browser.Schema)
	}
	bindings = append(bindings,
		Keys.Browser.Editor, Keys.Browser.History, Keys.Browser.Refresh,
	)
	return bindings
}

// ---------------------------------------------------------------------------
// ConnectionView
// ---------------------------------------------------------------------------

func (v *ConnectionView) UpdatePane(msg tea.Msg) tea.Cmd  { _, cmd := v.Update(msg); return cmd }
func (v *ConnectionView) SetDimensions(width, height int) { v.width = width; v.height = height }
func (v *ConnectionView) Focusable() bool                 { return true }
func (v *ConnectionView) OnFocus()                        {}
func (v *ConnectionView) OnBlur()                         {}
func (v *ConnectionView) SetCompact(bool)                 {}
func (v *ConnectionView) HelpBindings() []key.Binding     { return nil }

// ---------------------------------------------------------------------------
// EditorView
// ---------------------------------------------------------------------------

func (v *EditorView) UpdatePane(msg tea.Msg) tea.Cmd { _, cmd := v.Update(msg); return cmd }
func (v *EditorView) SetDimensions(width, height int) {
	v.width = width
	v.height = height
	v.applyWindowSize(width, height)
}
func (v *EditorView) Focusable() bool   { return true }
func (v *EditorView) OnFocus()          { v.textarea.Focus() }
func (v *EditorView) OnBlur()           { v.textarea.Blur() }
func (v *EditorView) SetCompact(c bool) { v.compact = c }
func (v *EditorView) HelpBindings() []key.Binding {
	bindings := []key.Binding{
		key.NewBinding(key.WithKeys(styles.KeyExecute), key.WithHelp(styles.KeyExecute, styles.KeyExecuteDesc)),
		Keys.Editor.Explain,
		Keys.Editor.Format,
		Keys.Editor.OpenEditor,
		Keys.Editor.Bookmarks,
		Keys.Editor.NewTab,
	}
	if len(v.buffers) > 1 {
		bindings = append(bindings, Keys.Editor.PrevTab, Keys.Editor.NextTab, Keys.Editor.CloseTab)
	}
	bindings = append(bindings, Keys.Editor.Clear, Keys.Global.Back)
	return bindings
}

// ---------------------------------------------------------------------------
// ResultsView
// ---------------------------------------------------------------------------

func (v *ResultsView) UpdatePane(msg tea.Msg) tea.Cmd  { _, cmd := v.Update(msg); return cmd }
func (v *ResultsView) SetDimensions(width, height int) { v.width = width; v.height = height }
func (v *ResultsView) Focusable() bool                 { return true }
func (v *ResultsView) OnFocus()                        {}
func (v *ResultsView) OnBlur()                         {}
func (v *ResultsView) SetCompact(c bool)               { v.compact = c }
func (v *ResultsView) HelpBindings() []key.Binding {
	return []key.Binding{
		Keys.Results.NextPage, Keys.Results.ColLeft,
		Keys.Results.ViewCell, Keys.Results.Where, Keys.Results.Columns,
		Keys.Results.Export, Keys.Results.PageSize, Keys.Global.Back,
	}
}

// ---------------------------------------------------------------------------
// HistoryView
// ---------------------------------------------------------------------------

func (v *HistoryView) UpdatePane(msg tea.Msg) tea.Cmd  { _, cmd := v.Update(msg); return cmd }
func (v *HistoryView) SetDimensions(width, height int) { v.width = width; v.height = height }
func (v *HistoryView) Focusable() bool                 { return true }
func (v *HistoryView) OnFocus()                        { v.refreshList() }
func (v *HistoryView) OnBlur()                         {}
func (v *HistoryView) SetCompact(bool)                 {}
func (v *HistoryView) HelpBindings() []key.Binding     { return nil }

// ---------------------------------------------------------------------------
// ExportView
// ---------------------------------------------------------------------------

func (v *ExportView) UpdatePane(msg tea.Msg) tea.Cmd  { _, cmd := v.Update(msg); return cmd }
func (v *ExportView) SetDimensions(width, height int) { v.width = width; v.height = height }
func (v *ExportView) Focusable() bool                 { return true }
func (v *ExportView) OnFocus()                        {}
func (v *ExportView) OnBlur()                         {}
func (v *ExportView) SetCompact(bool)                 {}
func (v *ExportView) HelpBindings() []key.Binding     { return nil }

// ---------------------------------------------------------------------------
// WhereView
// ---------------------------------------------------------------------------

func (v *WhereView) UpdatePane(msg tea.Msg) tea.Cmd  { _, cmd := v.Update(msg); return cmd }
func (v *WhereView) SetDimensions(width, height int) { v.width = width; v.height = height }
func (v *WhereView) Focusable() bool                 { return true }
func (v *WhereView) OnFocus()                        {}
func (v *WhereView) OnBlur()                         {}
func (v *WhereView) SetCompact(bool)                 {}
func (v *WhereView) HelpBindings() []key.Binding     { return nil }

// ---------------------------------------------------------------------------
// ColumnsView
// ---------------------------------------------------------------------------

func (v *ColumnsView) UpdatePane(msg tea.Msg) tea.Cmd  { _, cmd := v.Update(msg); return cmd }
func (v *ColumnsView) SetDimensions(width, height int) { v.width = width; v.height = height }
func (v *ColumnsView) Focusable() bool                 { return true }
func (v *ColumnsView) OnFocus()                        {}
func (v *ColumnsView) OnBlur()                         {}
func (v *ColumnsView) SetCompact(bool)                 {}
func (v *ColumnsView) HelpBindings() []key.Binding     { return nil }

// ---------------------------------------------------------------------------
// ChatView
// ---------------------------------------------------------------------------

func (v *ChatView) UpdatePane(msg tea.Msg) tea.Cmd  { _, cmd := v.Update(msg); return cmd }
func (v *ChatView) SetDimensions(width, height int) { v.width = width; v.height = height }
func (v *ChatView) Focusable() bool                 { return true }
func (v *ChatView) OnFocus()                        {}
func (v *ChatView) OnBlur()                         {}
func (v *ChatView) SetCompact(bool)                 {}
func (v *ChatView) HelpBindings() []key.Binding     { return nil }

// ---------------------------------------------------------------------------
// SchemaView
// ---------------------------------------------------------------------------

func (v *SchemaView) UpdatePane(msg tea.Msg) tea.Cmd  { _, cmd := v.Update(msg); return cmd }
func (v *SchemaView) SetDimensions(width, height int) { v.width = width; v.height = height }
func (v *SchemaView) Focusable() bool                 { return true }
func (v *SchemaView) OnFocus()                        {}
func (v *SchemaView) OnBlur()                         {}
func (v *SchemaView) SetCompact(bool)                 {}
func (v *SchemaView) HelpBindings() []key.Binding     { return nil }

// ---------------------------------------------------------------------------
// ImportView
// ---------------------------------------------------------------------------

func (v *ImportView) UpdatePane(msg tea.Msg) tea.Cmd  { _, cmd := v.Update(msg); return cmd }
func (v *ImportView) SetDimensions(width, height int) { v.width = width; v.height = height }
func (v *ImportView) Focusable() bool                 { return true }
func (v *ImportView) OnFocus()                        {}
func (v *ImportView) OnBlur()                         {}
func (v *ImportView) SetCompact(bool)                 {}
func (v *ImportView) HelpBindings() []key.Binding     { return nil }

// ---------------------------------------------------------------------------
// JSONViewer
// ---------------------------------------------------------------------------

func (v *JSONViewer) UpdatePane(msg tea.Msg) tea.Cmd  { _, cmd := v.Update(msg); return cmd }
func (v *JSONViewer) SetDimensions(width, height int) { v.width = width; v.height = height }
func (v *JSONViewer) Focusable() bool                 { return true }
func (v *JSONViewer) OnFocus()                        {}
func (v *JSONViewer) OnBlur()                         {}
func (v *JSONViewer) SetCompact(bool)                 {}
func (v *JSONViewer) HelpBindings() []key.Binding     { return nil }

// ---------------------------------------------------------------------------
// BookmarksView
// ---------------------------------------------------------------------------

func (v *BookmarksView) UpdatePane(msg tea.Msg) tea.Cmd  { _, cmd := v.Update(msg); return cmd }
func (v *BookmarksView) SetDimensions(width, height int) { v.width = width; v.height = height }
func (v *BookmarksView) Focusable() bool                 { return true }
func (v *BookmarksView) OnFocus()                        {}
func (v *BookmarksView) OnBlur()                         {}
func (v *BookmarksView) SetCompact(bool)                 {}
func (v *BookmarksView) HelpBindings() []key.Binding     { return nil }

// ---------------------------------------------------------------------------
// CmdLogView
// ---------------------------------------------------------------------------

func (v *CmdLogView) UpdatePane(msg tea.Msg) tea.Cmd  { _, cmd := v.Update(msg); return cmd }
func (v *CmdLogView) SetDimensions(width, height int) { v.width = width; v.height = height }
func (v *CmdLogView) Focusable() bool                 { return true }
func (v *CmdLogView) OnFocus()                        {}
func (v *CmdLogView) OnBlur()                         {}
func (v *CmdLogView) SetCompact(bool)                 {}
func (v *CmdLogView) HelpBindings() []key.Binding     { return nil }

// ---------------------------------------------------------------------------
// ExplainView
// ---------------------------------------------------------------------------

func (v *ExplainView) UpdatePane(msg tea.Msg) tea.Cmd  { _, cmd := v.Update(msg); return cmd }
func (v *ExplainView) SetDimensions(width, height int) { v.width = width; v.height = height }
func (v *ExplainView) Focusable() bool                 { return true }
func (v *ExplainView) OnFocus()                        {}
func (v *ExplainView) OnBlur()                         {}
func (v *ExplainView) SetCompact(bool)                 {}
func (v *ExplainView) HelpBindings() []key.Binding     { return nil }

// ---------------------------------------------------------------------------
// ERDView
// ---------------------------------------------------------------------------

// UpdatePane wraps the ERDView's Update method for polymorphic dispatch.
func (v *ERDView) UpdatePane(msg tea.Msg) tea.Cmd { _, cmd := v.Update(msg); return cmd }

// SetDimensions sets the available width and height for the ERD view.
func (v *ERDView) SetDimensions(width, height int) { v.width = width; v.height = height }

// Focusable returns true because the ERD view can receive keyboard focus.
func (v *ERDView) Focusable() bool { return true }

// OnFocus is called when the ERD view gains keyboard focus.
func (v *ERDView) OnFocus() {}

// OnBlur is called when the ERD view loses keyboard focus.
func (v *ERDView) OnBlur() {}

// SetCompact is a no-op for the ERD view (it has its own zoom toggle).
func (v *ERDView) SetCompact(bool) {}

// HelpBindings returns the key bindings to display in the global help bar.
func (v *ERDView) HelpBindings() []key.Binding { return nil }
