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

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/pkg/styles"
)

// auditResultMsg is sent when the audit scan completes.
type auditResultMsg struct {
	results []*database.TableAudit
	err     error
	schema  string
}

// AuditView displays data quality audit results for the connected database.
// Accessible via Ctrl+U.
type AuditView struct {
	parent       *MainModel
	results      []*database.TableAudit
	loading      bool
	err          error
	viewport     viewport.Model
	width        int
	height       int
	ready        bool
	schema       string
	cursor       int // index into the flattened issue list
	issueQueries []string
}

// NewAuditView creates a new AuditView attached to the given parent model.
func NewAuditView(parent *MainModel) *AuditView {
	return &AuditView{
		parent: parent,
		width:  80,
		height: 20,
	}
}

// Update handles input and messages for the audit view.
func (v *AuditView) Update(msg tea.Msg) (*AuditView, tea.Cmd) {
	switch msg := msg.(type) {
	case auditResultMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.results = msg.results
		v.schema = msg.schema
		v.cursor = 0
		v.rebuildViewport()
		return v, nil

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		if !v.loading && v.err == nil && v.results != nil {
			v.rebuildViewport()
		}
		return v, nil

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, Keys.Global.Back):
			if !v.parent.PopView() {
				v.parent.mode = ViewBrowser
			}
			return v, nil

		case key.Matches(msg, Keys.Audit.Up):
			if v.cursor > 0 {
				v.cursor--
				v.rebuildViewport()
			}
			return v, nil

		case key.Matches(msg, Keys.Audit.Down):
			if len(v.issueQueries) > 0 && v.cursor < len(v.issueQueries)-1 {
				v.cursor++
				v.rebuildViewport()
			}
			return v, nil

		case key.Matches(msg, Keys.Audit.DrillDown):
			if v.cursor >= 0 && v.cursor < len(v.issueQueries) {
				q := v.issueQueries[v.cursor]
				if q != "" {
					v.parent.editorView.textarea.SetValue(q)
					if !v.parent.PopView() {
						v.parent.mode = ViewEditor
					} else {
						v.parent.mode = ViewEditor
					}
					return v, v.parent.editorView.executeQuery()
				}
			}
			return v, nil
		}
	}

	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return v, cmd
}

// View renders the audit view.
func (v *AuditView) View() string {
	var b strings.Builder

	title := "Data Quality Audit"
	if v.schema != "" {
		title += " -- " + v.schema
	}
	b.WriteString(styles.RenderTitle(title))
	b.WriteString("\n\n")

	if v.err != nil {
		b.WriteString(styles.RenderErrorBox(v.err.Error()))
		b.WriteString("\n\n")
		b.WriteString(RenderBindingHelpWidth(v.width, Keys.Global.Back))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	if v.loading {
		b.WriteString(v.parent.SpinnerView() + styles.RenderMuted(" Scanning tables..."))
		b.WriteString("\n\n")
		b.WriteString(RenderBindingHelpWidth(v.width, Keys.Global.Back))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	if len(v.results) == 0 {
		b.WriteString(styles.RenderMuted("No tables found."))
		b.WriteString("\n\n")
		b.WriteString(RenderBindingHelpWidth(v.width, Keys.Global.Back))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	// Summary
	totalIssues := 0
	for _, t := range v.results {
		totalIssues += len(t.Issues)
	}
	summary := fmt.Sprintf("%d tables scanned, %d issues found", len(v.results), totalIssues)
	b.WriteString(styles.RenderMuted(summary))
	b.WriteString("\n\n")

	if !v.ready {
		v.rebuildViewport()
	}
	b.WriteString(v.viewport.View())
	b.WriteString("\n\n")

	b.WriteString(RenderBindingHelpWidth(v.width,
		Keys.Audit.Up,
		Keys.Audit.Down,
		Keys.Audit.DrillDown,
		Keys.Global.Back,
	))

	if v.viewport.TotalLineCount() > v.viewport.VisibleLineCount() {
		pct := v.viewport.ScrollPercent() * 100
		var scrollPct string
		if pct >= 99.5 {
			scrollPct = "bottom"
		} else {
			scrollPct = fmt.Sprintf("%.0f%%", pct)
		}
		b.WriteString("  ")
		b.WriteString(styles.RenderMuted(scrollPct))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

// loadAuditData returns a tea.Cmd that fetches audit data for all tables.
func (v *AuditView) loadAuditData() tea.Cmd {
	browserSchema := v.parent.browserView.currentSchema

	return func() tea.Msg {
		conn := v.parent.dbManager.GetCurrentConnection()
		if conn == nil {
			return auditResultMsg{err: fmt.Errorf("no connection")}
		}

		schema := browserSchema
		if schema == "" {
			schemas, err := v.parent.dbManager.GetSchemas()
			if err != nil {
				schemas = []string{}
			}
			if len(schemas) > 0 {
				schema = selectBestSchema(schemas)
			}
		}

		config := database.DefaultAuditConfig()
		results, err := v.parent.dbManager.AuditSchema(schema, config)
		if err != nil {
			return auditResultMsg{err: fmt.Errorf("audit failed: %w", err)}
		}

		return auditResultMsg{results: results, schema: schema}
	}
}

// rebuildViewport re-renders the audit results and sets the viewport content.
func (v *AuditView) rebuildViewport() {
	contentWidth := v.width - 8
	if contentWidth < 20 {
		contentWidth = 20
	}
	contentHeight := v.height - 14
	if contentHeight < 3 {
		contentHeight = 3
	}

	content, queries := v.renderAuditContent(contentWidth)
	v.issueQueries = queries

	v.viewport = viewport.New(viewport.WithWidth(contentWidth), viewport.WithHeight(contentHeight))
	v.viewport.SetContent(content)
	v.ready = true
}

// renderAuditContent builds the display string for all audit results.
// Returns the rendered string and a parallel slice of query strings for drill-down.
func (v *AuditView) renderAuditContent(width int) (string, []string) {
	var b strings.Builder
	var queries []string
	issueIdx := 0

	for _, tbl := range v.results {
		// Table header
		tableSeverity := v.tableSeverity(tbl)
		icon := severityIcon(tableSeverity)
		tableHeader := fmt.Sprintf("%s %s (%d rows)", icon, tbl.TableName, tbl.RowCount)
		b.WriteString(styles.RenderKey(tableHeader))
		b.WriteString("\n")

		if !tbl.HasPrimaryKey {
			b.WriteString(styles.RenderMuted("  No primary key"))
			b.WriteString("\n")
		}

		// Column health
		for _, col := range tbl.Columns {
			colIcon := severityIcon(col.Severity)
			pkLabel := ""
			if col.IsPrimary {
				pkLabel = " [PK]"
			}
			line := fmt.Sprintf("  %s %-20s %-12s nulls:%.0f%% distinct:%d%s",
				colIcon, col.Name, col.Type, col.NullPct, col.DistinctCount, pkLabel)
			b.WriteString(line)
			b.WriteString("\n")
			for _, issue := range col.Issues {
				b.WriteString(styles.RenderMuted(fmt.Sprintf("      %s", issue)))
				b.WriteString("\n")
			}
		}

		// FK audit results
		for _, fk := range tbl.ForeignKeys {
			fkIcon := severityIcon(fk.Severity)
			fkLine := fmt.Sprintf("  %s FK %s.%s -> %s.%s",
				fkIcon, fk.SourceTable, fk.SourceColumn, fk.TargetTable, fk.TargetColumn)
			if fk.OrphanCount > 0 {
				fkLine += fmt.Sprintf(" (%d orphans)", fk.OrphanCount)
			}
			b.WriteString(fkLine)
			b.WriteString("\n")
		}

		// Duplicate results
		for _, dup := range tbl.Duplicates {
			dupLine := fmt.Sprintf("  ! Duplicates in %s: %d groups, %d rows",
				strings.Join(dup.Columns, ", "), dup.DuplicateCount, dup.TotalDuplicateRows)
			b.WriteString(styles.RenderMuted(dupLine))
			b.WriteString("\n")
		}

		// Issues list (navigable)
		for _, issue := range tbl.Issues {
			prefix := "  "
			if issueIdx == v.cursor {
				prefix = "> "
			}
			issueIcon := severityIcon(issue.Severity)
			line := fmt.Sprintf("%s%s %s", prefix, issueIcon, issue.Message)
			if issueIdx == v.cursor {
				b.WriteString(styles.RenderKey(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
			queries = append(queries, issue.Query)
			issueIdx++
		}

		b.WriteString("\n")
	}

	return b.String(), queries
}

// tableSeverity returns the worst severity across all issues in a table audit.
func (v *AuditView) tableSeverity(tbl *database.TableAudit) database.AuditSeverity {
	worst := database.SeverityOK
	for _, issue := range tbl.Issues {
		if issue.Severity == database.SeverityError {
			return database.SeverityError
		}
		if issue.Severity == database.SeverityWarning {
			worst = database.SeverityWarning
		}
	}
	for _, col := range tbl.Columns {
		if col.Severity == database.SeverityError {
			return database.SeverityError
		}
		if col.Severity == database.SeverityWarning {
			worst = database.SeverityWarning
		}
	}
	return worst
}

// severityIcon returns a color-coded icon for the given severity.
func severityIcon(severity database.AuditSeverity) string {
	switch severity {
	case database.SeverityOK:
		return styles.RenderOk("ok")
	case database.SeverityWarning:
		return styles.RenderWarn("!!")
	case database.SeverityError:
		return styles.RenderErr("XX")
	default:
		return styles.RenderMuted("--")
	}
}
