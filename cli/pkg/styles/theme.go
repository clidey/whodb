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

import "github.com/charmbracelet/lipgloss"

// Theme defines a complete set of colors for the CLI UI.
type Theme struct {
	Name string

	// Core semantic colors
	Primary   lipgloss.AdaptiveColor
	Secondary lipgloss.AdaptiveColor
	Success   lipgloss.AdaptiveColor
	Error     lipgloss.AdaptiveColor
	Warning   lipgloss.AdaptiveColor
	Info      lipgloss.AdaptiveColor
	Muted     lipgloss.AdaptiveColor

	// UI chrome colors
	Background lipgloss.AdaptiveColor
	Foreground lipgloss.AdaptiveColor
	Border     lipgloss.AdaptiveColor
	Accent     lipgloss.AdaptiveColor

	// Syntax highlighting
	Keyword lipgloss.AdaptiveColor
	String  lipgloss.AdaptiveColor
	Comment lipgloss.AdaptiveColor
	Number  lipgloss.AdaptiveColor
}

// solid returns an AdaptiveColor where both Light and Dark are the same hex.
// Used for named themes that define their own colors regardless of terminal background.
func solid(hex string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: hex, Dark: hex}
}

// currentTheme is the active theme. Default is set in init().
var currentTheme *Theme

func init() {
	currentTheme = &ThemeDefault
}

// GetTheme returns the currently active theme.
func GetTheme() *Theme {
	return currentTheme
}

// SetTheme applies a theme by updating all global color and style variables.
func SetTheme(t *Theme) {
	currentTheme = t

	// Update color variables
	Primary = t.Primary
	Secondary = t.Secondary
	Success = t.Success
	Error = t.Error
	Warning = t.Warning
	Info = t.Info
	Muted = t.Muted
	Background = t.Background
	Foreground = t.Foreground
	Border = t.Border
	Accent = t.Accent

	// Update syntax highlighting colors
	KeywordStyle = lipgloss.NewStyle().Foreground(t.Keyword)
	StringStyle = lipgloss.NewStyle().Foreground(t.String)
	CommentStyle = lipgloss.NewStyle().Foreground(t.Comment).Italic(true)
	NumberStyle = lipgloss.NewStyle().Foreground(t.Number)

	// Rebuild all derived styles
	rebuildStyles()
}

// rebuildStyles reconstructs all package-level style variables from current colors.
func rebuildStyles() {
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
}

// ListThemes returns the names of all built-in themes in display order.
func ListThemes() []string {
	return []string{
		ThemeDefault.Name,
		ThemeLight.Name,
		ThemeMonokai.Name,
		ThemeDracula.Name,
		ThemeNord.Name,
		ThemeGruvbox.Name,
		ThemeTokyoNight.Name,
		ThemeCatppuccin.Name,
	}
}

// GetThemeByName returns the built-in theme with the given name, or nil.
func GetThemeByName(name string) *Theme {
	for _, t := range builtinThemes {
		if t.Name == name {
			return t
		}
	}
	return nil
}

var builtinThemes = []*Theme{
	&ThemeDefault,
	&ThemeLight,
	&ThemeMonokai,
	&ThemeDracula,
	&ThemeNord,
	&ThemeGruvbox,
	&ThemeTokyoNight,
	&ThemeCatppuccin,
}

// ---------------------------------------------------------------------------
// Built-in themes
// ---------------------------------------------------------------------------

// ThemeDefault is the original WhoDB dark/light adaptive theme.
var ThemeDefault = Theme{
	Name:       "default",
	Primary:    lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#fafafa"},
	Secondary:  lipgloss.AdaptiveColor{Light: "#6b6b73", Dark: "#a1a1aa"},
	Success:    lipgloss.AdaptiveColor{Light: "#16a34a", Dark: "#22c55e"},
	Error:      lipgloss.AdaptiveColor{Light: "#dc2626", Dark: "#ef4444"},
	Warning:    lipgloss.AdaptiveColor{Light: "#d97706", Dark: "#f59e0b"},
	Info:       lipgloss.AdaptiveColor{Light: "#2563eb", Dark: "#3b82f6"},
	Muted:      lipgloss.AdaptiveColor{Light: "#a1a1aa", Dark: "#71717a"},
	Background: lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#09090b"},
	Foreground: lipgloss.AdaptiveColor{Light: "#09090b", Dark: "#fafafa"},
	Border:     lipgloss.AdaptiveColor{Light: "#d4d4d8", Dark: "#27272a"},
	Accent:     lipgloss.AdaptiveColor{Light: "#f4f4f5", Dark: "#18181b"},
	Keyword:    lipgloss.AdaptiveColor{Light: "#3f3f46", Dark: "#d4d4d8"},
	String:     lipgloss.AdaptiveColor{Light: "#6b6b73", Dark: "#a1a1aa"},
	Comment:    lipgloss.AdaptiveColor{Light: "#a1a1aa", Dark: "#71717a"},
	Number:     lipgloss.AdaptiveColor{Light: "#3f3f46", Dark: "#d4d4d8"},
}

// ThemeLight uses the light palette for both modes.
var ThemeLight = Theme{
	Name:       "light",
	Primary:    solid("#1a1a1a"),
	Secondary:  solid("#6b6b73"),
	Success:    solid("#16a34a"),
	Error:      solid("#dc2626"),
	Warning:    solid("#d97706"),
	Info:       solid("#2563eb"),
	Muted:      solid("#a1a1aa"),
	Background: solid("#ffffff"),
	Foreground: solid("#09090b"),
	Border:     solid("#d4d4d8"),
	Accent:     solid("#f4f4f5"),
	Keyword:    solid("#7c3aed"),
	String:     solid("#059669"),
	Comment:    solid("#9ca3af"),
	Number:     solid("#d97706"),
}

// ThemeMonokai — classic dark theme from Sublime Text.
var ThemeMonokai = Theme{
	Name:       "monokai",
	Primary:    solid("#F8F8F2"),
	Secondary:  solid("#75715E"),
	Success:    solid("#A6E22E"),
	Error:      solid("#F92672"),
	Warning:    solid("#FD971F"),
	Info:       solid("#66D9EF"),
	Muted:      solid("#75715E"),
	Background: solid("#272822"),
	Foreground: solid("#F8F8F2"),
	Border:     solid("#3E3D32"),
	Accent:     solid("#3E3D32"),
	Keyword:    solid("#F92672"),
	String:     solid("#E6DB74"),
	Comment:    solid("#75715E"),
	Number:     solid("#AE81FF"),
}

// ThemeDracula — popular dark theme.
var ThemeDracula = Theme{
	Name:       "dracula",
	Primary:    solid("#F8F8F2"),
	Secondary:  solid("#6272A4"),
	Success:    solid("#50FA7B"),
	Error:      solid("#FF5555"),
	Warning:    solid("#FFB86C"),
	Info:       solid("#8BE9FD"),
	Muted:      solid("#6272A4"),
	Background: solid("#282A36"),
	Foreground: solid("#F8F8F2"),
	Border:     solid("#44475A"),
	Accent:     solid("#44475A"),
	Keyword:    solid("#FF79C6"),
	String:     solid("#F1FA8C"),
	Comment:    solid("#6272A4"),
	Number:     solid("#BD93F9"),
}

// ThemeNord — Arctic, north-bluish color palette.
var ThemeNord = Theme{
	Name:       "nord",
	Primary:    solid("#ECEFF4"),
	Secondary:  solid("#D8DEE9"),
	Success:    solid("#A3BE8C"),
	Error:      solid("#BF616A"),
	Warning:    solid("#EBCB8B"),
	Info:       solid("#81A1C1"),
	Muted:      solid("#4C566A"),
	Background: solid("#2E3440"),
	Foreground: solid("#ECEFF4"),
	Border:     solid("#3B4252"),
	Accent:     solid("#3B4252"),
	Keyword:    solid("#81A1C1"),
	String:     solid("#A3BE8C"),
	Comment:    solid("#616E88"),
	Number:     solid("#B48EAD"),
}

// ThemeGruvbox — retro groove color scheme.
var ThemeGruvbox = Theme{
	Name:       "gruvbox",
	Primary:    solid("#EBDBB2"),
	Secondary:  solid("#A89984"),
	Success:    solid("#B8BB26"),
	Error:      solid("#FB4934"),
	Warning:    solid("#FABD2F"),
	Info:       solid("#83A598"),
	Muted:      solid("#665C54"),
	Background: solid("#282828"),
	Foreground: solid("#EBDBB2"),
	Border:     solid("#3C3836"),
	Accent:     solid("#3C3836"),
	Keyword:    solid("#FB4934"),
	String:     solid("#B8BB26"),
	Comment:    solid("#928374"),
	Number:     solid("#D3869B"),
}

// ThemeTokyoNight — clean dark theme inspired by Tokyo at night.
var ThemeTokyoNight = Theme{
	Name:       "tokyo-night",
	Primary:    solid("#C0CAF5"),
	Secondary:  solid("#565F89"),
	Success:    solid("#9ECE6A"),
	Error:      solid("#F7768E"),
	Warning:    solid("#E0AF68"),
	Info:       solid("#7AA2F7"),
	Muted:      solid("#565F89"),
	Background: solid("#1A1B26"),
	Foreground: solid("#C0CAF5"),
	Border:     solid("#292E42"),
	Accent:     solid("#292E42"),
	Keyword:    solid("#BB9AF7"),
	String:     solid("#9ECE6A"),
	Comment:    solid("#565F89"),
	Number:     solid("#FF9E64"),
}

// ThemeCatppuccin — Catppuccin Mocha flavor, soothing pastel theme.
var ThemeCatppuccin = Theme{
	Name:       "catppuccin",
	Primary:    solid("#CDD6F4"),
	Secondary:  solid("#A6ADC8"),
	Success:    solid("#A6E3A1"),
	Error:      solid("#F38BA8"),
	Warning:    solid("#F9E2AF"),
	Info:       solid("#89B4FA"),
	Muted:      solid("#6C7086"),
	Background: solid("#1E1E2E"),
	Foreground: solid("#CDD6F4"),
	Border:     solid("#313244"),
	Accent:     solid("#313244"),
	Keyword:    solid("#CBA6F7"),
	String:     solid("#A6E3A1"),
	Comment:    solid("#6C7086"),
	Number:     solid("#FAB387"),
}
