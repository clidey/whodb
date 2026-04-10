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

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/pkg/styles"
)

// saveCurrentOption is a sentinel index used to represent the
// "Save current query" action at the top of the bookmarks list.
const saveCurrentOption = -1

// BookmarksView displays saved query bookmarks and allows loading,
// saving, and deleting them.
type BookmarksView struct {
	parent      *MainModel
	width       int
	height      int
	cursor      int
	editorQuery string // snapshot of editor content when bookmarks were opened
	naming      bool   // true when prompting for a bookmark name
	nameInput   textinput.Model
}

// NewBookmarksView creates a new BookmarksView.
func NewBookmarksView(parent *MainModel) *BookmarksView {
	ni := textinput.New()
	ni.Placeholder = "bookmark name"
	ni.CharLimit = 50
	ni.Width = 30
	ni.PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	ni.TextStyle = lipgloss.NewStyle().Foreground(styles.Foreground)
	ni.Cursor.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	return &BookmarksView{
		parent:    parent,
		cursor:    0,
		nameInput: ni,
	}
}

// itemCount returns the total number of selectable rows, including
// the "Save current query" option when the editor has content.
func (v *BookmarksView) itemCount() int {
	n := len(v.parent.config.GetSavedQueries())
	if v.editorQuery != "" {
		n++ // "Save current query" row at position 0
	}
	return n
}

// isSaveRow returns true when the cursor points at the save action.
func (v *BookmarksView) isSaveRow() bool {
	return v.editorQuery != "" && v.cursor == 0
}

// queryIndex translates the visual cursor into an index into the
// saved queries slice, accounting for the optional save row.
func (v *BookmarksView) queryIndex() int {
	if v.editorQuery != "" {
		return v.cursor - 1
	}
	return v.cursor
}

// nextBookmarkName generates an auto-incremented bookmark name like
// "Bookmark 1", "Bookmark 2", etc.
func (v *BookmarksView) nextBookmarkName() string {
	existing := v.parent.config.GetSavedQueries()
	max := 0
	for _, sq := range existing {
		var n int
		if _, err := fmt.Sscanf(sq.Name, "Bookmark %d", &n); err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("Bookmark %d", max+1)
}

// Update handles input for the bookmarks view.
func (v *BookmarksView) Update(msg tea.Msg) (*BookmarksView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		return v, nil

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if v.cursor > 0 {
				v.cursor--
			}
			return v, nil
		case tea.MouseButtonWheelDown:
			if v.cursor < v.itemCount()-1 {
				v.cursor++
			}
			return v, nil
		}

	case tea.KeyMsg:
		// Handle name input mode
		if v.naming {
			switch msg.String() {
			case "enter":
				name := strings.TrimSpace(v.nameInput.Value())
				if name == "" {
					name = v.nextBookmarkName()
				}
				v.parent.config.AddSavedQuery(name, v.editorQuery)
				v.parent.config.Save()
				v.naming = false
				v.nameInput.Blur()
				v.nameInput.SetValue("")
				return v, v.parent.SetStatus("Saved: " + name)
			case "esc":
				v.naming = false
				v.nameInput.Blur()
				v.nameInput.SetValue("")
				return v, nil
			default:
				v.nameInput, _ = v.nameInput.Update(msg)
				return v, nil
			}
		}

		switch {
		case key.Matches(msg, Keys.Global.Back):
			if !v.parent.PopView() {
				v.parent.mode = ViewEditor
			}
			return v, nil

		case key.Matches(msg, Keys.Bookmarks.Up):
			if v.cursor > 0 {
				v.cursor--
			}
			return v, nil

		case key.Matches(msg, Keys.Bookmarks.Down):
			if v.cursor < v.itemCount()-1 {
				v.cursor++
			}
			return v, nil

		case key.Matches(msg, Keys.Bookmarks.Load):
			if v.isSaveRow() {
				// Prompt for name
				v.naming = true
				v.nameInput.SetValue("")
				v.nameInput.Focus()
				return v, nil
			}
			// Load selected bookmark into editor
			queries := v.parent.config.GetSavedQueries()
			idx := v.queryIndex()
			if idx >= 0 && idx < len(queries) {
				v.parent.editorView.textarea.SetValue(queries[idx].Query)
				if !v.parent.PopView() {
					v.parent.mode = ViewEditor
				}
				return v, v.parent.SetStatus("Loaded: " + queries[idx].Name)
			}
			return v, nil

		case key.Matches(msg, Keys.Bookmarks.Save):
			// Save current editor query — prompt for name
			if v.editorQuery != "" {
				v.naming = true
				v.nameInput.SetValue("")
				v.nameInput.Focus()
				return v, nil
			}
			return v, nil

		case key.Matches(msg, Keys.Bookmarks.Delete):
			if v.isSaveRow() {
				return v, nil
			}
			queries := v.parent.config.GetSavedQueries()
			idx := v.queryIndex()
			if idx >= 0 && idx < len(queries) {
				name := queries[idx].Name
				v.parent.config.DeleteSavedQuery(name)
				v.parent.config.Save()
				// Adjust cursor if it fell off the end
				if v.cursor >= v.itemCount() && v.cursor > 0 {
					v.cursor--
				}
				return v, v.parent.SetStatus("Deleted: " + name)
			}
			return v, nil
		}
	}

	return v, nil
}

// View renders the bookmarks list.
func (v *BookmarksView) View() string {
	var b strings.Builder

	b.WriteString(styles.RenderTitle("Bookmarks"))
	b.WriteString("\n\n")

	// Name input prompt
	if v.naming {
		b.WriteString("  Bookmark name:\n")
		b.WriteString("  " + v.nameInput.View())
		b.WriteString("\n\n")
		b.WriteString(styles.RenderMuted("  Press Enter to save, Esc to cancel"))
		b.WriteString("\n\n")
		b.WriteString(RenderBindingHelpWidth(v.width, Keys.Global.Back))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	queries := v.parent.config.GetSavedQueries()
	row := 0

	// "Save current query" option when editor has content
	if v.editorQuery != "" {
		prefix := "  "
		if v.cursor == row {
			prefix = styles.RenderKey("> ")
		}
		label := "+ Save current query"
		if v.cursor == row {
			b.WriteString(prefix + styles.ActiveListItemStyle.Render(label))
		} else {
			b.WriteString(prefix + styles.RenderKey(label))
		}
		b.WriteString("\n\n")
		row++
	}

	if len(queries) == 0 {
		b.WriteString(styles.RenderMuted("No saved bookmarks"))
	} else {
		for i, sq := range queries {
			prefix := "  "
			if v.cursor == row+i {
				prefix = styles.RenderKey("> ")
			}

			// Truncate preview
			preview := strings.ReplaceAll(sq.Query, "\n", " ")
			maxPreview := 50
			if v.width > 30 {
				maxPreview = v.width - 30
			}
			if len(preview) > maxPreview {
				preview = preview[:maxPreview] + "..."
			}

			nameStr := sq.Name
			line := nameStr + "  " + styles.MutedStyle.Render(preview)

			if v.cursor == row+i {
				b.WriteString(prefix + styles.ActiveListItemStyle.Render(nameStr) + "  " + styles.MutedStyle.Render(preview))
			} else {
				b.WriteString(prefix + line)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n\n")

	bindings := []key.Binding{
		Keys.Bookmarks.Up,
		Keys.Bookmarks.Down,
		Keys.Bookmarks.Load,
		Keys.Bookmarks.Delete,
	}
	if v.editorQuery != "" {
		bindings = append(bindings, Keys.Bookmarks.Save)
	}
	bindings = append(bindings, Keys.Global.Back, Keys.Global.Quit)
	b.WriteString(RenderBindingHelpWidth(v.width, bindings...))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}
