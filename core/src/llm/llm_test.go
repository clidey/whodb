package llm

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
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

func TestNormalizeLLMTypeHandlesChatGPT(t *testing.T) {
	result := NormalizeLLMType("ChatGPT")
	if result != OpenAI_LLMType {
		t.Fatalf("expected ChatGPT to normalize to OpenAI, got %s", result)
	}

	result = NormalizeLLMType("OpenAI")
	if result != OpenAI_LLMType {
		t.Fatalf("expected OpenAI to stay as OpenAI, got %s", result)
	}
}
