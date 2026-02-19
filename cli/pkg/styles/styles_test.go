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

package styles

import (
	"runtime"
	"strings"
	"testing"
)

func TestKeyExecute_PlatformSpecific(t *testing.T) {
	if runtime.GOOS == "darwin" {
		if KeyExecute != "opt+enter" {
			t.Errorf("KeyExecute on darwin = %s, want opt+enter", KeyExecute)
		}
	} else {
		if KeyExecute != "alt+enter" {
			t.Errorf("KeyExecute on %s = %s, want alt+enter", runtime.GOOS, KeyExecute)
		}
	}
}

func TestDisableColor(t *testing.T) {
	// Store original state
	origEnabled := ColorEnabled()

	DisableColor()

	if ColorEnabled() {
		t.Error("ColorEnabled() should return false after DisableColor()")
	}

	// Note: We can't easily restore the color state since lipgloss profile
	// is a global setting, but this is acceptable for tests
	_ = origEnabled
}

func TestColorEnabled(t *testing.T) {
	// Just verify it doesn't panic and returns a bool
	_ = ColorEnabled()
}

func TestRenderTitle(t *testing.T) {
	result := RenderTitle("Test Title")
	if result == "" {
		t.Error("RenderTitle should not return empty string")
	}
	if !strings.Contains(result, "Test Title") {
		t.Errorf("RenderTitle result should contain the title text")
	}
}

func TestRenderSubtitle(t *testing.T) {
	result := RenderSubtitle("Test Subtitle")
	if result == "" {
		t.Error("RenderSubtitle should not return empty string")
	}
	if !strings.Contains(result, "Test Subtitle") {
		t.Errorf("RenderSubtitle result should contain the subtitle text")
	}
}

func TestRenderBox(t *testing.T) {
	result := RenderBox("Box Content")
	if result == "" {
		t.Error("RenderBox should not return empty string")
	}
	if !strings.Contains(result, "Box Content") {
		t.Errorf("RenderBox result should contain the content")
	}
}

func TestRenderActiveBox(t *testing.T) {
	result := RenderActiveBox("Active Content")
	if result == "" {
		t.Error("RenderActiveBox should not return empty string")
	}
	if !strings.Contains(result, "Active Content") {
		t.Errorf("RenderActiveBox result should contain the content")
	}
}

func TestRenderError(t *testing.T) {
	result := RenderError("Something went wrong")
	if result == "" {
		t.Error("RenderError should not return empty string")
	}
	if !strings.Contains(result, "Something went wrong") {
		t.Errorf("RenderError result should contain the message")
	}
}

func TestRenderSuccess(t *testing.T) {
	result := RenderSuccess("Operation completed")
	if result == "" {
		t.Error("RenderSuccess should not return empty string")
	}
	if !strings.Contains(result, "Operation completed") {
		t.Errorf("RenderSuccess result should contain the message")
	}
}

func TestRenderHelp_Empty(t *testing.T) {
	result := RenderHelp()
	if result != "" {
		t.Errorf("RenderHelp() with no args should return empty string, got %q", result)
	}
}

func TestRenderHelp_WithPairs(t *testing.T) {
	result := RenderHelp("ctrl+c", "quit", "enter", "select")
	if result == "" {
		t.Error("RenderHelp with args should not return empty string")
	}
	if !strings.Contains(result, "ctrl+c") {
		t.Error("RenderHelp result should contain key")
	}
	if !strings.Contains(result, "quit") {
		t.Error("RenderHelp result should contain description")
	}
}

func TestRenderHelp_OddArgs(t *testing.T) {
	// With odd number of args, the last one should be ignored
	result := RenderHelp("ctrl+c", "quit", "orphan")
	if !strings.Contains(result, "ctrl+c") {
		t.Error("RenderHelp should still render complete pairs")
	}
}

func TestRenderHelpParts_Empty(t *testing.T) {
	result := RenderHelpParts([]string{})
	if result != "" {
		t.Errorf("RenderHelpParts with empty slice should return empty string, got %q", result)
	}
}

func TestRenderHelpParts_WithParts(t *testing.T) {
	parts := []string{"part1", "part2", "part3"}
	result := RenderHelpParts(parts)
	if result == "" {
		t.Error("RenderHelpParts with args should not return empty string")
	}
}

func TestRenderHelpWithMaxItems(t *testing.T) {
	result := RenderHelpWithMaxItems(2, "a", "1", "b", "2", "c", "3", "d", "4")
	if result == "" {
		t.Error("RenderHelpWithMaxItems should not return empty string")
	}
	// Should have multiple lines since we set max 2 per line
	lines := strings.Split(result, "\n")
	if len(lines) < 2 {
		t.Errorf("RenderHelpWithMaxItems should create multiple lines, got %d", len(lines))
	}
}

func TestRenderHelpWithMaxItems_Empty(t *testing.T) {
	result := RenderHelpWithMaxItems(2)
	if result != "" {
		t.Errorf("RenderHelpWithMaxItems with no keys should return empty string, got %q", result)
	}
}

func TestRenderErrorBox(t *testing.T) {
	result := RenderErrorBox("Error message here")
	if result == "" {
		t.Error("RenderErrorBox should not return empty string")
	}
	if !strings.Contains(result, "Error message here") {
		t.Error("RenderErrorBox should contain the message")
	}
	if !strings.Contains(result, "Error") {
		t.Error("RenderErrorBox should contain 'Error' title")
	}
}

func TestRenderInfoBox(t *testing.T) {
	result := RenderInfoBox("Info message here")
	if result == "" {
		t.Error("RenderInfoBox should not return empty string")
	}
	if !strings.Contains(result, "Info message here") {
		t.Error("RenderInfoBox should contain the message")
	}
	if !strings.Contains(result, "Info") {
		t.Error("RenderInfoBox should contain 'Info' title")
	}
}

func TestColorConstants(t *testing.T) {
	// Verify color constants are defined
	colors := []struct {
		name  string
		color interface{}
	}{
		{"Primary", Primary},
		{"Secondary", Secondary},
		{"Success", Success},
		{"Error", Error},
		{"Warning", Warning},
		{"Info", Info},
		{"Muted", Muted},
		{"Background", Background},
		{"Foreground", Foreground},
		{"Border", Border},
		{"Accent", Accent},
	}

	for _, c := range colors {
		if c.color == nil {
			t.Errorf("Color %s should not be nil", c.name)
		}
	}
}

func TestRenderShorthands(t *testing.T) {
	tests := []struct {
		name   string
		render func(string) string
		input  string
	}{
		{"RenderMuted", RenderMuted, "muted text"},
		{"RenderKey", RenderKey, "key text"},
		{"RenderErr", RenderErr, "error text"},
		{"RenderOk", RenderOk, "success text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.render(tt.input)
			if result == "" {
				t.Errorf("%s returned empty string", tt.name)
			}
			if !strings.Contains(result, tt.input) {
				t.Errorf("%s result should contain %q", tt.name, tt.input)
			}
		})
	}
}
