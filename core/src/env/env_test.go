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

package env

import (
	"fmt"
	"strings"
	"testing"
)

func TestGetBasePathNormalizesInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "root", input: "/", want: ""},
		{name: "prefix without leading slash", input: "whodb", want: "/whodb"},
		{name: "prefix with trailing slash", input: "/whodb/", want: "/whodb"},
		{name: "trim whitespace", input: " /nested/path/ ", want: "/nested/path"},
		{name: "allows safe punctuation", input: "/v1.2/api_gateway/", want: "/v1.2/api_gateway"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("WHODB_BASE_PATH", tt.input)
			if got := getBasePath(); got != tt.want {
				t.Fatalf("expected normalized base path %q, got %q", tt.want, got)
			}
		})
	}
}

func TestGetBasePathRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "double slash", input: "/whodb//admin"},
		{name: "dot segment", input: "/whodb/./admin"},
		{name: "dot segment", input: "/whodb/../admin"},
		{name: "space in segment", input: "/who db"},
		{name: "query delimiter", input: "/whodb?next=/"},
		{name: "fragment delimiter", input: "/whodb#frag"},
		{name: "quote", input: `/who"db`},
		{name: "angle bracket", input: "/who<db"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("WHODB_BASE_PATH", tt.input)

			defer func() {
				panicValue := recover()
				if panicValue == nil {
					t.Fatalf("expected invalid base path %q to panic", tt.input)
				}
				if !strings.Contains(fmt.Sprint(panicValue), "invalid WHODB_BASE_PATH") {
					t.Fatalf("expected panic to mention invalid WHODB_BASE_PATH, got %v", panicValue)
				}
			}()

			_ = getBasePath()
		})
	}
}

func TestGetOllamaEndpointRespectsOverrides(t *testing.T) {
	origHost, origPort := OllamaHost, OllamaPort
	t.Cleanup(func() { OllamaHost, OllamaPort = origHost, origPort })

	OllamaHost = "ollama.example.com"
	OllamaPort = "9999"

	endpoint := GetOllamaEndpoint()
	if endpoint != "http://ollama.example.com:9999/api" {
		t.Fatalf("expected custom ollama endpoint to be honored, got %s", endpoint)
	}
}

func TestGetGeminiEndpointRespectsOverrides(t *testing.T) {
	origEndpoint := GeminiEndpoint
	t.Cleanup(func() { GeminiEndpoint = origEndpoint })

	GeminiEndpoint = "https://gemini.example.com/v1beta/openai/"

	endpoint := GetGeminiEndpoint()
	if endpoint != GeminiEndpoint {
		t.Fatalf("expected custom gemini endpoint to be honored, got %s", endpoint)
	}
}
