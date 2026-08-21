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
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/clidey/whodb/cli/internal/config"
)

func TestBookmarksView_ScrollsIntoView(t *testing.T) {
	setupTestEnv(t)

	parent := NewMainModel()
	if parent.err != nil {
		t.Fatalf("Failed to create MainModel: %v", parent.err)
	}
	v := parent.bookmarksView

	for i := 0; i < 20; i++ {
		parent.config.AddSavedQuery(fmt.Sprintf("Bookmark %d", i), fmt.Sprintf("SELECT %d", i))
	}

	v, _ = v.Update(tea.WindowSizeMsg{Width: 100, Height: 15})

	view := v.View()
	if !strings.Contains(view, "Bookmark 0") {
		t.Errorf("Expected first bookmark visible initially, got: %s", view)
	}
	if strings.Contains(view, "Bookmark 19") {
		t.Errorf("Expected last bookmark hidden initially, got: %s", view)
	}
	if !strings.Contains(view, "more") {
		t.Errorf("Expected a scroll indicator, got: %s", view)
	}

	for i := 0; i < 19; i++ {
		v, _ = v.Update(tea.KeyPressMsg{Text: "j"})
	}
	if v.cursor != 19 {
		t.Fatalf("Expected cursor at 19, got %d", v.cursor)
	}

	view = v.View()
	if !strings.Contains(view, "Bookmark 19") {
		t.Errorf("Expected last bookmark visible after scrolling down, got: %s", view)
	}

	// The footer must always be present, regardless of list length.
	if !strings.Contains(view, "load") && !strings.Contains(view, "delete") {
		t.Errorf("Expected footer shortcuts to remain visible, got: %s", view)
	}
}

func TestProfilesView_ScrollsIntoView(t *testing.T) {
	setupTestEnv(t)

	parent := NewMainModel()
	if parent.err != nil {
		t.Fatalf("Failed to create MainModel: %v", parent.err)
	}
	v := parent.profilesView

	for i := 0; i < 20; i++ {
		parent.config.AddProfile(config.Profile{
			Name:       fmt.Sprintf("Profile %d", i),
			Connection: "test-conn",
		})
	}

	v, _ = v.Update(tea.WindowSizeMsg{Width: 100, Height: 15})

	view := v.View()
	if !strings.Contains(view, "Profile 0") {
		t.Errorf("Expected first profile visible initially, got: %s", view)
	}
	if strings.Contains(view, "Profile 19") {
		t.Errorf("Expected last profile hidden initially, got: %s", view)
	}
	if !strings.Contains(view, "more") {
		t.Errorf("Expected a scroll indicator, got: %s", view)
	}

	for i := 0; i < 19; i++ {
		v, _ = v.Update(tea.KeyPressMsg{Text: "j"})
	}
	if v.cursor != 19 {
		t.Fatalf("Expected cursor at 19, got %d", v.cursor)
	}

	view = v.View()
	if !strings.Contains(view, "Profile 19") {
		t.Errorf("Expected last profile visible after scrolling down, got: %s", view)
	}
}
