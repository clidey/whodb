package llm

import (
	"io"
	"strings"
	"testing"

	"github.com/clidey/whodb/core/src/env"
)

func TestParseChatGPTStreamingResponseAggregatesAndStreams(t *testing.T) {
	stream := make(chan string, 4)
	body := strings.NewReader(`{"choices":[{"delta":{"content":"Hello"}}],"finish_reason":""}
{"choices":[{"delta":{"content":"!"}}],"finish_reason":"stop"}
`)
	builder := &strings.Builder{}

	resp, err := parseChatGPTResponse(io.NopCloser(body), &stream, builder)
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

func TestParseChatGPTModelsResponseFiltersGPT(t *testing.T) {
	body := strings.NewReader(`{"data":[{"id":"gpt-4"},{"id":"text-embedding"}]}`)
	models, err := parseChatGPTModelsResponse(io.NopCloser(body))
	if err != nil {
		t.Fatalf("unexpected error parsing models: %v", err)
	}
	if len(models) != 1 || models[0] != "gpt-4" {
		t.Fatalf("expected only gpt-* models, got %v", models)
	}
}

func TestParseOllamaResponseAggregatesChunks(t *testing.T) {
	stream := make(chan string, 4)
	body := strings.NewReader(`{"response":"Hi","done":false}
{"response":" there","done":true}
`)
	builder := &strings.Builder{}

	resp, err := parseOllamaResponse(io.NopCloser(body), &stream, builder)
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

func TestParseAnthropicResponseAggregatesUntilEndTurn(t *testing.T) {
	stream := make(chan string, 4)
	body := strings.NewReader(`{"content":[{"text":"Hello","type":"text"}],"stop_reason":"end_turn"}
`)
	builder := &strings.Builder{}

	resp, err := parseAnthropicResponse(io.NopCloser(body), &stream, builder)
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

func TestParseOllamaModelsResponseParsesTags(t *testing.T) {
	body := strings.NewReader(`{"models":[{"model":"phi"},{"model":"gemma"}]}`)
	models, err := parseOllamaModelsResponse(io.NopCloser(body))
	if err != nil {
		t.Fatalf("unexpected error parsing ollama models: %v", err)
	}
	if len(models) != 2 || models[0] != "phi" || models[1] != "gemma" {
		t.Fatalf("expected models parsed from tags, got %v", models)
	}
}

func TestPrepareChatGPTRequestUsesCompatibleEndpoint(t *testing.T) {
	original := env.OpenAICompatibleEndpoint
	t.Cleanup(func() { env.OpenAICompatibleEndpoint = original })
	env.OpenAICompatibleEndpoint = "http://compat.local"

	client := &LLMClient{Type: OpenAICompatible_LLMType, APIKey: "token"}
	url, _, _, err := prepareChatGPTRequest(client, "hello", "compat-model", nil, true)
	if err != nil {
		t.Fatalf("unexpected error preparing request: %v", err)
	}
	if !strings.HasPrefix(url, env.OpenAICompatibleEndpoint) {
		t.Fatalf("expected compatible endpoint to be used, got %s", url)
	}
}

func TestParseChatGPTNonStreamingErrorsOnMissingChoices(t *testing.T) {
	body := strings.NewReader(`{"choices":[]}`)
	builder := &strings.Builder{}
	if resp, err := parseChatGPTResponse(io.NopCloser(body), nil, builder); err == nil || resp != nil {
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
