package llm

import (
	"testing"

	"github.com/clidey/whodb/core/src/env"
)

func TestResolveModelLookupUsesConfiguredGenericModels(t *testing.T) {
	originalGenericProviders := env.GenericProviders
	t.Cleanup(func() { env.GenericProviders = originalGenericProviders })
	env.GenericProviders = []env.GenericProviderConfig{{
		ProviderId: "test-generic",
		Models:     []string{"model-a", "model-b"},
	}}

	model, models := resolveModelLookup(ModelLookupOptions{
		ProviderID: "test-generic",
		ModelType:  "test-generic",
		ConfiguredProviders: []env.ChatProvider{{
			ProviderId: "test-generic",
			APIKey:     "configured-token",
			IsGeneric:  true,
		}},
	})

	if model.Token != "configured-token" {
		t.Fatalf("expected configured token, got %q", model.Token)
	}
	if len(models) != 2 || models[0] != "model-a" || models[1] != "model-b" {
		t.Fatalf("expected configured models, got %#v", models)
	}
}

func TestResolveModelLookupPrefersConfiguredProviderToken(t *testing.T) {
	model, models := resolveModelLookup(ModelLookupOptions{
		ProviderID: "openai-1",
		ModelType:  "OpenAI",
		Token:      "user-token",
		ConfiguredProviders: []env.ChatProvider{{
			ProviderId: "openai-1",
			APIKey:     "configured-token",
		}},
	})

	if model.Token != "configured-token" {
		t.Fatalf("expected configured token, got %q", model.Token)
	}
	if models != nil {
		t.Fatalf("expected dynamic model lookup, got %#v", models)
	}
}

func TestResolveModelLookupTokenFallbackIsOptional(t *testing.T) {
	withFallback, _ := resolveModelLookup(ModelLookupOptions{
		ProviderID:         "unknown-provider",
		ModelType:          "OpenAI",
		Token:              "user-token",
		AllowTokenFallback: true,
	})
	if withFallback.Token != "user-token" {
		t.Fatalf("expected token fallback, got %q", withFallback.Token)
	}

	withoutFallback, _ := resolveModelLookup(ModelLookupOptions{
		ProviderID: "unknown-provider",
		ModelType:  "OpenAI",
		Token:      "user-token",
	})
	if withoutFallback.Token != "" {
		t.Fatalf("expected no token fallback, got %q", withoutFallback.Token)
	}
}
