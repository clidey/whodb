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

func TestGetConfiguredChatProviders(t *testing.T) {
	originalOpenAI := OpenAIAPIKey
	originalOpenAIEndpoint := OpenAIEndpoint
	originalAnthropic := AnthropicAPIKey
	originalOpenAICompatKey := OpenAICompatibleAPIKey
	originalOpenAICompatEndpoint := OpenAICompatibleEndpoint
	originalCustomModels := CustomModels
	origHost, origPort := OllamaHost, OllamaPort

	t.Cleanup(func() {
		OpenAIAPIKey = originalOpenAI
		OpenAIEndpoint = originalOpenAIEndpoint
		AnthropicAPIKey = originalAnthropic
		OpenAICompatibleAPIKey = originalOpenAICompatKey
		OpenAICompatibleEndpoint = originalOpenAICompatEndpoint
		CustomModels = originalCustomModels
		OllamaHost, OllamaPort = origHost, origPort
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
		t.Fatalf("expected four providers (OpenAI, Anthropic, OpenAI-Compatible, Ollama), got %d", len(providers))
	}

	if providers[0].Type != "OpenAI" || providers[0].Endpoint != OpenAIEndpoint {
		t.Fatalf("expected OpenAI provider to use custom endpoint, got %+v", providers[0])
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
