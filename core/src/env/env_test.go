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

package env

import "testing"

func TestGetOllamaEndpointRespectsOverrides(t *testing.T) {
	origHost, origPort := OllamaHost, OllamaPort
	t.Cleanup(func() { OllamaHost, OllamaPort = origHost, origPort })

	OllamaHost = "ollama.example.com"
	OllamaPort = "9999"

	endpoint := GetOllamaEndpoint()
	if endpoint != "http://ollama.example.com:9999/api" {
		t.Fatalf("expected custom ollama endpoint to be honored, got %s", endpoint)
	}
}

func TestGetSealosBootstrapEnabledDefaultsToTrue(t *testing.T) {
	t.Run("default enabled", func(t *testing.T) {
		t.Setenv("WHODB_SEALOS_BOOTSTRAP_ENABLED", "")
		if !GetSealosBootstrapEnabled() {
			t.Fatalf("expected sealos bootstrap to default to enabled")
		}
	})

	t.Run("explicit false disables", func(t *testing.T) {
		t.Setenv("WHODB_SEALOS_BOOTSTRAP_ENABLED", "false")
		if GetSealosBootstrapEnabled() {
			t.Fatalf("expected explicit false to disable sealos bootstrap")
		}
	})

	t.Run("explicit true enables", func(t *testing.T) {
		t.Setenv("WHODB_SEALOS_BOOTSTRAP_ENABLED", "true")
		if !GetSealosBootstrapEnabled() {
			t.Fatalf("expected explicit true to enable sealos bootstrap")
		}
	})
}

func TestGetStandaloneLoginEnabledDefaultsToTrue(t *testing.T) {
	t.Run("default enabled", func(t *testing.T) {
		t.Setenv("WHODB_STANDALONE_LOGIN_ENABLED", "")
		if !GetStandaloneLoginEnabled() {
			t.Fatalf("expected standalone login to default to enabled")
		}
	})

	t.Run("explicit false disables", func(t *testing.T) {
		t.Setenv("WHODB_STANDALONE_LOGIN_ENABLED", "false")
		if GetStandaloneLoginEnabled() {
			t.Fatalf("expected explicit false to disable standalone login")
		}
	})
}
