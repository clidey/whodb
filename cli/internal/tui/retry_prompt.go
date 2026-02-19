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
	"time"

	"github.com/clidey/whodb/cli/pkg/styles"
)

// RetryPromptResult represents the user's timeout selection from the retry prompt.
type RetryPromptResult struct {
	Timeout time.Duration
	Save    bool // Whether to save this as the preferred timeout
}

// RetryPrompt is a shared component for handling timeout retry prompts.
// It displays options [1] 60s [2] 2min [3] 5min [4] No limit and handles key input.
type RetryPrompt struct {
	active        bool
	timedOutQuery string
	autoRetried   bool
}

// Show activates the retry prompt for the given query.
func (r *RetryPrompt) Show(query string) {
	r.active = true
	r.timedOutQuery = query
}

// IsActive returns whether the retry prompt is currently shown.
func (r *RetryPrompt) IsActive() bool {
	return r.active
}

// TimedOutQuery returns the query that timed out.
func (r *RetryPrompt) TimedOutQuery() string {
	return r.timedOutQuery
}

// AutoRetried returns whether an auto-retry has already been attempted.
func (r *RetryPrompt) AutoRetried() bool {
	return r.autoRetried
}

// SetAutoRetried marks that an auto-retry has been attempted.
func (r *RetryPrompt) SetAutoRetried(v bool) {
	r.autoRetried = v
}

// HandleKeyMsg processes a key press while the retry prompt is active.
// Returns (result, handled). If handled is true, the caller should act on the result.
// If result is non-nil, the caller should retry with the given timeout.
func (r *RetryPrompt) HandleKeyMsg(keyStr string) (*RetryPromptResult, bool) {
	if !r.active {
		return nil, false
	}

	switch keyStr {
	case "1":
		r.active = false
		return &RetryPromptResult{Timeout: 60 * time.Second, Save: true}, true
	case "2":
		r.active = false
		return &RetryPromptResult{Timeout: 2 * time.Minute, Save: true}, true
	case "3":
		r.active = false
		return &RetryPromptResult{Timeout: 5 * time.Minute, Save: true}, true
	case "4":
		r.active = false
		// No limit applies once but doesn't save
		return &RetryPromptResult{Timeout: 24 * time.Hour, Save: false}, true
	case "esc":
		r.active = false
		r.timedOutQuery = ""
		return nil, true
	}

	// Ignore other keys while in retry prompt
	return nil, true
}

// View renders the retry prompt UI.
func (r *RetryPrompt) View() string {
	var b string
	b += styles.ErrorStyle.Render("Request timed out")
	b += "\n\n"
	b += styles.MutedStyle.Render("Retry with longer timeout:")
	b += "\n"
	b += styles.KeyStyle.Render("[1]")
	b += styles.MutedStyle.Render(" 60 seconds  ")
	b += styles.KeyStyle.Render("[2]")
	b += styles.MutedStyle.Render(" 2 minutes  ")
	b += styles.KeyStyle.Render("[3]")
	b += styles.MutedStyle.Render(" 5 minutes  ")
	b += styles.KeyStyle.Render("[4]")
	b += styles.MutedStyle.Render(" No limit")
	b += "\n\n"
	b += styles.RenderHelp("esc", "cancel")
	return b
}
