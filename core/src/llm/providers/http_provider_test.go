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

package providers

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func withTestHTTPClient(t *testing.T, transport roundTripFunc) {
	t.Helper()

	originalFactory := httpClientFactory
	httpClientFactory = func() *http.Client {
		return &http.Client{Transport: transport}
	}
	t.Cleanup(func() {
		httpClientFactory = originalFactory
	})
}

func TestDefaultHTTPClientHasTimeout(t *testing.T) {
	client := httpClientFactory()
	if client.Timeout <= 0 {
		t.Fatalf("expected default provider HTTP client to have a timeout")
	}
}

func httpResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func clearResponsesProbeCache() {
	responsesAPICache.Range(func(key, _ any) bool {
		responsesAPICache.Delete(key)
		return true
	})
}

func TestOpenAIProviderGetSupportedModelsFiltersNonChatModels(t *testing.T) {
	provider := NewOpenAIProvider()
	withTestHTTPClient(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://openai.test/v1/models" {
			t.Fatalf("unexpected URL %s", r.URL.String())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("expected bearer auth header, got %q", got)
		}
		return httpResponse(http.StatusOK, `{"data":[
			{"id":"gpt-4o"},
			{"id":"custom-chat"},
			{"id":"tts-1"},
			{"id":"text-embedding-3-small"},
			{"id":"gpt-4o-realtime-preview"},
			{"id":"davinci-002"}
		]}`), nil
	})

	models, err := provider.GetSupportedModels(&ProviderConfig{
		APIKey:   "test-key",
		Endpoint: "https://openai.test/v1",
	})
	if err != nil {
		t.Fatalf("GetSupportedModels returned error: %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("expected two chat-capable models, got %#v", models)
	}
	if models[0] != "gpt-4o" || models[1] != "custom-chat" {
		t.Fatalf("unexpected filtered models: %#v", models)
	}
}

func TestOpenAIProviderGetSupportedModelsErrorsOnBadStatus(t *testing.T) {
	provider := NewOpenAIProvider()
	withTestHTTPClient(t, func(r *http.Request) (*http.Response, error) {
		return httpResponse(http.StatusBadGateway, "upstream unavailable\n"), nil
	})

	_, err := provider.GetSupportedModels(&ProviderConfig{
		APIKey:   "test-key",
		Endpoint: "https://openai.test/v1",
	})
	if err == nil || !strings.Contains(err.Error(), "upstream unavailable") {
		t.Fatalf("expected response body in error, got %v", err)
	}
}

func TestOpenAIProviderCreateBAMLClientCachesResponsesProbe(t *testing.T) {
	clearResponsesProbeCache()
	t.Cleanup(clearResponsesProbeCache)

	provider := NewOpenAIProvider()
	probeHits := 0
	withTestHTTPClient(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://responses.test/v1/responses" {
			t.Fatalf("unexpected probe URL %s", r.URL.String())
		}
		probeHits++
		return httpResponse(http.StatusBadRequest, `{"error":"missing input"}`), nil
	})

	config := &ProviderConfig{
		APIKey:   "test-key",
		Endpoint: "https://responses.test/v1",
	}

	firstType, firstOpts, err := provider.CreateBAMLClient(config, "gpt-4o")
	if err != nil {
		t.Fatalf("first CreateBAMLClient returned error: %v", err)
	}
	secondType, secondOpts, err := provider.CreateBAMLClient(config, "gpt-4o")
	if err != nil {
		t.Fatalf("second CreateBAMLClient returned error: %v", err)
	}

	if firstType != "openai-responses" || secondType != "openai-responses" {
		t.Fatalf("expected responses client type, got %q and %q", firstType, secondType)
	}
	if probeHits != 1 {
		t.Fatalf("expected one cached probe, got %d", probeHits)
	}
	if firstOpts["model"] != "gpt-4o" || secondOpts["api_key"] != "test-key" {
		t.Fatalf("unexpected BAML options: %#v %#v", firstOpts, secondOpts)
	}
}

func TestOpenAIProviderCreateBAMLClientFallsBackWhenResponsesMissing(t *testing.T) {
	clearResponsesProbeCache()
	t.Cleanup(clearResponsesProbeCache)

	provider := NewOpenAIProvider()
	probeHits := 0
	withTestHTTPClient(t, func(r *http.Request) (*http.Response, error) {
		probeHits++
		return httpResponse(http.StatusNotFound, "not found"), nil
	})

	clientType, opts, err := provider.CreateBAMLClient(&ProviderConfig{
		Endpoint: "https://fallback.test/v1",
	}, "gpt-4.1")
	if err != nil {
		t.Fatalf("CreateBAMLClient returned error: %v", err)
	}

	if clientType != "openai" {
		t.Fatalf("expected chat-completions fallback, got %q", clientType)
	}
	if probeHits != 1 {
		t.Fatalf("expected one probe, got %d", probeHits)
	}
	if opts["model"] != "gpt-4.1" {
		t.Fatalf("expected model option, got %#v", opts)
	}
}

func TestLMStudioProviderGetSupportedModelsUsesOpenAICompatEndpoint(t *testing.T) {
	provider := NewLMStudioProvider()
	withTestHTTPClient(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://lmstudio.test/v1/models" {
			t.Fatalf("unexpected URL %s", r.URL.String())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer lmstudio-key" {
			t.Fatalf("expected auth header, got %q", got)
		}
		return httpResponse(http.StatusOK, `{"data":[
			{"id":"local-model"},
			{"id":""},
			{"id":"custom/model"}
		]}`), nil
	})

	models, err := provider.GetSupportedModels(&ProviderConfig{
		Endpoint: "https://lmstudio.test/v1",
		APIKey:   "lmstudio-key",
	})
	if err != nil {
		t.Fatalf("GetSupportedModels returned error: %v", err)
	}

	if len(models) != 2 || models[0] != "local-model" || models[1] != "custom/model" {
		t.Fatalf("unexpected LM Studio models: %#v", models)
	}
}

func TestOllamaProviderModelsAndBAMLClient(t *testing.T) {
	provider := NewOllamaProvider()
	withTestHTTPClient(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://ollama.test/api/tags" {
			t.Fatalf("unexpected URL %s", r.URL.String())
		}
		return httpResponse(http.StatusOK, `{"models":[{"model":"llama3:latest"},{"model":"mistral:7b"}]}`), nil
	})

	config := &ProviderConfig{Endpoint: "https://ollama.test/api"}
	models, err := provider.GetSupportedModels(config)
	if err != nil {
		t.Fatalf("GetSupportedModels returned error: %v", err)
	}
	if len(models) != 2 || models[0] != "llama3:latest" || models[1] != "mistral:7b" {
		t.Fatalf("unexpected Ollama models: %#v", models)
	}

	clientType, opts, err := provider.CreateBAMLClient(config, "llama3:latest")
	if err != nil {
		t.Fatalf("CreateBAMLClient returned error: %v", err)
	}
	if clientType != "openai-generic" {
		t.Fatalf("expected openai-generic client type, got %q", clientType)
	}
	if opts["base_url"] != "https://ollama.test/v1" {
		t.Fatalf("expected trimmed OpenAI-compatible base URL, got %#v", opts["base_url"])
	}
}
