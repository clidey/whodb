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

package graph

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"

	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/llm/providers"
)

type decisionRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn decisionRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func TestGraphQLRunAIDecisionMutation(t *testing.T) {
	var sentBody []byte
	originalFactory := providers.HTTPClientFactory
	providers.HTTPClientFactory = func() *http.Client {
		return &http.Client{Transport: decisionRoundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "https://api.openai.com/v1/decisions" {
				t.Fatalf("unexpected decisions URL %s", r.URL.String())
			}
			sentBody, _ = io.ReadAll(r.Body)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{"model":"gpt-6-luna","answers":[
					{"type":"predicate","name":"refund_requested","probability":0.97},
					{"type":"choice","name":"department","choice":"billing","confidence":0.93,"probabilities":[
						{"value":"billing","probability":0.95},
						{"value":"other","probability":0.05}
					]},
					{"type":"score","name":"severity","score":0.2,"confidence":0.6,"probabilities":[
						{"value":0,"label":"Minor","probability":0.8},
						{"value":1,"label":"Blocking","probability":0.2}
					]},
					{"type":"refusal","name":"abusive_language"}
				]}`)),
			}, nil
		})}
	}
	t.Cleanup(func() { providers.HTTPClientFactory = originalFactory })

	query := `
	mutation RunAIDecision($input: AIDecisionInput!) {
		RunAIDecision(input: $input) {
			Answers {
				Type
				Name
				Probability
				Choice
				Score
				Confidence
				ChoiceProbabilities { Value Probability }
				ScoreProbabilities { Value Label Probability }
			}
		}
	}`
	variables := `{"input":{
		"ModelType":"OpenAI",
		"Token":"test-key",
		"Model":"gpt-6-luna",
		"Input":"I was charged twice for my order.",
		"Questions":[
			{"Type":"Predicate","Name":"refund_requested","Instructions":"Does the customer ask for a refund?"},
			{"Type":"Choice","Name":"department","Instructions":"Which department should handle this?","Choices":[
				{"Value":"billing","Description":"Payments, invoices, and refunds."},
				{"Value":"other"}
			]},
			{"Type":"Score","Name":"severity","Instructions":"How severe is this issue?","Levels":[
				{"Label":"Minor"},
				{"Label":"Blocking"}
			]},
			{"Type":"Predicate","Name":"abusive_language","Instructions":"Is the message abusive?"}
		]
	}}`
	w := serveGraphQL(t, query, variables)
	requireJSONEqual(t, "decision request", sentBody, `{
		"model":"gpt-6-luna",
		"input":"I was charged twice for my order.",
		"questions":[
			{"type":"predicate","name":"refund_requested","instructions":"Does the customer ask for a refund?"},
			{"type":"choice","name":"department","instructions":"Which department should handle this?","choices":[
				{"value":"billing","description":"Payments, invoices, and refunds."},
				{"value":"other"}
			]},
			{"type":"score","name":"severity","instructions":"How severe is this issue?","levels":[
				{"label":"Minor"},
				{"label":"Blocking"}
			]},
			{"type":"predicate","name":"abusive_language","instructions":"Is the message abusive?"}
		]
	}`)
	requireJSONEqual(t, "GraphQL response", w.Body.Bytes(), `{"data":{"RunAIDecision":{"Answers":[
		{"Type":"Predicate","Name":"refund_requested","Probability":0.97,"Choice":null,"Score":null,"Confidence":null,
			"ChoiceProbabilities":null,"ScoreProbabilities":null},
		{"Type":"Choice","Name":"department","Probability":null,"Choice":"billing","Score":null,"Confidence":0.93,
			"ChoiceProbabilities":[{"Value":"billing","Probability":0.95},{"Value":"other","Probability":0.05}],"ScoreProbabilities":null},
		{"Type":"Score","Name":"severity","Probability":null,"Choice":null,"Score":0.2,"Confidence":0.6,"ChoiceProbabilities":null,
			"ScoreProbabilities":[{"Value":0,"Label":"Minor","Probability":0.8},{"Value":1,"Label":"Blocking","Probability":0.2}]},
		{"Type":"Refusal","Name":"abusive_language","Probability":null,"Choice":null,"Score":null,"Confidence":null,
			"ChoiceProbabilities":null,"ScoreProbabilities":null}
	]}}}`)
}

func TestGraphQLRunAIDecisionUsesClientEndpointWithoutProvider(t *testing.T) {
	var requestedURL string
	originalFactory := providers.HTTPClientFactory
	providers.HTTPClientFactory = func() *http.Client {
		return &http.Client{Transport: decisionRoundTripFunc(func(r *http.Request) (*http.Response, error) {
			requestedURL = r.URL.String()
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"answers":[{"type":"predicate","name":"urgent","probability":0.4}]}`)),
			}, nil
		})}
	}
	t.Cleanup(func() { providers.HTTPClientFactory = originalFactory })

	w := serveGraphQL(t, `mutation RunAIDecision($input: AIDecisionInput!) { RunAIDecision(input: $input) { Answers { Name Probability } } }`, `{"input":{
		"ModelType":"OpenAI",
		"Token":"test-key",
		"Model":"gpt-6-luna",
		"Endpoint":"https://gateway.test/v1",
		"Input":"The site is down.",
		"Questions":[{"Type":"Predicate","Name":"urgent","Instructions":"Is this urgent?"}]
	}}`)
	if requestedURL != "https://gateway.test/v1/decisions" {
		t.Fatalf("expected the client endpoint to be used, got %q", requestedURL)
	}
	requireJSONEqual(t, "GraphQL response", w.Body.Bytes(), `{"data":{"RunAIDecision":{"Answers":[{"Name":"urgent","Probability":0.4}]}}}`)
}

func TestGraphQLAIDecisionModelsQuery(t *testing.T) {
	for modelType, want := range map[string]string{
		"OpenAI":    `{"data":{"AIDecisionModels":["gpt-6-luna"]}}`,
		"Anthropic": `{"data":{"AIDecisionModels":[]}}`,
		"Ollama":    `{"data":{"AIDecisionModels":[]}}`,
	} {
		w := serveGraphQL(t, `query AIDecisionModels($modelType: String!) { AIDecisionModels(modelType: $modelType) }`, `{"modelType":"`+modelType+`"}`)
		requireJSONEqual(t, modelType+" decision models", w.Body.Bytes(), want)
	}
}

func TestGraphQLDecisionsTurnedOff(t *testing.T) {
	original := env.DisableAIDecisions
	env.DisableAIDecisions = true
	t.Cleanup(func() { env.DisableAIDecisions = original })
	originalFactory := providers.HTTPClientFactory
	providers.HTTPClientFactory = func() *http.Client {
		t.Fatal("expected no request to the provider")
		return nil
	}
	t.Cleanup(func() { providers.HTTPClientFactory = originalFactory })

	w := serveGraphQL(t, `query AIDecisionModels($modelType: String!) { AIDecisionModels(modelType: $modelType) }`, `{"modelType":"OpenAI"}`)
	requireJSONEqual(t, "decision models", w.Body.Bytes(), `{"data":{"AIDecisionModels":[]}}`)

	w = serveGraphQL(t, `mutation RunAIDecision($input: AIDecisionInput!) { RunAIDecision(input: $input) { Answers { Name } } }`, `{"input":{
		"ModelType":"OpenAI",
		"Token":"test-key",
		"Model":"gpt-6-luna",
		"Input":"The site is down.",
		"Questions":[{"Type":"Predicate","Name":"urgent","Instructions":"Is this urgent?"}]
	}}`)
	if !strings.Contains(w.Body.String(), "decisions are turned off on this server") || strings.Contains(w.Body.String(), `"Answers"`) {
		t.Fatalf("expected the mutation to be refused, got %s", w.Body.String())
	}
}

// serveGraphQL runs one GraphQL document through the executable schema and requires a 200 response.
func serveGraphQL(t *testing.T, query string, variables string) *httptest.ResponseRecorder {
	t.Helper()
	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	payload, err := json.Marshal(map[string]any{"query": query, "variables": json.RawMessage(variables)})
	if err != nil {
		t.Fatalf("failed to encode GraphQL request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d: %s", w.Code, w.Body.String())
	}
	return w
}

func requireJSONEqual(t *testing.T, label string, got []byte, want string) {
	t.Helper()
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("failed to parse %s %q: %v", label, got, err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("failed to parse expected %s: %v", label, err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("unexpected %s:\n got %s\nwant %s", label, got, want)
	}
}
