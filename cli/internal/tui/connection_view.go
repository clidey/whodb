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

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/internal/config"
	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/internal/docker"
	"github.com/clidey/whodb/cli/pkg/styles"
)

type connectionItem struct {
	conn   config.Connection
	source string
}

func (i connectionItem) Title() string { return i.conn.Name }

// ConnectionSourceDocker identifies a connection detected from a running Docker container.
const ConnectionSourceDocker = "docker"

func (i connectionItem) Description() string {
	desc := fmt.Sprintf("%s@%s", i.conn.Type, i.conn.Host)
	if i.source == dbmgr.ConnectionSourceEnv {
		desc += " (env)"
	} else if i.source == ConnectionSourceDocker {
		desc += " (docker)"
	}
	return desc
}
func (i connectionItem) FilterValue() string { return i.conn.Name }

// connectionPingResult tracks the reachability status of a saved connection.
type connectionPingResult struct {
	checked bool
	online  bool
}

// connectionPingMsg is sent when a background ping completes.
type connectionPingMsg struct {
	name   string
	online bool
}

type connectionDelegate struct {
	pingResults map[string]connectionPingResult
}

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

	desc := styles.RenderMuted(i.Description())
	if result, ok := d.pingResults[i.conn.Name]; ok && result.checked {
		if result.online {
			desc += " " + styles.SuccessStyle.Render("●")
		} else {
			desc += " " + styles.ErrorStyle.Render("●")
		}
	} else if i.conn.SSHHost != "" {
		desc += " " + styles.MutedStyle.Render("○")
	}
	str += "\n  " + desc
	fmt.Fprint(w, str)
}

// ConnectionView provides the TUI for managing database connections.
// It supports both a list view (selecting from saved connections) and
// a form view (creating new connections).
// Form field indices for text inputs.
const (
	fieldName        = 0
	fieldHost        = 1
	fieldPort        = 2
	fieldUsername    = 3
	fieldPassword    = 4
	fieldDatabase    = 5
	fieldSchema      = 6
	fieldSSHHost     = 7
	fieldSSHUser     = 8
	fieldSSHKeyFile  = 9
	fieldSSHPassword = 10
)

// Virtual focus indices (not backed by text inputs).
const (
	focusDBType    = 11
	focusSSHToggle = 12
	focusConnect   = 13
)

type ConnectionView struct {
	parent        *MainModel
	list          list.Model
	mode          string // "list" or "form"
	inputs        []textinput.Model
	focusIndex    int
	dbTypes       []string
	dbTypeIndex   int
	visibleFields []int // indices of visible input fields for current db type
	sshEnabled    bool  // whether the SSH tunnel section is expanded
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
	// Background ping status for each connection
	pingResults map[string]connectionPingResult
}

// NewConnectionView creates a connection view initialized with saved connections
// from the parent's config. If no connections exist, it starts in form mode.
func NewConnectionView(parent *MainModel) *ConnectionView {
	var items []list.Item
	for _, info := range parent.dbManager.ListConnectionsWithSource() {
		items = append(items, connectionItem{conn: info.Connection, source: info.Source})
	}

	// Append running Docker database containers as connection options
	for _, c := range docker.DetectContainers() {
		items = append(items, connectionItem{
			conn: config.Connection{
				Name: c.Name,
				Type: c.Type,
				Host: "localhost",
				Port: c.Port,
			},
			source: ConnectionSourceDocker,
		})
	}

	pingResults := make(map[string]connectionPingResult)
	l := list.New(items, connectionDelegate{pingResults: pingResults}, 0, 0)
	l.Title = ""
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.SetStatusBarItemName("saved connection", "saved connections")

	// Initialize form inputs
	newInput := func(placeholder string, charLimit int) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = charLimit
		ti.Width = 40
		ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
		ti.TextStyle = lipgloss.NewStyle().Foreground(styles.Foreground)
		ti.Cursor.Style = lipgloss.NewStyle().Foreground(styles.Primary)
		return ti
	}

	inputs := make([]textinput.Model, 11)
	inputs[fieldName] = newInput("My Connection", 50)
	inputs[fieldHost] = newInput("localhost", 100)
	inputs[fieldPort] = newInput("5432", 5)
	inputs[fieldUsername] = newInput("postgres", 50)

	inputs[fieldPassword] = newInput("password", 100)
	inputs[fieldPassword].EchoMode = textinput.EchoPassword
	inputs[fieldPassword].EchoCharacter = '•'

	inputs[fieldDatabase] = newInput("mydb", 50)
	inputs[fieldSchema] = newInput("Schema name (optional)", 50)

	// SSH tunnel fields
	inputs[fieldSSHHost] = newInput("ssh.example.com", 100)
	inputs[fieldSSHUser] = newInput("ssh-user", 50)
	inputs[fieldSSHKeyFile] = newInput("~/.ssh/id_rsa", 200)

	inputs[fieldSSHPassword] = newInput("SSH password (optional)", 100)
	inputs[fieldSSHPassword].EchoMode = textinput.EchoPassword
	inputs[fieldSSHPassword].EchoCharacter = '•'

	mode := "list"
	fi := focusDBType // Start on db type selector
	if len(items) == 0 {
		mode = "form"
	}

	// Password prompt (shown after pressing Connect if password is empty)
	prompt := newInput("enter password", 100)
	prompt.EchoMode = textinput.EchoPassword
	prompt.EchoCharacter = '•'

	dbTypes := []string{"Postgres", "MySQL", "Sqlite3", "DuckDB", "MongoDB", "Redis", "MariaDB", "ClickHouse", "ElasticSearch", "TiDB"}

	return &ConnectionView{
		parent:           parent,
		list:             l,
		mode:             mode,
		inputs:           inputs,
		focusIndex:       fi,
		dbTypes:          dbTypes,
		dbTypeIndex:      0,
		visibleFields:    getVisibleFields(dbTypes[0]),
		connecting:       false,
		awaitingPassword: false,
		passwordPrompt:   prompt,
		pingResults:      pingResults,
	}
}

func (v *ConnectionView) Update(msg tea.Msg) (*ConnectionView, tea.Cmd) {
	if v.mode == "form" {
		return v.updateForm(msg)
	}
	return v.updateList(msg)
}

// Init returns a command to start background ping checks for all connections.
func (v *ConnectionView) Init() tea.Cmd {
	if v.mode != "list" || len(v.list.Items()) == 0 {
		return nil
	}
	return v.pingAllConnections()
}

// pingAllConnections fires background pings for every connection in the list.
func (v *ConnectionView) pingAllConnections() tea.Cmd {
	var cmds []tea.Cmd
	for _, item := range v.list.Items() {
		ci, ok := item.(connectionItem)
		if !ok {
			continue
		}
		if ci.conn.SSHHost != "" {
			continue
		}
		conn := ci.conn
		mgr := v.parent.dbManager
		cmds = append(cmds, func() tea.Msg {
			online := mgr.Ping(&conn)
			return connectionPingMsg{name: conn.Name, online: online}
		})
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (v *ConnectionView) updateList(msg tea.Msg) (*ConnectionView, tea.Cmd) {
	switch msg := msg.(type) {
	case connectionPingMsg:
		v.pingResults[msg.name] = connectionPingResult{checked: true, online: msg.online}
		return v, nil

	case connectionResultMsg:
		v.connecting = false
		if msg.err != nil {
			v.parent.err = msg.err
			return v, nil
		}
		v.parent.mode = ViewBrowser
		v.parent.initLayout()
		conn := v.parent.dbManager.GetCurrentConnection()
		connDesc := ""
		if conn != nil {
			connDesc = fmt.Sprintf("Connected to %s@%s", conn.Type, conn.Host)
		}
		return v, tea.Batch(v.parent.browserView.Init(), v.parent.SetStatus(connDesc))

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
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
		switch {
		case key.Matches(msg, Keys.ConnectionList.Down):
			v.list.CursorDown()
			return v, nil

		case key.Matches(msg, Keys.ConnectionList.Up):
			v.list.CursorUp()
			return v, nil

		case key.Matches(msg, Keys.ConnectionList.Connect):
			if item, ok := v.list.SelectedItem().(connectionItem); ok {
				// Docker containers: open form pre-filled so user can add credentials
				if item.source == ConnectionSourceDocker {
					v.mode = "form"
					v.resetForm()
					v.prefillFromConnection(item.conn)
					return v, nil
				}
				v.connecting = true
				v.connError = nil
				conn := item.conn
				return v, func() tea.Msg {
					if err := v.parent.dbManager.Connect(&conn); err != nil {
						return connectionResultMsg{err: err}
					}
					return connectionResultMsg{err: nil}
				}
			}

		case key.Matches(msg, Keys.ConnectionList.New):
			v.mode = "form"
			v.resetForm()
			v.inputs[0].Focus()
			return v, nil

		case key.Matches(msg, Keys.ConnectionList.DeleteConn):
			if item, ok := v.list.SelectedItem().(connectionItem); ok {
				v.parent.config.RemoveConnection(item.conn.Name)
				v.parent.config.Save()
				v.refreshList()
				return v, v.pingAllConnections()
			}

		case key.Matches(msg, Keys.ConnectionList.QuitEsc):
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

	// Clear ESC confirmation on any non-ESC key press
	if km, ok := msg.(tea.KeyMsg); ok && km.String() != "esc" {
		v.escPressed = false
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

	// Absorb ping results that arrive while in form mode
	if pm, ok := msg.(connectionPingMsg); ok {
		v.pingResults[pm.name] = connectionPingResult{checked: true, online: pm.online}
		return v, nil
	}

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
			v.parent.initLayout()
			conn := v.parent.dbManager.GetCurrentConnection()
			connDesc := ""
			if conn != nil {
				connDesc = fmt.Sprintf("Connected to %s@%s", conn.Type, conn.Host)
			}
			return v, tea.Batch(v.parent.browserView.Init(), v.parent.SetStatus(connDesc))
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
				v.refreshList()
				return v, v.pingAllConnections()
			}
			return v, tea.Quit

		case "tab", "down":
			v.nextInput()
			return v, nil

		case "shift+tab", "up":
			v.prevInput()
			return v, nil

		case "left":
			if v.focusIndex == focusDBType {
				v.dbTypeIndex--
				if v.dbTypeIndex < 0 {
					v.dbTypeIndex = len(v.dbTypes) - 1
				}
				v.onDbTypeChanged()
			}
			return v, nil

		case "right":
			if v.focusIndex == focusDBType {
				v.dbTypeIndex++
				if v.dbTypeIndex >= len(v.dbTypes) {
					v.dbTypeIndex = 0
				}
				v.onDbTypeChanged()
			}
			return v, nil

		case "enter", " ":
			if v.focusIndex == focusSSHToggle {
				v.sshEnabled = !v.sshEnabled
				v.visibleFields = getVisibleFields(v.dbTypes[v.dbTypeIndex])
				return v, nil
			}
			if msg.String() == " " {
				// Space only toggles SSH; don't propagate to other fields
				break
			}
			if v.focusIndex == focusConnect {
				// If password field is visible and empty, prompt securely before connecting
				if v.isFieldVisible(fieldPassword) && v.inputs[fieldPassword].Value() == "" {
					v.awaitingPassword = true
					v.passwordPrompt.SetValue("")
					v.passwordPrompt.Focus()
					return v, nil
				}
				v.connecting = true
				v.connError = nil
				return v, v.connect()
			}
			if msg.String() == "enter" {
				v.nextInput()
			}
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
	subtitle := styles.RenderMuted("Select an existing connection below, or create a new one with [n]")
	helpText := RenderBindingHelpWidth(v.parent.width,
		Keys.ConnectionList.Up,
		Keys.ConnectionList.Down,
		Keys.ConnectionList.Connect,
		Keys.ConnectionList.New,
		Keys.ConnectionList.DeleteConn,
		Keys.Global.CycleTheme,
		Keys.ConnectionList.QuitEsc,
		Keys.Global.Quit,
	)
	sep := styles.MutedStyle.Render("  ")
	legend := styles.SuccessStyle.Render("●") + styles.RenderMuted(" available") + sep +
		styles.ErrorStyle.Render("●") + styles.RenderMuted(" not available") + sep +
		styles.MutedStyle.Render("○") + styles.RenderMuted(" inactive tunnel")

	// Measure chrome within this view: title + subtitle + legend + help + padding(2) + separators(3)
	chromeHeight := lipgloss.Height(title) + lipgloss.Height(subtitle) + lipgloss.Height(helpText) + 1 + 5
	listHeight := v.parent.ContentHeight() - chromeHeight
	if listHeight < 3 {
		listHeight = 3
	}
	v.list.SetSize(v.parent.width-4, listHeight)

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(subtitle)
	b.WriteString("\n\n")
	if v.connecting {
		b.WriteString(v.parent.SpinnerView() + styles.RenderMuted(" Connecting..."))
	} else {
		b.WriteString(v.list.View())
	}
	b.WriteString("\n\n")
	if v.escPressed {
		b.WriteString(styles.RenderErr(fmt.Sprintf("Press ESC again to quit (%ds)", v.escTimeoutSecs)))
	} else {
		b.WriteString(legend)
	}
	b.WriteString("\n")
	b.WriteString(helpText)

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
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

	if v.connecting {
		var cb strings.Builder
		cb.WriteString(styles.RenderTitle("New Database Connection"))
		cb.WriteString(v.parent.SpinnerView() + styles.RenderMuted(" Connecting..."))
		return lipgloss.NewStyle().Padding(1, 2).Render(cb.String())
	}

	// Set responsive input widths before rendering
	iw := v.inputWidth()
	for i := range v.inputs {
		v.inputs[i].Width = iw
	}
	v.passwordPrompt.Width = iw

	// Build form body for the viewport
	var body strings.Builder

	// Database Type — rendered inline like the other fields.
	// Type options wrap to multiple lines to stay within viewport width.
	// The label is written as plain text (not RenderKey) to avoid a viewport
	// bug where ANSI-styled text on the first visible line gets misaligned.
	if v.focusIndex == focusDBType {
		body.WriteString("▶ Database Type:")
	} else {
		body.WriteString("  Database Type:")
	}
	body.WriteString("\n  ")
	for i, dbType := range v.dbTypes {
		if i > 0 {
			body.WriteString("  ")
		}
		if i == v.dbTypeIndex {
			if v.focusIndex == focusDBType {
				body.WriteString(styles.ActiveListItemStyle.Render(dbType))
			} else {
				body.WriteString(styles.RenderKey(dbType))
			}
		} else {
			body.WriteString(styles.RenderMuted(dbType))
		}
	}
	body.WriteString("\n\n")

	fieldLabels := map[int]string{
		fieldName:     "Connection Name:",
		fieldHost:     "Host:",
		fieldPort:     "Port:",
		fieldUsername: "Username:",
		fieldPassword: "Password:",
		fieldDatabase: "Database:",
		fieldSchema:   "Schema:",
	}
	for _, i := range v.visibleFields {
		label := fieldLabels[i]
		if v.focusIndex == i {
			label = styles.RenderKey("▶ " + label)
		} else {
			label = "  " + label
		}
		body.WriteString(label)
		body.WriteString("\n  ")
		body.WriteString(v.inputs[i].View())
		body.WriteString("\n\n")
	}

	// SSH Tunnel toggle (only for network databases)
	if isNetworkDatabase(v.dbTypes[v.dbTypeIndex]) {
		toggleLabel := "SSH Tunnel:"
		toggleValue := "Off"
		if v.sshEnabled {
			toggleValue = "On"
		}
		if v.focusIndex == focusSSHToggle {
			body.WriteString(styles.RenderKey("▶ " + toggleLabel))
			body.WriteString("  ")
			body.WriteString(styles.ActiveListItemStyle.Render(toggleValue))
		} else {
			body.WriteString("  " + toggleLabel)
			body.WriteString("  ")
			if v.sshEnabled {
				body.WriteString(styles.RenderKey(toggleValue))
			} else {
				body.WriteString(styles.RenderMuted(toggleValue))
			}
		}
		body.WriteString("\n\n")

		// SSH fields (shown when toggle is on)
		if v.sshEnabled {
			sshLabels := map[int]string{
				fieldSSHHost:     "SSH Host:",
				fieldSSHUser:     "SSH User:",
				fieldSSHKeyFile:  "SSH Key File:",
				fieldSSHPassword: "SSH Password:",
			}
			for _, i := range []int{fieldSSHHost, fieldSSHUser, fieldSSHKeyFile, fieldSSHPassword} {
				label := sshLabels[i]
				if v.focusIndex == i {
					label = styles.RenderKey("▶ " + label)
				} else {
					label = "  " + label
				}
				body.WriteString(label)
				body.WriteString("\n  ")
				body.WriteString(v.inputs[i].View())
				body.WriteString("\n\n")
			}
		}
	}

	// Connect button
	connectBtn := "[Connect]"
	if v.focusIndex == focusConnect {
		connectBtn = styles.ActiveListItemStyle.Render(" Connect ")
	} else {
		connectBtn = styles.RenderKey(connectBtn)
	}
	body.WriteString("  " + connectBtn)

	// Render title and help first, measure them, give remaining height to viewport
	title := styles.RenderTitle("New Database Connection")
	helpWidth := v.width
	if helpWidth == 0 {
		helpWidth = v.parent.width
	}
	helpText := ""
	if len(v.parent.config.Connections) > 0 {
		helpText = styles.RenderHelpWidth(helpWidth,
			Keys.ConnectionForm.Navigate.Help().Key, Keys.ConnectionForm.Navigate.Help().Desc,
			Keys.ConnectionForm.TypeLeft.Help().Key, Keys.ConnectionForm.TypeLeft.Help().Desc,
			Keys.ConnectionForm.ConnectForm.Help().Key, Keys.ConnectionForm.ConnectForm.Help().Desc,
			Keys.Global.CycleTheme.Help().Key, Keys.Global.CycleTheme.Help().Desc,
			Keys.Global.Back.Help().Key, Keys.Global.Back.Help().Desc,
			Keys.Global.Quit.Help().Key, Keys.Global.Quit.Help().Desc,
		)
	} else {
		helpText = styles.RenderHelpWidth(helpWidth,
			Keys.ConnectionForm.Navigate.Help().Key, Keys.ConnectionForm.Navigate.Help().Desc,
			Keys.ConnectionForm.TypeLeft.Help().Key, Keys.ConnectionForm.TypeLeft.Help().Desc,
			Keys.ConnectionForm.ConnectForm.Help().Key, Keys.ConnectionForm.ConnectForm.Help().Desc,
			Keys.Global.CycleTheme.Help().Key, Keys.Global.CycleTheme.Help().Desc,
			Keys.Global.Quit.Help().Key, Keys.Global.Quit.Help().Desc,
		)
	}

	// Render error outside viewport so it's always fully visible
	errorBlock := ""
	if v.connError != nil {
		errorBlock = styles.RenderErrorBox(v.connError.Error()) + "\n"
	}

	// Measure chrome within this view: title + error + help + padding(2) + separators(1)
	chromeHeight := lipgloss.Height(title) + lipgloss.Height(errorBlock) + lipgloss.Height(helpText) + 3

	// Size viewport to fill remaining space
	if v.formReady {
		vpHeight := v.parent.ContentHeight() - chromeHeight
		if vpHeight < 3 {
			vpHeight = 3
		}
		v.formViewport.Height = vpHeight
		v.formViewport.Width = v.width - 4
		v.formViewport.SetContent(body.String())
	}

	// Build output with manual left-padding instead of lipgloss.Padding,
	// which miscalculates widths when combining styled title with viewport output.
	pad := "  "
	var out strings.Builder
	out.WriteString("\n") // top padding
	for _, line := range strings.Split(title, "\n") {
		out.WriteString(pad)
		out.WriteString(line)
		out.WriteString("\n")
	}
	out.WriteString(errorBlock)
	var content string
	if v.formReady {
		content = v.formViewport.View()
	} else {
		content = body.String()
	}
	for _, line := range strings.Split(content, "\n") {
		out.WriteString(pad)
		out.WriteString(line)
		out.WriteString("\n")
	}
	for _, line := range strings.Split(helpText, "\n") {
		out.WriteString(pad)
		out.WriteString(line)
		out.WriteString("\n")
	}
	return out.String()
}

func (v *ConnectionView) refreshList() {
	var items []list.Item
	for _, info := range v.parent.dbManager.ListConnectionsWithSource() {
		items = append(items, connectionItem{conn: info.Connection, source: info.Source})
	}
	for _, c := range docker.DetectContainers() {
		items = append(items, connectionItem{
			conn: config.Connection{
				Name: c.Name,
				Type: c.Type,
				Host: "localhost",
				Port: c.Port,
			},
			source: ConnectionSourceDocker,
		})
	}
	v.list.SetItems(items)
	// Reset ping results so they get refreshed
	for k := range v.pingResults {
		delete(v.pingResults, k)
	}
}

// getFocusOrder returns the ordered list of focusable indices:
// db type first, then visible fields, then SSH toggle (for network DBs),
// then SSH fields (if toggle is on), then connect.
func (v *ConnectionView) getFocusOrder() []int {
	order := []int{focusDBType}
	order = append(order, v.visibleFields...)

	if isNetworkDatabase(v.dbTypes[v.dbTypeIndex]) {
		order = append(order, focusSSHToggle)
		if v.sshEnabled {
			order = append(order, fieldSSHHost, fieldSSHUser, fieldSSHKeyFile, fieldSSHPassword)
		}
	}

	order = append(order, focusConnect)
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
	if v.focusIndex == focusDBType {
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

	// SSH toggle and fields
	if v.focusIndex == focusSSHToggle || v.focusIndex >= fieldSSHHost {
		// SSH section comes after visible fields
		for range v.visibleFields {
			// already counted above unless we hit the focused field
		}
	}

	// Connect button — scroll to bottom
	if v.focusIndex == focusConnect {
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
	v.focusIndex = focusDBType
	v.dbTypeIndex = 0
	v.sshEnabled = false
	v.connError = nil
	v.onDbTypeChanged()
}

// prefillFromConnection populates the form fields from a Connection (e.g. Docker-detected).
func (v *ConnectionView) prefillFromConnection(conn config.Connection) {
	// Set database type
	for i, t := range v.dbTypes {
		if strings.EqualFold(t, conn.Type) {
			v.dbTypeIndex = i
			break
		}
	}
	v.onDbTypeChanged()

	if conn.Name != "" {
		v.inputs[fieldName].SetValue(conn.Name)
	}
	if conn.Host != "" {
		v.inputs[fieldHost].SetValue(conn.Host)
	}
	if conn.Port > 0 {
		v.inputs[fieldPort].SetValue(strconv.Itoa(conn.Port))
	}
	if conn.Username != "" {
		v.inputs[fieldUsername].SetValue(conn.Username)
	}
	if conn.Database != "" {
		v.inputs[fieldDatabase].SetValue(conn.Database)
	}

	// Prefill SSH fields if present
	if conn.SSHHost != "" {
		v.sshEnabled = true
		v.inputs[fieldSSHHost].SetValue(conn.SSHHost)
		if conn.SSHUser != "" {
			v.inputs[fieldSSHUser].SetValue(conn.SSHUser)
		}
		if conn.SSHKeyFile != "" {
			v.inputs[fieldSSHKeyFile].SetValue(conn.SSHKeyFile)
		}
	}

	// Focus on the first empty required field (usually username or database)
	v.focusIndex = fieldUsername
	v.inputs[fieldUsername].Focus()
}

func (v *ConnectionView) updatePortPlaceholder() {
	defaultPort := v.getDefaultPort(v.dbTypes[v.dbTypeIndex])
	v.inputs[fieldPort].Placeholder = strconv.Itoa(defaultPort)
}

func (v *ConnectionView) getDefaultPort(dbType string) int {
	switch dbType {
	case "Postgres":
		return 5432
	case "MySQL", "MariaDB":
		return 3306
	case "TiDB":
		return 4000
	case "MongoDB":
		return 27017
	case "Redis":
		return 6379
	case "ClickHouse":
		return 9000
	case "ElasticSearch":
		return 9200
	case "Sqlite3", "DuckDB":
		return 0
	default:
		return 5432
	}
}

// isNetworkDatabase returns true for database types that connect over a network,
// i.e. those where SSH tunneling is applicable.
func isNetworkDatabase(dbType string) bool {
	switch dbType {
	case "Sqlite3", "DuckDB":
		return false
	default:
		return true
	}
}

// getVisibleFields returns the input field indices visible for the given database type.
// SSH fields are not included here; they are managed separately via the SSH toggle.
func getVisibleFields(dbType string) []int {
	switch dbType {
	case "Sqlite3", "DuckDB":
		return []int{fieldName, fieldDatabase}
	case "MongoDB":
		return []int{fieldName, fieldHost, fieldPort, fieldUsername, fieldPassword, fieldDatabase}
	case "Redis":
		return []int{fieldName, fieldHost, fieldPort, fieldPassword, fieldDatabase}
	case "ElasticSearch":
		return []int{fieldName, fieldHost, fieldPort, fieldUsername, fieldPassword}
	case "Postgres":
		return []int{fieldName, fieldHost, fieldPort, fieldUsername, fieldPassword, fieldDatabase, fieldSchema}
	default:
		// MySQL, MariaDB, TiDB, ClickHouse
		return []int{fieldName, fieldHost, fieldPort, fieldUsername, fieldPassword, fieldDatabase}
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

	// Update database placeholder for file-based databases
	if v.dbTypes[v.dbTypeIndex] == "Sqlite3" || v.dbTypes[v.dbTypeIndex] == "DuckDB" {
		v.inputs[fieldDatabase].Placeholder = "/path/to/database.db"
	} else {
		v.inputs[fieldDatabase].Placeholder = "mydb"
	}

	// Disable SSH toggle for non-network databases
	if !isNetworkDatabase(v.dbTypes[v.dbTypeIndex]) {
		v.sshEnabled = false
	}

	// If current focus is on a hidden field, move to next visible
	if v.focusIndex < len(v.inputs) && !v.isFieldVisible(v.focusIndex) {
		v.nextInput()
	}
}

func (v *ConnectionView) connect() tea.Cmd {
	// Capture all form values before the closure to avoid data races
	name := v.inputs[fieldName].Value()
	dbType := v.dbTypes[v.dbTypeIndex]

	host := ""
	if v.isFieldVisible(fieldHost) {
		host = v.inputs[fieldHost].Value()
	}
	if host == "" {
		host = "localhost"
	}

	var port int
	if v.isFieldVisible(fieldPort) {
		portStr := v.inputs[fieldPort].Value()
		if portStr == "" {
			port = v.getDefaultPort(dbType)
		} else {
			portNum, err := strconv.Atoi(portStr)
			if err != nil || portNum < 1 || portNum > 65535 {
				return func() tea.Msg {
					return connectionResultMsg{err: fmt.Errorf("invalid port number: must be between 1 and 65535")}
				}
			}
			port = portNum
		}
	} else {
		port = v.getDefaultPort(dbType)
	}

	username := ""
	if v.isFieldVisible(fieldUsername) {
		username = v.inputs[fieldUsername].Value()
	}
	password := ""
	if v.isFieldVisible(fieldPassword) {
		password = v.inputs[fieldPassword].Value()
	}
	database := ""
	if v.isFieldVisible(fieldDatabase) {
		database = v.inputs[fieldDatabase].Value()
	}
	schema := ""
	if v.isFieldVisible(fieldSchema) {
		schema = v.inputs[fieldSchema].Value()
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

	// Capture SSH tunnel fields if enabled
	if v.sshEnabled && isNetworkDatabase(dbType) {
		conn.SSHHost = v.inputs[fieldSSHHost].Value()
		conn.SSHUser = v.inputs[fieldSSHUser].Value()
		conn.SSHKeyFile = v.inputs[fieldSSHKeyFile].Value()
		conn.SSHPassword = v.inputs[fieldSSHPassword].Value()
		if portStr := v.inputs[fieldPort].Value(); portStr != "" {
			// SSHPort defaults to 22 in the tunnel; leave 0 here to use that default
		}
	}

	dbManager := v.parent.dbManager
	cfg := v.parent.config

	return func() tea.Msg {
		if err := dbManager.Connect(&conn); err != nil {
			return connectionResultMsg{err: err}
		}

		// Save connection if name is provided
		if name != "" {
			cfg.AddConnection(conn)
			cfg.Save()
		}

		return connectionResultMsg{err: nil}
	}
}
