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

package envconfig

import (
	"testing"

	"github.com/clidey/whodb/core/src/env"
)

func TestResolveProviderCredentials_PreservesModelType(t *testing.T) {
	origProviders := env.GenericProviders
	t.Cleanup(func() { env.GenericProviders = origProviders })

	env.GenericProviders = []env.GenericProviderConfig{
		{
			ProviderId: "lmstudio",
			Name:       "LM Studio",
			ClientType: "openai-generic",
			BaseURL:    "http://localhost:1234/v1",
			APIKey:     "lms-key",
		},
	}

	result := ResolveProviderCredentials("lmstudio", "", "", "lmstudio")

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
	origProviders := env.GenericProviders
	t.Cleanup(func() { env.GenericProviders = origProviders })

	env.GenericProviders = []env.GenericProviderConfig{
		{
			ProviderId: "test-provider",
			Name:       "Test",
			ClientType: "openai-generic",
			BaseURL:    "http://config-url",
			APIKey:     "config-key",
		},
	}

	result := ResolveProviderCredentials("test-provider", "request-key", "http://request-url", "test-provider")

	if result.Token != "request-key" {
		t.Fatalf("expected request-level token to take precedence, got %q", result.Token)
	}
	if result.Endpoint != "http://request-url" {
		t.Fatalf("expected request-level endpoint to take precedence, got %q", result.Endpoint)
	}
}

func TestGetConfiguredChatProviders(t *testing.T) {
	originalOpenAI := env.OpenAIAPIKey
	originalOpenAIEndpoint := env.OpenAIEndpoint
	originalAnthropic := env.AnthropicAPIKey
	origHost, origPort := env.OllamaHost, env.OllamaPort

	t.Cleanup(func() {
		env.OpenAIAPIKey = originalOpenAI
		env.OpenAIEndpoint = originalOpenAIEndpoint
		env.AnthropicAPIKey = originalAnthropic
		env.OllamaHost, env.OllamaPort = origHost, origPort
	})

	env.OpenAIAPIKey = "openai-key"
	env.OpenAIEndpoint = "https://custom.openai/api"
	env.AnthropicAPIKey = "anthropic-key"
	env.OllamaHost = "ollama.local"
	env.OllamaPort = "1234"

	providers := GetConfiguredChatProviders()
	if len(providers) != 3 {
		t.Fatalf("expected three providers (OpenAI, Anthropic, Ollama), got %d", len(providers))
	}

	if providers[0].Type != "OpenAI" || providers[0].Endpoint != env.OpenAIEndpoint {
		t.Fatalf("expected OpenAI provider to use custom endpoint, got %+v", providers[0])
	}
	if providers[1].Type != "Anthropic" {
		t.Fatalf("expected Anthropic provider present, got %+v", providers[1])
	}
	if providers[2].Type != "Ollama" || providers[2].Endpoint != "http://ollama.local:1234/api" {
		t.Fatalf("expected Ollama provider to use overridden host/port, got %+v", providers[2])
	}
}
