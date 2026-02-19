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

package styles

import (
	"os"
	"runtime"
	"sync/atomic"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Platform-aware keyboard shortcut labels
var (
	// KeyExecute is the shortcut to execute a query (Alt+Enter / Option+Enter)
	KeyExecute string
	// KeyExecuteDesc is the description for the execute shortcut
	KeyExecuteDesc = "run query"
)

func init() {
	// Set platform-appropriate shortcut labels
	if runtime.GOOS == "darwin" {
		KeyExecute = "opt+enter"
	} else {
		KeyExecute = "alt+enter"
	}
}

var colorDisabled atomic.Bool

func init() {
	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		colorDisabled.Store(true)
	}
	if os.Getenv("TERM") == "dumb" {
		colorDisabled.Store(true)
	}
	if colorDisabled.Load() {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
}

func DisableColor() {
	colorDisabled.Store(true)
	lipgloss.SetColorProfile(termenv.Ascii)
}

func ColorEnabled() bool {
	return !colorDisabled.Load()
}

var (
	Primary   = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#fafafa"}
	Secondary = lipgloss.AdaptiveColor{Light: "#6b6b73", Dark: "#a1a1aa"}
	Success   = lipgloss.AdaptiveColor{Light: "#16a34a", Dark: "#22c55e"}
	Error     = lipgloss.AdaptiveColor{Light: "#dc2626", Dark: "#ef4444"}
	Warning   = lipgloss.AdaptiveColor{Light: "#d97706", Dark: "#f59e0b"}
	Info      = lipgloss.AdaptiveColor{Light: "#2563eb", Dark: "#3b82f6"}
	Muted     = lipgloss.AdaptiveColor{Light: "#a1a1aa", Dark: "#71717a"}

	Background = lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#09090b"}
	Foreground = lipgloss.AdaptiveColor{Light: "#09090b", Dark: "#fafafa"}
	Border     = lipgloss.AdaptiveColor{Light: "#d4d4d8", Dark: "#27272a"}
	Accent     = lipgloss.AdaptiveColor{Light: "#f4f4f5", Dark: "#18181b"}
)

var (
	BaseStyle = lipgloss.NewStyle().
			Foreground(Foreground).
			Background(Background)

	TitleStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			MarginBottom(1)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Border).
			Padding(1, 2)

	ActiveBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2)

	ListItemStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Padding(0, 2)

	ActiveListItemStyle = lipgloss.NewStyle().
				Foreground(Primary).
				Background(Accent).
				Padding(0, 2)

	StatusBarStyle = lipgloss.NewStyle().
			Background(Accent).
			Foreground(Primary).
			Padding(0, 1)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(Info).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	MutedStyle = lipgloss.NewStyle().
			Foreground(Muted)

	HelpStyle = lipgloss.NewStyle().
			Foreground(Muted).
			MarginTop(1)

	KeyStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	ValueStyle = lipgloss.NewStyle().
			Foreground(Foreground)

	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(Primary).
				Bold(true).
				BorderBottom(true).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(Border)

	TableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	CodeStyle = lipgloss.NewStyle().
			Background(Accent).
			Foreground(Foreground).
			Padding(1, 2).
			MarginBottom(1)

	KeywordStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#3f3f46", Dark: "#d4d4d8"})

	StringStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#6b6b73", Dark: "#a1a1aa"})

	CommentStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	NumberStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#3f3f46", Dark: "#d4d4d8"})
)

func RenderTitle(title string) string {
	return TitleStyle.Render(title)
}

func RenderSubtitle(subtitle string) string {
	return SubtitleStyle.Render(subtitle)
}

func RenderBox(content string) string {
	return BoxStyle.Render(content)
}

func RenderActiveBox(content string) string {
	return ActiveBoxStyle.Render(content)
}

func RenderError(message string) string {
	return ErrorStyle.Render("✗ " + message)
}

func RenderSuccess(message string) string {
	return SuccessStyle.Render("✓ " + message)
}

func RenderHelp(keys ...string) string {
	return RenderHelpWidth(80, keys...)
}

// RenderHelpWidth renders help key/value pairs, wrapping at the given maxWidth.
func RenderHelpWidth(maxWidth int, keys ...string) string {
	if len(keys) == 0 {
		return ""
	}

	var parts []string
	for i := 0; i < len(keys); i += 2 {
		if i+1 < len(keys) {
			keyPart := KeyStyle.Render(keys[i])
			descPart := MutedStyle.Render(keys[i+1])
			parts = append(parts, keyPart+" "+descPart)
		}
	}

	return RenderHelpPartsWidth(parts, maxWidth)
}

func RenderHelpParts(parts []string) string {
	return RenderHelpPartsWidth(parts, 80)
}

// RenderHelpPartsWidth renders help parts wrapping at the given maxWidth.
func RenderHelpPartsWidth(parts []string, maxWidth int) string {
	if len(parts) == 0 {
		return ""
	}

	separator := MutedStyle.Render(" • ")
	maxLineWidth := maxWidth

	var lines []string
	var currentLine string
	var currentWidth int

	for i, part := range parts {
		partWidth := lipgloss.Width(part)
		sepWidth := 0
		if i > 0 {
			sepWidth = 3
		}

		if currentWidth+sepWidth+partWidth > maxLineWidth && currentLine != "" {
			lines = append(lines, currentLine)
			currentLine = part
			currentWidth = partWidth
		} else {
			if currentLine != "" {
				currentLine += separator
				currentWidth += sepWidth
			}
			currentLine += part
			currentWidth += partWidth
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += HelpStyle.Render(line)
	}

	return result
}

func RenderHelpWithMaxItems(maxPerLine int, keys ...string) string {
	if len(keys) == 0 {
		return ""
	}

	var parts []string
	for i := 0; i < len(keys); i += 2 {
		if i+1 < len(keys) {
			keyPart := KeyStyle.Render(keys[i])
			descPart := MutedStyle.Render(keys[i+1])
			parts = append(parts, keyPart+" "+descPart)
		}
	}

	separator := MutedStyle.Render(" • ")
	var lines []string
	var currentLine []string

	for i, part := range parts {
		currentLine = append(currentLine, part)
		if (i+1)%maxPerLine == 0 || i == len(parts)-1 {
			lineStr := ""
			for j, p := range currentLine {
				if j > 0 {
					lineStr += separator
				}
				lineStr += p
			}
			lines = append(lines, HelpStyle.Render(lineStr))
			currentLine = nil
		}
	}

	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}

	return result
}

func RenderErrorBox(message string) string {
	return RenderErrorBoxWidth(message, 64)
}

// RenderErrorBoxWidth renders an error box clamped to the given maxWidth.
func RenderErrorBoxWidth(message string, maxWidth int) string {
	w := maxWidth - 4
	if w > 80 {
		w = 80
	}
	if w < 30 {
		w = 30
	}

	errorBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Error).
		Padding(1, 2).
		Width(w)

	errorTitleStyle := lipgloss.NewStyle().
		Foreground(Error).
		Bold(true).
		MarginBottom(1)

	errorDescStyle := lipgloss.NewStyle().
		Foreground(Foreground)

	content := errorTitleStyle.Render("Error") + "\n" + errorDescStyle.Render(message)
	return errorBoxStyle.Render(content)
}

func RenderInfoBox(message string) string {
	return RenderInfoBoxWidth(message, 64)
}

// RenderInfoBoxWidth renders an info box clamped to the given maxWidth.
func RenderInfoBoxWidth(message string, maxWidth int) string {
	w := maxWidth - 4
	if w > 80 {
		w = 80
	}
	if w < 30 {
		w = 30
	}

	infoBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Info).
		Padding(1, 2).
		Width(w)

	infoTitleStyle := lipgloss.NewStyle().
		Foreground(Info).
		Bold(true).
		MarginBottom(1)

	infoDescStyle := lipgloss.NewStyle().
		Foreground(Foreground)

	content := infoTitleStyle.Render("Info") + "\n" + infoDescStyle.Render(message)
	return infoBoxStyle.Render(content)
}

// RenderMuted renders text in muted style.
func RenderMuted(s string) string { return MutedStyle.Render(s) }

// RenderKey renders text in key/bold style.
func RenderKey(s string) string { return KeyStyle.Render(s) }

// RenderErr renders text in error style.
func RenderErr(s string) string { return ErrorStyle.Render(s) }

// RenderOk renders text in success style.
func RenderOk(s string) string { return SuccessStyle.Render(s) }

