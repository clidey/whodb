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

package config

import "testing"

func TestSetOnlyPlatformHostReplacesExistingHosts(t *testing.T) {
	cfg := &Config{
		CLISection: CLISection{
			Platform: PlatformConfig{
				DefaultHost: "https://old.whodb.com",
				Hosts: []PlatformHost{
					{URL: "https://old.whodb.com", AccountID: "old-user"},
					{URL: "https://stale.whodb.com", AccountID: "stale-user"},
				},
			},
		},
	}

	cfg.SetOnlyPlatformHost(PlatformHost{
		URL:       "https://app.whodb.com",
		AccountID: "new-user",
		Email:     "new@example.com",
	})

	if cfg.Platform.DefaultHost != "https://app.whodb.com" {
		t.Fatalf("DefaultHost = %q, want app host", cfg.Platform.DefaultHost)
	}
	if len(cfg.Platform.Hosts) != 1 {
		t.Fatalf("len(Hosts) = %d, want 1", len(cfg.Platform.Hosts))
	}
	if cfg.Platform.Hosts[0].AccountID != "new-user" {
		t.Fatalf("AccountID = %q, want new user", cfg.Platform.Hosts[0].AccountID)
	}
}
