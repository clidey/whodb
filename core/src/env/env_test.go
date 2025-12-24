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

package env

import "testing"

func TestIsMockDataGenerationAllowed(t *testing.T) {
	original := DisableMockDataGeneration
	t.Cleanup(func() {
		DisableMockDataGeneration = original
	})

	DisableMockDataGeneration = ""
	if !IsMockDataGenerationAllowed("users") {
		t.Fatalf("mock data generation should be allowed when unset")
	}

	DisableMockDataGeneration = "*"
	if IsMockDataGenerationAllowed("anything") {
		t.Fatalf("mock data generation should be disabled when wildcard is used")
	}

	DisableMockDataGeneration = "logs, metrics"
	if IsMockDataGenerationAllowed("logs") {
		t.Fatalf("logs should be disabled when listed")
	}
	if !IsMockDataGenerationAllowed("orders") {
		t.Fatalf("orders should remain enabled when not listed")
	}
}

func TestGetLogLevel(t *testing.T) {
	original := LogLevel
	t.Cleanup(func() {
		LogLevel = original
	})

	cases := []struct {
		name     string
		envValue string
		expected string
	}{
		{name: "info", envValue: "INFO", expected: "info"},
		{name: "warning", envValue: "warn", expected: "warning"},
		{name: "error", envValue: "Error", expected: "error"},
		{name: "default", envValue: "", expected: "info"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envValue != "" {
				t.Setenv("WHODB_LOG_LEVEL", tc.envValue)
			} else {
				t.Setenv("WHODB_LOG_LEVEL", "")
			}

			if level := getLogLevel(); level != tc.expected {
				t.Fatalf("getLogLevel(%s) = %s, expected %s", tc.envValue, level, tc.expected)
			}
		})
	}
}

func TestGetOllamaEndpointRespectsOverrides(t *testing.T) {
	originalHost := OllamaHost
	originalPort := OllamaPort
	t.Cleanup(func() {
		OllamaHost = originalHost
		OllamaPort = originalPort
	})

	OllamaHost = "ollama.example.com"
	OllamaPort = "9999"

	endpoint := GetOllamaEndpoint()
	if endpoint != "http://ollama.example.com:9999/api" {
		t.Fatalf("expected custom ollama endpoint to be honored, got %s", endpoint)
	}
}

func TestGetConfiguredChatProviders(t *testing.T) {
	originalOpenAI := OpenAIAPIKey
	originalOpenAIEndpoint := OpenAIEndpoint
	originalAnthropic := AnthropicAPIKey
	originalOpenAICompatKey := OpenAICompatibleAPIKey
	originalOpenAICompatEndpoint := OpenAICompatibleEndpoint
	originalCustomModels := CustomModels
	originalOllamaHost := OllamaHost
	originalOllamaPort := OllamaPort

	t.Cleanup(func() {
		OpenAIAPIKey = originalOpenAI
		OpenAIEndpoint = originalOpenAIEndpoint
		AnthropicAPIKey = originalAnthropic
		OpenAICompatibleAPIKey = originalOpenAICompatKey
		OpenAICompatibleEndpoint = originalOpenAICompatEndpoint
		CustomModels = originalCustomModels
		OllamaHost = originalOllamaHost
		OllamaPort = originalOllamaPort
	})

	OpenAIAPIKey = "openai-key"
	OpenAIEndpoint = "https://custom.openai/api"
	AnthropicAPIKey = "anthropic-key"
	OpenAICompatibleAPIKey = "compat-key"
	OpenAICompatibleEndpoint = "https://compat.example.com"
	CustomModels = []string{"mixtral"}
	OllamaHost = "ollama.local"
	OllamaPort = "1234"

	providers := GetConfiguredChatProviders()
	if len(providers) != 4 {
		t.Fatalf("expected four providers (ChatGPT, Anthropic, OpenAI-Compatible, Ollama), got %d", len(providers))
	}

	if providers[0].Type != "ChatGPT" || providers[0].Endpoint != OpenAIEndpoint {
		t.Fatalf("expected ChatGPT provider to use custom endpoint, got %+v", providers[0])
	}
	if providers[1].Type != "Anthropic" {
		t.Fatalf("expected Anthropic provider present, got %+v", providers[1])
	}
	if providers[2].Type != "OpenAI-Compatible" || providers[2].Endpoint != OpenAICompatibleEndpoint {
		t.Fatalf("expected OpenAI-Compatible provider to use configured endpoint, got %+v", providers[2])
	}
	if providers[3].Type != "Ollama" || providers[3].Endpoint != "http://ollama.local:1234/api" {
		t.Fatalf("expected Ollama provider to use overridden host/port, got %+v", providers[3])
	}
}

func TestGetDefaultDatabaseCredentialsParsesEnv(t *testing.T) {
	t.Setenv("WHODB_POSTGRES", `[{"host":"db.local","user":"alice","password":"secret","database":"app"}]`)

	creds := GetDefaultDatabaseCredentials("postgres")
	if len(creds) != 1 {
		t.Fatalf("expected one credential parsed from env, got %d", len(creds))
	}

	if creds[0].Hostname != "db.local" || creds[0].Username != "alice" || creds[0].Database != "app" {
		t.Fatalf("unexpected credentials parsed: %+v", creds[0])
	}
}

func TestFindAllDatabaseCredentialsFallback(t *testing.T) {
	t.Setenv("WHODB_MYSQL", "")
	t.Setenv("WHODB_MYSQL_1", `{"host":"mysql.local","user":"bob","password":"pw","database":"northwind"}`)

	creds := GetDefaultDatabaseCredentials("mysql")
	if len(creds) != 1 {
		t.Fatalf("expected fallback credentials to be discovered, got %d", len(creds))
	}
	if creds[0].Hostname != "mysql.local" || creds[0].Username != "bob" {
		t.Fatalf("unexpected fallback credential: %+v", creds[0])
	}
}
