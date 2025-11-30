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
	"github.com/charmbracelet/lipgloss"
)

var (
	Primary   = lipgloss.Color("#fafafa")
	Secondary = lipgloss.Color("#a1a1aa")
	Success   = lipgloss.Color("#22c55e")
	Error     = lipgloss.Color("#ef4444")
	Warning   = lipgloss.Color("#f59e0b")
	Info      = lipgloss.Color("#3b82f6")
	Muted     = lipgloss.Color("#71717a")

	Background = lipgloss.Color("#09090b")
	Foreground = lipgloss.Color("#fafafa")
	Border     = lipgloss.Color("#27272a")
	Accent     = lipgloss.Color("#18181b")
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
			Foreground(lipgloss.Color("#d4d4d8"))

	StringStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a1a1aa"))

	CommentStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	NumberStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#d4d4d8"))
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

	return RenderHelpParts(parts)
}

func RenderHelpParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}

	separator := MutedStyle.Render(" • ")
	const maxLineWidth = 80

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
	errorBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Error).
		Padding(1, 2).
		Width(60)

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
	infoBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Info).
		Padding(1, 2).
		Width(60)

	infoTitleStyle := lipgloss.NewStyle().
		Foreground(Info).
		Bold(true).
		MarginBottom(1)

	infoDescStyle := lipgloss.NewStyle().
		Foreground(Foreground)

	content := infoTitleStyle.Render("Info") + "\n" + infoDescStyle.Render(message)
	return infoBoxStyle.Render(content)
}
