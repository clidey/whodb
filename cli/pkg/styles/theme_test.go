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

package styles

import "testing"

func TestDefaultThemeIsSet(t *testing.T) {
	theme := GetTheme()
	if theme == nil {
		t.Fatal("GetTheme() returned nil")
	}
	if theme.Name != "default" {
		t.Errorf("Expected default theme name 'default', got %q", theme.Name)
	}
}

func TestListThemes(t *testing.T) {
	names := ListThemes()
	if len(names) != 8 {
		t.Errorf("Expected 8 themes, got %d", len(names))
	}

	expected := []string{"default", "light", "monokai", "dracula", "nord", "gruvbox", "tokyo-night", "catppuccin"}
	for i, name := range expected {
		if i >= len(names) {
			break
		}
		if names[i] != name {
			t.Errorf("Theme[%d] = %q, want %q", i, names[i], name)
		}
	}
}

func TestGetThemeByName(t *testing.T) {
	for _, name := range ListThemes() {
		theme := GetThemeByName(name)
		if theme == nil {
			t.Errorf("GetThemeByName(%q) returned nil", name)
			continue
		}
		if theme.Name != name {
			t.Errorf("GetThemeByName(%q).Name = %q", name, theme.Name)
		}
	}

	if GetThemeByName("nonexistent") != nil {
		t.Error("Expected nil for unknown theme name")
	}
}

func TestSetThemeUpdatesColors(t *testing.T) {
	// Remember original
	original := GetTheme()
	defer SetTheme(original)

	// Switch to Dracula
	dracula := GetThemeByName("dracula")
	if dracula == nil {
		t.Fatal("Dracula theme not found")
	}
	SetTheme(dracula)

	if GetTheme().Name != "dracula" {
		t.Errorf("Expected active theme 'dracula', got %q", GetTheme().Name)
	}

	// Verify a color was updated
	if Primary != dracula.Primary {
		t.Error("Primary color not updated after SetTheme")
	}
	if Background != dracula.Background {
		t.Error("Background color not updated after SetTheme")
	}
}

func TestSetThemeRebuildsDerivedStyles(t *testing.T) {
	original := GetTheme()
	defer SetTheme(original)

	// Switch to Nord
	nord := GetThemeByName("nord")
	SetTheme(nord)

	// Verify that derived styles were rebuilt (they should not be nil/empty)
	output := RenderTitle("test")
	if output == "" {
		t.Error("RenderTitle returned empty after theme change")
	}

	output = RenderError("test")
	if output == "" {
		t.Error("RenderError returned empty after theme change")
	}

	output = RenderSuccess("test")
	if output == "" {
		t.Error("RenderSuccess returned empty after theme change")
	}
}

func TestAllThemesHaveAllColors(t *testing.T) {
	for _, name := range ListThemes() {
		theme := GetThemeByName(name)

		// Check that no color slot is entirely empty
		if theme.Primary.Light == "" && theme.Primary.Dark == "" {
			t.Errorf("Theme %q: Primary has no colors", name)
		}
		if theme.Background.Light == "" && theme.Background.Dark == "" {
			t.Errorf("Theme %q: Background has no colors", name)
		}
		if theme.Foreground.Light == "" && theme.Foreground.Dark == "" {
			t.Errorf("Theme %q: Foreground has no colors", name)
		}
		if theme.Error.Light == "" && theme.Error.Dark == "" {
			t.Errorf("Theme %q: Error has no colors", name)
		}
		if theme.Keyword.Light == "" && theme.Keyword.Dark == "" {
			t.Errorf("Theme %q: Keyword has no colors", name)
		}
	}
}

func TestSetThemeCycleDoesNotPanic(t *testing.T) {
	original := GetTheme()
	defer SetTheme(original)

	// Cycle through all themes rapidly — catch any panic in rebuild
	for _, name := range ListThemes() {
		theme := GetThemeByName(name)
		SetTheme(theme)
		// Render something with the new theme
		_ = RenderTitle("test")
		_ = RenderError("test")
		_ = RenderHelp("key", "desc")
		_ = RenderErrorBox("error message")
		_ = RenderInfoBox("info message")
	}
}
