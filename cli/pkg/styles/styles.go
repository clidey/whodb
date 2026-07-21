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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
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

var (
	helpRenderCacheMu    sync.RWMutex
	helpRenderWidthCache = make(map[string]string)
	helpRenderPartsCache = make(map[string]string)
)

func init() {
	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		colorDisabled.Store(true)
	}
	if os.Getenv("TERM") == "dumb" {
		colorDisabled.Store(true)
	}
	if colorDisabled.Load() {
		lipgloss.Writer.Profile = colorprofile.Ascii
	}
}

func DisableColor() {
	colorDisabled.Store(true)
	lipgloss.Writer.Profile = colorprofile.Ascii
}

func ColorEnabled() bool {
	return !colorDisabled.Load()
}

var (
	Primary   = adaptive("#1a1a1a", "#fafafa")
	Secondary = adaptive("#6b6b73", "#a1a1aa")
	Success   = adaptive("#16a34a", "#22c55e")
	Error     = adaptive("#dc2626", "#ef4444")
	Warning   = adaptive("#d97706", "#f59e0b")
	Info      = adaptive("#2563eb", "#3b82f6")
	Muted     = adaptive("#a1a1aa", "#71717a")

	Background = adaptive("#ffffff", "#09090b")
	Foreground = adaptive("#09090b", "#fafafa")
	Border     = adaptive("#d4d4d8", "#27272a")
	Accent     = adaptive("#f4f4f5", "#18181b")
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

	WarningStyle = lipgloss.NewStyle().
			Foreground(Warning).
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
			Foreground(adaptive("#3f3f46", "#d4d4d8"))

	StringStyle = lipgloss.NewStyle().
			Foreground(adaptive("#6b6b73", "#a1a1aa"))

	CommentStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	NumberStyle = lipgloss.NewStyle().
			Foreground(adaptive("#3f3f46", "#d4d4d8"))
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

	cacheKey := helpWidthCacheKey(maxWidth, keys)
	helpRenderCacheMu.RLock()
	if cached, ok := helpRenderWidthCache[cacheKey]; ok {
		helpRenderCacheMu.RUnlock()
		return cached
	}
	helpRenderCacheMu.RUnlock()

	var parts []string
	for i := 0; i < len(keys); i += 2 {
		if i+1 < len(keys) {
			keyPart := KeyStyle.Render(keys[i])
			descPart := MutedStyle.Render(keys[i+1])
			parts = append(parts, keyPart+" "+descPart)
		}
	}

	result := RenderHelpPartsWidth(parts, maxWidth)

	helpRenderCacheMu.Lock()
	helpRenderWidthCache[cacheKey] = result
	helpRenderCacheMu.Unlock()

	return result
}

func RenderHelpParts(parts []string) string {
	return RenderHelpPartsWidth(parts, 80)
}

// RenderHelpPartsWidth renders help parts wrapping at the given maxWidth.
func RenderHelpPartsWidth(parts []string, maxWidth int) string {
	if len(parts) == 0 {
		return ""
	}

	cacheKey := helpPartsCacheKey(maxWidth, parts)
	helpRenderCacheMu.RLock()
	if cached, ok := helpRenderPartsCache[cacheKey]; ok {
		helpRenderCacheMu.RUnlock()
		return cached
	}
	helpRenderCacheMu.RUnlock()

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

	// Apply HelpStyle once to the entire block so MarginTop is not
	// repeated per wrapped line.
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	rendered := HelpStyle.Render(result)

	helpRenderCacheMu.Lock()
	helpRenderPartsCache[cacheKey] = rendered
	helpRenderCacheMu.Unlock()

	return rendered
}

func helpWidthCacheKey(maxWidth int, keys []string) string {
	return currentHelpThemeName() + "\x00" + renderHelpKeyWidth(maxWidth) + "\x00" + strings.Join(keys, "\x1f")
}

func helpPartsCacheKey(maxWidth int, parts []string) string {
	return currentHelpThemeName() + "\x00parts\x00" + renderHelpKeyWidth(maxWidth) + "\x00" + strings.Join(parts, "\x1f")
}

func renderHelpKeyWidth(width int) string {
	return strconv.Itoa(width)
}

func currentHelpThemeName() string {
	if theme := GetTheme(); theme != nil {
		return theme.Name
	}
	return ""
}

func clearHelpRenderCaches() {
	helpRenderCacheMu.Lock()
	defer helpRenderCacheMu.Unlock()

	helpRenderWidthCache = make(map[string]string)
	helpRenderPartsCache = make(map[string]string)
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

// RenderWarn renders text in warning style.
func RenderWarn(s string) string { return WarningStyle.Render(s) }
