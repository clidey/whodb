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
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Pane is the interface that all TUI views implement for polymorphic layout
// dispatch. It enables the layout engine to treat views interchangeably
// when composing split-pane layouts.
//
// Existing concrete Update/View methods on views are preserved as-is.
// UpdatePane wraps the concrete Update to return only a tea.Cmd, since
// views use pointer receivers and mutate in place.
type Pane interface {
	// UpdatePane handles a Bubble Tea message and returns a command.
	// Wraps the view's concrete Update method for polymorphic dispatch.
	UpdatePane(msg tea.Msg) tea.Cmd

	// View renders the pane's content as a string.
	View() string

	// SetDimensions sets the available width and height for this pane.
	// Called by the layout engine before each render pass.
	SetDimensions(width, height int)

	// Focusable returns true if this pane can receive keyboard focus.
	Focusable() bool

	// OnFocus is called when the pane gains keyboard focus.
	OnFocus()

	// OnBlur is called when the pane loses keyboard focus.
	OnBlur()

	// SetCompact enables compact mode (suppresses help text in multi-pane layout).
	SetCompact(compact bool)

	// HelpBindings returns the key bindings to display in the global help bar.
	HelpBindings() []key.Binding
}

// PaneID identifies a pane slot within a layout.
type PaneID int

const (
	PaneLeft PaneID = iota
	PaneTopRight
	PaneBottomRight
	PaneCenter
	PaneFull
)
