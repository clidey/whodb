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

package cmd

import (
	"io"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
)

func TestPlatformHostsWithLogin(t *testing.T) {
	cfg := &config.Config{
		CLISection: config.CLISection{
			Platform: config.PlatformConfig{
				Hosts: []config.PlatformHost{
					{URL: "https://app.whodb.com", AccountID: "user-1", Email: "a@example.com"},
					{URL: "https://stale.whodb.com"},
					{AccountID: "user-2"},
				},
			},
		},
	}

	hosts := platformHostsWithLogin(cfg)
	if len(hosts) != 1 {
		t.Fatalf("len(hosts) = %d, want 1", len(hosts))
	}
	if hosts[0].URL != "https://app.whodb.com" {
		t.Fatalf("host URL = %q, want app host", hosts[0].URL)
	}
}

func TestConfirmPlatformLoginReplacementSkipsPromptWhenApprovedByFlag(t *testing.T) {
	approved, err := confirmPlatformLoginReplacement(io.Discard, []config.PlatformHost{
		{URL: "https://app.whodb.com", AccountID: "user-1"},
	}, true)
	if err != nil {
		t.Fatalf("confirmPlatformLoginReplacement() error = %v", err)
	}
	if !approved {
		t.Fatal("confirmPlatformLoginReplacement() approved = false, want true")
	}
}

func TestIsAffirmativeConfirmation(t *testing.T) {
	tests := []struct {
		answer string
		want   bool
	}{
		{"y", true},
		{"Y", true},
		{" yes ", true},
		{"no", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.answer, func(t *testing.T) {
			if got := isAffirmativeConfirmation(tt.answer); got != tt.want {
				t.Fatalf("isAffirmativeConfirmation(%q) = %v, want %v", tt.answer, got, tt.want)
			}
		})
	}
}
