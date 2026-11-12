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

package src

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/types"
)

func TestGetSourceProfilesReportsRequiresPassword(t *testing.T) {
	t.Cleanup(func() {
		MainEngine = nil
	})
	MainEngine = &engine.Engine{}

	t.Setenv("WHODB_POSTGRES", `[{"alias":"pg-no-password","host":"db.internal","user":"reader","database":"app"},{"alias":"pg-with-password","host":"db.internal","user":"reader","password":"secret","database":"app"}]`)
	t.Setenv("WHODB_SQLITE3", `[{"alias":"sqlite-file","database":"/tmp/app.db"}]`)

	profiles := GetSourceProfiles()
	byAlias := map[string]source.Profile{}
	for _, profile := range profiles {
		byAlias[profile.DisplayName] = profile
	}

	expected := map[string]bool{
		"pg-no-password":   true,
		"pg-with-password": false,
		"sqlite-file":      false,
	}
	for alias, requiresPassword := range expected {
		profile, ok := byAlias[alias]
		if !ok {
			t.Fatalf("expected profile %q to be listed, got %#v", alias, profiles)
		}
		if profile.RequiresPassword != requiresPassword {
			t.Fatalf("expected %q RequiresPassword=%v, got %v", alias, requiresPassword, profile.RequiresPassword)
		}
	}

	found, _, ok := FindSourceProfile("pg-no-password")
	if !ok || !found.RequiresPassword {
		t.Fatalf("expected FindSourceProfile to report RequiresPassword, got %#v ok=%v", found, ok)
	}
}

func TestProfileRequiresPasswordIgnoresUnknownSourceTypes(t *testing.T) {
	if profileRequiresPassword(types.DatabaseCredentials{Type: "Unknown"}) {
		t.Fatal("expected unknown source types to never require a password")
	}
}
