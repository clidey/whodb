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
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/pkg/styles"
)

type connectionItem struct {
	conn config.Connection
}

func (i connectionItem) Title() string       { return i.conn.Name }
func (i connectionItem) Description() string { return fmt.Sprintf("%s@%s", i.conn.Type, i.conn.Host) }
func (i connectionItem) FilterValue() string { return i.conn.Name }

type connectionDelegate struct{}

func (d connectionDelegate) Height() int                             { return 2 }
func (d connectionDelegate) Spacing() int                            { return 1 }
func (d connectionDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d connectionDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(connectionItem)
	if !ok {
		return
	}

	str := ""
	if index == m.Index() {
		str = styles.ActiveListItemStyle.Render("▶ " + i.Title())
	} else {
		str = "  " + i.Title()
	}
	str += "\n  " + styles.MutedStyle.Render(i.Description())
	fmt.Fprint(w, str)
}

type connectionResultMsg struct {
	err error
}

type escTimeoutTickMsg struct{}

// ConnectionView provides the TUI for managing database connections.
// It supports both a list view (selecting from saved connections) and
// a form view (creating new connections).
type ConnectionView struct {
	parent      *MainModel
	list        list.Model
	mode        string // "list" or "form"
	inputs      []textinput.Model
	focusIndex  int
	dbTypes     []string
	dbTypeIndex int
	connecting  bool
	connError   error
	// Deferred password prompt when connecting with empty password
	awaitingPassword bool
	passwordPrompt   textinput.Model
	// ESC confirmation
	escPressed     bool
	escTimeoutSecs int
}

// NewConnectionView creates a connection view initialized with saved connections
// from the parent's config. If no connections exist, it starts in form mode.
func NewConnectionView(parent *MainModel) *ConnectionView {
	var items []list.Item
	for _, conn := range parent.config.Connections {
		items = append(items, connectionItem{conn: conn})
	}

	l := list.New(items, connectionDelegate{}, 0, 0)
	l.Title = ""
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetStatusBarItemName("connection available", "connections available")

	// Initialize form inputs
	inputs := make([]textinput.Model, 5)

	// Connection name
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "My Connection"
	inputs[0].CharLimit = 50
	inputs[0].Width = 40
	inputs[0].PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	inputs[0].TextStyle = lipgloss.NewStyle().Foreground(styles.Foreground)
	inputs[0].Cursor.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	// Host
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "localhost"
	inputs[1].CharLimit = 100
	inputs[1].Width = 40
	inputs[1].PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	inputs[1].TextStyle = lipgloss.NewStyle().Foreground(styles.Foreground)
	inputs[1].Cursor.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	// Port
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "5432"
	inputs[2].CharLimit = 5
	inputs[2].Width = 40
	inputs[2].PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	inputs[2].TextStyle = lipgloss.NewStyle().Foreground(styles.Foreground)
	inputs[2].Cursor.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	// Username
	inputs[3] = textinput.New()
	inputs[3].Placeholder = "postgres"
	inputs[3].CharLimit = 50
	inputs[3].Width = 40
	inputs[3].PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	inputs[3].TextStyle = lipgloss.NewStyle().Foreground(styles.Foreground)
	inputs[3].Cursor.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	// Password
	inputs[4] = textinput.New()
	inputs[4].Placeholder = "password"
	inputs[4].EchoMode = textinput.EchoPassword
	inputs[4].EchoCharacter = '•'
	inputs[4].CharLimit = 100
	inputs[4].Width = 40
	inputs[4].PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	inputs[4].TextStyle = lipgloss.NewStyle().Foreground(styles.Foreground)
	inputs[4].Cursor.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	// Database name
	dbInput := textinput.New()
	dbInput.Placeholder = "mydb"
	dbInput.CharLimit = 50
	dbInput.Width = 40
	dbInput.PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	dbInput.TextStyle = lipgloss.NewStyle().Foreground(styles.Foreground)
	dbInput.Cursor.Style = lipgloss.NewStyle().Foreground(styles.Primary)
	inputs = append(inputs, dbInput)

	// Schema name (optional)
	schemaInput := textinput.New()
	schemaInput.Placeholder = "Schema name (optional)"
	schemaInput.CharLimit = 50
	schemaInput.Width = 40
	schemaInput.PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	schemaInput.TextStyle = lipgloss.NewStyle().Foreground(styles.Foreground)
	schemaInput.Cursor.Style = lipgloss.NewStyle().Foreground(styles.Primary)
	inputs = append(inputs, schemaInput)

	mode := "list"
	if len(parent.config.Connections) == 0 {
		mode = "form"
		inputs[0].Focus()
	}

	// Password prompt (shown after pressing Connect if password is empty)
	prompt := textinput.New()
	prompt.Placeholder = "enter password"
	prompt.EchoMode = textinput.EchoPassword
	prompt.EchoCharacter = '•'
	prompt.CharLimit = 100
	prompt.Width = 40
	prompt.PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	prompt.TextStyle = lipgloss.NewStyle().Foreground(styles.Foreground)
	prompt.Cursor.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	return &ConnectionView{
		parent:           parent,
		list:             l,
		mode:             mode,
		inputs:           inputs,
		focusIndex:       0,
		dbTypes:          []string{"Postgres", "MySQL", "SQLite", "MongoDB", "Redis", "MariaDB", "ClickHouse", "ElasticSearch"},
		dbTypeIndex:      0,
		connecting:       false,
		awaitingPassword: false,
		passwordPrompt:   prompt,
	}
}

func (v *ConnectionView) Update(msg tea.Msg) (*ConnectionView, tea.Cmd) {
	if v.mode == "form" {
		return v.updateForm(msg)
	}
	return v.updateList(msg)
}

func (v *ConnectionView) updateList(msg tea.Msg) (*ConnectionView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.list.SetSize(msg.Width, msg.Height-8)
		return v, nil

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			v.list.CursorUp()
			return v, nil
		case tea.MouseButtonWheelDown:
			v.list.CursorDown()
			return v, nil
		}

	case escTimeoutTickMsg:
		if v.escPressed {
			v.escTimeoutSecs--
			if v.escTimeoutSecs <= 0 {
				v.escPressed = false
				v.escTimeoutSecs = 0
			}
			return v, tea.Tick(time.Second, func(time.Time) tea.Msg { return escTimeoutTickMsg{} })
		}
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			v.list.CursorDown()
			return v, nil

		case "shift+tab":
			v.list.CursorUp()
			return v, nil

		case "enter":
			if item, ok := v.list.SelectedItem().(connectionItem); ok {
				if err := v.parent.dbManager.Connect(&item.conn); err != nil {
					v.parent.err = err
					return v, nil
				}
				v.parent.mode = ViewBrowser
				return v, v.parent.browserView.Init()
			}

		case "n":
			v.mode = "form"
			v.resetForm()
			v.inputs[0].Focus()
			return v, nil

		case "d":
			if item, ok := v.list.SelectedItem().(connectionItem); ok {
				v.parent.config.RemoveConnection(item.conn.Name)
				v.parent.config.Save()
				v.refreshList()
				return v, nil
			}

		case "esc":
			if v.escPressed {
				// Second ESC press - confirm quit
				return v, tea.Quit
			}
			// First ESC press - show confirmation
			v.escPressed = true
			v.escTimeoutSecs = 3
			return v, tea.Tick(time.Second, func(time.Time) tea.Msg { return escTimeoutTickMsg{} })
		}
	}

	// Clear ESC confirmation on any other key press
	if _, ok := msg.(tea.KeyMsg); ok && msg.(tea.KeyMsg).String() != "esc" && !v.escPressed {
		// Not specifically checking escPressed here since it's already reset by timeout or second press
	}

	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

func (v *ConnectionView) updateForm(msg tea.Msg) (*ConnectionView, tea.Cmd) {
	var cmd tea.Cmd

	// Handle deferred password prompt overlay
	if v.awaitingPassword {
		switch m := msg.(type) {
		case tea.KeyMsg:
			switch m.String() {
			case "enter":
				// Set the password and proceed to connect
				v.inputs[4].SetValue(v.passwordPrompt.Value())
				v.passwordPrompt.SetValue("")
				v.awaitingPassword = false
				v.connecting = true
				v.connError = nil
				return v, v.connect()
			case "esc":
				// Cancel password prompt
				v.passwordPrompt.SetValue("")
				v.awaitingPassword = false
				return v, nil
			}
		}
		v.passwordPrompt, cmd = v.passwordPrompt.Update(msg)
		return v, cmd
	}

	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			v.prevInput()
			return v, nil
		case tea.MouseButtonWheelDown:
			v.nextInput()
			return v, nil
		}
	case connectionResultMsg:
		if msg.err != nil {
			v.connError = msg.err
			v.connecting = false
		} else {
			v.parent.mode = ViewBrowser
			return v, v.parent.browserView.Init()
		}
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return v, tea.Quit

		case "esc":
			if len(v.parent.config.Connections) > 0 {
				v.mode = "list"
				v.connError = nil
				return v, nil
			}
			return v, tea.Quit

		case "tab", "down":
			v.nextInput()
			return v, nil

		case "shift+tab", "up":
			v.prevInput()
			return v, nil

		case "left":
			if v.focusIndex == 7 {
				v.dbTypeIndex--
				if v.dbTypeIndex < 0 {
					v.dbTypeIndex = len(v.dbTypes) - 1
				}
				v.updatePortPlaceholder()
			}
			return v, nil

		case "right":
			if v.focusIndex == 7 {
				v.dbTypeIndex++
				if v.dbTypeIndex >= len(v.dbTypes) {
					v.dbTypeIndex = 0
				}
				v.updatePortPlaceholder()
			}
			return v, nil

		case "enter":
			if v.focusIndex == 8 {
				// If password is empty, prompt securely before connecting
				if v.inputs[4].Value() == "" {
					v.awaitingPassword = true
					v.passwordPrompt.SetValue("")
					v.passwordPrompt.Focus()
					return v, nil
				}
				v.connecting = true
				v.connError = nil
				return v, v.connect()
			}
			v.nextInput()
			return v, nil
		}
	}

	if v.focusIndex >= 0 && v.focusIndex < len(v.inputs) {
		v.inputs[v.focusIndex], cmd = v.inputs[v.focusIndex].Update(msg)
	}

	return v, cmd
}

func (v *ConnectionView) View() string {
	if v.mode == "form" {
		return v.renderForm()
	}

	var b strings.Builder

	b.WriteString(styles.RenderTitle("Welcome to WhoDB!"))
	b.WriteString("\n")
	b.WriteString(styles.MutedStyle.Render("Select an existing connection below, or create a new one with [n]"))
	b.WriteString("\n\n")
	b.WriteString(v.list.View())
	b.WriteString("\n\n")
	b.WriteString(styles.RenderHelp(
		"↑/k/shift+tab", "up",
		"↓/j/tab", "down",
		"enter", "connect",
		"[n]", "new",
		"[d]", "delete",
		"esc", "quit",
		"ctrl+c", "force quit",
	))

	content := lipgloss.NewStyle().Padding(1, 2).Render(b.String())

	if v.escPressed {
		confirmMsg := fmt.Sprintf("Press ESC again to quit (%ds)", v.escTimeoutSecs)
		confirmBox := styles.RenderErrorBox(confirmMsg)
		return content + "\n" + confirmBox
	}

	return content
}

func (v *ConnectionView) renderForm() string {
	var b strings.Builder

	b.WriteString(styles.RenderTitle("New Database Connection"))
	b.WriteString("\n\n")

	if v.connError != nil {
		b.WriteString(styles.RenderErrorBox(v.connError.Error()))
		b.WriteString("\n\n")
	}

	// Connection Name (index 0)
	label := "Connection Name:"
	if v.focusIndex == 0 {
		label = styles.KeyStyle.Render("▶ " + label)
	} else {
		label = "  " + label
	}
	b.WriteString(label)
	b.WriteString("\n  ")
	b.WriteString(v.inputs[0].View())
	b.WriteString("\n\n")

	// Host (index 1)
	label = "Host:"
	if v.focusIndex == 1 {
		label = styles.KeyStyle.Render("▶ " + label)
	} else {
		label = "  " + label
	}
	b.WriteString(label)
	b.WriteString("\n  ")
	b.WriteString(v.inputs[1].View())
	b.WriteString("\n\n")

	// Port (index 2)
	label = "Port:"
	if v.focusIndex == 2 {
		label = styles.KeyStyle.Render("▶ " + label)
	} else {
		label = "  " + label
	}
	b.WriteString(label)
	b.WriteString("\n  ")
	b.WriteString(v.inputs[2].View())
	b.WriteString("\n\n")

	// Username (index 3)
	label = "Username:"
	if v.focusIndex == 3 {
		label = styles.KeyStyle.Render("▶ " + label)
	} else {
		label = "  " + label
	}
	b.WriteString(label)
	b.WriteString("\n  ")
	b.WriteString(v.inputs[3].View())
	b.WriteString("\n\n")

	// Password (index 4)
	label = "Password:"
	if v.focusIndex == 4 {
		label = styles.KeyStyle.Render("▶ " + label)
	} else {
		label = "  " + label
	}
	b.WriteString(label)
	b.WriteString("\n  ")
	b.WriteString(v.inputs[4].View())
	b.WriteString("\n\n")

	// Database (index 5)
	label = "Database:"
	if v.focusIndex == 5 {
		label = styles.KeyStyle.Render("▶ " + label)
	} else {
		label = "  " + label
	}
	b.WriteString(label)
	b.WriteString("\n  ")
	b.WriteString(v.inputs[5].View())
	b.WriteString("\n\n")

	// Schema (index 6)
	label = "Schema:"
	if v.focusIndex == 6 {
		label = styles.KeyStyle.Render("▶ " + label)
	} else {
		label = "  " + label
	}
	b.WriteString(label)
	b.WriteString("\n  ")
	b.WriteString(v.inputs[6].View())
	b.WriteString("\n\n")

	// Database Type (index 7)
	label = "Database Type:"
	if v.focusIndex == 7 {
		label = styles.KeyStyle.Render("▶ " + label)
	} else {
		label = "  " + label
	}
	b.WriteString(label)
	b.WriteString("\n  ")
	for i, dbType := range v.dbTypes {
		if i == v.dbTypeIndex {
			if v.focusIndex == 7 {
				b.WriteString(styles.ActiveListItemStyle.Render(" " + dbType + " "))
			} else {
				b.WriteString(styles.KeyStyle.Render("[" + dbType + "]"))
			}
		} else {
			b.WriteString(styles.MutedStyle.Render(" " + dbType + " "))
		}
		b.WriteString(" ")
	}
	b.WriteString("\n\n")

	// Connect button (index 8)
	connectBtn := "[Connect]"
	if v.focusIndex == 8 {
		connectBtn = styles.ActiveListItemStyle.Render(" Connect ")
	} else {
		connectBtn = styles.KeyStyle.Render(connectBtn)
	}
	b.WriteString("  " + connectBtn)
	b.WriteString("\n\n")

	// If awaiting password, render overlay prompt
	if v.awaitingPassword {
		b.WriteString(styles.RenderTitle("Enter Password"))
		b.WriteString("\n  ")
		b.WriteString(v.passwordPrompt.View())
		b.WriteString("\n\n")
		b.WriteString(styles.RenderHelp(
			"enter", "confirm",
			"esc", "cancel",
		))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	helpText := ""
	if len(v.parent.config.Connections) > 0 {
		helpText = styles.RenderHelp(
			"↑/↓/tab", "navigate",
			"←/→", "change type",
			"enter", "connect",
			"esc", "back",
			"ctrl+c", "quit",
		)
	} else {
		helpText = styles.RenderHelp(
			"↑/↓/tab", "navigate",
			"←/→", "change type",
			"enter", "connect",
			"ctrl+c", "quit",
		)
	}
	b.WriteString(helpText)

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (v *ConnectionView) refreshList() {
	var items []list.Item
	for _, conn := range v.parent.config.Connections {
		items = append(items, connectionItem{conn: conn})
	}
	v.list.SetItems(items)
}

func (v *ConnectionView) nextInput() {
	if v.focusIndex < len(v.inputs) {
		v.inputs[v.focusIndex].Blur()
	}
	v.focusIndex++
	if v.focusIndex > 8 {
		v.focusIndex = 0
	}
	if v.focusIndex < len(v.inputs) {
		v.inputs[v.focusIndex].Focus()
	}
}

func (v *ConnectionView) prevInput() {
	if v.focusIndex < len(v.inputs) {
		v.inputs[v.focusIndex].Blur()
	}
	v.focusIndex--
	if v.focusIndex < 0 {
		v.focusIndex = 8
	}
	if v.focusIndex < len(v.inputs) {
		v.inputs[v.focusIndex].Focus()
	}
}

func (v *ConnectionView) resetForm() {
	for i := range v.inputs {
		v.inputs[i].SetValue("")
		v.inputs[i].Blur()
	}
	v.focusIndex = 0
	v.dbTypeIndex = 0
	v.connError = nil
	v.inputs[0].Focus()
	v.updatePortPlaceholder()
}

func (v *ConnectionView) updatePortPlaceholder() {
	defaultPort := v.getDefaultPort(v.dbTypes[v.dbTypeIndex])
	v.inputs[2].Placeholder = strconv.Itoa(defaultPort)
}

func (v *ConnectionView) getDefaultPort(dbType string) int {
	switch dbType {
	case "Postgres":
		return 5432
	case "MySQL", "MariaDB":
		return 3306
	case "MongoDB":
		return 27017
	case "Redis":
		return 6379
	case "ClickHouse":
		return 9000
	case "ElasticSearch":
		return 9200
	case "SQLite":
		return 0
	default:
		return 5432
	}
}

func (v *ConnectionView) connect() tea.Cmd {
	return func() tea.Msg {
		name := v.inputs[0].Value()
		host := v.inputs[1].Value()
		if host == "" {
			host = "localhost"
		}

		portStr := v.inputs[2].Value()
		var port int
		if portStr == "" {
			port = v.getDefaultPort(v.dbTypes[v.dbTypeIndex])
		} else {
			portNum, err := strconv.Atoi(portStr)
			if err != nil || portNum < 1024 || portNum > 65535 {
				return connectionResultMsg{err: fmt.Errorf("invalid port number: must be between 1024 and 65535 (ports below 1024 are system reserved)")}
			}
			port = portNum
		}

		username := v.inputs[3].Value()
		password := v.inputs[4].Value()
		database := v.inputs[5].Value()
		schema := v.inputs[6].Value()
		dbType := v.dbTypes[v.dbTypeIndex]

		conn := config.Connection{
			Name:     name,
			Type:     dbType,
			Host:     host,
			Port:     port,
			Username: username,
			Password: password,
			Database: database,
			Schema:   schema,
		}

		// Try to connect
		if err := v.parent.dbManager.Connect(&conn); err != nil {
			return connectionResultMsg{err: err}
		}

		// Save connection if name is provided
		if name != "" {
			v.parent.config.AddConnection(conn)
			v.parent.config.Save()
		}

		return connectionResultMsg{err: nil}
	}
}
