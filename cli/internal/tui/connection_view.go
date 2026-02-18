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
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/internal/config"
	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/pkg/styles"
)

type connectionItem struct {
	conn   config.Connection
	source string
}

func (i connectionItem) Title() string { return i.conn.Name }
func (i connectionItem) Description() string {
	desc := fmt.Sprintf("%s@%s", i.conn.Type, i.conn.Host)
	if i.source == dbmgr.ConnectionSourceEnv {
		desc += " (env)"
	}
	return desc
}
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
	parent        *MainModel
	list          list.Model
	mode          string // "list" or "form"
	inputs        []textinput.Model
	focusIndex    int
	dbTypes       []string
	dbTypeIndex   int
	visibleFields []int // indices of visible input fields for current db type
	connecting    bool
	connError     error
	// Deferred password prompt when connecting with empty password
	awaitingPassword bool
	passwordPrompt   textinput.Model
	// ESC confirmation
	escPressed     bool
	escTimeoutSecs int
	// Viewport for scrollable form
	formViewport viewport.Model
	formReady    bool
	width        int
	height       int
}

// NewConnectionView creates a connection view initialized with saved connections
// from the parent's config. If no connections exist, it starts in form mode.
func NewConnectionView(parent *MainModel) *ConnectionView {
	var items []list.Item
	for _, info := range parent.dbManager.ListConnectionsWithSource() {
		items = append(items, connectionItem{conn: info.Connection, source: info.Source})
	}

	l := list.New(items, connectionDelegate{}, 0, 0)
	l.Title = ""
	l.SetShowTitle(false)
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
	focusIndex := 7 // Start on db type selector
	if len(items) == 0 {
		mode = "form"
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

	dbTypes := []string{"Postgres", "MySQL", "SQLite", "MongoDB", "Redis", "MariaDB", "ClickHouse", "ElasticSearch"}

	return &ConnectionView{
		parent:           parent,
		list:             l,
		mode:             mode,
		inputs:           inputs,
		focusIndex:       focusIndex,
		dbTypes:          dbTypes,
		dbTypeIndex:      0,
		visibleFields:    getVisibleFields(dbTypes[0]),
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
		// Store dimensions; actual list sizing happens in View() using lipgloss.Height() measurements
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

func (v *ConnectionView) inputWidth() int {
	return clamp(v.width-8, 20, 60)
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
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		if !v.formReady {
			v.formViewport = viewport.New(msg.Width-4, msg.Height-8)
			v.formViewport.MouseWheelEnabled = true
			v.formReady = true
		}
		// Actual sizing happens in renderForm using lipgloss.Height() measurements
		return v, nil

	case tea.MouseMsg:
		// Forward mouse events to viewport for scroll handling
		if v.formReady {
			v.formViewport, cmd = v.formViewport.Update(msg)
			return v, cmd
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
				v.onDbTypeChanged()
			}
			return v, nil

		case "right":
			if v.focusIndex == 7 {
				v.dbTypeIndex++
				if v.dbTypeIndex >= len(v.dbTypes) {
					v.dbTypeIndex = 0
				}
				v.onDbTypeChanged()
			}
			return v, nil

		case "enter":
			if v.focusIndex == 8 {
				// If password field is visible and empty, prompt securely before connecting
				if v.isFieldVisible(4) && v.inputs[4].Value() == "" {
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

	// Render chrome first, measure heights, give remainder to list
	title := styles.RenderTitle("Welcome to WhoDB!")
	subtitle := styles.MutedStyle.Render("Select an existing connection below, or create a new one with [n]")
	helpText := styles.RenderHelp(
		"↑/k/shift+tab", "up",
		"↓/j/tab", "down",
		"enter", "connect",
		"[n]", "new",
		"[d]", "delete",
		"esc", "quit",
		"ctrl+c", "force quit",
	)

	// Measure chrome: title + subtitle + help + padding(2) + view indicator(2) + separators(2)
	chromeHeight := lipgloss.Height(title) + lipgloss.Height(subtitle) + lipgloss.Height(helpText) + 6
	listHeight := v.parent.height - chromeHeight
	if listHeight < 3 {
		listHeight = 3
	}
	v.list.SetSize(v.parent.width-4, listHeight)

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(subtitle)
	b.WriteString("\n\n")
	b.WriteString(v.list.View())
	b.WriteString("\n\n")
	b.WriteString(helpText)

	content := lipgloss.NewStyle().Padding(1, 2).Render(b.String())

	if v.escPressed {
		confirmMsg := fmt.Sprintf("Press ESC again to quit (%ds)", v.escTimeoutSecs)
		confirmBox := styles.RenderErrorBox(confirmMsg)
		return content + "\n" + confirmBox
	}

	return content
}

func (v *ConnectionView) renderForm() string {
	// If awaiting password, render overlay prompt (outside viewport)
	if v.awaitingPassword {
		var b strings.Builder
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

	// Set responsive input widths before rendering
	iw := v.inputWidth()
	for i := range v.inputs {
		v.inputs[i].Width = iw
	}
	v.passwordPrompt.Width = iw

	// Build form body for the viewport
	var body strings.Builder

	if v.connError != nil {
		body.WriteString(styles.RenderErrorBoxWidth(v.connError.Error(), v.width))
		body.WriteString("\n")
	}

	// Database Type (index 7)
	dbTypeLabel := "Database Type:"
	if v.focusIndex == 7 {
		dbTypeLabel = styles.KeyStyle.Render("▶ " + dbTypeLabel)
	} else {
		dbTypeLabel = "  " + dbTypeLabel
	}
	body.WriteString(dbTypeLabel)
	body.WriteString("\n  ")
	for i, dbType := range v.dbTypes {
		if i == v.dbTypeIndex {
			if v.focusIndex == 7 {
				body.WriteString(styles.ActiveListItemStyle.Render(" " + dbType + " "))
			} else {
				body.WriteString(styles.KeyStyle.Render("[" + dbType + "]"))
			}
		} else {
			body.WriteString(styles.MutedStyle.Render(" " + dbType + " "))
		}
		body.WriteString(" ")
	}
	body.WriteString("\n\n")

	fieldLabels := []string{"Connection Name:", "Host:", "Port:", "Username:", "Password:", "Database:", "Schema:"}
	for i, fieldLabel := range fieldLabels {
		if !v.isFieldVisible(i) {
			continue
		}
		label := fieldLabel
		if v.focusIndex == i {
			label = styles.KeyStyle.Render("▶ " + label)
		} else {
			label = "  " + label
		}
		body.WriteString(label)
		body.WriteString("\n  ")
		body.WriteString(v.inputs[i].View())
		body.WriteString("\n\n")
	}

	// Connect button (index 8)
	connectBtn := "[Connect]"
	if v.focusIndex == 8 {
		connectBtn = styles.ActiveListItemStyle.Render(" Connect ")
	} else {
		connectBtn = styles.KeyStyle.Render(connectBtn)
	}
	body.WriteString("  " + connectBtn)

	// Render title and help first, measure them, give remaining height to viewport
	title := styles.RenderTitle("New Database Connection")
	helpText := ""
	if len(v.parent.config.Connections) > 0 {
		helpText = styles.RenderHelpWidth(v.width,
			"↑/↓/tab", "navigate",
			"←/→", "change type",
			"enter", "connect",
			"esc", "back",
			"ctrl+c", "quit",
		)
	} else {
		helpText = styles.RenderHelpWidth(v.width,
			"↑/↓/tab", "navigate",
			"←/→", "change type",
			"enter", "connect",
			"ctrl+c", "quit",
		)
	}

	// Measure chrome height: title + help + padding(2) + view indicator(2) + separators(1)
	chromeHeight := lipgloss.Height(title) + lipgloss.Height(helpText) + 5

	// Size viewport to fill remaining space
	if v.formReady {
		vpHeight := v.height - chromeHeight
		if vpHeight < 3 {
			vpHeight = 3
		}
		v.formViewport.Height = vpHeight
		v.formViewport.Width = v.width - 4
		v.formViewport.SetContent(body.String())
	}

	var b strings.Builder
	b.WriteString(title)
	if v.formReady {
		b.WriteString(v.formViewport.View())
	} else {
		b.WriteString(body.String())
	}
	b.WriteString("\n")
	b.WriteString(helpText)

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (v *ConnectionView) refreshList() {
	var items []list.Item
	for _, info := range v.parent.dbManager.ListConnectionsWithSource() {
		items = append(items, connectionItem{conn: info.Connection, source: info.Source})
	}
	v.list.SetItems(items)
}

// getFocusOrder returns the ordered list of focusable indices: db type first, then visible fields, then connect.
func (v *ConnectionView) getFocusOrder() []int {
	order := []int{7} // db type selector first
	order = append(order, v.visibleFields...)
	order = append(order, 8) // connect button last
	return order
}

func (v *ConnectionView) nextInput() {
	if v.focusIndex < len(v.inputs) {
		v.inputs[v.focusIndex].Blur()
	}
	order := v.getFocusOrder()
	currentPos := -1
	for i, idx := range order {
		if idx == v.focusIndex {
			currentPos = i
			break
		}
	}
	nextPos := (currentPos + 1) % len(order)
	v.focusIndex = order[nextPos]
	if v.focusIndex < len(v.inputs) {
		v.inputs[v.focusIndex].Focus()
	}
	v.scrollToFocused()
}

func (v *ConnectionView) prevInput() {
	if v.focusIndex < len(v.inputs) {
		v.inputs[v.focusIndex].Blur()
	}
	order := v.getFocusOrder()
	currentPos := -1
	for i, idx := range order {
		if idx == v.focusIndex {
			currentPos = i
			break
		}
	}
	prevPos := (currentPos - 1 + len(order)) % len(order)
	v.focusIndex = order[prevPos]
	if v.focusIndex < len(v.inputs) {
		v.inputs[v.focusIndex].Focus()
	}
	v.scrollToFocused()
}

// scrollToFocused adjusts the viewport offset to keep the focused field visible.
func (v *ConnectionView) scrollToFocused() {
	if !v.formReady {
		return
	}

	// Estimate the line position of the focused field in the form body.
	// Each section: db type ~3 lines, each field ~3 lines (label + input + blank).
	line := 0
	if v.connError != nil {
		line += 5
	}

	// DB type selector
	if v.focusIndex == 7 {
		v.formViewport.GotoTop()
		return
	}
	line += 3 // db type label + options + blank

	// Visible fields before the focused one
	for _, idx := range v.visibleFields {
		if idx == v.focusIndex {
			break
		}
		line += 3
	}

	// Connect button — scroll to bottom
	if v.focusIndex == 8 {
		v.formViewport.GotoBottom()
		return
	}

	vpHeight := v.formViewport.Height
	offset := v.formViewport.YOffset

	if line < offset+1 {
		newOffset := line - 1
		if newOffset < 0 {
			newOffset = 0
		}
		v.formViewport.SetYOffset(newOffset)
	} else if line+2 >= offset+vpHeight {
		newOffset := line - vpHeight + 3
		if newOffset < 0 {
			newOffset = 0
		}
		v.formViewport.SetYOffset(newOffset)
	}
}

func (v *ConnectionView) resetForm() {
	for i := range v.inputs {
		v.inputs[i].SetValue("")
		v.inputs[i].Blur()
	}
	v.focusIndex = 7 // Start on db type selector
	v.dbTypeIndex = 0
	v.connError = nil
	v.onDbTypeChanged()
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

// Field indices: 0=name, 1=host, 2=port, 3=username, 4=password, 5=database, 6=schema
func getVisibleFields(dbType string) []int {
	switch dbType {
	case "SQLite":
		return []int{0, 5} // name, database
	case "MongoDB":
		return []int{0, 1, 2, 3, 4, 5} // all except schema
	case "Redis":
		return []int{0, 1, 2, 4, 5} // all except username, schema
	case "ElasticSearch":
		return []int{0, 1, 2, 3, 4} // all except database, schema
	default:
		// Postgres, MySQL, MariaDB, ClickHouse
		return []int{0, 1, 2, 3, 4, 5, 6}
	}
}

func (v *ConnectionView) isFieldVisible(index int) bool {
	for _, vi := range v.visibleFields {
		if vi == index {
			return true
		}
	}
	return false
}

func (v *ConnectionView) onDbTypeChanged() {
	v.updatePortPlaceholder()
	v.visibleFields = getVisibleFields(v.dbTypes[v.dbTypeIndex])

	// Update database placeholder for SQLite
	if v.dbTypes[v.dbTypeIndex] == "SQLite" {
		v.inputs[5].Placeholder = "/path/to/database.db"
	} else {
		v.inputs[5].Placeholder = "mydb"
	}

	// If current focus is on a hidden field, move to next visible
	if v.focusIndex < len(v.inputs) && !v.isFieldVisible(v.focusIndex) {
		v.nextInput()
	}
}

func (v *ConnectionView) connect() tea.Cmd {
	return func() tea.Msg {
		name := v.inputs[0].Value()
		dbType := v.dbTypes[v.dbTypeIndex]

		host := ""
		if v.isFieldVisible(1) {
			host = v.inputs[1].Value()
		}
		if host == "" {
			host = "localhost"
		}

		var port int
		if v.isFieldVisible(2) {
			portStr := v.inputs[2].Value()
			if portStr == "" {
				port = v.getDefaultPort(dbType)
			} else {
				portNum, err := strconv.Atoi(portStr)
				if err != nil || portNum < 1024 || portNum > 65535 {
					return connectionResultMsg{err: fmt.Errorf("invalid port number: must be between 1024 and 65535 (ports below 1024 are system reserved)")}
				}
				port = portNum
			}
		} else {
			port = v.getDefaultPort(dbType)
		}

		username := ""
		if v.isFieldVisible(3) {
			username = v.inputs[3].Value()
		}
		password := ""
		if v.isFieldVisible(4) {
			password = v.inputs[4].Value()
		}
		database := ""
		if v.isFieldVisible(5) {
			database = v.inputs[5].Value()
		}
		schema := ""
		if v.isFieldVisible(6) {
			schema = v.inputs[6].Value()
		}

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
