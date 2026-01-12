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
	"io"
	"strings"
	"testing"

	"github.com/clidey/whodb/core/src/llm/providers"
)

func TestOpenAIStreamingResponseAggregatesAndStreams(t *testing.T) {
	stream := make(chan string, 4)
	body := strings.NewReader(`{"choices":[{"delta":{"content":"Hello"}}],"finish_reason":""}
{"choices":[{"delta":{"content":"!"}}],"finish_reason":"stop"}
`)

	provider := providers.NewOpenAIProvider()
	resp, err := provider.ParseResponse(io.NopCloser(body), &stream)
	if err != nil {
		t.Fatalf("unexpected error parsing streaming response: %v", err)
	}
	if resp == nil || *resp != "Hello!" {
		t.Fatalf("expected aggregated content 'Hello!', got %v", resp)
	}
	if strings.Join(drain(stream), "") != "Hello!" {
		t.Fatalf("expected streamed chunks to match response")
	}
}

func TestOllamaResponseAggregatesChunks(t *testing.T) {
	stream := make(chan string, 4)
	body := strings.NewReader(`{"response":"Hi","done":false}
{"response":" there","done":true}
`)

	provider := providers.NewOllamaProvider()
	resp, err := provider.ParseResponse(io.NopCloser(body), &stream)
	if err != nil {
		t.Fatalf("unexpected error parsing ollama response: %v", err)
	}
	if resp == nil || *resp != "Hi there" {
		t.Fatalf("expected aggregated response 'Hi there', got %v", resp)
	}
	if strings.Join(drain(stream), "") != "Hi there" {
		t.Fatalf("expected streamed chunks to match aggregated response")
	}
}

func TestAnthropicResponseAggregatesUntilEndTurn(t *testing.T) {
	stream := make(chan string, 4)
	body := strings.NewReader(`{"content":[{"text":"Hello","type":"text"}],"stop_reason":"end_turn"}
`)

	provider := providers.NewAnthropicProvider()
	resp, err := provider.ParseResponse(io.NopCloser(body), &stream)
	if err != nil {
		t.Fatalf("unexpected error parsing anthropic response: %v", err)
	}
	if resp == nil || *resp != "Hello" {
		t.Fatalf("expected aggregated response 'Hello', got %v", resp)
	}
	if strings.Join(drain(stream), "") != "Hello" {
		t.Fatalf("expected streamed anthropic text to match response")
	}
}

func TestGenericProviderUsesConfiguredEndpoint(t *testing.T) {
	// Test that GenericProvider validates endpoint is required
	provider := providers.NewGenericProvider("test-provider", "Test", []string{"model1"}, "openai-generic")

	config := &providers.ProviderConfig{
		Type:     provider.GetType(),
		APIKey:   "token",
		Endpoint: "", // Empty endpoint should fail validation
	}

	err := provider.ValidateConfig(config)
	if err == nil {
		t.Fatalf("expected error when endpoint is empty for generic provider")
	}

	// With endpoint configured, validation should pass
	config.Endpoint = "http://test.local"
	err = provider.ValidateConfig(config)
	if err != nil {
		t.Fatalf("expected no error when endpoint is configured, got %v", err)
	}
}

func TestOpenAINonStreamingErrorsOnMissingChoices(t *testing.T) {
	body := strings.NewReader(`{"choices":[]}`)

	provider := providers.NewOpenAIProvider()
	resp, err := provider.ParseResponse(io.NopCloser(body), nil)
	if err == nil || resp != nil {
		t.Fatalf("expected error when no choices returned, got resp=%v err=%v", resp, err)
	}
}

func drain(ch chan string) []string {
	var out []string
	for {
		select {
		case v := <-ch:
			out = append(out, v)
		default:
			return out
		}
	}
}
