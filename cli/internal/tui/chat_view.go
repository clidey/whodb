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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/cli/pkg/styles"
	"github.com/clidey/whodb/core/src/engine"
)

type chatMessage struct {
	Role    string
	Content string
	Type    string
	Result  *engine.GetRowsResult
}

type chatResponseMsg struct {
	messages []*database.ChatMessage
	err      error
}

type modelsLoadedMsg struct {
	models []string
	err    error
}

type ChatView struct {
	parent           *MainModel
	providers        []database.AIProvider
	selectedProvider int
	models           []string
	selectedModel    int
	loadingModels    bool
	messages         []chatMessage
	input            textarea.Model
	sending          bool
	err              error
	width            int
	height           int
	scrollOffset     int
	selectedMessage  int
	viewingResult    bool
	focusField       int
	// Consent gate for data governance
	consented bool
}

const (
	focusFieldProvider = iota
	focusFieldModel
	focusFieldMessage
)

func NewChatView(parent *MainModel) *ChatView {
	ti := textarea.New()
	ti.Placeholder = "Ask a question about your database..."
	ti.Focus()
	ti.CharLimit = 1000
	ti.ShowLineNumbers = false
	ti.SetHeight(3)
	ti.SetWidth(70)
	ti.Prompt = ""
	ti.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ti.FocusedStyle.Base = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Padding(0, 1)
	ti.BlurredStyle.Base = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Border).
		Padding(0, 1)

	providers := parent.dbManager.GetAIProviders()
	selectedProvider := 0

	for i, p := range providers {
		if p.Type == "Ollama" {
			selectedProvider = i
			break
		}
	}

	// Load consent from config
	consentGiven := parent.config.GetAIConsent()

	return &ChatView{
		parent:           parent,
		providers:        providers,
		selectedProvider: selectedProvider,
		models:           []string{},
		selectedModel:    0,
		messages:         []chatMessage{},
		input:            ti,
		sending:          false,
		loadingModels:    false,
		err:              nil,
		width:            80,
		height:           24,
		scrollOffset:     0,
		selectedMessage:  -1,
		viewingResult:    false,
		focusField:       focusFieldMessage,
		consented:        consentGiven,
	}
}

func (v *ChatView) Update(msg tea.Msg) (*ChatView, tea.Cmd) {
	var cmd tea.Cmd

	// Data governance consent handling
	if !v.consented {
		switch m := msg.(type) {
		case tea.KeyMsg:
			switch m.String() {
			case "a":
				v.consented = true
				// Persist consent to config
				v.parent.config.SetAIConsent(true)
				if err := v.parent.config.Save(); err != nil {
					v.err = fmt.Errorf("failed to save consent: %w", err)
					return v, nil
				}
				if len(v.providers) > 0 {
					return v, v.loadModels()
				}
				return v, nil
			case "esc", "q", "d":
				v.parent.mode = ViewBrowser
				return v, nil
			}
		case tea.WindowSizeMsg:
			v.width = m.Width
			v.height = m.Height
			v.input.SetWidth(m.Width - 12)
			return v, nil
		}
		return v, nil
	}

	switch msg := msg.(type) {
	case chatResponseMsg:
		v.sending = false
		maxVisibleMessages := 6 // Must match View()
		if msg.err != nil {
			v.err = msg.err
			v.messages = append(v.messages, chatMessage{
				Role:    "system",
				Content: fmt.Sprintf("Error: %s", msg.err.Error()),
				Type:    "error",
			})
			if len(v.messages) > maxVisibleMessages {
				v.scrollOffset = len(v.messages) - maxVisibleMessages
			}
			return v, nil
		}

		for _, m := range msg.messages {
			v.messages = append(v.messages, chatMessage{
				Role:    "system",
				Content: m.Text,
				Type:    m.Type,
				Result:  m.Result,
			})
		}
		v.err = nil

		// Auto-scroll to show latest messages
		if len(v.messages) > maxVisibleMessages {
			v.scrollOffset = len(v.messages) - maxVisibleMessages
		}
		return v, nil

	case modelsLoadedMsg:
		v.loadingModels = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.models = msg.models
		if len(v.models) > 0 {
			v.selectedModel = 0
		}
		v.err = nil
		return v, nil

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.input.SetWidth(msg.Width - 12)
		return v, nil

	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseWheelUp:
			if v.focusField == focusFieldMessage {
				if v.scrollOffset > 0 {
					v.scrollOffset--
				}
			}
			return v, nil
		case tea.MouseWheelDown:
			if v.focusField == focusFieldMessage {
				maxMsgHeight := v.height - 18
				maxScroll := len(v.messages) - maxMsgHeight
				if maxScroll < 0 {
					maxScroll = 0
				}
				if v.scrollOffset < maxScroll {
					v.scrollOffset++
				}
			}
			return v, nil
		}
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+r":
			// Revoke consent
			v.consented = false
			v.parent.config.SetAIConsent(false)
			if err := v.parent.config.Save(); err != nil {
				v.err = fmt.Errorf("failed to revoke consent: %w", err)
			}
			return v, nil

		case "esc":
			if v.viewingResult {
				v.viewingResult = false
				return v, nil
			}
			v.parent.mode = ViewBrowser
			return v, nil

		case "ctrl+i", "/":
			// Focus chat input
			v.focusField = focusFieldMessage
			v.input.Focus()
			return v, nil

		case "up":
			// Cycle backward through fields: message -> model -> provider
			if v.focusField > focusFieldProvider {
				v.focusField--
				if v.focusField == focusFieldMessage {
					v.input.Focus()
				} else {
					v.input.Blur()
				}
			}
			return v, nil

		case "down":
			// Cycle forward through fields: provider -> model -> message
			if v.focusField < focusFieldMessage {
				v.focusField++
				if v.focusField == focusFieldMessage {
					v.input.Focus()
				} else {
					v.input.Blur()
				}
			}
			return v, nil

		case "ctrl+p":
			// Select previous message in conversation
			if len(v.messages) > 0 {
				if v.selectedMessage < 0 {
					v.selectedMessage = len(v.messages) - 1
				} else if v.selectedMessage > 0 {
					v.selectedMessage--
				}
			}
			return v, nil

		case "ctrl+n":
			// Select next message in conversation
			if len(v.messages) > 0 {
				if v.selectedMessage < 0 {
					v.selectedMessage = 0
				} else if v.selectedMessage < len(v.messages)-1 {
					v.selectedMessage++
				}
			}
			return v, nil

		case "left":
			if v.focusField == focusFieldProvider {
				if v.selectedProvider > 0 {
					v.selectedProvider--
				} else {
					v.selectedProvider = len(v.providers) - 1
				}
				return v, nil
			} else if v.focusField == focusFieldModel && len(v.models) > 0 {
				if v.selectedModel > 0 {
					v.selectedModel--
				} else {
					v.selectedModel = len(v.models) - 1
				}
				return v, nil
			}

		case "right":
			if v.focusField == focusFieldProvider {
				if v.selectedProvider < len(v.providers)-1 {
					v.selectedProvider++
				} else {
					v.selectedProvider = 0
				}
				return v, nil
			} else if v.focusField == focusFieldModel && len(v.models) > 0 {
				if v.selectedModel < len(v.models)-1 {
					v.selectedModel++
				} else {
					v.selectedModel = 0
				}
				return v, nil
			}

		case "ctrl+l":
			if !v.loadingModels {
				v.loadingModels = true
				return v, v.loadModels()
			}
			return v, nil

		case "enter":
			// View table if a message with result is selected
			if v.selectedMessage >= 0 && v.selectedMessage < len(v.messages) {
				msg := v.messages[v.selectedMessage]
				if msg.Result != nil && strings.HasPrefix(msg.Type, "sql") {
					v.parent.resultsView.SetResults(msg.Result, "")
					v.parent.resultsView.returnTo = ViewChat
					v.parent.mode = ViewResults
					return v, nil
				}
			}
			if v.focusField == focusFieldProvider {
				// Confirm provider selection, load models, and move to model field
				if !v.loadingModels {
					v.loadingModels = true
					v.focusField = focusFieldModel
					return v, v.loadModels()
				}
				return v, nil
			} else if v.focusField == focusFieldModel {
				// Confirm model selection and move to message field
				v.focusField = focusFieldMessage
				v.input.Focus()
				return v, nil
			} else if v.focusField == focusFieldMessage && !v.sending {
				query := strings.TrimSpace(v.input.Value())
				if query != "" {
					v.messages = append(v.messages, chatMessage{
						Role:    "user",
						Content: query,
						Type:    "message",
					})
					v.input.Reset()
					v.scrollOffset = 0
					v.selectedMessage = -1
					return v, v.sendChat(query)
				}
			}
			return v, nil
		}
	}

	if v.focusField == focusFieldMessage {
		v.input, cmd = v.input.Update(msg)
	}

	return v, cmd
}

func (v *ChatView) View() string {
	var b strings.Builder

	b.WriteString(styles.RenderTitle("AI Chat"))
	b.WriteString("\n\n")

	if !v.consented {
		msg := "You are about to use AI chat. Your prompts and related metadata may be sent to the selected AI provider. Avoid including secrets or sensitive data."
		b.WriteString(styles.RenderInfoBox(msg))
		b.WriteString("\n\n")
		b.WriteString(styles.RenderHelp(
			"[a]", "accept",
			"esc", "cancel",
		))
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	if v.err != nil && !v.sending {
		b.WriteString(styles.RenderErrorBox(v.err.Error()))
		b.WriteString("\n\n")
	}

	providerLabel := "Provider:"
	if v.focusField == focusFieldProvider {
		providerLabel = styles.KeyStyle.Render("▶ Provider:")
	} else {
		providerLabel = "  Provider:"
	}
	b.WriteString(providerLabel)
	b.WriteString(" ")
	for i, provider := range v.providers {
		if i == v.selectedProvider {
			b.WriteString(styles.ActiveListItemStyle.Render(fmt.Sprintf(" %s ", provider.Type)))
		} else {
			b.WriteString(styles.MutedStyle.Render(fmt.Sprintf(" %s ", provider.Type)))
		}
		if i < len(v.providers)-1 {
			b.WriteString(" ")
		}
	}
	b.WriteString("\n")

	modelLabel := "Model:"
	if v.focusField == focusFieldModel {
		modelLabel = styles.KeyStyle.Render("▶ Model:")
	} else {
		modelLabel = "  Model:"
	}
	b.WriteString(modelLabel)
	b.WriteString(" ")
	if v.loadingModels {
		b.WriteString(styles.MutedStyle.Render("Loading models..."))
	} else if len(v.models) == 0 {
		b.WriteString(styles.MutedStyle.Render("Press Ctrl+L to load models"))
	} else {
		for i, model := range v.models {
			displayName := model
			if len(displayName) > 20 {
				displayName = displayName[:17] + "..."
			}
			if i == v.selectedModel {
				b.WriteString(styles.ActiveListItemStyle.Render(fmt.Sprintf(" %s ", displayName)))
			} else {
				b.WriteString(styles.MutedStyle.Render(fmt.Sprintf(" %s ", displayName)))
			}
			if i < len(v.models)-1 && i < 3 {
				b.WriteString(" ")
			}
			if i == 3 {
				break
			}
		}
		if len(v.models) > 4 {
			b.WriteString(styles.MutedStyle.Render(fmt.Sprintf(" +%d more", len(v.models)-4)))
		}
	}
	b.WriteString("\n\n")

	if len(v.messages) > 0 {
		// Fixed max messages to display (keeps window size stable)
		maxVisibleMessages := 6
		b.WriteString(styles.RenderSubtitle("Conversation"))
		b.WriteString("\n")

		// Auto-scroll to show latest messages
		startIdx := v.scrollOffset
		if startIdx < 0 {
			startIdx = 0
		}
		if startIdx >= len(v.messages) {
			startIdx = len(v.messages) - 1
		}

		endIdx := startIdx + maxVisibleMessages
		if endIdx > len(v.messages) {
			endIdx = len(v.messages)
		}

		for i := startIdx; i < endIdx; i++ {
			msg := v.messages[i]

			isSelected := i == v.selectedMessage

			if msg.Role == "user" {
				prefix := "  "
				if isSelected {
					prefix = styles.KeyStyle.Render("▶ ")
				}
				b.WriteString(prefix)
				b.WriteString(styles.KeyStyle.Render("You: "))

				content := v.wrapText(msg.Content, 7) // 2 (prefix) + 5 ("You: ")
				if isSelected {
					content = styles.ActiveListItemStyle.Render(content)
				}
				b.WriteString(content)
				b.WriteString("\n\n")
			} else {
				prefix := "  "
				if isSelected {
					prefix = styles.KeyStyle.Render("▶ ")
				}
				b.WriteString(prefix)

				if msg.Type == "error" {
					b.WriteString(styles.ErrorStyle.Render("Error: "))
					b.WriteString(v.wrapText(msg.Content, 9)) // 2 (prefix) + 7 ("Error: ")
					b.WriteString("\n\n")
				} else if strings.HasPrefix(msg.Type, "sql") {
					b.WriteString(styles.SuccessStyle.Render("Assistant: "))
					if msg.Content != "" {
						b.WriteString(v.wrapText(msg.Content, 14)) // 2 (prefix) + 12 ("Assistant: ")
						b.WriteString("\n")
					}
					if msg.Result != nil {
						b.WriteString("  ")
						b.WriteString(v.renderTableSummary(msg.Result))
						if isSelected {
							b.WriteString("\n  ")
							b.WriteString(styles.MutedStyle.Render("Press 'v' to view full table"))
						}
					}
					b.WriteString("\n")
				} else {
					b.WriteString(styles.SuccessStyle.Render("Assistant: "))
					b.WriteString(v.wrapText(msg.Content, 14)) // 2 (prefix) + 12 ("Assistant: ")
					b.WriteString("\n\n")
				}
			}
		}

		if v.scrollOffset > 0 || endIdx < len(v.messages) {
			scrollInfo := fmt.Sprintf("Messages %d-%d of %d", startIdx+1, endIdx, len(v.messages))
			if v.scrollOffset > 0 {
				scrollInfo += " • ↑ scroll up"
			}
			if endIdx < len(v.messages) {
				scrollInfo += " • ↓ scroll down"
			}
			b.WriteString("\n")
			b.WriteString(styles.MutedStyle.Render(scrollInfo))
			b.WriteString("\n")
		}
	}

	if v.sending {
		b.WriteString(styles.MutedStyle.Render("Thinking..."))
		b.WriteString("\n\n")
	}

	b.WriteString(styles.KeyStyle.Render("Message:"))
	b.WriteString("\n")
	b.WriteString(v.input.View())
	b.WriteString("\n\n")

	b.WriteString(styles.RenderHelp(
		"↑/↓", "cycle fields",
		"←/→", "change selection",
		"enter", "confirm/send/view",
		"ctrl+p/n", "select message",
		"ctrl+r", "revoke consent",
		"esc", "back",
	))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

// wrapText wraps text to fit within the available width and limits lines
func (v *ChatView) wrapText(text string, indent int) string {
	availableWidth := v.width - 8 - indent // 8 = padding (4) + margin (4)
	if availableWidth < 20 {
		availableWidth = 20
	}
	wrapped := lipgloss.NewStyle().Width(availableWidth).Render(text)

	// Limit to max 4 lines to keep view stable
	lines := strings.Split(wrapped, "\n")
	maxLines := 4
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines[maxLines-1] = lines[maxLines-1] + "..."
		wrapped = strings.Join(lines, "\n")
	}
	return wrapped
}

func (v *ChatView) renderTableSummary(result *engine.GetRowsResult) string {
	if result == nil || len(result.Columns) == 0 {
		return styles.MutedStyle.Render("No results")
	}

	return styles.MutedStyle.Render(fmt.Sprintf("📊 Table: %d rows × %d columns", len(result.Rows), len(result.Columns)))
}

func (v *ChatView) renderTable(result *engine.GetRowsResult) string {
	if result == nil || len(result.Columns) == 0 {
		return ""
	}

	maxCols := 5
	cols := result.Columns
	if len(cols) > maxCols {
		cols = cols[:maxCols]
	}

	columns := make([]table.Column, len(cols))
	for i, col := range cols {
		columns[i] = table.Column{
			Title: col.Name,
			Width: 15,
		}
	}

	maxRows := 5
	rows := result.Rows
	if len(rows) > maxRows {
		rows = rows[:maxRows]
	}

	tableRows := make([]table.Row, len(rows))
	for i, row := range rows {
		tableRow := make([]string, len(cols))
		for j := range cols {
			if j < len(row) {
				val := row[j]
				if len(val) > 15 {
					val = val[:12] + "..."
				}
				tableRow[j] = val
			} else {
				tableRow[j] = ""
			}
		}
		tableRows[i] = table.Row(tableRow)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(tableRows),
		table.WithHeight(len(tableRows)),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Border).
		BorderBottom(true).
		Bold(true).
		Foreground(styles.Primary)
	s.Selected = s.Selected.
		Foreground(styles.Background).
		Background(styles.Primary).
		Bold(false)
	t.SetStyles(s)

	info := ""
	if len(result.Rows) > maxRows {
		info = styles.MutedStyle.Render(fmt.Sprintf("Showing %d of %d rows", maxRows, len(result.Rows)))
	}
	if len(result.Columns) > maxCols {
		if info != "" {
			info += " • "
		}
		info += styles.MutedStyle.Render(fmt.Sprintf("Showing %d of %d columns", maxCols, len(result.Columns)))
	}

	if info != "" {
		return t.View() + "\n" + info
	}
	return t.View()
}

func (v *ChatView) loadModels() tea.Cmd {
	return func() tea.Msg {
		if v.selectedProvider >= len(v.providers) {
			return modelsLoadedMsg{models: []string{}, err: fmt.Errorf("invalid provider selected")}
		}

		provider := v.providers[v.selectedProvider]
		modelType := provider.Type

		models, err := v.parent.dbManager.GetAIModels(provider.ProviderId, modelType, "")
		if err != nil {
			return modelsLoadedMsg{models: []string{}, err: err}
		}

		return modelsLoadedMsg{models: models, err: nil}
	}
}

func (v *ChatView) sendChat(query string) tea.Cmd {
	return func() tea.Msg {
		v.sending = true

		if v.selectedProvider >= len(v.providers) {
			return chatResponseMsg{messages: nil, err: fmt.Errorf("invalid provider selected")}
		}
		if len(v.models) == 0 || v.selectedModel >= len(v.models) {
			return chatResponseMsg{messages: nil, err: fmt.Errorf("please select a model first")}
		}

		provider := v.providers[v.selectedProvider]
		modelType := provider.Type
		model := v.models[v.selectedModel]

		// Use the schema selected in browser view if available
		schema := v.parent.browserView.currentSchema
		if schema == "" {
			schemas, err := v.parent.dbManager.GetSchemas()
			if err != nil {
				return chatResponseMsg{messages: nil, err: fmt.Errorf("failed to get schema: %w", err)}
			}
			schema = selectBestSchema(schemas)
		}

		previousConversation := ""
		if len(v.messages) > 1 {
			convMessages := []map[string]string{}
			for _, msg := range v.messages {
				if msg.Type != "error" {
					convMessages = append(convMessages, map[string]string{
						"role":    msg.Role,
						"content": msg.Content,
					})
				}
			}
			convBytes, _ := json.Marshal(convMessages)
			previousConversation = string(convBytes)
		}

		messages, err := v.parent.dbManager.SendAIChat(
			provider.ProviderId,
			modelType,
			"",
			schema,
			model,
			previousConversation,
			query,
		)

		if err != nil {
			return chatResponseMsg{messages: nil, err: err}
		}

		return chatResponseMsg{messages: messages, err: nil}
	}
}

func (v *ChatView) Init() tea.Cmd {
	if v.consented && len(v.providers) > 0 {
		return v.loadModels()
	}
	return nil
}
