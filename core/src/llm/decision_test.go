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
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/llm/providers"
	"github.com/clidey/whodb/core/src/source"
)

func TestDecisionModels(t *testing.T) {
	if got := DecisionModels(string(providers.OpenAI_LLMType)); !reflect.DeepEqual(got, []string{"gpt-6-luna"}) {
		t.Fatalf("expected gpt-6-luna for OpenAI, got %v", got)
	}
	for _, modelType := range []string{string(providers.Anthropic_LLMType), string(providers.Ollama_LLMType), "Unknown"} {
		if got := DecisionModels(modelType); got != nil {
			t.Fatalf("expected no decision models for %s, got %v", modelType, got)
		}
	}
}

func TestCreateDecisionRejectsProvidersWithoutDecisions(t *testing.T) {
	model := &source.ExternalModel{Type: string(providers.Anthropic_LLMType), Token: "key", Model: "claude-sonnet-5-5"}
	_, err := CreateDecision(context.Background(), model, "text", nil)
	if err == nil || !strings.Contains(err.Error(), "does not support decisions") {
		t.Fatalf("expected unsupported provider error, got %v", err)
	}
}

func TestDecisionsCanBeTurnedOff(t *testing.T) {
	original := env.DisableAIDecisions
	env.DisableAIDecisions = true
	t.Cleanup(func() { env.DisableAIDecisions = original })
	originalFactory := providers.HTTPClientFactory
	providers.HTTPClientFactory = func() *http.Client {
		t.Fatal("expected no request to the provider")
		return nil
	}
	t.Cleanup(func() { providers.HTTPClientFactory = originalFactory })

	if got := DecisionModels(string(providers.OpenAI_LLMType)); got != nil {
		t.Fatalf("expected no decision models, got %v", got)
	}
	model := &source.ExternalModel{Type: string(providers.OpenAI_LLMType), Token: "key", Model: "gpt-6-luna"}
	_, err := CreateDecision(context.Background(), model, "text", []providers.DecisionQuestion{{Type: "predicate", Name: "urgent", Instructions: "Is this urgent?"}})
	if err == nil || err.Error() != "decisions are turned off on this server" {
		t.Fatalf("expected the turned-off error, got %v", err)
	}
}
