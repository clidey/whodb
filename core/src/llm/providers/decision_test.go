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
	"context"
	"net/http"
	"reflect"
	"testing"
)

func TestOpenAIProviderCreateDecisionParsesAnswers(t *testing.T) {
	provider := NewOpenAIProvider()
	withTestHTTPClient(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.String() != "https://openai.test/v1/decisions" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("expected bearer auth header, got %q", got)
		}
		return httpResponse(http.StatusOK, `{"model":"gpt-6-luna","answers":[
			{"type":"predicate","name":"visible_damage","probability":0.92},
			{"type":"choice","name":"department","choice":"billing","confidence":0.93,"probabilities":[
				{"value":"billing","probability":0.95},
				{"value":"other","probability":0.05}
			]},
			{"type":"score","name":"severity","score":1.1,"confidence":0.55,"probabilities":[
				{"value":0,"label":"Cosmetic","probability":0.1},
				{"value":1,"label":"Workaround available","probability":0.7},
				{"value":2,"label":"Fully blocked","probability":0.2}
			]},
			{"type":"refusal","name":"intent"}
		]}`), nil
	})

	answers, err := provider.CreateDecision(context.Background(), &ProviderConfig{
		APIKey:   "test-key",
		Endpoint: "https://openai.test/v1/",
	}, &DecisionRequest{Model: "gpt-6-luna", Input: "Export fails in Safari but works in Chrome."})
	if err != nil {
		t.Fatalf("CreateDecision returned error: %v", err)
	}

	expected := []DecisionAnswer{
		{Type: DecisionAnswerPredicate, Name: "visible_damage", Probability: 0.92},
		{Type: DecisionAnswerChoice, Name: "department", Choice: "billing", Confidence: 0.93, ChoiceProbabilities: []DecisionChoiceProbability{
			{Value: "billing", Probability: 0.95},
			{Value: "other", Probability: 0.05},
		}},
		{Type: DecisionAnswerScore, Name: "severity", Score: 1.1, Confidence: 0.55, ScoreProbabilities: []DecisionScoreProbability{
			{Value: 0, Label: "Cosmetic", Probability: 0.1},
			{Value: 1, Label: "Workaround available", Probability: 0.7},
			{Value: 2, Label: "Fully blocked", Probability: 0.2},
		}},
		{Type: DecisionAnswerRefusal, Name: "intent"},
	}
	if !reflect.DeepEqual(answers, expected) {
		t.Fatalf("unexpected answers:\n got %+v\nwant %+v", answers, expected)
	}
}

func TestOpenAIProviderCreateDecisionErrorsOnBadStatus(t *testing.T) {
	provider := NewOpenAIProvider()
	withTestHTTPClient(t, func(r *http.Request) (*http.Response, error) {
		return httpResponse(http.StatusBadRequest, `{"error":{"message":"unsupported model","type":"invalid_request_error"}}`), nil
	})

	_, err := provider.CreateDecision(context.Background(), &ProviderConfig{
		APIKey:   "test-key",
		Endpoint: "https://openai.test/v1",
	}, &DecisionRequest{Model: "gpt-4o", Input: "text"})
	if err == nil || err.Error() != "failed to create decision: unsupported model" {
		t.Fatalf("expected only the API error message, got %v", err)
	}
}

func TestOpenAIProviderCreateDecisionHidesUnparsedErrorBodies(t *testing.T) {
	provider := NewOpenAIProvider()
	withTestHTTPClient(t, func(r *http.Request) (*http.Response, error) {
		return httpResponse(http.StatusBadGateway, `<html>upstream echoed customer@example.com</html>`), nil
	})

	_, err := provider.CreateDecision(context.Background(), &ProviderConfig{
		APIKey:   "test-key",
		Endpoint: "https://openai.test/v1",
	}, &DecisionRequest{Model: "gpt-6-luna", Input: "customer@example.com"})
	if err == nil || err.Error() != "failed to create decision: HTTP 502" {
		t.Fatalf("expected a status-only error, got %v", err)
	}
}

func TestOpenAIProviderDecisionModels(t *testing.T) {
	if got := NewOpenAIProvider().DecisionModels(); !reflect.DeepEqual(got, []string{"gpt-6-luna"}) {
		t.Fatalf("unexpected decision models %v", got)
	}
}
