// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package src

import (
	"encoding/json"
	"testing"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/types"
)

func TestInitializeEngineCollectsRegisteredPlugins(t *testing.T) {
	// Register test plugins via the global registry (simulates what init() does in each plugin package)
	engine.RegisterPlugin(&engine.Plugin{Type: "TestDB1"})
	engine.RegisterPlugin(&engine.Plugin{Type: "TestDB2"})

	t.Cleanup(func() {
		MainEngine = nil
	})

	eng := InitializeEngine()

	if eng.Choose("TestDB1") == nil {
		t.Fatalf("expected TestDB1 to be registered via global registry")
	}
	if eng.Choose("TestDB2") == nil {
		t.Fatalf("expected TestDB2 to be registered via global registry")
	}
}

func TestGetLoginProfilesMergesSources(t *testing.T) {
	t.Cleanup(func() {
		MainEngine = nil
	})

	MainEngine = &engine.Engine{}
	MainEngine.RegistryPlugin(&engine.Plugin{Type: "Test"})

	MainEngine.AddLoginProfile(types.DatabaseCredentials{
		Alias:     "saved-profile",
		Hostname:  "host1",
		Username:  "alice",
		Password:  "pw",
		Database:  "db1",
		IsProfile: true,
		Type:      "Test",
	})

	MainEngine.RegisterProfileRetriever(func() ([]types.DatabaseCredentials, error) {
		return []types.DatabaseCredentials{{
			Hostname: "host2",
			Username: "retrieved",
			Database: "db2",
			Type:     "Test",
		}}, nil
	})

	envCreds := []types.DatabaseCredentials{{
		Hostname: "env-host",
		Username: "env-user",
		Database: "env-db",
		Password: "env-pw",
	}}
	envValue, err := json.Marshal(envCreds)
	if err != nil {
		t.Fatalf("failed to marshal env credentials: %v", err)
	}
	t.Setenv("WHODB_TEST", string(envValue))

	profiles := GetLoginProfiles()
	if len(profiles) != 3 {
		t.Fatalf("expected 3 profiles (stored + retriever + env), got %d", len(profiles))
	}

	foundEnvProfile := false
	for _, profile := range profiles {
		if profile.Hostname == "env-host" && profile.IsProfile {
			foundEnvProfile = true
		}
	}
	if !foundEnvProfile {
		t.Fatalf("expected env-provided profile to be marked as profile and returned")
	}
}

func TestGetLoginProfileIdPrioritizesFields(t *testing.T) {
	profile := types.DatabaseCredentials{
		CustomId: "custom-id",
		Alias:    "alias-id",
		Username: "user",
		Hostname: "host",
		Database: "db",
	}
	if got := GetLoginProfileId(0, profile); got != "custom-id" {
		t.Fatalf("expected custom id to take priority, got %s", got)
	}

	profile.CustomId = ""
	if got := GetLoginProfileId(1, profile); got != "alias-id" {
		t.Fatalf("expected alias to be used when custom id is empty, got %s", got)
	}

	profile.Alias = ""
	if got := GetLoginProfileId(2, profile); got == "" {
		t.Fatalf("expected fallback id to be generated when no custom id or alias is present")
	}
}
