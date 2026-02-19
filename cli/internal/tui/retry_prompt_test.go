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
	"time"
)

func TestRetryPrompt_InitialState(t *testing.T) {
	var rp RetryPrompt

	if rp.IsActive() {
		t.Error("Expected new RetryPrompt to be inactive")
	}
	if rp.TimedOutQuery() != "" {
		t.Error("Expected empty timed out query")
	}
	if rp.AutoRetried() {
		t.Error("Expected AutoRetried to be false initially")
	}
}

func TestRetryPrompt_Show(t *testing.T) {
	var rp RetryPrompt
	rp.Show("SELECT 1")

	if !rp.IsActive() {
		t.Error("Expected prompt to be active after Show")
	}
	if rp.TimedOutQuery() != "SELECT 1" {
		t.Errorf("Expected timed out query 'SELECT 1', got %q", rp.TimedOutQuery())
	}
}

func TestRetryPrompt_HandleKeyMsg_WhenInactive(t *testing.T) {
	var rp RetryPrompt

	result, handled := rp.HandleKeyMsg("1")
	if handled {
		t.Error("Expected handled=false when prompt is inactive")
	}
	if result != nil {
		t.Error("Expected nil result when prompt is inactive")
	}
}

func TestRetryPrompt_HandleKeyMsg_Options(t *testing.T) {
	tests := []struct {
		key         string
		wantTimeout time.Duration
		wantSave    bool
	}{
		{"1", 60 * time.Second, true},
		{"2", 2 * time.Minute, true},
		{"3", 5 * time.Minute, true},
		{"4", 24 * time.Hour, false},
	}

	for _, tt := range tests {
		t.Run("key_"+tt.key, func(t *testing.T) {
			var rp RetryPrompt
			rp.Show("SELECT 1")

			result, handled := rp.HandleKeyMsg(tt.key)
			if !handled {
				t.Error("Expected handled=true")
			}
			if result == nil {
				t.Fatal("Expected non-nil result")
			}
			if result.Timeout != tt.wantTimeout {
				t.Errorf("Expected timeout %v, got %v", tt.wantTimeout, result.Timeout)
			}
			if result.Save != tt.wantSave {
				t.Errorf("Expected save=%v, got %v", tt.wantSave, result.Save)
			}
			if rp.IsActive() {
				t.Error("Expected prompt to be deactivated after selection")
			}
		})
	}
}

func TestRetryPrompt_HandleKeyMsg_Escape(t *testing.T) {
	var rp RetryPrompt
	rp.Show("SELECT 1")

	result, handled := rp.HandleKeyMsg("esc")
	if !handled {
		t.Error("Expected handled=true for esc")
	}
	if result != nil {
		t.Error("Expected nil result for esc (cancel)")
	}
	if rp.IsActive() {
		t.Error("Expected prompt to be deactivated after esc")
	}
	if rp.TimedOutQuery() != "" {
		t.Error("Expected timed out query to be cleared after esc")
	}
}

func TestRetryPrompt_HandleKeyMsg_OtherKeys(t *testing.T) {
	var rp RetryPrompt
	rp.Show("SELECT 1")

	result, handled := rp.HandleKeyMsg("x")
	if !handled {
		t.Error("Expected handled=true for unrecognized keys (swallowed)")
	}
	if result != nil {
		t.Error("Expected nil result for unrecognized key")
	}
	if !rp.IsActive() {
		t.Error("Expected prompt to remain active for unrecognized key")
	}
}

func TestRetryPrompt_AutoRetried(t *testing.T) {
	var rp RetryPrompt

	if rp.AutoRetried() {
		t.Error("Expected false initially")
	}

	rp.SetAutoRetried(true)
	if !rp.AutoRetried() {
		t.Error("Expected true after SetAutoRetried(true)")
	}

	rp.SetAutoRetried(false)
	if rp.AutoRetried() {
		t.Error("Expected false after SetAutoRetried(false)")
	}
}

func TestRetryPrompt_View(t *testing.T) {
	var rp RetryPrompt
	view := rp.View()

	if !strings.Contains(view, "timed out") {
		t.Error("Expected view to contain 'timed out'")
	}
	if !strings.Contains(view, "60 seconds") {
		t.Error("Expected view to contain '60 seconds'")
	}
	if !strings.Contains(view, "2 minutes") {
		t.Error("Expected view to contain '2 minutes'")
	}
	if !strings.Contains(view, "5 minutes") {
		t.Error("Expected view to contain '5 minutes'")
	}
	if !strings.Contains(view, "No limit") {
		t.Error("Expected view to contain 'No limit'")
	}
}
