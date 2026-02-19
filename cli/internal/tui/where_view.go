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

type WhereCondition struct {
	Field    string
	Operator string
	Value    string
}

type WhereView struct {
	parent        *MainModel
	conditions    []WhereCondition
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
}

func NewWhereView(parent *MainModel) *WhereView {
	ti := textinput.New()
	ti.Placeholder = "value"
	ti.CharLimit = 100
	ti.Width = 30
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.Foreground)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	return &WhereView{
		parent:        parent,
		conditions:    []WhereCondition{},
		valueInput:    ti,
		focusIndex:    0,
		operators:     []string{"=", "!=", ">", "<", ">=", "<=", "LIKE", "IN", "BETWEEN", "IS NULL", "IS NOT NULL"},
		addingNew:     false,
		selectedIndex: -1,
	}
}

func (v *WhereView) SetTableContext(schema, tableName string, columns []engine.Column, existingCondition *model.WhereCondition) {
	v.schema = schema
	v.tableName = tableName
	v.columns = columns

	// Load existing conditions if provided
	if existingCondition != nil && existingCondition.Type == model.WhereConditionTypeAnd && existingCondition.And != nil {
		v.conditions = []WhereCondition{}
		for _, child := range existingCondition.And.Children {
			if child.Type == model.WhereConditionTypeAtomic && child.Atomic != nil {
				v.conditions = append(v.conditions, WhereCondition{
					Field:    child.Atomic.Key,
					Operator: child.Atomic.Operator,
					Value:    child.Atomic.Value,
				})
			}
		}
	} else {
		v.conditions = []WhereCondition{}
	}

	v.currentField = ""
	v.currentOp = ""
	v.valueInput.SetValue("")
	v.addingNew = false
	v.selectedIndex = -1
}

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
				if v.selectedIndex > 0 {
					v.selectedIndex--
				}
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
				if v.selectedIndex < len(v.conditions)-1 {
					v.selectedIndex++
				} else if v.selectedIndex == -1 && len(v.conditions) > 0 {
					v.selectedIndex = 0
				}
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

		case key.Matches(msg, Keys.WhereList.Add):
			if !v.addingNew {
				v.addingNew = true
				v.selectedIndex = -1
				v.focusIndex = 0
				v.currentField = ""
				v.currentOp = ""
				v.valueInput.SetValue("")
				return v, nil
			}

		case key.Matches(msg, Keys.WhereList.Delete):
			if !v.addingNew && v.selectedIndex >= 0 && v.selectedIndex < len(v.conditions) {
				v.conditions = append(v.conditions[:v.selectedIndex], v.conditions[v.selectedIndex+1:]...)
				if v.selectedIndex >= len(v.conditions) && v.selectedIndex > 0 {
					v.selectedIndex--
				}
				return v, nil
			}

		case key.Matches(msg, Keys.WhereList.EditCond):
			if !v.addingNew && v.selectedIndex >= 0 && v.selectedIndex < len(v.conditions) {
				// Load the selected condition into edit mode
				cond := v.conditions[v.selectedIndex]
				v.addingNew = true
				v.focusIndex = 0
				v.currentField = cond.Field
				v.currentOp = cond.Operator
				v.valueInput.SetValue(cond.Value)
				// Remove the condition being edited
				v.conditions = append(v.conditions[:v.selectedIndex], v.conditions[v.selectedIndex+1:]...)
				v.selectedIndex = -1
				return v, nil
			}

		case key.Matches(msg, Keys.WhereList.Apply):
			if v.addingNew {
				if v.focusIndex == 3 {
					if v.currentField != "" && v.currentOp != "" && v.valueInput.Value() != "" {
						v.conditions = append(v.conditions, WhereCondition{
							Field:    v.currentField,
							Operator: v.currentOp,
							Value:    v.valueInput.Value(),
						})
						v.addingNew = false
						v.currentField = ""
						v.currentOp = ""
						v.valueInput.SetValue("")
						v.valueInput.Blur()
					}
					return v, nil
				}
			} else {
				v.parent.resultsView.whereCondition = v.buildWhereCondition()
				v.parent.resultsView.loadWithWhere()
				if !v.parent.PopView() {
					v.parent.mode = ViewResults
				}
				return v, nil
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
				if v.selectedIndex > 0 {
					v.selectedIndex--
				}
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
				if v.selectedIndex < len(v.conditions)-1 {
					v.selectedIndex++
				} else if v.selectedIndex == -1 && len(v.conditions) > 0 {
					v.selectedIndex = 0
				}
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

func (v *WhereView) View() string {
	var b strings.Builder

	b.WriteString(styles.RenderTitle("WHERE Conditions"))
	b.WriteString("\n")
	b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("Table: %s.%s", v.schema, v.tableName)))
	b.WriteString("\n\n")

	if len(v.conditions) == 0 {
		b.WriteString(styles.MutedStyle.Render("No conditions added yet"))
	} else {
		b.WriteString(styles.KeyStyle.Render("Current Conditions:"))
		b.WriteString("\n\n")
		for i, cond := range v.conditions {
			prefix := "  "
			if i == v.selectedIndex {
				prefix = styles.KeyStyle.Render("▶ ")
			}
			condStr := fmt.Sprintf("%s %s %s", cond.Field, cond.Operator, cond.Value)
			if i == v.selectedIndex {
				b.WriteString(prefix + styles.ActiveListItemStyle.Render(condStr))
			} else {
				b.WriteString(prefix + styles.ListItemStyle.Render(condStr))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	if v.addingNew {
		b.WriteString(styles.KeyStyle.Render("Add New Condition:"))
		b.WriteString("\n\n")

		fieldLabel := "Field:"
		if v.focusIndex == 0 {
			fieldLabel = styles.KeyStyle.Render("▶ Field:")
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
			b.WriteString(styles.MutedStyle.Render("(use ← → to select)"))
		}
		b.WriteString("\n\n")

		opLabel := "Operator:"
		if v.focusIndex == 1 {
			opLabel = styles.KeyStyle.Render("▶ Operator:")
		} else {
			opLabel = "  Operator:"
		}
		b.WriteString(opLabel)
		b.WriteString("\n  ")
		if v.currentOp != "" {
			if v.focusIndex == 1 {
				b.WriteString(styles.ActiveListItemStyle.Render(v.currentOp))
			} else {
				b.WriteString(styles.KeyStyle.Render(v.currentOp))
			}
		} else {
			b.WriteString(styles.MutedStyle.Render("(use ← → to select)"))
		}
		b.WriteString("\n\n")

		valueLabel := "Value:"
		if v.focusIndex == 2 {
			valueLabel = styles.KeyStyle.Render("▶ Value:")
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
			addLabel = styles.KeyStyle.Render("[" + addLabel + "]")
		}
		b.WriteString("  " + addLabel)
		b.WriteString("\n\n")

		b.WriteString(RenderBindingHelp(
			Keys.WhereAdd.Prev,
			Keys.WhereAdd.Next,
			Keys.WhereAdd.Change,
			Keys.WhereAdd.Confirm,
			Keys.Global.Back,
		))
	} else {
		b.WriteString("\n")
		b.WriteString(RenderBindingHelp(
			Keys.WhereList.Up,
			Keys.WhereList.Down,
			Keys.WhereList.Add,
			Keys.WhereList.EditCond,
			Keys.WhereList.Delete,
			Keys.WhereList.Apply,
			Keys.Global.Back,
			Keys.Global.Quit,
		))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
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

func (v *WhereView) buildWhereCondition() *model.WhereCondition {
	if len(v.conditions) == 0 {
		return nil
	}

	children := make([]*model.WhereCondition, len(v.conditions))
	for i, cond := range v.conditions {
		children[i] = &model.WhereCondition{
			Type: model.WhereConditionTypeAtomic,
			Atomic: &model.AtomicWhereCondition{
				Key:        cond.Field,
				Operator:   cond.Operator,
				Value:      cond.Value,
				ColumnType: "string",
			},
		}
	}

	return &model.WhereCondition{
		Type: model.WhereConditionTypeAnd,
		And: &model.OperationWhereCondition{
			Children: children,
		},
	}
}
