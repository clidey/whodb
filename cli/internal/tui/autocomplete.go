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
	"time"
	"unicode/utf8"

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

// autocompleteDebounceDelay is the delay before triggering autocomplete after a keystroke.
// This prevents excessive database calls during fast typing.
const autocompleteDebounceDelay = 100 * time.Millisecond

// updateCursorPosition computes the absolute byte offset of the cursor
// in the textarea's text using the textarea's own row/column state.
func (v *EditorView) updateCursorPosition() {
	text := v.textarea.Value()
	row := v.textarea.Line()
	li := v.textarea.LineInfo()
	col := li.ColumnOffset

	// Split text into lines and sum lengths of lines before the cursor row,
	// adding 1 per line for the newline character.
	lines := strings.Split(text, "\n")
	pos := 0
	for i := 0; i < row && i < len(lines); i++ {
		pos += len(lines[i]) + 1 // +1 for newline
	}
	// Add column offset within the current line (convert rune offset to byte offset)
	if row < len(lines) {
		runes := []rune(lines[row])
		if col > len(runes) {
			col = len(runes)
		}
		pos += len(string(runes[:col]))
	}

	if pos > len(text) {
		pos = len(text)
	}

	v.cursorPos = pos
	v.lastText = text
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
	// Use the schema selected in browser view if available
	if v.parent.browserView.currentSchema != "" {
		return v.parent.browserView.currentSchema
	}

	schemas, err := v.parent.dbManager.GetSchemas()
	if err != nil || len(schemas) == 0 {
		return ""
	}

	return selectBestSchema(schemas)
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

	startIdx := 0
	if v.selectedSuggestion >= maxDisplay {
		startIdx = v.selectedSuggestion - maxDisplay + 1
	}

	for i := startIdx; i < startIdx+maxDisplay && i < len(v.filteredSuggestions); i++ {
		sug := v.filteredSuggestions[i]
		line := fmt.Sprintf("%s (%s)", sug.label, sug.detail)

		if i == v.selectedSuggestion {
			panel.WriteString(styles.ActiveListItemStyle.Render("> " + line))
		} else {
			panel.WriteString(styles.MutedStyle.Render("  " + line))
		}
		panel.WriteString("\n")
	}

	if len(v.filteredSuggestions) > startIdx+maxDisplay {
		panel.WriteString(styles.MutedStyle.Render(fmt.Sprintf("  ... and %d more", len(v.filteredSuggestions)-startIdx-maxDisplay)))
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

		// Calculate replacement position and new cursor position
		var newText string
		var newCursorPos int
		if tokenMatch != "" {
			// If the suggestion doesn't contain a dot but the token does (e.g. token is
			// "test_schema." and suggestion is "products"), only replace the part after the
			// last dot to preserve the qualifier prefix.
			if strings.Contains(tokenMatch, ".") && !strings.Contains(sug.apply, ".") {
				lastDot := strings.LastIndex(tokenMatch, ".")
				// Keep everything up to and including the dot
				prefixToKeep := tokenMatch[:lastDot+1]
				startPos := v.cursorPos - len(tokenMatch)
				newText = text[:startPos] + prefixToKeep + sug.apply + afterCursor
				newCursorPos = startPos + len(prefixToKeep) + len(sug.apply)
			} else {
				// Replace the entire token
				startPos := v.cursorPos - len(tokenMatch)
				newText = text[:startPos] + sug.apply + afterCursor
				newCursorPos = startPos + len(sug.apply)
			}
		} else {
			// Insert at cursor position
			newText = beforeCursor + sug.apply + afterCursor
			newCursorPos = v.cursorPos + len(sug.apply)
		}

		v.textarea.SetValue(newText)

		// Sync cursor tracking state to prevent updateCursorPosition() miscalculations
		v.cursorPos = newCursorPos
		v.lastText = newText

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
	if len(text) == 0 {
		return ""
	}

	// If text ends with a delimiter, there's no partial word to filter by
	lastChar, _ := utf8.DecodeLastRuneInString(text)
	if lastChar == ' ' || lastChar == '\n' || lastChar == '\t' || lastChar == ',' || lastChar == '(' || lastChar == ')' {
		return ""
	}

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
