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
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/pkg/styles"
)

// ProfilesView displays saved connection profiles and allows applying,
// saving, and deleting them.
type ProfilesView struct {
	parent    *MainModel
	width     int
	height    int
	cursor    int
	naming    bool // true when prompting for a profile name
	nameInput textinput.Model
}

// NewProfilesView creates a new ProfilesView.
func NewProfilesView(parent *MainModel) *ProfilesView {
	ni := textinput.New()
	ni.Placeholder = "profile name"
	ni.CharLimit = 50
	ni.SetWidth(30)
	niStyles := ni.Styles()
	niStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.Primary)
	niStyles.Focused.Text = lipgloss.NewStyle().Foreground(styles.Foreground)
	niStyles.Cursor.Color = styles.Primary
	ni.SetStyles(niStyles)

	return &ProfilesView{
		parent:    parent,
		cursor:    0,
		nameInput: ni,
	}
}

// Update handles input for the profiles view.
func (v *ProfilesView) Update(msg tea.Msg) (*ProfilesView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		return v, nil

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			if v.cursor > 0 {
				v.cursor--
			}
			return v, nil
		case tea.MouseWheelDown:
			profiles := v.parent.config.GetProfiles()
			if v.cursor < len(profiles)-1 {
				v.cursor++
			}
			return v, nil
		}

	case tea.KeyPressMsg:
		// Handle name input mode
		if v.naming {
			switch msg.String() {
			case "enter":
				name := strings.TrimSpace(v.nameInput.Value())
				if name == "" {
					name = v.nextProfileName()
				}
				saveCmd := v.saveCurrentAsProfile(name)
				v.naming = false
				v.nameInput.Blur()
				v.nameInput.SetValue("")
				return v, tea.Batch(saveCmd, v.parent.SetStatus("Profile saved: "+name))
			case "esc":
				v.naming = false
				v.nameInput.Blur()
				v.nameInput.SetValue("")
				return v, nil
			default:
				v.nameInput, _ = v.nameInput.Update(msg)
				return v, nil
			}
		}

		switch {
		case key.Matches(msg, Keys.Global.Back):
			if !v.parent.PopView() {
				v.parent.mode = ViewBrowser
			}
			return v, nil

		case key.Matches(msg, Keys.Profiles.Up):
			if v.cursor > 0 {
				v.cursor--
			}
			return v, nil

		case key.Matches(msg, Keys.Profiles.Down):
			profiles := v.parent.config.GetProfiles()
			if v.cursor < len(profiles)-1 {
				v.cursor++
			}
			return v, nil

		case key.Matches(msg, Keys.Profiles.Apply):
			return v.applySelected()

		case key.Matches(msg, Keys.Profiles.Save):
			v.naming = true
			v.nameInput.SetValue("")
			v.nameInput.Focus()
			return v, nil

		case key.Matches(msg, Keys.Profiles.Delete):
			profiles := v.parent.config.GetProfiles()
			if len(profiles) == 0 {
				return v, nil
			}
			if v.cursor >= 0 && v.cursor < len(profiles) {
				name := profiles[v.cursor].Name
				v.parent.config.DeleteProfile(name)
				if v.cursor >= len(v.parent.config.GetProfiles()) && v.cursor > 0 {
					v.cursor--
				}
				return v, tea.Batch(v.parent.requestConfigSave(), v.parent.SetStatus("Deleted: "+name))
			}
			return v, nil
		}
	}

	return v, nil
}

// applySelected applies the profile at the current cursor position.
func (v *ProfilesView) applySelected() (*ProfilesView, tea.Cmd) {
	profiles := v.parent.config.GetProfiles()
	if len(profiles) == 0 || v.cursor < 0 || v.cursor >= len(profiles) {
		return v, nil
	}

	profile := profiles[v.cursor]

	// Look up the connection
	conn, err := v.parent.config.GetConnection(profile.Connection)
	if err != nil {
		v.parent.err = fmt.Errorf("profile %q: connection %q not found", profile.Name, profile.Connection)
		return v, nil
	}

	// Disconnect current database
	_ = v.parent.dbManager.Disconnect()
	v.parent.activeLayout = ""
	v.parent.layoutRoot = nil
	v.parent.viewHistory = nil

	// Apply theme
	if profile.Theme != "" {
		if t := styles.GetThemeByName(profile.Theme); t != nil {
			styles.SetTheme(t)
			v.parent.config.SetThemeName(profile.Theme)
		}
	}

	// Apply page size
	if profile.PageSize > 0 {
		v.parent.config.SetPageSize(profile.PageSize)
	}

	// Apply timeout
	if profile.TimeoutSeconds > 0 {
		v.parent.config.Query.TimeoutSeconds = profile.TimeoutSeconds
	}

	saveCmd := v.parent.requestConfigSave()

	// Connect to the profile's connection
	if err := v.parent.dbManager.Connect(conn); err != nil {
		v.parent.err = fmt.Errorf("profile %q: %w", profile.Name, err)
		v.parent.mode = ViewConnection
		return v, saveCmd
	}

	v.parent.currentProfileName = profile.Name
	v.parent.mode = ViewBrowser
	v.parent.initLayout()
	return v, tea.Batch(
		saveCmd,
		v.parent.browserView.loadTables(),
		v.parent.SetStatus("Profile applied: "+profile.Name),
	)
}

// saveCurrentAsProfile saves the current connection and settings as a profile.
func (v *ProfilesView) saveCurrentAsProfile(name string) tea.Cmd {
	conn := v.parent.dbManager.GetCurrentConnection()
	if conn == nil {
		return nil
	}

	// Use connection name if available; otherwise save the connection
	// to config so the profile can reference it.
	connName := conn.Name
	if connName == "" {
		// Generate a name from the connection details and save it
		connName = fmt.Sprintf("%s-%s-%s", conn.Type, conn.Host, conn.Database)
		conn.Name = connName
		v.parent.config.AddConnection(*conn)
	}

	profile := config.Profile{
		Name:           name,
		Connection:     connName,
		Theme:          v.parent.config.GetThemeName(),
		PageSize:       v.parent.config.GetPageSize(),
		TimeoutSeconds: v.parent.config.Query.TimeoutSeconds,
	}

	v.parent.config.AddProfile(profile)
	return v.parent.requestConfigSave()
}

// nextProfileName generates an auto-incremented profile name.
func (v *ProfilesView) nextProfileName() string {
	existing := v.parent.config.GetProfiles()
	max := 0
	for _, p := range existing {
		var n int
		if _, err := fmt.Sscanf(p.Name, "Profile %d", &n); err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("Profile %d", max+1)
}

// View renders the profiles list.
func (v *ProfilesView) View() string {
	var b strings.Builder

	b.WriteString(styles.RenderTitle("Profiles"))
	b.WriteString("\n\n")

	// Name input prompt
	if v.naming {
		conn := v.parent.dbManager.GetCurrentConnection()
		if conn == nil {
			b.WriteString(styles.RenderMuted("  No active connection to save as profile"))
			b.WriteString("\n\n")
			b.WriteString(styles.RenderMuted("  Press Esc to cancel"))
			return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
		}
		b.WriteString("  Profile name:\n")
		b.WriteString("  " + v.nameInput.View())
		b.WriteString("\n\n")
		b.WriteString(styles.RenderMuted(fmt.Sprintf("  Connection: %s | Theme: %s | Page size: %d | Timeout: %ds",
			conn.Name, v.parent.config.GetThemeName(), v.parent.config.GetPageSize(), v.parent.config.Query.TimeoutSeconds)))
		b.WriteString("\n\n")
		b.WriteString(styles.RenderMuted("  Press Enter to save, Esc to cancel"))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	profiles := v.parent.config.GetProfiles()

	if len(profiles) == 0 {
		b.WriteString(styles.RenderMuted("  No saved profiles"))
		b.WriteString("\n\n")
		b.WriteString(styles.RenderMuted("  Press [s] to save current settings as a profile"))
	} else {
		for i, p := range profiles {
			prefix := "  "
			if v.cursor == i {
				prefix = styles.RenderKey("> ")
			}

			nameStr := p.Name
			details := fmt.Sprintf("conn:%s", p.Connection)
			if p.Theme != "" {
				details += fmt.Sprintf(" theme:%s", p.Theme)
			}
			if p.PageSize > 0 {
				details += fmt.Sprintf(" page:%d", p.PageSize)
			}
			if p.TimeoutSeconds > 0 {
				details += fmt.Sprintf(" timeout:%ds", p.TimeoutSeconds)
			}

			if v.cursor == i {
				b.WriteString(prefix + styles.ActiveListItemStyle.Render(nameStr) + "  " + styles.MutedStyle.Render(details))
			} else {
				b.WriteString(prefix + nameStr + "  " + styles.MutedStyle.Render(details))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n\n")

	bindings := []key.Binding{
		Keys.Profiles.Up,
		Keys.Profiles.Down,
		Keys.Profiles.Apply,
		Keys.Profiles.Save,
		Keys.Profiles.Delete,
		Keys.Global.Back,
		Keys.Global.Quit,
	}
	b.WriteString(RenderBindingHelpWidth(v.width, bindings...))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}
