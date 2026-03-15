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
)

func TestInstanceReturnsCorrectClient(t *testing.T) {
	cfg := &engine.PluginConfig{ExternalModel: &engine.ExternalModel{Type: string(OpenAI_LLMType), Token: "key1"}}
	client := Instance(cfg)
	if client.Type != OpenAI_LLMType {
		t.Fatalf("expected type %s, got %s", OpenAI_LLMType, client.Type)
	}
	if client.APIKey != "key1" {
		t.Fatalf("expected API key 'key1', got %s", client.APIKey)
	}

	// Different config produces a client with the new values
	cfg2 := &engine.PluginConfig{ExternalModel: &engine.ExternalModel{Type: string(Ollama_LLMType), Token: "key2"}}
	client2 := Instance(cfg2)
	if client2.Type != Ollama_LLMType {
		t.Fatalf("expected type %s, got %s", Ollama_LLMType, client2.Type)
	}
	if client2.APIKey != "key2" {
		t.Fatalf("expected API key 'key2', got %s", client2.APIKey)
	}
}

func TestGetSupportedModelsReturnsErrorForUnsupportedType(t *testing.T) {
	client := LLMClient{Type: LLMType("Unknown")}
	if _, err := client.GetSupportedModels(); err == nil {
		t.Fatalf("expected error for unsupported type")
	}
}
