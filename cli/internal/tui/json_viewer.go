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
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/pkg/styles"
)

// JSONViewer is a modal view that displays cell content with JSON pretty-printing
// and syntax highlighting. It is opened from the Results view by pressing `z`.
type JSONViewer struct {
	parent     *MainModel
	viewport   viewport.Model
	columnName string
	rawContent string
	width      int
	height     int
	ready      bool
}

// NewJSONViewer creates a new JSONViewer attached to the given parent model.
func NewJSONViewer(parent *MainModel) *JSONViewer {
	return &JSONViewer{
		parent: parent,
	}
}

// SetContent configures the viewer with a column name and cell value.
func (v *JSONViewer) SetContent(columnName, content string) {
	v.columnName = columnName
	v.rawContent = content
	v.ready = false
}

// Update handles input for the JSON viewer modal.
func (v *JSONViewer) Update(msg tea.Msg) (*JSONViewer, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.initViewport()
		return v, nil

	case tea.KeyMsg:
		if key.Matches(msg, Keys.Global.Back) {
			if !v.parent.PopView() {
				v.parent.mode = ViewResults
			}
			return v, nil
		}
	}

	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// View renders the JSON viewer modal.
func (v *JSONViewer) View() string {
	if !v.ready {
		v.initViewport()
	}

	var b strings.Builder

	title := "Cell Viewer"
	if v.columnName != "" {
		title += " — " + v.columnName
	}
	b.WriteString(styles.RenderTitle(title))
	b.WriteString("\n\n")
	b.WriteString(v.viewport.View())
	b.WriteString("\n\n")

	b.WriteString(styles.RenderHelp(
		"↑/↓", "scroll",
		"esc", "close",
	))

	if v.viewport.TotalLineCount() > v.viewport.VisibleLineCount() {
		pct := v.viewport.ScrollPercent() * 100
		var scrollPct string
		if pct >= 99.5 {
			scrollPct = "bottom"
		} else {
			scrollPct = formatFloat(pct) + "%"
		}
		b.WriteString("  ")
		b.WriteString(styles.RenderMuted(scrollPct))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

// initViewport creates (or resizes) the viewport and fills it with highlighted content.
func (v *JSONViewer) initViewport() {
	contentWidth := v.width - 8
	if contentWidth < 20 {
		contentWidth = 20
	}
	contentHeight := v.height - 10
	if contentHeight < 3 {
		contentHeight = 3
	}

	v.viewport = viewport.New(contentWidth, contentHeight)
	v.viewport.SetContent(highlightJSON(v.rawContent))
	v.ready = true
}

// highlightJSON attempts to pretty-print and syntax-highlight the input as JSON.
// If the input is not valid JSON, it is returned as-is.
func highlightJSON(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return styles.MutedStyle.Render("(empty)")
	}

	// Attempt JSON pretty-print
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(trimmed), "", "  "); err != nil {
		// Not valid JSON — return raw text
		return raw
	}

	return colorizeJSON(buf.String())
}

// colorizeJSON applies syntax highlighting to pre-formatted JSON text.
// Keys are rendered with KeywordStyle, strings with StringStyle,
// numbers with NumberStyle, and booleans/null with InfoStyle.
func colorizeJSON(formatted string) string {
	var out strings.Builder
	runes := []rune(formatted)
	i := 0

	for i < len(runes) {
		ch := runes[i]

		switch {
		// String literal (key or value)
		case ch == '"':
			str, end := consumeJSONString(runes, i)
			// Determine if this is a key: look ahead past whitespace for ':'
			isKey := false
			for j := end; j < len(runes); j++ {
				if runes[j] == ':' {
					isKey = true
					break
				}
				if !unicode.IsSpace(runes[j]) {
					break
				}
			}
			if isKey {
				out.WriteString(styles.KeywordStyle.Render(str))
			} else {
				out.WriteString(styles.StringStyle.Render(str))
			}
			i = end

		// Number
		case ch == '-' || (ch >= '0' && ch <= '9'):
			start := i
			for i < len(runes) && isNumberRune(runes[i]) {
				i++
			}
			out.WriteString(styles.NumberStyle.Render(string(runes[start:i])))

		// Boolean or null
		case ch == 't' || ch == 'f' || ch == 'n':
			word, end := consumeKeyword(runes, i)
			if word == "true" || word == "false" || word == "null" {
				out.WriteString(lipgloss.NewStyle().Foreground(styles.Info).Render(word))
				i = end
			} else {
				out.WriteRune(ch)
				i++
			}

		default:
			out.WriteRune(ch)
			i++
		}
	}

	return out.String()
}

// consumeJSONString reads a JSON string starting at position i (which must be '"')
// and returns the full string (including quotes) and the position after the closing quote.
func consumeJSONString(runes []rune, i int) (string, int) {
	j := i + 1
	for j < len(runes) {
		if runes[j] == '\\' {
			j += 2
			continue
		}
		if runes[j] == '"' {
			j++
			return string(runes[i:j]), j
		}
		j++
	}
	// Unterminated string — return what we have
	return string(runes[i:]), len(runes)
}

// consumeKeyword reads an alphabetic keyword starting at position i.
func consumeKeyword(runes []rune, i int) (string, int) {
	j := i
	for j < len(runes) && ((runes[j] >= 'a' && runes[j] <= 'z') || (runes[j] >= 'A' && runes[j] <= 'Z')) {
		j++
	}
	return string(runes[i:j]), j
}

// isNumberRune returns true for runes that can appear in a JSON number.
func isNumberRune(r rune) bool {
	return (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '+' || r == 'e' || r == 'E'
}

// formatFloat formats a float64 as a compact percentage string (e.g. "42.5").
func formatFloat(f float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", f), "0"), ".")
}
