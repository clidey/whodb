package llm

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

func TestInstanceCachesPerTypeAndUpdatesAPIKey(t *testing.T) {
	llmInstance = nil
	t.Cleanup(func() { llmInstance = nil })

	cfg := &engine.PluginConfig{ExternalModel: &engine.ExternalModel{Type: string(ChatGPT_LLMType), Token: "key1"}}
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

func TestValidateAPIKeyRequirements(t *testing.T) {
	cases := []struct {
		name    string
		client  LLMClient
		wantErr bool
	}{
		{name: "chatgpt missing key", client: LLMClient{Type: ChatGPT_LLMType, APIKey: ""}, wantErr: true},
		{name: "anthropic missing key", client: LLMClient{Type: Anthropic_LLMType, APIKey: ""}, wantErr: true},
		{name: "ollama no key needed", client: LLMClient{Type: Ollama_LLMType, APIKey: ""}, wantErr: false},
		{name: "openai-compatible no key validation", client: LLMClient{Type: OpenAICompatible_LLMType, APIKey: ""}, wantErr: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.client.validateAPIKey()
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
