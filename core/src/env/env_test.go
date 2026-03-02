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

func TestResolveProviderCredentials_PreservesModelType(t *testing.T) {
	// Save and restore global state
	origProviders := GenericProviders
	t.Cleanup(func() { GenericProviders = origProviders })

	// Register a generic provider with ClientType "openai-generic"
	GenericProviders = []GenericProviderConfig{
		{
			ProviderId: "lmstudio",
			Name:       "LM Studio",
			ClientType: "openai-generic",
			BaseURL:    "http://localhost:1234/v1",
			APIKey:     "lms-key",
		},
	}

	// Frontend sends modelType = provider ID ("lmstudio"), NOT the ClientType
	result := ResolveProviderCredentials("lmstudio", "", "", "lmstudio")

	// The regression: ModelType was being overridden to "openai-generic" (the ClientType),
	// which broke provider registry lookups keyed by ProviderId.
	// After the fix, ModelType stays as the frontend sent it.
	if result.ModelType != "lmstudio" {
		t.Fatalf("expected ModelType to stay 'lmstudio' (ProviderId), got %q — ClientType override regression", result.ModelType)
	}
	if result.Token != "lms-key" {
		t.Fatalf("expected Token to be filled from provider config, got %q", result.Token)
	}
	if result.Endpoint == "" {
		t.Fatalf("expected Endpoint to be filled from provider config, got empty string")
	}
}

func TestResolveProviderCredentials_RequestValuesOverrideConfig(t *testing.T) {
	origProviders := GenericProviders
	t.Cleanup(func() { GenericProviders = origProviders })

	GenericProviders = []GenericProviderConfig{
		{
			ProviderId: "test-provider",
			Name:       "Test",
			ClientType: "openai-generic",
			BaseURL:    "http://config-url",
			APIKey:     "config-key",
		},
	}

	// Request-level values take precedence
	result := ResolveProviderCredentials("test-provider", "request-key", "http://request-url", "test-provider")

	if result.Token != "request-key" {
		t.Fatalf("expected request-level token to take precedence, got %q", result.Token)
	}
	if result.Endpoint != "http://request-url" {
		t.Fatalf("expected request-level endpoint to take precedence, got %q", result.Endpoint)
	}
}

func TestGetConfiguredChatProviders(t *testing.T) {
	originalOpenAI := OpenAIAPIKey
	originalOpenAIEndpoint := OpenAIEndpoint
	originalAnthropic := AnthropicAPIKey
	origHost, origPort := OllamaHost, OllamaPort

	t.Cleanup(func() {
		OpenAIAPIKey = originalOpenAI
		OpenAIEndpoint = originalOpenAIEndpoint
		AnthropicAPIKey = originalAnthropic
		OllamaHost, OllamaPort = origHost, origPort
	})

	OpenAIAPIKey = "openai-key"
	OpenAIEndpoint = "https://custom.openai/api"
	AnthropicAPIKey = "anthropic-key"
	OllamaHost = "ollama.local"
	OllamaPort = "1234"

	providers := GetConfiguredChatProviders()
	if len(providers) != 3 {
		t.Fatalf("expected three providers (OpenAI, Anthropic, Ollama), got %d", len(providers))
	}

	if providers[0].Type != "OpenAI" || providers[0].Endpoint != OpenAIEndpoint {
		t.Fatalf("expected OpenAI provider to use custom endpoint, got %+v", providers[0])
	}
	if providers[1].Type != "Anthropic" {
		t.Fatalf("expected Anthropic provider present, got %+v", providers[1])
	}
	if providers[2].Type != "Ollama" || providers[2].Endpoint != "http://ollama.local:1234/api" {
		t.Fatalf("expected Ollama provider to use overridden host/port, got %+v", providers[2])
	}
}
