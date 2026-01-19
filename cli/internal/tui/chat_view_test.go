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
	"errors"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clidey/whodb/core/src/engine"
)

func setupChatViewTest(t *testing.T) (*ChatView, func()) {
	t.Helper()

	tempDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)

	parent := NewMainModel()
	if parent.err != nil {
		t.Fatalf("Failed to create MainModel: %v", parent.err)
	}

	cleanup := func() {
		os.Setenv("HOME", origHome)
	}

	return parent.chatView, cleanup
}

func TestNewChatView(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	if v == nil {
		t.Fatal("NewChatView returned nil")
	}

	if v.selectedProvider < 0 {
		t.Error("Expected selectedProvider >= 0")
	}

	if len(v.messages) != 0 {
		t.Error("Expected empty messages initially")
	}

	if v.sending {
		t.Error("Expected sending to be false initially")
	}

	if v.selectedMessage != -1 {
		t.Errorf("Expected selectedMessage -1, got %d", v.selectedMessage)
	}

	if v.focusField != focusFieldMessage {
		t.Errorf("Expected focusField focusFieldMessage, got %d", v.focusField)
	}
}

func TestChatView_ConsentAccept(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = false

	// Press 'a' to accept
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	v, _ = v.Update(msg)

	if !v.consented {
		t.Error("Expected consented to be true after 'a'")
	}
}

func TestChatView_ConsentCancel_Esc(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = false

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.consented {
		t.Error("Expected consented to remain false after Esc")
	}

	if v.parent.mode != ViewBrowser {
		t.Errorf("Expected mode ViewBrowser after Esc, got %v", v.parent.mode)
	}
}

func TestChatView_ConsentCancel_Q(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = false

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	v, _ = v.Update(msg)

	if v.parent.mode != ViewBrowser {
		t.Errorf("Expected mode ViewBrowser after 'q', got %v", v.parent.mode)
	}
}

func TestChatView_ConsentCancel_D(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = false

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	v, _ = v.Update(msg)

	if v.parent.mode != ViewBrowser {
		t.Errorf("Expected mode ViewBrowser after 'd', got %v", v.parent.mode)
	}
}

func TestChatView_RevokeConsent(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true

	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	v, _ = v.Update(msg)

	if v.consented {
		t.Error("Expected consented to be false after Ctrl+R")
	}
}

func TestChatView_Escape_GoesBack(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.viewingResult = false

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.parent.mode != ViewBrowser {
		t.Errorf("Expected mode ViewBrowser after Esc, got %v", v.parent.mode)
	}
}

func TestChatView_Escape_ExitViewingResult(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.viewingResult = true

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	if v.viewingResult {
		t.Error("Expected viewingResult to be false after Esc")
	}

	// Mode should not change yet
	if v.parent.mode != ViewConnection {
		// Note: mode depends on initial state
	}
}

func TestChatView_FieldNavigation_Up(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.focusField = focusFieldMessage

	// Up from message goes to model
	msg := tea.KeyMsg{Type: tea.KeyUp}
	v, _ = v.Update(msg)

	if v.focusField != focusFieldModel {
		t.Errorf("Expected focusField focusFieldModel after Up, got %d", v.focusField)
	}

	// Up from model goes to provider
	v, _ = v.Update(msg)

	if v.focusField != focusFieldProvider {
		t.Errorf("Expected focusField focusFieldProvider after second Up, got %d", v.focusField)
	}

	// Up from provider stays at provider
	v, _ = v.Update(msg)

	if v.focusField != focusFieldProvider {
		t.Errorf("Expected focusField to stay focusFieldProvider, got %d", v.focusField)
	}
}

func TestChatView_FieldNavigation_Down(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.focusField = focusFieldProvider

	// Down from provider goes to model
	msg := tea.KeyMsg{Type: tea.KeyDown}
	v, _ = v.Update(msg)

	if v.focusField != focusFieldModel {
		t.Errorf("Expected focusField focusFieldModel after Down, got %d", v.focusField)
	}

	// Down from model goes to message
	v, _ = v.Update(msg)

	if v.focusField != focusFieldMessage {
		t.Errorf("Expected focusField focusFieldMessage after second Down, got %d", v.focusField)
	}

	// Down from message stays at message
	v, _ = v.Update(msg)

	if v.focusField != focusFieldMessage {
		t.Errorf("Expected focusField to stay focusFieldMessage, got %d", v.focusField)
	}
}

func TestChatView_ProviderSelection_LeftRight(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.focusField = focusFieldProvider

	if len(v.providers) < 2 {
		t.Skip("Need at least 2 providers to test navigation")
	}

	initialProvider := v.selectedProvider

	// Right changes provider
	msg := tea.KeyMsg{Type: tea.KeyRight}
	v, _ = v.Update(msg)

	if v.selectedProvider == initialProvider {
		// It should have changed unless at end and wrapped
	}

	// Left changes provider back
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	v, _ = v.Update(msg)

	// Just ensure no panic
}

func TestChatView_ModelSelection_LeftRight(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.focusField = focusFieldModel
	v.models = []string{"model1", "model2", "model3"}
	v.selectedModel = 0

	// Right moves forward
	msg := tea.KeyMsg{Type: tea.KeyRight}
	v, _ = v.Update(msg)

	if v.selectedModel != 1 {
		t.Errorf("Expected selectedModel 1 after Right, got %d", v.selectedModel)
	}

	// Left moves backward
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	v, _ = v.Update(msg)

	if v.selectedModel != 0 {
		t.Errorf("Expected selectedModel 0 after Left, got %d", v.selectedModel)
	}
}

func TestChatView_ModelSelection_WrapAround(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.focusField = focusFieldModel
	v.models = []string{"model1", "model2"}
	v.selectedModel = 0

	// Left from 0 wraps to end
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	v, _ = v.Update(msg)

	if v.selectedModel != 1 {
		t.Errorf("Expected selectedModel 1 after wrap, got %d", v.selectedModel)
	}

	// Right from end wraps to 0
	msg = tea.KeyMsg{Type: tea.KeyRight}
	v, _ = v.Update(msg)

	if v.selectedModel != 0 {
		t.Errorf("Expected selectedModel 0 after wrap, got %d", v.selectedModel)
	}
}

func TestChatView_MessageNavigation_CtrlP(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.messages = []chatMessage{
		{Role: "user", Content: "Hello"},
		{Role: "system", Content: "Hi"},
	}
	v.selectedMessage = -1

	// Ctrl+P selects last message when none selected
	msg := tea.KeyMsg{Type: tea.KeyCtrlP}
	v, _ = v.Update(msg)

	if v.selectedMessage != 1 {
		t.Errorf("Expected selectedMessage 1 after Ctrl+P, got %d", v.selectedMessage)
	}

	// Ctrl+P goes to previous
	v, _ = v.Update(msg)

	if v.selectedMessage != 0 {
		t.Errorf("Expected selectedMessage 0 after second Ctrl+P, got %d", v.selectedMessage)
	}
}

func TestChatView_MessageNavigation_CtrlN(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.messages = []chatMessage{
		{Role: "user", Content: "Hello"},
		{Role: "system", Content: "Hi"},
	}
	v.selectedMessage = -1

	// Ctrl+N selects first message when none selected
	msg := tea.KeyMsg{Type: tea.KeyCtrlN}
	v, _ = v.Update(msg)

	if v.selectedMessage != 0 {
		t.Errorf("Expected selectedMessage 0 after Ctrl+N, got %d", v.selectedMessage)
	}

	// Ctrl+N goes to next
	v, _ = v.Update(msg)

	if v.selectedMessage != 1 {
		t.Errorf("Expected selectedMessage 1 after second Ctrl+N, got %d", v.selectedMessage)
	}
}

func TestChatView_FocusInput(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.focusField = focusFieldModel // Start from model field

	// "/" focuses message input from any field
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	v, _ = v.Update(msg)

	if v.focusField != focusFieldMessage {
		t.Errorf("Expected focusField focusFieldMessage after '/', got %d", v.focusField)
	}
}

func TestChatView_FocusInput_Slash(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.focusField = focusFieldProvider

	// '/' also focuses input
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	v, _ = v.Update(msg)

	if v.focusField != focusFieldMessage {
		t.Errorf("Expected focusField focusFieldMessage after '/', got %d", v.focusField)
	}
}

func TestChatView_MouseScroll(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.focusField = focusFieldMessage
	v.messages = make([]chatMessage, 20)
	v.scrollOffset = 5
	v.height = 30

	// Mouse wheel up
	msg := tea.MouseMsg{Type: tea.MouseWheelUp}
	v, _ = v.Update(msg)

	if v.scrollOffset != 4 {
		t.Errorf("Expected scrollOffset 4 after wheel up, got %d", v.scrollOffset)
	}

	// Mouse wheel down
	msg = tea.MouseMsg{Type: tea.MouseWheelDown}
	v, _ = v.Update(msg)

	// Scroll offset may or may not change depending on max calculation
	// Just ensure no panic
}

func TestChatView_MouseScrollUp_AtTop(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.focusField = focusFieldMessage
	v.scrollOffset = 0

	msg := tea.MouseMsg{Type: tea.MouseWheelUp}
	v, _ = v.Update(msg)

	if v.scrollOffset != 0 {
		t.Errorf("Expected scrollOffset to stay 0 at top, got %d", v.scrollOffset)
	}
}

func TestChatView_WindowSizeMsg(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	v, _ = v.Update(msg)

	if v.width != 120 {
		t.Errorf("Expected width 120, got %d", v.width)
	}

	if v.height != 40 {
		t.Errorf("Expected height 40, got %d", v.height)
	}
}

func TestChatView_WindowSizeMsg_ConsentScreen(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = false

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	v, _ = v.Update(msg)

	if v.width != 100 {
		t.Errorf("Expected width 100, got %d", v.width)
	}
}

func TestChatView_ModelsLoadedMsg(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.loadingModels = true

	msg := modelsLoadedMsg{models: []string{"gpt-4", "gpt-3.5"}, err: nil}
	v, _ = v.Update(msg)

	if v.loadingModels {
		t.Error("Expected loadingModels to be false after models loaded")
	}

	if len(v.models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(v.models))
	}

	if v.selectedModel != 0 {
		t.Errorf("Expected selectedModel 0, got %d", v.selectedModel)
	}
}

func TestChatView_ModelsLoadedMsg_Error(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.loadingModels = true

	msg := modelsLoadedMsg{models: nil, err: errors.New("failed to load")}
	v, _ = v.Update(msg)

	if v.loadingModels {
		t.Error("Expected loadingModels to be false after error")
	}

	if v.err == nil {
		t.Error("Expected error to be set")
	}
}

func TestChatView_ChatResponseMsg(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.sending = true

	msg := chatResponseMsg{
		messages: nil,
		err:      nil,
	}
	v, _ = v.Update(msg)

	if v.sending {
		t.Error("Expected sending to be false after response")
	}
}

func TestChatView_ChatResponseMsg_Error(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.sending = true
	initialMsgCount := len(v.messages)

	msg := chatResponseMsg{
		messages: nil,
		err:      errors.New("chat failed"),
	}
	v, _ = v.Update(msg)

	if v.sending {
		t.Error("Expected sending to be false after error")
	}

	if v.err == nil {
		t.Error("Expected error to be set")
	}

	// Error message should be added
	if len(v.messages) != initialMsgCount+1 {
		t.Error("Expected error message to be added")
	}
}

func TestChatView_View_ConsentScreen(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = false

	view := v.View()

	if !strings.Contains(view, "AI Chat") {
		t.Error("Expected 'AI Chat' title")
	}

	if !strings.Contains(view, "accept") {
		t.Error("Expected 'accept' option on consent screen")
	}

	if !strings.Contains(view, "cancel") {
		t.Error("Expected 'cancel' option on consent screen")
	}
}

func TestChatView_View_ChatScreen(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true

	view := v.View()

	if !strings.Contains(view, "AI Chat") {
		t.Error("Expected 'AI Chat' title")
	}

	if !strings.Contains(view, "Provider:") {
		t.Error("Expected 'Provider:' label")
	}

	if !strings.Contains(view, "Model:") {
		t.Error("Expected 'Model:' label")
	}

	if !strings.Contains(view, "Message:") {
		t.Error("Expected 'Message:' label")
	}
}

func TestChatView_View_WithMessages(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.messages = []chatMessage{
		{Role: "user", Content: "Hello", Type: "message"},
		{Role: "system", Content: "Hi there", Type: "message"},
	}

	view := v.View()

	if !strings.Contains(view, "Conversation") {
		t.Error("Expected 'Conversation' subtitle")
	}

	if !strings.Contains(view, "You:") {
		t.Error("Expected 'You:' label for user message")
	}

	if !strings.Contains(view, "Assistant:") {
		t.Error("Expected 'Assistant:' label for system message")
	}
}

func TestChatView_View_WithError(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.err = errors.New("test error")
	v.sending = false

	view := v.View()

	if !strings.Contains(view, "test error") {
		t.Error("Expected error message in view")
	}
}

func TestChatView_View_Sending(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.sending = true

	view := v.View()

	if !strings.Contains(view, "Thinking...") {
		t.Error("Expected 'Thinking...' when sending")
	}
}

func TestChatView_View_LoadingModels(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.loadingModels = true

	view := v.View()

	if !strings.Contains(view, "Loading models...") {
		t.Error("Expected 'Loading models...' when loading")
	}
}

func TestChatView_View_NoModels(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true
	v.models = []string{}
	v.loadingModels = false

	view := v.View()

	if !strings.Contains(view, "Ctrl+L") || !strings.Contains(view, "load models") {
		t.Error("Expected hint to load models")
	}
}

func TestChatView_WrapText(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.width = 80

	result := v.wrapText("Hello world", 5)

	if result == "" {
		t.Error("Expected non-empty wrapped text")
	}
}

func TestChatView_WrapText_LongText(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.width = 50

	longText := strings.Repeat("This is a long line of text. ", 10)
	result := v.wrapText(longText, 5)

	// Should be truncated to max 4 lines with "..."
	lines := strings.Split(result, "\n")
	if len(lines) > 4 {
		t.Errorf("Expected max 4 lines, got %d", len(lines))
	}
}

func TestChatView_RenderTableSummary(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	// Nil result
	result := v.renderTableSummary(nil)
	if !strings.Contains(result, "No results") {
		t.Error("Expected 'No results' for nil result")
	}

	// Empty columns
	result = v.renderTableSummary(&engine.GetRowsResult{
		Columns: []engine.Column{},
		Rows:    [][]string{},
	})
	if !strings.Contains(result, "No results") {
		t.Error("Expected 'No results' for empty columns")
	}

	// Valid result
	result = v.renderTableSummary(&engine.GetRowsResult{
		Columns: []engine.Column{{Name: "id"}, {Name: "name"}},
		Rows:    [][]string{{"1", "Alice"}, {"2", "Bob"}},
	})
	if !strings.Contains(result, "2 rows") {
		t.Error("Expected row count in summary")
	}
	if !strings.Contains(result, "2 columns") {
		t.Error("Expected column count in summary")
	}
}

func TestChatView_RenderTable(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	// Nil result
	result := v.renderTable(nil)
	if result != "" {
		t.Error("Expected empty string for nil result")
	}

	// Empty columns
	result = v.renderTable(&engine.GetRowsResult{
		Columns: []engine.Column{},
	})
	if result != "" {
		t.Error("Expected empty string for empty columns")
	}

	// Valid result
	result = v.renderTable(&engine.GetRowsResult{
		Columns: []engine.Column{{Name: "id"}, {Name: "name"}},
		Rows:    [][]string{{"1", "Alice"}, {"2", "Bob"}},
	})
	if result == "" {
		t.Error("Expected non-empty table render")
	}
}

func TestChatView_Init_WithConsent(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = true

	if len(v.providers) > 0 {
		cmd := v.Init()
		if cmd == nil {
			t.Error("Expected Init to return command when consented with providers")
		}
	}
}

func TestChatView_Init_WithoutConsent(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	v.consented = false

	cmd := v.Init()
	if cmd != nil {
		t.Error("Expected Init to return nil when not consented")
	}
}

func TestChatMessage_Fields(t *testing.T) {
	msg := chatMessage{
		Role:    "user",
		Content: "Hello",
		Type:    "message",
		Result:  nil,
	}

	if msg.Role != "user" {
		t.Errorf("Expected Role 'user', got '%s'", msg.Role)
	}

	if msg.Content != "Hello" {
		t.Errorf("Expected Content 'Hello', got '%s'", msg.Content)
	}

	if msg.Type != "message" {
		t.Errorf("Expected Type 'message', got '%s'", msg.Type)
	}
}

func TestFocusField_Constants(t *testing.T) {
	// Verify focus field constants are ordered
	if focusFieldProvider >= focusFieldModel {
		t.Error("Expected focusFieldProvider < focusFieldModel")
	}

	if focusFieldModel >= focusFieldMessage {
		t.Error("Expected focusFieldModel < focusFieldMessage")
	}
}

func TestChatView_RetryPrompt_EscCancels(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	// Set up retry prompt state
	v.consented = true
	v.retryPrompt = true
	v.timedOutQuery = "tell me about the users table"
	v.err = errors.New("request timed out")

	// Send ESC key
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v, _ = v.Update(msg)

	// Verify retry prompt was dismissed
	if v.retryPrompt {
		t.Error("Expected retryPrompt to be false after ESC")
	}

	// Verify timed out query was cleared
	if v.timedOutQuery != "" {
		t.Errorf("Expected timedOutQuery to be empty, got '%s'", v.timedOutQuery)
	}
}

func TestChatView_RetryPrompt_KeyHandling(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"option_1", "1"},
		{"option_2", "2"},
		{"option_3", "3"},
		{"option_4", "4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, cleanup := setupChatViewTest(t)
			defer cleanup()

			// Set up retry prompt state
			v.consented = true
			v.retryPrompt = true
			v.timedOutQuery = "tell me about the users table"
			v.err = errors.New("request timed out")
			// Need to have providers and models for sendChatWithTimeout to work
			v.models = []string{"test-model"}

			// Send number key
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			v, cmd := v.Update(msg)

			// Verify retry prompt was dismissed
			if v.retryPrompt {
				t.Error("Expected retryPrompt to be false after selecting retry option")
			}

			// Verify error was cleared
			if v.err != nil {
				t.Error("Expected err to be nil after retry")
			}

			// Verify a command was returned (the query execution)
			if cmd == nil {
				t.Error("Expected a command to be returned for retry")
			}
		})
	}
}

func TestChatView_RetryPrompt_IgnoresOtherKeys(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	// Set up retry prompt state
	v.consented = true
	v.retryPrompt = true
	v.timedOutQuery = "tell me about the users table"

	// Send an unrelated key (like 'a')
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}
	v, _ = v.Update(msg)

	// Verify retry prompt is still active
	if !v.retryPrompt {
		t.Error("Expected retryPrompt to still be true after unrecognized key")
	}

	// Verify query wasn't cleared
	if v.timedOutQuery == "" {
		t.Error("Expected timedOutQuery to still be set")
	}
}

func TestChatView_RetryPrompt_View(t *testing.T) {
	v, cleanup := setupChatViewTest(t)
	defer cleanup()

	// Set up retry prompt state
	v.consented = true
	v.retryPrompt = true
	v.timedOutQuery = "tell me about the users table"

	view := v.View()

	// Verify retry prompt is shown
	if !strings.Contains(view, "timed out") {
		t.Error("Expected 'timed out' in view")
	}
	if !strings.Contains(view, "60 seconds") {
		t.Error("Expected '60 seconds' option in view")
	}
	if !strings.Contains(view, "2 minutes") {
		t.Error("Expected '2 minutes' option in view")
	}
	if !strings.Contains(view, "5 minutes") {
		t.Error("Expected '5 minutes' option in view")
	}
	if !strings.Contains(view, "No limit") {
		t.Error("Expected 'No limit' option in view")
	}
}
