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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
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

type ChatView struct {
	parent           *MainModel
	providers        []database.AIProvider
	selectedProvider int
	models           []string
	selectedModel    int
	loadingModels    bool
	modelsProvider   int // provider index models were loaded for
	messages         []chatMessage
	input            textarea.Model
	sending          bool
	err              error
	width            int
	height           int
	scrollOffset     int
	selectedMessage  int
	focusField       int
	// Consent gate for data governance
	consented bool
	// Cancellation support
	chatCancel   context.CancelFunc
	modelsCancel context.CancelFunc
	retryPrompt  RetryPrompt
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

	// Try to restore last used provider
	lastProvider := parent.config.GetLastAIProvider()
	providerRestored := false
	if lastProvider != "" {
		for i, p := range providers {
			if p.Type == lastProvider {
				selectedProvider = i
				providerRestored = true
				break
			}
		}
	}

	// Fall back to Ollama if no saved provider
	if !providerRestored {
		for i, p := range providers {
			if p.Type == "Ollama" {
				selectedProvider = i
				break
			}
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
				if !v.parent.PopView() {
					v.parent.mode = ViewBrowser
				}
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
		v.chatCancel = nil
		maxVisibleMessages := v.maxVisibleMessages()
		if msg.err != nil {
			// Check for cancellation - don't show error
			if errors.Is(msg.err, context.Canceled) {
				return v, nil
			}
			// Check for timeout - auto-retry with saved preference or show menu
			if errors.Is(msg.err, context.DeadlineExceeded) {
				preferred := v.parent.config.GetPreferredTimeout()
				if preferred > 0 && !v.retryPrompt.AutoRetried() {
					v.retryPrompt.SetAutoRetried(true)
					return v, v.sendChatWithTimeout(msg.query, time.Duration(preferred)*time.Second)
				}
				v.err = fmt.Errorf("request timed out")
				v.retryPrompt.Show(msg.query)
				return v, nil
			}
			v.err = msg.err
			v.messages = append(v.messages, chatMessage{
				Role:    "system",
				Content: fmt.Sprintf("Error: %s", v.err.Error()),
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

		// Save last used provider+model on successful response
		if v.selectedProvider < len(v.providers) {
			v.parent.config.SetLastAIProvider(v.providers[v.selectedProvider].Type)
		}
		if v.selectedModel < len(v.models) {
			v.parent.config.SetLastAIModel(v.models[v.selectedModel])
		}
		v.parent.config.Save()

		// Auto-scroll to show latest messages
		if len(v.messages) > maxVisibleMessages {
			v.scrollOffset = len(v.messages) - maxVisibleMessages
		}
		return v, v.parent.SetStatus("Response received")

	case modelsLoadedMsg:
		v.loadingModels = false
		v.modelsCancel = nil
		if msg.err != nil {
			// Check for cancellation - don't show error
			if errors.Is(msg.err, context.Canceled) {
				return v, nil
			}
			// Check for timeout
			if errors.Is(msg.err, context.DeadlineExceeded) {
				v.err = fmt.Errorf("loading models timed out")
			} else {
				v.err = msg.err
			}
			return v, nil
		}
		v.models = msg.models
		v.modelsProvider = v.selectedProvider
		if len(v.models) > 0 {
			v.selectedModel = 0
			// Try to restore last used model
			lastModel := v.parent.config.GetLastAIModel()
			if lastModel != "" {
				for i, m := range v.models {
					if m == lastModel {
						v.selectedModel = i
						// Auto-focus message field when both provider and model are restored
						v.focusField = focusFieldMessage
						v.input.Focus()
						break
					}
				}
			}
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
			if v.scrollOffset > 0 {
				v.scrollOffset--
			}
			return v, nil
		case tea.MouseWheelDown:
			maxVisible := v.maxVisibleMessages()
			maxScroll := len(v.messages) - maxVisible
			if maxScroll < 0 {
				maxScroll = 0
			}
			if v.scrollOffset < maxScroll {
				v.scrollOffset++
			}
			return v, nil
		}
		return v, nil

	case tea.KeyMsg:
		// Handle retry prompt for timed out requests
		if v.retryPrompt.IsActive() {
			result, handled := v.retryPrompt.HandleKeyMsg(msg.String())
			if handled {
				if result != nil {
					v.err = nil
					if result.Save {
						v.parent.config.SetPreferredTimeout(int(result.Timeout.Seconds()))
						v.parent.config.Save()
					}
					return v, v.sendChatWithTimeout(v.retryPrompt.TimedOutQuery(), result.Timeout)
				}
				return v, nil
			}
			return v, nil
		}

		switch {
		case key.Matches(msg, Keys.Chat.RevokeConsent):
			// Revoke consent
			v.consented = false
			v.parent.config.SetAIConsent(false)
			if err := v.parent.config.Save(); err != nil {
				v.err = fmt.Errorf("failed to revoke consent: %w", err)
			}
			return v, nil

		case key.Matches(msg, Keys.Global.Back):
			// First priority: cancel ongoing operations
			if v.sending && v.chatCancel != nil {
				v.chatCancel()
				return v, nil
			}
			if v.loadingModels && v.modelsCancel != nil {
				v.modelsCancel()
				return v, nil
			}
			if !v.parent.PopView() {
				v.parent.mode = ViewBrowser
			}
			return v, nil

		case key.Matches(msg, Keys.Chat.FocusInput):
			// Focus chat input
			v.focusField = focusFieldMessage
			v.input.Focus()
			return v, nil

		case key.Matches(msg, Keys.Chat.CycleFieldUp):
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

		case key.Matches(msg, Keys.Chat.CycleFieldDown):
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

		case key.Matches(msg, Keys.Chat.SelectPrevMsg):
			// Select previous message in conversation
			if len(v.messages) > 0 {
				if v.selectedMessage < 0 {
					v.selectedMessage = len(v.messages) - 1
				} else if v.selectedMessage > 0 {
					v.selectedMessage--
				}
				v.ensureMessageVisible()
			}
			return v, nil

		case key.Matches(msg, Keys.Chat.SelectNextMsg):
			// Select next message in conversation
			if len(v.messages) > 0 {
				if v.selectedMessage < 0 {
					v.selectedMessage = 0
				} else if v.selectedMessage < len(v.messages)-1 {
					v.selectedMessage++
				}
				v.ensureMessageVisible()
			}
			return v, nil

		case key.Matches(msg, Keys.Chat.ChangeLeft):
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

		case key.Matches(msg, Keys.Chat.ChangeRight):
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

		case key.Matches(msg, Keys.Chat.LoadModels):
			if !v.loadingModels {
				v.loadingModels = true
				return v, v.loadModels()
			}
			return v, nil

		case key.Matches(msg, Keys.Chat.Send):
			// View table if a message with result is selected
			if v.selectedMessage >= 0 && v.selectedMessage < len(v.messages) {
				chatMsg := v.messages[v.selectedMessage]
				if chatMsg.Result != nil && strings.HasPrefix(chatMsg.Type, "sql") {
					v.parent.resultsView.SetResults(chatMsg.Result, "")
					v.parent.PushView(ViewResults)
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
					v.retryPrompt.SetAutoRetried(false)
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

	// Show retry prompt for timed out requests
	if v.retryPrompt.IsActive() {
		b.WriteString(v.retryPrompt.View())
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	if v.err != nil && !v.sending {
		b.WriteString(styles.RenderErrorBox(v.err.Error()))
		b.WriteString("\n\n")
	}

	providerLabel := "Provider:"
	if v.focusField == focusFieldProvider {
		providerLabel = styles.RenderKey("▶ Provider:")
	} else {
		providerLabel = "  Provider:"
	}
	b.WriteString(providerLabel)
	b.WriteString(" ")
	for i, provider := range v.providers {
		if i == v.selectedProvider {
			b.WriteString(styles.ActiveListItemStyle.Render(fmt.Sprintf(" %s ", provider.Type)))
		} else {
			b.WriteString(styles.RenderMuted(fmt.Sprintf(" %s ", provider.Type)))
		}
		if i < len(v.providers)-1 {
			b.WriteString(" ")
		}
	}
	b.WriteString("\n")

	modelLabel := "Model:"
	if v.focusField == focusFieldModel {
		modelLabel = styles.RenderKey("▶ Model:")
	} else {
		modelLabel = "  Model:"
	}
	b.WriteString(modelLabel)
	b.WriteString(" ")
	if v.loadingModels {
		b.WriteString(v.parent.SpinnerView() + styles.RenderMuted(" Loading models... Press ESC to cancel"))
	} else if len(v.models) == 0 {
		b.WriteString(styles.RenderMuted("Press Ctrl+L to load models"))
	} else {
		for i, model := range v.models {
			displayName := model
			if len(displayName) > 20 {
				displayName = displayName[:17] + "..."
			}
			if i == v.selectedModel {
				b.WriteString(styles.ActiveListItemStyle.Render(fmt.Sprintf(" %s ", displayName)))
			} else {
				b.WriteString(styles.RenderMuted(fmt.Sprintf(" %s ", displayName)))
			}
			if i < len(v.models)-1 && i < 3 {
				b.WriteString(" ")
			}
			if i == 3 {
				break
			}
		}
		if len(v.models) > 4 {
			b.WriteString(styles.RenderMuted(fmt.Sprintf(" +%d more", len(v.models)-4)))
		}
	}
	b.WriteString("\n\n")

	if len(v.messages) > 0 {
		maxVisibleMessages := v.maxVisibleMessages()
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
					prefix = styles.RenderKey("▶ ")
				}
				b.WriteString(prefix)
				b.WriteString(styles.RenderKey("You: "))

				content := v.wrapText(msg.Content, 7) // 2 (prefix) + 5 ("You: ")
				if isSelected {
					content = styles.ActiveListItemStyle.Render(content)
				}
				b.WriteString(content)
				b.WriteString("\n\n")
			} else {
				prefix := "  "
				if isSelected {
					prefix = styles.RenderKey("▶ ")
				}
				b.WriteString(prefix)

				if msg.Type == "error" {
					b.WriteString(styles.RenderErr("Error: "))
					b.WriteString(v.wrapText(msg.Content, 9)) // 2 (prefix) + 7 ("Error: ")
					b.WriteString("\n\n")
				} else if strings.HasPrefix(msg.Type, "sql") {
					b.WriteString(styles.RenderOk("Assistant: "))
					if msg.Content != "" {
						b.WriteString(v.wrapText(msg.Content, 14)) // 2 (prefix) + 12 ("Assistant: ")
						b.WriteString("\n")
					}
					if msg.Result != nil {
						b.WriteString("  ")
						b.WriteString(v.renderTableSummary(msg.Result))
						if isSelected {
							b.WriteString("\n  ")
							b.WriteString(styles.RenderMuted("Press Enter to view full table"))
						}
					}
					b.WriteString("\n")
				} else {
					b.WriteString(styles.RenderOk("Assistant: "))
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
			b.WriteString(styles.RenderMuted(scrollInfo))
			b.WriteString("\n")
		}
	}

	if v.sending {
		b.WriteString(v.parent.SpinnerView() + styles.RenderMuted(" Thinking... Press ESC to cancel"))
		b.WriteString("\n\n")
	}

	b.WriteString(styles.RenderKey("Message:"))
	b.WriteString("\n")
	b.WriteString(v.input.View())
	b.WriteString("\n\n")

	b.WriteString(RenderBindingHelp(
		Keys.Chat.CycleFieldUp,
		Keys.Chat.ChangeLeft,
		Keys.Chat.Send,
		Keys.Chat.SelectPrevMsg,
		Keys.Chat.FocusInput,
		Keys.Chat.RevokeConsent,
		Keys.Global.Back,
	))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (v *ChatView) maxVisibleMessages() int {
	return clamp((v.height-18)/3, 2, 10)
}

// ensureMessageVisible adjusts scrollOffset so the selected message is within the visible window.
func (v *ChatView) ensureMessageVisible() {
	if v.selectedMessage < 0 {
		return
	}
	maxVisible := v.maxVisibleMessages()
	if v.selectedMessage < v.scrollOffset {
		v.scrollOffset = v.selectedMessage
	} else if v.selectedMessage >= v.scrollOffset+maxVisible {
		v.scrollOffset = v.selectedMessage - maxVisible + 1
	}
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
		return styles.RenderMuted("No results")
	}

	return styles.RenderMuted(fmt.Sprintf("📊 Table: %d rows × %d columns", len(result.Rows), len(result.Columns)))
}

func (v *ChatView) loadModels() tea.Cmd {
	if v.selectedProvider >= len(v.providers) {
		return func() tea.Msg {
			return modelsLoadedMsg{models: []string{}, err: fmt.Errorf("invalid provider selected")}
		}
	}

	provider := v.providers[v.selectedProvider]
	modelType := provider.Type

	// Get timeout from config
	timeout := v.parent.config.GetQueryTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	v.modelsCancel = cancel

	return func() tea.Msg {
		defer cancel()
		models, err := v.parent.dbManager.GetAIModelsWithContext(ctx, provider.ProviderId, modelType, "")
		if err != nil {
			return modelsLoadedMsg{models: []string{}, err: err}
		}

		return modelsLoadedMsg{models: models, err: nil}
	}
}

func (v *ChatView) sendChat(query string) tea.Cmd {
	return v.sendChatWithTimeout(query, v.parent.config.GetQueryTimeout())
}

func (v *ChatView) sendChatWithTimeout(query string, timeout time.Duration) tea.Cmd {
	// Validate inputs before creating closure
	if v.selectedProvider >= len(v.providers) {
		return func() tea.Msg {
			return chatResponseMsg{messages: nil, query: query, err: fmt.Errorf("invalid provider selected")}
		}
	}
	if len(v.models) == 0 || v.selectedModel >= len(v.models) {
		return func() tea.Msg {
			return chatResponseMsg{messages: nil, query: query, err: fmt.Errorf("please select a model first")}
		}
	}

	// Capture values for closure
	provider := v.providers[v.selectedProvider]
	modelType := provider.Type
	model := v.models[v.selectedModel]
	schema := v.parent.browserView.currentSchema
	msgs := make([]chatMessage, len(v.messages))
	copy(msgs, v.messages)

	// Set sending state and create context
	v.sending = true
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	v.chatCancel = cancel

	return func() tea.Msg {
		defer cancel()

		// Use the schema selected in browser view if available
		currentSchema := schema
		if currentSchema == "" {
			schemas, err := v.parent.dbManager.GetSchemas()
			if err != nil {
				// Schema-less databases (SQLite, Redis, etc.) don't support schemas.
				schemas = []string{}
			}
			currentSchema = selectBestSchema(schemas)
		}

		previousConversation := ""
		if len(msgs) > 1 {
			var convMessages []map[string]string
			for _, msg := range msgs {
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

		result, err := v.parent.dbManager.SendAIChatWithContext(
			ctx,
			provider.ProviderId,
			modelType,
			"",
			currentSchema,
			model,
			previousConversation,
			query,
		)

		if err != nil {
			return chatResponseMsg{messages: nil, query: query, err: err}
		}

		return chatResponseMsg{messages: result, query: query, err: nil}
	}
}

func (v *ChatView) Init() tea.Cmd {
	if v.consented && len(v.providers) > 0 {
		// Only fetch if models haven't been loaded or provider changed
		if len(v.models) == 0 || v.modelsProvider != v.selectedProvider {
			return v.loadModels()
		}
	}
	return nil
}
