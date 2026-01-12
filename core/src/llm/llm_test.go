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

package llm

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/llm/providers"
)

func TestInstanceCachesPerTypeAndUpdatesAPIKey(t *testing.T) {
	llmInstance = nil
	t.Cleanup(func() { llmInstance = nil })

	cfg := &engine.PluginConfig{ExternalModel: &engine.ExternalModel{Type: string(OpenAI_LLMType), Token: "key1"}}
	first := Instance(cfg)
	if first.APIKey != "key1" {
		t.Fatalf("expected API key to be set on first instance")
	}

	// Same type: should reuse instance and update key
	cfg.ExternalModel.Token = "key2"
	second := Instance(cfg)
	if first != second {
		t.Fatalf("expected same instance for same type")
	}
	if second.APIKey != "key2" {
		t.Fatalf("expected API key to be updated on reuse, got %s", second.APIKey)
	}

	// Different type: new instance
	other := Instance(&engine.PluginConfig{ExternalModel: &engine.ExternalModel{Type: string(Ollama_LLMType), Token: ""}})
	if other == first {
		t.Fatalf("expected different instance for different LLM type")
	}
}

func TestProviderValidateConfigRequirements(t *testing.T) {
	cases := []struct {
		name     string
		provider providers.AIProvider
		config   *providers.ProviderConfig
		wantErr  bool
	}{
		{
			name:     "openai missing key",
			provider: providers.NewOpenAIProvider(),
			config:   &providers.ProviderConfig{Type: providers.OpenAI_LLMType, APIKey: ""},
			wantErr:  true,
		},
		{
			name:     "anthropic missing key",
			provider: providers.NewAnthropicProvider(),
			config:   &providers.ProviderConfig{Type: providers.Anthropic_LLMType, APIKey: ""},
			wantErr:  true,
		},
		{
			name:     "ollama no key needed",
			provider: providers.NewOllamaProvider(),
			config:   &providers.ProviderConfig{Type: providers.Ollama_LLMType, APIKey: ""},
			wantErr:  false,
		},
		{
			name:     "generic provider requires endpoint",
			provider: providers.NewGenericProvider("test", "Test", []string{}, ""),
			config:   &providers.ProviderConfig{Type: "test", APIKey: "", Endpoint: ""},
			wantErr:  true,
		},
		{
			name:     "generic provider with endpoint valid",
			provider: providers.NewGenericProvider("test", "Test", []string{}, ""),
			config:   &providers.ProviderConfig{Type: "test", APIKey: "", Endpoint: "http://localhost"},
			wantErr:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.provider.ValidateConfig(tc.config)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error but got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestCompleteReturnsErrorForUnsupportedType(t *testing.T) {
	client := LLMClient{Type: LLMType("Unknown")}
	if _, err := client.Complete("hi", "model", nil); err == nil {
		t.Fatalf("expected error for unsupported type")
	}
}

func TestGetSupportedModelsReturnsErrorForUnsupportedType(t *testing.T) {
	client := LLMClient{Type: LLMType("Unknown")}
	if _, err := client.GetSupportedModels(); err == nil {
		t.Fatalf("expected error for unsupported type")
	}
}

func TestNormalizeLLMTypeHandlesDeprecatedChatGPT(t *testing.T) {
	// Test that "ChatGPT" is normalized to "OpenAI"
	normalized := NormalizeLLMType("ChatGPT")
	if normalized != OpenAI_LLMType {
		t.Fatalf("expected 'ChatGPT' to normalize to 'OpenAI', got %s", normalized)
	}

	// Test that other types pass through unchanged
	normalized = NormalizeLLMType("Ollama")
	if normalized != Ollama_LLMType {
		t.Fatalf("expected 'Ollama' to pass through unchanged, got %s", normalized)
	}
}
