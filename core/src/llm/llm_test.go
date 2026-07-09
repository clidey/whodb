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

	"github.com/clidey/whodb/core/src/llm/providers"
	"github.com/clidey/whodb/core/src/source"
)

func TestInstanceReturnsCorrectClient(t *testing.T) {
	client := ClientForModel(&source.ExternalModel{Type: string(providers.OpenAI_LLMType), Token: "key1"})
	if client.Type != providers.OpenAI_LLMType {
		t.Fatalf("expected type %s, got %s", providers.OpenAI_LLMType, client.Type)
	}
	if client.APIKey != "key1" {
		t.Fatalf("expected API key 'key1', got %s", client.APIKey)
	}

	// Different config produces a client with the new values
	client2 := ClientForModel(&source.ExternalModel{Type: string(providers.Ollama_LLMType), Token: "key2"})
	if client2.Type != providers.Ollama_LLMType {
		t.Fatalf("expected type %s, got %s", providers.Ollama_LLMType, client2.Type)
	}
	if client2.APIKey != "key2" {
		t.Fatalf("expected API key 'key2', got %s", client2.APIKey)
	}

	client3 := ClientForModel(&source.ExternalModel{Type: string(providers.Gemini_LLMType), Token: "key3"})
	if client3.Type != providers.Gemini_LLMType {
		t.Fatalf("expected type %s, got %s", providers.Gemini_LLMType, client3.Type)
	}
	if client3.APIKey != "key3" {
		t.Fatalf("expected API key 'key3', got %s", client3.APIKey)
	}
}

func TestGetSupportedModelsReturnsErrorForUnsupportedType(t *testing.T) {
	client := LLMClient{Type: providers.LLMType("Unknown")}
	if _, err := client.GetSupportedModels(); err == nil {
		t.Fatalf("expected error for unsupported type")
	}
}
