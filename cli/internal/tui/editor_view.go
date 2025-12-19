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
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/pkg/styles"
)

type suggestionType string

const (
	suggestionTypeKeyword  suggestionType = "keyword"
	suggestionTypeTable    suggestionType = "table"
	suggestionTypeColumn   suggestionType = "column"
	suggestionTypeSchema   suggestionType = "schema"
	suggestionTypeMixed    suggestionType = "mixed"
	suggestionTypeFunction suggestionType = "function"
	suggestionTypeSnippet  suggestionType = "snippet"
)

type suggestion struct {
	label  string
	typ    suggestionType
	detail string
	apply  string
}

type sqlContext struct {
	contextType    suggestionType
	schema         string
	table          string
	alias          string
	tablesInQuery  []tableInfo
	tokenBeforeDot string
}

type tableInfo struct {
	schema string
	table  string
	alias  string
}

const (
	minSuggestionHeight = 3
	maxSuggestionHeight = 12
)

type EditorView struct {
	parent              *MainModel
	textarea            textarea.Model
	err                 error
	allSuggestions      []suggestion
	filteredSuggestions []suggestion
	showSuggestions     bool
	selectedSuggestion  int
	currentSchema       string
	cursorPos           int
	lastText            string
	lastWidth           int
	lastHeight          int
	suggestionHeight    int
}

func NewEditorView(parent *MainModel) *EditorView {
	ta := textarea.New()
	ta.Placeholder = "Enter SQL query..."
	ta.Focus()
	ta.SetHeight(10)
	ta.SetWidth(80)
	ta.CharLimit = 0

	return &EditorView{
		parent:              parent,
		textarea:            ta,
		allSuggestions:      []suggestion{},
		filteredSuggestions: []suggestion{},
		selectedSuggestion:  0,
	}
}

func (v *EditorView) Update(msg tea.Msg) (*EditorView, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.applyWindowSize(msg.Width, msg.Height)
		return v, nil

	case tea.MouseMsg:
		if v.showSuggestions && len(v.filteredSuggestions) > 0 {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				v.selectedSuggestion--
				if v.selectedSuggestion < 0 {
					v.selectedSuggestion = len(v.filteredSuggestions) - 1
				}
				return v, nil
			case tea.MouseButtonWheelDown:
				v.selectedSuggestion = (v.selectedSuggestion + 1) % len(v.filteredSuggestions)
				return v, nil
			}
		}

	case tea.KeyMsg:
		// IMPORTANT: Check for execute query shortcut FIRST before passing to textarea
		// Alt+Enter (Option+Enter on macOS) - works reliably across all platforms
		if msg.Type == tea.KeyEnter && msg.Alt {
			return v, v.executeQuery()
		}

		// Ctrl+Space to manually trigger autocomplete
		// Ctrl+@ is how Ctrl+Space is typically represented in terminals (ASCII 0)
		if msg.Type == tea.KeyCtrlAt {
			v.triggerAutocomplete()
			return v, nil
		}
		// Also check for null rune
		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 && msg.Runes[0] == 0 {
			v.triggerAutocomplete()
			return v, nil
		}

		// Ctrl+L to clear
		if msg.Type == tea.KeyCtrlL {
			v.textarea.Reset()
			v.err = nil
			v.showSuggestions = false
			v.allSuggestions = []suggestion{}
			v.filteredSuggestions = []suggestion{}
			v.selectedSuggestion = 0
			v.cursorPos = 0
			v.lastText = ""
			v.refreshLayout()
			return v, nil
		}

		// Handle autocomplete navigation when suggestions are shown
		if v.showSuggestions && len(v.filteredSuggestions) > 0 {
			switch msg.Type {
			case tea.KeyTab:
				v.selectedSuggestion = (v.selectedSuggestion + 1) % len(v.filteredSuggestions)
				return v, nil

			case tea.KeyShiftTab:
				v.selectedSuggestion--
				if v.selectedSuggestion < 0 {
					v.selectedSuggestion = len(v.filteredSuggestions) - 1
				}
				return v, nil

			case tea.KeyDown, tea.KeyCtrlN:
				v.selectedSuggestion = (v.selectedSuggestion + 1) % len(v.filteredSuggestions)
				return v, nil

			case tea.KeyUp, tea.KeyCtrlP:
				v.selectedSuggestion--
				if v.selectedSuggestion < 0 {
					v.selectedSuggestion = len(v.filteredSuggestions) - 1
				}
				return v, nil

			case tea.KeyEnter:
				if !msg.Alt {
					v.acceptSuggestion()
					return v, nil
				}
			}
		}

		switch msg.Type {
		case tea.KeyEsc:
			if v.showSuggestions {
				v.showSuggestions = false
				v.selectedSuggestion = 0
				v.refreshLayout()
				return v, nil
			}
			v.parent.mode = ViewBrowser
			return v, nil

		case tea.KeyCtrlS:
			return v, v.saveQuery()
		}

		// Handle "e" key for export (when not in autocomplete mode)
		if msg.String() == "e" && !v.showSuggestions {
			if v.parent.resultsView.results != nil && v.parent.resultsView.query != "" {
				v.parent.exportView.SetExportDataFromQuery(v.parent.resultsView.results)
				v.parent.mode = ViewExport
				return v, nil
			}
		}
	}

	// Pass to textarea
	v.textarea, cmd = v.textarea.Update(msg)

	// Calculate cursor position based on current line and column
	v.updateCursorPosition()

	// Update autocomplete in real-time as user types
	text := v.textarea.Value()
	if v.textarea.Focused() {
		v.updateAutocomplete(text, v.cursorPos)
	}

	return v, cmd
}

func (v *EditorView) View() string {
	var b strings.Builder

	b.WriteString(styles.RenderTitle("SQL Editor"))
	b.WriteString("\n\n")

	b.WriteString(v.textarea.View())
	b.WriteString("\n")

	if v.err != nil {
		b.WriteString("\n\n")
		b.WriteString(styles.RenderErrorBox(v.err.Error()))
	}

	b.WriteString("\n")
	b.WriteString(v.renderSuggestionArea())

	b.WriteString("\n\n")
	b.WriteString(styles.RenderHelp(
		styles.KeyExecute, styles.KeyExecuteDesc,
		"ctrl+@", "autocomplete",
		"[e]", "export results",
		"ctrl+l", "clear",
		"tab", "next view",
		"esc", "back",
		"ctrl+c", "quit",
	))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (v *EditorView) executeQuery() tea.Cmd {
	return func() tea.Msg {
		query := v.textarea.Value()
		if query == "" {
			v.err = fmt.Errorf("query is empty")
			return nil
		}

		result, err := v.parent.dbManager.ExecuteQuery(query)
		if err != nil {
			v.err = err
			conn := v.parent.dbManager.GetCurrentConnection()
			dbName := ""
			if conn != nil {
				dbName = conn.Database
			}
			v.parent.histMgr.Add(query, false, dbName)
			return nil
		}

		conn := v.parent.dbManager.GetCurrentConnection()
		dbName := ""
		if conn != nil {
			dbName = conn.Database
		}
		v.parent.histMgr.Add(query, true, dbName)

		v.parent.resultsView.SetResults(result, query)
		v.parent.mode = ViewResults
		v.err = nil

		return nil
	}
}

func (v *EditorView) saveQuery() tea.Cmd {
	return func() tea.Msg {
		return nil
	}
}

// updateCursorPosition infers cursor position by comparing text changes
func (v *EditorView) updateCursorPosition() {
	text := v.textarea.Value()

	// If text is empty, cursor is at start
	if len(text) == 0 {
		v.cursorPos = 0
		v.lastText = text
		return
	}

	// If this is the first time or text was cleared, assume cursor at end
	if v.lastText == "" {
		v.cursorPos = len(text)
		v.lastText = text
		return
	}

	// Find where the text changed by comparing with last known text
	oldLen := len(v.lastText)
	newLen := len(text)

	if newLen > oldLen {
		// Text was inserted - find insertion point
		insertPos := findDiffPosition(v.lastText, text)
		v.cursorPos = insertPos + (newLen - oldLen)
	} else if newLen < oldLen {
		// Text was deleted - find deletion point
		deletePos := findDiffPosition(text, v.lastText)
		v.cursorPos = deletePos
	} else {
		// Same length - text might have been replaced or cursor just moved
		// Try to find the change position
		diffPos := findDiffPosition(v.lastText, text)
		if diffPos < len(text) {
			v.cursorPos = diffPos + 1
		} else {
			// No change detected, cursor likely moved - keep current position
			// but clamp to text length
			if v.cursorPos > len(text) {
				v.cursorPos = len(text)
			}
		}
	}

	// Clamp cursor position to valid range
	if v.cursorPos < 0 {
		v.cursorPos = 0
	}
	if v.cursorPos > len(text) {
		v.cursorPos = len(text)
	}

	v.lastText = text
}

// findDiffPosition finds the first position where two strings differ
func findDiffPosition(s1, s2 string) int {
	minLen := len(s1)
	if len(s2) < minLen {
		minLen = len(s2)
	}

	for i := 0; i < minLen; i++ {
		if s1[i] != s2[i] {
			return i
		}
	}

	return minLen
}

// triggerAutocomplete manually triggers autocomplete at current cursor position
func (v *EditorView) triggerAutocomplete() {
	v.updateCursorPosition()
	text := v.textarea.Value()
	v.updateAutocomplete(text, v.cursorPos)
	if len(v.filteredSuggestions) > 0 {
		v.showSuggestions = true
		v.refreshLayout()
	}
}

func (v *EditorView) applyWindowSize(width, height int) {
	v.lastWidth = width
	v.lastHeight = height

	v.textarea.SetWidth(width - 8)

	v.suggestionHeight = v.computeSuggestionHeight(height)

	targetHeight := height - 20 - v.suggestionHeight
	if targetHeight < 5 {
		targetHeight = 5
	}
	v.textarea.SetHeight(targetHeight)
}

func (v *EditorView) refreshLayout() {
	if v.lastWidth > 0 && v.lastHeight > 0 {
		v.applyWindowSize(v.lastWidth, v.lastHeight)
	}
}

func (v *EditorView) computeSuggestionHeight(totalHeight int) int {
	if !v.showSuggestions || len(v.filteredSuggestions) == 0 {
		return 0
	}

	// Leave room for title, spacing, and help footer (~20 lines previously)
	available := totalHeight - 20 - 8 // reserve at least 8 lines for the textarea
	if available <= 0 {
		return 0
	}

	height := available
	if height > maxSuggestionHeight {
		height = maxSuggestionHeight
	}
	if height < minSuggestionHeight {
		height = available
	}

	return height
}

// updateAutocomplete updates the autocomplete suggestions based on the current context
func (v *EditorView) updateAutocomplete(text string, pos int) {
	previouslyShown := v.showSuggestions

	if pos == 0 {
		v.showSuggestions = false
		if previouslyShown {
			v.refreshLayout()
		}
		return
	}

	beforeCursor := text[:pos]
	tokenMatch := regexp.MustCompile(`[A-Za-z0-9_\.` + "`" + `]+$`).FindString(beforeCursor)
	token := tokenMatch

	// Parse context first to understand what we're working with
	ctx := v.parseSQLContext(text, pos)

	// Always load suggestions for the context
	v.loadSuggestionsForContext(ctx, text)

	// Filter suggestions by token (empty token shows all)
	v.filterSuggestions(getLastWord(beforeCursor))

	// Show suggestions if we have any and we're in a meaningful context
	// Don't show for keywords-only context unless user is typing something
	if len(v.filteredSuggestions) > 0 {
		// Show if we have a token being typed OR if we're in a context that expects something specific
		if token != "" || ctx.contextType != suggestionTypeKeyword {
			v.showSuggestions = true
			if v.selectedSuggestion >= len(v.filteredSuggestions) {
				v.selectedSuggestion = 0
			}
			if !previouslyShown {
				v.refreshLayout()
			}
			return
		}
	}

	v.showSuggestions = false
	if previouslyShown {
		v.refreshLayout()
	}
}

// loadSuggestionsForContext loads all possible suggestions based on SQL context
func (v *EditorView) loadSuggestionsForContext(ctx sqlContext, fullText string) {
	v.allSuggestions = []suggestion{}

	switch ctx.contextType {
	case suggestionTypeSchema:
		// After FROM/JOIN - suggest schemas and tables
		schemas, _ := v.parent.dbManager.GetSchemas()
		for _, s := range schemas {
			v.allSuggestions = append(v.allSuggestions, suggestion{
				label:  s,
				typ:    suggestionTypeSchema,
				detail: "Schema",
				apply:  s,
			})
		}
		// Also suggest tables from current schema
		v.addTableSuggestions(v.getCurrentSchema())

	case suggestionTypeTable:
		// After schema. or in FROM/JOIN
		if ctx.schema != "" {
			v.addTableSuggestions(ctx.schema)
		} else {
			// No schema specified, show schemas
			schemas, _ := v.parent.dbManager.GetSchemas()
			for _, s := range schemas {
				v.allSuggestions = append(v.allSuggestions, suggestion{
					label:  s,
					typ:    suggestionTypeSchema,
					detail: "Schema",
					apply:  s,
				})
			}
		}

	case suggestionTypeColumn:
		// After table. or alias. or in SELECT/WHERE with tables
		if ctx.tokenBeforeDot != "" {
			// Handle alias.column or table.column
			v.addQualifiedColumnSuggestions(ctx, fullText)
		} else {
			// Handle unqualified column suggestions
			v.addUnqualifiedColumnSuggestions(ctx)
		}
		// Add SQL functions
		v.addFunctionSuggestions()

	case suggestionTypeMixed:
		// After WHERE/ON - show everything
		v.addMixedSuggestions(ctx, fullText)

	default:
		// Default: keywords + functions + snippets
		v.addKeywordSuggestions()
		v.addFunctionSuggestions()
		v.addSnippetSuggestions()
	}
}

func (v *EditorView) addTableSuggestions(schema string) {
	units, err := v.parent.dbManager.GetStorageUnits(schema)
	if err != nil {
		return
	}

	for _, unit := range units {
		v.allSuggestions = append(v.allSuggestions, suggestion{
			label:  unit.Name,
			typ:    suggestionTypeTable,
			detail: "Table",
			apply:  unit.Name,
		})
	}
}

func (v *EditorView) addQualifiedColumnSuggestions(ctx sqlContext, fullText string) {
	// tokenBeforeDot could be alias, table, or schema
	tokenBefore := ctx.tokenBeforeDot

	// Check if it matches an alias or table in the query
	for _, t := range ctx.tablesInQuery {
		if strings.EqualFold(t.alias, tokenBefore) || strings.EqualFold(t.table, tokenBefore) {
			// Fetch columns for this table
			schema := t.schema
			if schema == "" {
				schema = v.getCurrentSchema()
			}

			columns, err := v.parent.dbManager.GetColumns(schema, t.table)
			if err != nil {
				return
			}

			for _, col := range columns {
				// Add both qualified and unqualified versions
				v.allSuggestions = append(v.allSuggestions, suggestion{
					label:  fmt.Sprintf("%s.%s", tokenBefore, col.Name),
					typ:    suggestionTypeColumn,
					detail: col.Type,
					apply:  fmt.Sprintf("%s.%s", tokenBefore, col.Name),
				})
				v.allSuggestions = append(v.allSuggestions, suggestion{
					label:  col.Name,
					typ:    suggestionTypeColumn,
					detail: col.Type,
					apply:  col.Name,
				})
			}
			return
		}
	}

	// If no match, might be schema.table pattern - show tables
	v.addTableSuggestions(tokenBefore)
}

func (v *EditorView) addUnqualifiedColumnSuggestions(ctx sqlContext) {
	tables := ctx.tablesInQuery
	if len(tables) == 0 {
		return
	}

	if len(tables) == 1 {
		// Single table - show its columns
		t := tables[0]
		schema := t.schema
		if schema == "" {
			schema = v.getCurrentSchema()
		}

		columns, err := v.parent.dbManager.GetColumns(schema, t.table)
		if err != nil {
			return
		}

		for _, col := range columns {
			v.allSuggestions = append(v.allSuggestions, suggestion{
				label:  col.Name,
				typ:    suggestionTypeColumn,
				detail: col.Type,
				apply:  col.Name,
			})
		}
	} else {
		// Multiple tables - show aliases and qualified columns for first 3 tables
		for i, t := range tables {
			if i >= 3 {
				break
			}

			// Add alias
			if t.alias != "" {
				v.allSuggestions = append(v.allSuggestions, suggestion{
					label:  t.alias,
					typ:    suggestionTypeTable,
					detail: fmt.Sprintf("Alias for %s", t.table),
					apply:  t.alias,
				})
			}

			// Add table name
			v.allSuggestions = append(v.allSuggestions, suggestion{
				label:  t.table,
				typ:    suggestionTypeTable,
				detail: "Table",
				apply:  t.table,
			})
		}
	}
}

func (v *EditorView) addMixedSuggestions(ctx sqlContext, fullText string) {
	// Add aliases and table names
	for _, t := range ctx.tablesInQuery {
		if t.alias != "" {
			v.allSuggestions = append(v.allSuggestions, suggestion{
				label:  t.alias,
				typ:    suggestionTypeTable,
				detail: fmt.Sprintf("Alias for %s", t.table),
				apply:  t.alias,
			})
		}
		v.allSuggestions = append(v.allSuggestions, suggestion{
			label:  t.table,
			typ:    suggestionTypeTable,
			detail: "Table",
			apply:  t.table,
		})
	}

	// Add columns from first 3 tables
	for i, t := range ctx.tablesInQuery {
		if i >= 3 {
			break
		}

		schema := t.schema
		if schema == "" {
			schema = v.getCurrentSchema()
		}

		columns, err := v.parent.dbManager.GetColumns(schema, t.table)
		if err != nil {
			continue
		}

		for _, col := range columns {
			// Add alias-qualified if exists
			if t.alias != "" {
				v.allSuggestions = append(v.allSuggestions, suggestion{
					label:  fmt.Sprintf("%s.%s", t.alias, col.Name),
					typ:    suggestionTypeColumn,
					detail: col.Type,
					apply:  fmt.Sprintf("%s.%s", t.alias, col.Name),
				})
			}
			// Add table-qualified
			v.allSuggestions = append(v.allSuggestions, suggestion{
				label:  fmt.Sprintf("%s.%s", t.table, col.Name),
				typ:    suggestionTypeColumn,
				detail: col.Type,
				apply:  fmt.Sprintf("%s.%s", t.table, col.Name),
			})
			// Add unqualified
			v.allSuggestions = append(v.allSuggestions, suggestion{
				label:  col.Name,
				typ:    suggestionTypeColumn,
				detail: col.Type,
				apply:  col.Name,
			})
		}
	}

	// Add functions and snippets
	v.addFunctionSuggestions()
	v.addSnippetSuggestions()
}

func (v *EditorView) addKeywordSuggestions() {
	keywords := []string{
		"SELECT", "FROM", "WHERE", "JOIN", "INNER", "LEFT", "RIGHT", "OUTER", "FULL",
		"ON", "AND", "OR", "NOT", "NULL", "TRUE", "FALSE", "ORDER", "BY", "GROUP",
		"HAVING", "LIMIT", "OFFSET", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP",
		"ALTER", "TABLE", "INDEX", "VIEW", "DATABASE", "SCHEMA", "INTO", "VALUES",
		"SET", "AS", "DISTINCT", "COUNT", "SUM", "AVG", "MAX", "MIN", "CASE", "WHEN",
		"THEN", "ELSE", "END", "IF", "EXISTS", "LIKE", "IN", "BETWEEN", "IS", "ASC",
		"DESC", "UNION", "ALL", "ANY", "SOME", "WITH", "RECURSIVE", "CASCADE",
		"CONSTRAINT", "PRIMARY", "KEY", "FOREIGN", "REFERENCES", "UNIQUE", "CHECK",
		"DEFAULT", "TRUNCATE", "EXPLAIN", "ANALYZE", "GRANT", "REVOKE", "COMMIT",
		"ROLLBACK", "TRANSACTION", "BEGIN", "START",
	}

	for _, kw := range keywords {
		v.allSuggestions = append(v.allSuggestions, suggestion{
			label:  kw,
			typ:    suggestionTypeKeyword,
			detail: "SQL Keyword",
			apply:  kw,
		})
	}
}

func (v *EditorView) addFunctionSuggestions() {
	functions := []struct {
		name   string
		detail string
	}{
		{"COUNT", "COUNT(expr)"},
		{"SUM", "SUM(expr)"},
		{"AVG", "AVG(expr)"},
		{"MIN", "MIN(expr)"},
		{"MAX", "MAX(expr)"},
		{"COALESCE", "COALESCE(expr1, expr2)"},
		{"CAST", "CAST(expr AS type)"},
		{"CONCAT", "CONCAT(expr, ...)"},
	}

	for _, fn := range functions {
		v.allSuggestions = append(v.allSuggestions, suggestion{
			label:  fn.name,
			typ:    suggestionTypeFunction,
			detail: fn.detail,
			apply:  fn.name + "()",
		})
	}
}

func (v *EditorView) addSnippetSuggestions() {
	snippets := []struct {
		label  string
		detail string
		apply  string
	}{
		{"JOIN ... ON ...", "Snippet: JOIN with ON", "JOIN schema.table alias ON alias.column = other.column"},
		{"LEFT JOIN ... ON ...", "Snippet: LEFT JOIN with ON", "LEFT JOIN schema.table alias ON alias.column = other.column"},
		{"WHERE IN (...)", "Snippet: WHERE IN", "WHERE column IN (value1, value2)"},
		{"GROUP BY ...", "Snippet: GROUP BY", "GROUP BY column1, column2"},
		{"SELECT DISTINCT ...", "Snippet: SELECT DISTINCT", "SELECT DISTINCT column FROM schema.table"},
	}

	for _, snip := range snippets {
		v.allSuggestions = append(v.allSuggestions, suggestion{
			label:  snip.label,
			typ:    suggestionTypeSnippet,
			detail: snip.detail,
			apply:  snip.apply,
		})
	}
}

func (v *EditorView) getCurrentSchema() string {
	if v.currentSchema != "" {
		return v.currentSchema
	}

	// Use the schema selected in browser view if available
	if v.parent.browserView.currentSchema != "" {
		v.currentSchema = v.parent.browserView.currentSchema
		return v.currentSchema
	}

	schemas, err := v.parent.dbManager.GetSchemas()
	if err != nil || len(schemas) == 0 {
		return ""
	}

	v.currentSchema = selectBestSchema(schemas)
	return v.currentSchema
}

func (v *EditorView) filterSuggestions(prefix string) {
	if prefix == "" {
		v.filteredSuggestions = make([]suggestion, len(v.allSuggestions))
		copy(v.filteredSuggestions, v.allSuggestions)
		return
	}

	v.filteredSuggestions = []suggestion{}
	lowerPrefix := strings.ToLower(prefix)

	for _, sug := range v.allSuggestions {
		// Check if label starts with prefix (for simple matches)
		if strings.HasPrefix(strings.ToLower(sug.label), lowerPrefix) {
			v.filteredSuggestions = append(v.filteredSuggestions, sug)
			continue
		}

		// For qualified names (table.column), also check the part after the dot
		if strings.Contains(sug.label, ".") {
			parts := strings.Split(sug.label, ".")
			if len(parts) > 1 && strings.HasPrefix(strings.ToLower(parts[len(parts)-1]), lowerPrefix) {
				v.filteredSuggestions = append(v.filteredSuggestions, sug)
			}
		}
	}
}

func (v *EditorView) renderAutocompletePanel() string {
	var panel strings.Builder

	count := len(v.filteredSuggestions)
	header := styles.MutedStyle.Render(fmt.Sprintf("Suggestions (%d)", count))
	panel.WriteString(header)
	panel.WriteString("\n\n")

	maxDisplay := 8
	if len(v.filteredSuggestions) < maxDisplay {
		maxDisplay = len(v.filteredSuggestions)
	}

	for i := 0; i < maxDisplay; i++ {
		sug := v.filteredSuggestions[i]
		line := fmt.Sprintf("%s (%s)", sug.label, sug.detail)

		if i == v.selectedSuggestion {
			panel.WriteString(styles.ActiveListItemStyle.Render("> " + line))
		} else {
			panel.WriteString(styles.MutedStyle.Render("  " + line))
		}
		panel.WriteString("\n")
	}

	if len(v.filteredSuggestions) > maxDisplay {
		panel.WriteString(styles.MutedStyle.Render(fmt.Sprintf("  ... and %d more", len(v.filteredSuggestions)-maxDisplay)))
		panel.WriteString("\n")
	}

	panel.WriteString("\n")
	controlPanel := styles.HelpStyle.Render(
		styles.KeyStyle.Render("↑↓") + " " + styles.MutedStyle.Render("navigate") + " " +
			styles.KeyStyle.Render("tab") + " " + styles.MutedStyle.Render("cycle") + " " +
			styles.KeyStyle.Render("↵") + " " + styles.MutedStyle.Render("accept") + " " +
			styles.KeyStyle.Render("esc") + " " + styles.MutedStyle.Render("dismiss"),
	)
	panel.WriteString(controlPanel)

	return styles.BoxStyle.Render(panel.String())
}

func (v *EditorView) renderSuggestionArea() string {
	content := ""
	if v.showSuggestions && len(v.filteredSuggestions) > 0 {
		content = v.renderAutocompletePanel()
	}

	// Keep layout stable by reserving a fixed-height area for suggestions.
	return lipgloss.NewStyle().
		Height(v.suggestionHeight).
		MaxHeight(v.suggestionHeight).
		Render(content)
}

func (v *EditorView) acceptSuggestion() {
	if v.selectedSuggestion >= 0 && v.selectedSuggestion < len(v.filteredSuggestions) {
		sug := v.filteredSuggestions[v.selectedSuggestion]
		text := v.textarea.Value()

		// Use cursor position to find what to replace
		beforeCursor := text[:v.cursorPos]
		afterCursor := text[v.cursorPos:]

		// Find the token to replace before cursor
		tokenMatch := regexp.MustCompile(`[A-Za-z0-9_\.` + "`" + `]+$`).FindString(beforeCursor)

		// Calculate replacement position
		var newText string
		if tokenMatch != "" {
			// Replace the token
			startPos := v.cursorPos - len(tokenMatch)
			newText = text[:startPos] + sug.apply + afterCursor
		} else {
			// Insert at cursor position
			newText = beforeCursor + sug.apply + afterCursor
		}

		v.textarea.SetValue(newText)
		v.showSuggestions = false
		v.selectedSuggestion = 0
		v.refreshLayout()
	}
}

func (v *EditorView) parseSQLContext(text string, pos int) sqlContext {
	beforeCursor := text[:pos]
	beforeUpper := strings.ToUpper(beforeCursor)

	tablesInQuery := v.extractTablesAndAliases(text)

	// Check for dot notation
	tokenMatch := regexp.MustCompile(`[A-Za-z0-9_\.` + "`" + `]+$`).FindString(beforeCursor)
	if strings.Contains(tokenMatch, ".") {
		parts := strings.Split(tokenMatch, ".")
		if len(parts) >= 2 {
			tokenBeforeDot := strings.Trim(parts[0], "`")

			// Check if it's an alias or table name
			for _, t := range tablesInQuery {
				if strings.EqualFold(t.alias, tokenBeforeDot) || strings.EqualFold(t.table, tokenBeforeDot) {
					return sqlContext{
						contextType:    suggestionTypeColumn,
						schema:         t.schema,
						table:          t.table,
						alias:          t.alias,
						tablesInQuery:  tablesInQuery,
						tokenBeforeDot: tokenBeforeDot,
					}
				}
			}

			// Might be schema.table
			return sqlContext{
				contextType:    suggestionTypeTable,
				schema:         tokenBeforeDot,
				tablesInQuery:  tablesInQuery,
				tokenBeforeDot: tokenBeforeDot,
			}
		}
	}

	// Check if after FROM or JOIN
	if regexp.MustCompile(`\bFROM\s*$`).MatchString(beforeUpper) ||
		regexp.MustCompile(`\bJOIN\s*$`).MatchString(beforeUpper) {
		return sqlContext{
			contextType:   suggestionTypeSchema,
			tablesInQuery: tablesInQuery,
		}
	}

	if regexp.MustCompile(`\bFROM\s+\w*$`).MatchString(beforeUpper) ||
		regexp.MustCompile(`\bJOIN\s+\w*$`).MatchString(beforeUpper) {
		return sqlContext{
			contextType:   suggestionTypeTable,
			tablesInQuery: tablesInQuery,
		}
	}

	// Check if in SELECT clause with tables (expecting columns)
	lastSelect := strings.LastIndex(beforeUpper, "SELECT")
	lastFrom := strings.LastIndex(beforeUpper, "FROM")

	// Also check in full text to see if there's a FROM clause anywhere
	fullTextUpper := strings.ToUpper(text)
	hasFromInQuery := strings.Contains(fullTextUpper, "FROM")

	// If we're after SELECT and have tables in query (from parsing full text), suggest columns
	if lastSelect > -1 && len(tablesInQuery) > 0 {
		// Check if we're between SELECT and FROM, or after FROM with tables
		if (lastFrom > lastSelect) || (lastFrom == -1 && hasFromInQuery) {
			return sqlContext{
				contextType:   suggestionTypeColumn,
				tablesInQuery: tablesInQuery,
			}
		}
	}

	// Check if in WHERE/ON clause (expecting columns)
	lastWhere := strings.LastIndex(beforeUpper, "WHERE")
	lastOn := strings.LastIndex(beforeUpper, " ON ")

	if ((lastWhere > -1 && lastWhere > lastFrom) || (lastOn > -1)) && len(tablesInQuery) > 0 {
		return sqlContext{
			contextType:   suggestionTypeMixed,
			tablesInQuery: tablesInQuery,
		}
	}

	// If typing an identifier and we have tables, suggest columns
	if tokenMatch != "" && !strings.Contains(tokenMatch, ".") && len(tablesInQuery) > 0 {
		if len(tablesInQuery) == 1 {
			return sqlContext{
				contextType:   suggestionTypeColumn,
				schema:        tablesInQuery[0].schema,
				table:         tablesInQuery[0].table,
				alias:         tablesInQuery[0].alias,
				tablesInQuery: tablesInQuery,
			}
		}
		return sqlContext{
			contextType:   suggestionTypeMixed,
			tablesInQuery: tablesInQuery,
		}
	}

	// Default: keywords
	return sqlContext{
		contextType:   suggestionTypeKeyword,
		tablesInQuery: tablesInQuery,
	}
}

func (v *EditorView) extractTablesAndAliases(text string) []tableInfo {
	tables := []tableInfo{}

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bFROM\s+(?:` + "`" + `?(\w+)` + "`" + `?\.)?` + "`" + `?(\w+)` + "`" + `?(?:\s+(?:AS\s+)?` + "`" + `?(\w+)` + "`" + `?)?`),
		regexp.MustCompile(`(?i)\bJOIN\s+(?:` + "`" + `?(\w+)` + "`" + `?\.)?` + "`" + `?(\w+)` + "`" + `?(?:\s+(?:AS\s+)?` + "`" + `?(\w+)` + "`" + `?)?`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				schema := strings.Trim(match[1], "`")
				table := strings.Trim(match[2], "`")
				alias := ""
				if len(match) >= 4 {
					alias = strings.Trim(match[3], "`")
				}

				if !v.isKeyword(table) {
					tables = append(tables, tableInfo{
						schema: schema,
						table:  table,
						alias:  alias,
					})
				}
			}
		}
	}

	return tables
}

func (v *EditorView) isKeyword(word string) bool {
	keywords := map[string]bool{
		"SELECT": true, "FROM": true, "WHERE": true, "AND": true, "OR": true,
		"JOIN": true, "LEFT": true, "RIGHT": true, "INNER": true, "OUTER": true,
		"ON": true, "AS": true, "BY": true, "ORDER": true, "GROUP": true,
	}
	return keywords[strings.ToUpper(word)]
}

func getLastWord(text string) string {
	words := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t' || r == ',' || r == '(' || r == ')'
	})

	if len(words) == 0 {
		return ""
	}

	lastWord := words[len(words)-1]

	// If it contains a dot, get the part after the dot
	if strings.Contains(lastWord, ".") {
		parts := strings.Split(lastWord, ".")
		if len(parts) > 1 {
			return strings.Trim(parts[len(parts)-1], "`")
		}
	}

	return strings.Trim(lastWord, "`")
}
