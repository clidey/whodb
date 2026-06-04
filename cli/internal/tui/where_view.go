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
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/pkg/styles"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
)

// WhereCondition represents a single atomic condition (field operator value).
type WhereCondition struct {
	Field    string
	Operator string
	Value    string
}

// conditionGroup holds a logical grouping of conditions and nested sub-groups.
type conditionGroup struct {
	Logic      string           // "AND" or "OR"
	Conditions []WhereCondition // leaf conditions in this group
	Groups     []conditionGroup // nested sub-groups
}

// treeItem is a flattened reference to a node in the condition tree used for
// cursor-based navigation.
type treeItem struct {
	GroupIndex     int // index in parent's Groups slice (-1 for root)
	ConditionIndex int // index within the group's Conditions slice (-1 if this is a group header)
	Depth          int // nesting depth for indentation
	IsGroup        bool
}

// WhereView implements the nested WHERE condition builder.
type WhereView struct {
	parent        *MainModel
	width         int
	height        int
	groups        []conditionGroup // top-level groups
	currentField  string
	currentOp     string
	valueInput    textinput.Model
	focusIndex    int
	selectedIndex int
	columns       []engine.Column
	operators     []string
	schema        string
	tableName     string
	addingNew     bool

	// flatItems is the flattened tree for navigation; rebuilt on every mutation.
	flatItems []treeItem
	// addTargetGroup tracks which group index a new condition is being added to.
	addTargetGroup int
}

// NewWhereView creates a new WhereView attached to the given parent model.
func NewWhereView(parent *MainModel) *WhereView {
	ti := textinput.New()
	ti.Placeholder = "value"
	ti.CharLimit = 100
	ti.Width = 30
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.Foreground)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	return &WhereView{
		parent:         parent,
		groups:         []conditionGroup{},
		valueInput:     ti,
		focusIndex:     0,
		operators:      []string{"=", "!=", ">", "<", ">=", "<=", "LIKE", "IN", "BETWEEN", "IS NULL", "IS NOT NULL"},
		addingNew:      false,
		selectedIndex:  -1,
		addTargetGroup: 0,
	}
}

// SetTableContext configures the view for a specific table and optionally
// loads existing conditions from a model.WhereCondition tree.
func (v *WhereView) SetTableContext(schema, tableName string, columns []engine.Column, existingCondition *model.WhereCondition) {
	v.schema = schema
	v.tableName = tableName
	v.columns = columns

	v.groups = nil

	if existingCondition != nil {
		v.groups = groupsFromWhereCondition(existingCondition)
	}

	// Ensure at least one group exists.
	if len(v.groups) == 0 {
		v.groups = []conditionGroup{{Logic: "AND"}}
	}

	v.currentField = ""
	v.currentOp = ""
	v.valueInput.SetValue("")
	v.addingNew = false
	v.selectedIndex = -1
	v.rebuildFlatItems()
}

// Update handles input events for the WHERE view.
func (v *WhereView) Update(msg tea.Msg) (*WhereView, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.valueInput.Width = clamp(msg.Width-16, 15, 50)
		return v, nil

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if v.addingNew {
				v.focusIndex--
				if v.focusIndex < 0 {
					v.focusIndex = 3
				}
				if v.focusIndex == 2 {
					v.valueInput.Focus()
				} else {
					v.valueInput.Blur()
				}
			} else {
				v.moveSelectionUp()
			}
			return v, nil
		case tea.MouseButtonWheelDown:
			if v.addingNew {
				v.focusIndex++
				if v.focusIndex > 3 {
					v.focusIndex = 0
				}
				if v.focusIndex == 2 {
					v.valueInput.Focus()
				} else {
					v.valueInput.Blur()
				}
			} else {
				v.moveSelectionDown()
			}
			return v, nil
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, Keys.Global.Back):
			if v.addingNew {
				v.addingNew = false
				v.currentField = ""
				v.currentOp = ""
				v.valueInput.SetValue("")
				v.valueInput.Blur()
				v.selectedIndex = -1
				return v, nil
			}
			if !v.parent.PopView() {
				v.parent.mode = ViewResults
			}
			return v, nil

		case key.Matches(msg, Keys.WhereList.NewGroup):
			if !v.addingNew {
				v.groups = append(v.groups, conditionGroup{Logic: "AND"})
				v.rebuildFlatItems()
				// Select the newly added group header.
				v.selectedIndex = len(v.flatItems) - 1
				return v, nil
			}

		case key.Matches(msg, Keys.WhereList.ToggleLogic):
			if !v.addingNew && v.selectedIndex >= 0 && v.selectedIndex < len(v.flatItems) {
				item := v.flatItems[v.selectedIndex]
				gi := item.GroupIndex
				if gi >= 0 && gi < len(v.groups) {
					if v.groups[gi].Logic == "AND" {
						v.groups[gi].Logic = "OR"
					} else {
						v.groups[gi].Logic = "AND"
					}
					v.rebuildFlatItems()
				}
				return v, nil
			}

		case key.Matches(msg, Keys.WhereList.Add):
			if !v.addingNew {
				v.addingNew = true
				v.focusIndex = 0
				v.currentField = ""
				v.currentOp = ""
				v.valueInput.SetValue("")
				// Determine which group to add to based on selection.
				v.addTargetGroup = v.selectedGroupIndex()
				return v, nil
			}

		case key.Matches(msg, Keys.WhereList.Delete):
			if !v.addingNew && v.selectedIndex >= 0 && v.selectedIndex < len(v.flatItems) {
				item := v.flatItems[v.selectedIndex]
				gi := item.GroupIndex
				if gi < 0 || gi >= len(v.groups) {
					return v, nil
				}
				if item.IsGroup {
					// Delete entire group.
					v.groups = append(v.groups[:gi], v.groups[gi+1:]...)
				} else if item.ConditionIndex >= 0 && item.ConditionIndex < len(v.groups[gi].Conditions) {
					ci := item.ConditionIndex
					v.groups[gi].Conditions = append(v.groups[gi].Conditions[:ci], v.groups[gi].Conditions[ci+1:]...)
				}
				v.rebuildFlatItems()
				if v.selectedIndex >= len(v.flatItems) && v.selectedIndex > 0 {
					v.selectedIndex = len(v.flatItems) - 1
				}
				if len(v.flatItems) == 0 {
					v.selectedIndex = -1
				}
				return v, nil
			}

		case key.Matches(msg, Keys.WhereList.EditCond):
			if !v.addingNew && v.selectedIndex >= 0 && v.selectedIndex < len(v.flatItems) {
				item := v.flatItems[v.selectedIndex]
				if item.IsGroup {
					return v, nil
				}
				gi := item.GroupIndex
				ci := item.ConditionIndex
				if gi < 0 || gi >= len(v.groups) || ci < 0 || ci >= len(v.groups[gi].Conditions) {
					return v, nil
				}
				cond := v.groups[gi].Conditions[ci]
				v.addingNew = true
				v.focusIndex = 0
				v.currentField = cond.Field
				v.currentOp = cond.Operator
				v.valueInput.SetValue(cond.Value)
				v.addTargetGroup = gi
				// Remove the condition being edited.
				v.groups[gi].Conditions = append(v.groups[gi].Conditions[:ci], v.groups[gi].Conditions[ci+1:]...)
				v.rebuildFlatItems()
				v.selectedIndex = -1
				return v, nil
			}

		case key.Matches(msg, Keys.WhereList.Apply):
			if v.addingNew {
				if v.focusIndex == 3 {
					isNullOp := v.currentOp == "IS NULL" || v.currentOp == "IS NOT NULL"
					if v.currentField != "" && v.currentOp != "" && (v.valueInput.Value() != "" || isNullOp) {
						value := v.valueInput.Value()
						if isNullOp {
							value = ""
						}
						gi := v.addTargetGroup
						if gi < 0 || gi >= len(v.groups) {
							gi = 0
						}
						if len(v.groups) == 0 {
							v.groups = append(v.groups, conditionGroup{Logic: "AND"})
							gi = 0
						}
						v.groups[gi].Conditions = append(v.groups[gi].Conditions, WhereCondition{
							Field:    v.currentField,
							Operator: v.currentOp,
							Value:    value,
						})
						v.addingNew = false
						v.currentField = ""
						v.currentOp = ""
						v.valueInput.SetValue("")
						v.valueInput.Blur()
						v.rebuildFlatItems()
					}
					return v, nil
				}
			} else {
				v.parent.resultsView.whereCondition = v.buildWhereCondition()
				cmd := v.parent.resultsView.loadWithWhere()
				if !v.parent.PopView() {
					v.parent.mode = ViewResults
				}
				return v, cmd
			}

		case key.Matches(msg, Keys.WhereList.Up):
			if v.addingNew {
				v.focusIndex--
				if v.focusIndex < 0 {
					v.focusIndex = 3
				}
				if v.focusIndex == 2 {
					v.valueInput.Focus()
				} else {
					v.valueInput.Blur()
				}
			} else {
				v.moveSelectionUp()
			}
			return v, nil

		case key.Matches(msg, Keys.WhereList.Down):
			if v.addingNew {
				v.focusIndex++
				if v.focusIndex > 3 {
					v.focusIndex = 0
				}
				if v.focusIndex == 2 {
					v.valueInput.Focus()
				} else {
					v.valueInput.Blur()
				}
			} else {
				v.moveSelectionDown()
			}
			return v, nil

		case key.Matches(msg, Keys.WhereAdd.Change):
			if v.addingNew {
				if msg.String() == "left" {
					switch v.focusIndex {
					case 0:
						idx := v.findColumnIndex(v.currentField)
						if idx > 0 {
							v.currentField = v.columns[idx-1].Name
						}
					case 1:
						idx := v.findOperatorIndex(v.currentOp)
						if idx > 0 {
							v.currentOp = v.operators[idx-1]
						}
					}
				} else {
					switch v.focusIndex {
					case 0:
						idx := v.findColumnIndex(v.currentField)
						if idx < len(v.columns)-1 {
							v.currentField = v.columns[idx+1].Name
						} else if v.currentField == "" && len(v.columns) > 0 {
							v.currentField = v.columns[0].Name
						}
					case 1:
						idx := v.findOperatorIndex(v.currentOp)
						if idx < len(v.operators)-1 {
							v.currentOp = v.operators[idx+1]
						} else if v.currentOp == "" && len(v.operators) > 0 {
							v.currentOp = v.operators[0]
						}
					}
				}
			}
			return v, nil
		}
	}

	if v.addingNew && v.focusIndex == 2 {
		v.valueInput, cmd = v.valueInput.Update(msg)
	}

	return v, cmd
}

// View renders the WHERE condition builder.
func (v *WhereView) View() string {
	var b strings.Builder

	b.WriteString(styles.RenderTitle("WHERE Conditions"))
	b.WriteString("\n")
	b.WriteString(styles.RenderMuted(fmt.Sprintf("Table: %s.%s", v.schema, v.tableName)))
	b.WriteString("\n\n")

	if len(v.groups) == 0 || (v.totalConditionCount() == 0 && !v.hasMultipleGroups()) {
		b.WriteString(styles.RenderMuted("No conditions added yet"))
	} else {
		b.WriteString(styles.RenderKey("Current Conditions:"))
		b.WriteString("\n\n")
		b.WriteString(v.renderTree())
	}

	b.WriteString("\n")

	if v.addingNew {
		targetLabel := ""
		if v.addTargetGroup >= 0 && v.addTargetGroup < len(v.groups) {
			targetLabel = fmt.Sprintf(" (to %s group %d)", v.groups[v.addTargetGroup].Logic, v.addTargetGroup+1)
		}
		b.WriteString(styles.RenderKey("Add New Condition" + targetLabel + ":"))
		b.WriteString("\n\n")

		var fieldLabel string
		if v.focusIndex == 0 {
			fieldLabel = styles.RenderKey("▶ Field:")
		} else {
			fieldLabel = "  Field:"
		}
		b.WriteString(fieldLabel)
		b.WriteString("\n  ")
		if v.currentField != "" {
			if col := v.findColumn(v.currentField); col != nil {
				displayText := fmt.Sprintf("%s (%s)", v.currentField, col.Type)
				style := styles.KeyStyle
				if v.focusIndex == 0 {
					style = styles.ActiveListItemStyle
				}
				b.WriteString(style.Render(displayText))
			}
		} else {
			b.WriteString(styles.RenderMuted("(use ← → to select)"))
		}
		b.WriteString("\n\n")

		var opLabel string
		if v.focusIndex == 1 {
			opLabel = styles.RenderKey("▶ Operator:")
		} else {
			opLabel = "  Operator:"
		}
		b.WriteString(opLabel)
		b.WriteString("\n  ")
		if v.currentOp != "" {
			if v.focusIndex == 1 {
				b.WriteString(styles.ActiveListItemStyle.Render(v.currentOp))
			} else {
				b.WriteString(styles.RenderKey(v.currentOp))
			}
		} else {
			b.WriteString(styles.RenderMuted("(use ← → to select)"))
		}
		b.WriteString("\n\n")

		var valueLabel string
		if v.focusIndex == 2 {
			valueLabel = styles.RenderKey("▶ Value:")
		} else {
			valueLabel = "  Value:"
		}
		b.WriteString(valueLabel)
		b.WriteString("\n  ")
		b.WriteString(v.valueInput.View())
		b.WriteString("\n\n")

		addLabel := "Add Condition"
		if v.focusIndex == 3 {
			addLabel = styles.ActiveListItemStyle.Render("[" + addLabel + "]")
		} else {
			addLabel = styles.RenderKey("[" + addLabel + "]")
		}
		b.WriteString("  " + addLabel)
		b.WriteString("\n\n")

		b.WriteString(renderBindingHelpWidthNoHelp(v.width,
			Keys.WhereAdd.Prev,
			Keys.WhereAdd.Next,
			Keys.WhereAdd.Change,
			Keys.WhereAdd.Confirm,
			Keys.Global.Back,
		))
	} else {
		b.WriteString("\n")
		b.WriteString(renderBindingHelpWidthNoHelp(v.width,
			Keys.WhereList.Up,
			Keys.WhereList.Down,
			Keys.WhereList.Add,
			Keys.WhereList.NewGroup,
			Keys.WhereList.ToggleLogic,
			Keys.WhereList.EditCond,
			Keys.WhereList.Delete,
			Keys.WhereList.Apply,
			Keys.Global.Back,
			Keys.Global.Quit,
		))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

// renderTree produces the indented tree representation of all groups and
// conditions.
func (v *WhereView) renderTree() string {
	var b strings.Builder
	for i, item := range v.flatItems {
		selected := i == v.selectedIndex
		prefix := "  "
		if selected {
			prefix = styles.RenderKey("▶ ")
		}
		indent := strings.Repeat("  ", item.Depth)

		if item.IsGroup {
			gi := item.GroupIndex
			g := v.groups[gi]
			condCount := len(g.Conditions)
			label := g.Logic
			if condCount > 1 {
				label += " ─┬─"
			} else if condCount == 1 {
				label += " ── "
			} else {
				label += " (empty)"
			}
			text := fmt.Sprintf("%s%s", indent, label)
			if selected {
				b.WriteString(prefix + styles.ActiveListItemStyle.Render(text))
			} else {
				b.WriteString(prefix + styles.ListItemStyle.Render(text))
			}
		} else {
			gi := item.GroupIndex
			ci := item.ConditionIndex
			cond := v.groups[gi].Conditions[ci]
			isLast := ci == len(v.groups[gi].Conditions)-1
			connector := "├─ "
			if isLast {
				connector = "└─ "
			}
			// Single-condition groups don't need connectors since the group
			// header already shows "── ".
			if len(v.groups[gi].Conditions) == 1 {
				connector = ""
			}
			condStr := fmt.Sprintf("%s%s%s %s %s", indent, connector, cond.Field, cond.Operator, cond.Value)
			if selected {
				b.WriteString(prefix + styles.ActiveListItemStyle.Render(condStr))
			} else {
				b.WriteString(prefix + styles.ListItemStyle.Render(condStr))
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

// rebuildFlatItems rebuilds the flat navigation list from the groups tree.
func (v *WhereView) rebuildFlatItems() {
	v.flatItems = nil
	for gi, g := range v.groups {
		v.flatItems = append(v.flatItems, treeItem{
			GroupIndex:     gi,
			ConditionIndex: -1,
			Depth:          0,
			IsGroup:        true,
		})
		for ci := range g.Conditions {
			v.flatItems = append(v.flatItems, treeItem{
				GroupIndex:     gi,
				ConditionIndex: ci,
				Depth:          1,
				IsGroup:        false,
			})
		}
	}
}

// moveSelectionUp moves the tree cursor up by one item.
func (v *WhereView) moveSelectionUp() {
	if v.selectedIndex > 0 {
		v.selectedIndex--
	}
}

// moveSelectionDown moves the tree cursor down by one item.
func (v *WhereView) moveSelectionDown() {
	if v.selectedIndex < len(v.flatItems)-1 {
		v.selectedIndex++
	} else if v.selectedIndex == -1 && len(v.flatItems) > 0 {
		v.selectedIndex = 0
	}
}

// selectedGroupIndex returns the group index that the current selection
// belongs to, defaulting to 0 if nothing is selected.
func (v *WhereView) selectedGroupIndex() int {
	if v.selectedIndex >= 0 && v.selectedIndex < len(v.flatItems) {
		return v.flatItems[v.selectedIndex].GroupIndex
	}
	if len(v.groups) > 0 {
		return 0
	}
	return 0
}

// totalConditionCount returns the total number of leaf conditions across all
// groups.
func (v *WhereView) totalConditionCount() int {
	n := 0
	for _, g := range v.groups {
		n += len(g.Conditions)
	}
	return n
}

// hasMultipleGroups returns true if there are two or more groups.
func (v *WhereView) hasMultipleGroups() bool {
	return len(v.groups) > 1
}

func (v *WhereView) findColumnIndex(name string) int {
	for i, col := range v.columns {
		if col.Name == name {
			return i
		}
	}
	return -1
}

func (v *WhereView) findOperatorIndex(op string) int {
	for i, operator := range v.operators {
		if operator == op {
			return i
		}
	}
	return -1
}

func (v *WhereView) findColumn(name string) *engine.Column {
	for i := range v.columns {
		if v.columns[i].Name == name {
			return &v.columns[i]
		}
	}
	return nil
}

// buildWhereCondition converts the group tree into a *model.WhereCondition
// suitable for the GetRows GraphQL query.
func (v *WhereView) buildWhereCondition() *model.WhereCondition {
	if v.totalConditionCount() == 0 {
		return nil
	}

	// Collect non-empty group conditions.
	var groupConditions []*model.WhereCondition
	for _, g := range v.groups {
		if len(g.Conditions) == 0 {
			continue
		}
		children := v.conditionsToAtomics(g.Conditions)
		if len(children) == 1 {
			groupConditions = append(groupConditions, children[0])
		} else {
			groupConditions = append(groupConditions, groupToWhereCondition(g.Logic, children))
		}
	}

	if len(groupConditions) == 0 {
		return nil
	}
	if len(groupConditions) == 1 {
		// Wrap single group in AND for consistency.
		return &model.WhereCondition{
			Type: model.WhereConditionTypeAnd,
			And: &model.OperationWhereCondition{
				Children: groupConditions,
			},
		}
	}

	// Multiple groups are joined with OR at the top level — each group
	// is its own AND/OR subtree, and groups combine with OR.
	return &model.WhereCondition{
		Type: model.WhereConditionTypeOr,
		Or: &model.OperationWhereCondition{
			Children: groupConditions,
		},
	}
}

// conditionsToAtomics converts a slice of WhereCondition into atomic model
// conditions.
func (v *WhereView) conditionsToAtomics(conds []WhereCondition) []*model.WhereCondition {
	out := make([]*model.WhereCondition, len(conds))
	for i, cond := range conds {
		colType := "string"
		for _, col := range v.columns {
			if cond.Field == col.Name {
				colType = col.Type
				break
			}
		}
		out[i] = &model.WhereCondition{
			Type: model.WhereConditionTypeAtomic,
			Atomic: &model.AtomicWhereCondition{
				Key:        cond.Field,
				Operator:   cond.Operator,
				Value:      cond.Value,
				ColumnType: colType,
			},
		}
	}
	return out
}

// groupToWhereCondition wraps children in an AND or OR operation node.
func groupToWhereCondition(logic string, children []*model.WhereCondition) *model.WhereCondition {
	op := &model.OperationWhereCondition{Children: children}
	if logic == "OR" {
		return &model.WhereCondition{
			Type: model.WhereConditionTypeOr,
			Or:   op,
		}
	}
	return &model.WhereCondition{
		Type: model.WhereConditionTypeAnd,
		And:  op,
	}
}

// groupsFromWhereCondition reconstructs condition groups from an existing
// model.WhereCondition tree. This is used when re-opening the WHERE view
// with previously-applied conditions.
func groupsFromWhereCondition(wc *model.WhereCondition) []conditionGroup {
	if wc == nil {
		return nil
	}

	switch wc.Type {
	case model.WhereConditionTypeAtomic:
		if wc.Atomic != nil {
			return []conditionGroup{{
				Logic: "AND",
				Conditions: []WhereCondition{{
					Field:    wc.Atomic.Key,
					Operator: wc.Atomic.Operator,
					Value:    wc.Atomic.Value,
				}},
			}}
		}

	case model.WhereConditionTypeAnd:
		if wc.And != nil {
			return groupsFromOperation("AND", wc.And.Children)
		}

	case model.WhereConditionTypeOr:
		if wc.Or != nil {
			return groupsFromOperation("OR", wc.Or.Children)
		}
	}

	return nil
}

// groupsFromOperation reconstructs groups from an AND/OR operation's children.
func groupsFromOperation(parentLogic string, children []*model.WhereCondition) []conditionGroup {
	// If all children are atomics, this is a single flat group.
	allAtomic := true
	for _, c := range children {
		if c.Type != model.WhereConditionTypeAtomic {
			allAtomic = false
			break
		}
	}

	if allAtomic {
		g := conditionGroup{Logic: parentLogic}
		for _, c := range children {
			if c.Atomic != nil {
				g.Conditions = append(g.Conditions, WhereCondition{
					Field:    c.Atomic.Key,
					Operator: c.Atomic.Operator,
					Value:    c.Atomic.Value,
				})
			}
		}
		return []conditionGroup{g}
	}

	// Mixed children: each child becomes its own group or is unpacked.
	var groups []conditionGroup
	for _, c := range children {
		sub := groupsFromWhereCondition(c)
		groups = append(groups, sub...)
	}
	return groups
}
