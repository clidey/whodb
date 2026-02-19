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
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
)

func TestRenderBindingHelp_Empty(t *testing.T) {
	result := RenderBindingHelp()
	if result != "" {
		t.Errorf("Expected empty string for no bindings, got %q", result)
	}
}

func TestRenderBindingHelp_SingleBinding(t *testing.T) {
	b := key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm"))
	result := RenderBindingHelp(b)

	if !strings.Contains(result, "enter") {
		t.Errorf("Expected result to contain 'enter', got %q", result)
	}
	if !strings.Contains(result, "confirm") {
		t.Errorf("Expected result to contain 'confirm', got %q", result)
	}
}

func TestRenderBindingHelp_MultipleBindings(t *testing.T) {
	b1 := key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm"))
	b2 := key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))
	result := RenderBindingHelp(b1, b2)

	if !strings.Contains(result, "enter") {
		t.Errorf("Expected result to contain 'enter', got %q", result)
	}
	if !strings.Contains(result, "cancel") {
		t.Errorf("Expected result to contain 'cancel', got %q", result)
	}
}

func TestKeys_GlobalBindings(t *testing.T) {
	// Verify global bindings have help text
	h := Keys.Global.Quit.Help()
	if h.Key == "" {
		t.Error("Expected Global.Quit to have a help key")
	}
	if h.Desc == "" {
		t.Error("Expected Global.Quit to have a help description")
	}

	h = Keys.Global.Back.Help()
	if h.Key == "" {
		t.Error("Expected Global.Back to have a help key")
	}

	h = Keys.Global.NextView.Help()
	if h.Key == "" {
		t.Error("Expected Global.NextView to have a help key")
	}
}

func TestKeys_BrowserBindings(t *testing.T) {
	// Verify browser bindings are configured with correct keys
	h := Keys.Browser.Up.Help()
	if h.Key == "" {
		t.Error("Expected Browser.Up to have a help key")
	}

	h = Keys.Browser.Select.Help()
	if h.Key == "" {
		t.Error("Expected Browser.Select to have a help key")
	}

	h = Keys.Browser.Filter.Help()
	if h.Key == "" {
		t.Error("Expected Browser.Filter to have a help key")
	}
}

func TestKeys_EditorBindings(t *testing.T) {
	h := Keys.Editor.Execute.Help()
	if h.Key == "" {
		t.Error("Expected Editor.Execute to have a help key")
	}
	if h.Desc == "" {
		t.Error("Expected Editor.Execute to have a help description")
	}
}

func TestKeys_ResultsBindings(t *testing.T) {
	h := Keys.Results.NextPage.Help()
	if h.Key == "" {
		t.Error("Expected Results.NextPage to have a help key")
	}

	h = Keys.Results.Where.Help()
	if h.Key == "" {
		t.Error("Expected Results.Where to have a help key")
	}
}
