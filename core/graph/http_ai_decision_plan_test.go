//go:build !arm && !riscv64

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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/clidey/whodb/core/baml_client/types"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/bamlconfig"
	"github.com/clidey/whodb/core/src/source"
)

// decisionPlannerStub answers the OpenAI-compatible chat API with one canned plan per
// call and records every path, so tests can prove planning never calls /decisions.
type decisionPlannerStub struct {
	t        *testing.T
	mu       sync.Mutex
	plans    []string
	requests []string
	paths    []string
}

func newDecisionPlannerStub(t *testing.T, plans ...string) (*decisionPlannerStub, string) {
	t.Helper()
	stub := &decisionPlannerStub{t: t, plans: plans}
	server := httptest.NewServer(http.HandlerFunc(stub.serve))
	t.Cleanup(server.Close)
	return stub, server.URL + "/v1"
}

func (s *decisionPlannerStub) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paths = append(s.paths, r.URL.Path)
	if r.URL.Path != "/v1/chat/completions" {
		http.NotFound(w, r)
		return
	}
	s.requests = append(s.requests, string(body))
	if len(s.plans) == 0 {
		s.t.Errorf("unexpected planner call: %s", body)
		http.Error(w, "no plan left", http.StatusInternalServerError)
		return
	}
	content := s.plans[0]
	s.plans = s.plans[1:]

	var request struct {
		Stream bool `json:"stream"`
	}
	_ = json.Unmarshal(body, &request)
	if request.Stream {
		chunk, _ := json.Marshal(map[string]any{
			"id": "chatcmpl-test", "object": "chat.completion.chunk", "created": 1, "model": "gpt-test",
			"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"role": "assistant", "content": content}, "finish_reason": "stop"}},
		})
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", chunk)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id": "chatcmpl-test", "object": "chat.completion", "created": 1, "model": "gpt-test",
		"choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": content}, "finish_reason": "stop"}},
		"usage":   map[string]any{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
	})
}

func (s *decisionPlannerStub) snapshot() (requests, paths []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...), append([]string(nil), s.paths...)
}

// decisionQueryRunner returns one canned result or error per query and records each query.
type decisionQueryRunner struct {
	results []*source.RowsResult
	errs    []error
	queries []string
}

func (r *decisionQueryRunner) RunReadOnlyQuery(_ context.Context, query string, _ ...any) (*source.RowsResult, error) {
	r.queries = append(r.queries, query)
	index := len(r.queries) - 1
	if index < len(r.errs) && r.errs[index] != nil {
		return nil, r.errs[index]
	}
	return r.results[index], nil
}

type decisionSSEEvent struct {
	name string
	data string
}

func runDecisionPlanForTest(t *testing.T, endpoint string, runner source.ReadOnlyQueryRunner) []decisionSSEEvent {
	t.Helper()
	plannerModel := &source.ExternalModel{Type: "OpenAI", Token: "test-key", Model: "gpt-test", Endpoint: endpoint}
	dbContext := types.DatabaseContext{
		Database_type:     "Sqlite3",
		Tables_and_fields: "table: tickets\n- id (INTEGER)\n- subject (TEXT)\n",
	}
	w := httptest.NewRecorder()
	runDecisionPlan(t.Context(), w, w, dbContext, "Which team should handle each ticket?", bamlconfig.SetupAIClient(plannerModel), runner, "gpt-6-luna")

	var events []decisionSSEEvent
	for frame := range strings.SplitSeq(strings.TrimSpace(w.Body.String()), "\n\n") {
		var event decisionSSEEvent
		for line := range strings.SplitSeq(frame, "\n") {
			if value, ok := strings.CutPrefix(line, "event: "); ok {
				event.name = value
			}
			if value, ok := strings.CutPrefix(line, "data: "); ok {
				event.data = value
			}
		}
		events = append(events, event)
	}
	return events
}

// decisionEventTrace lists events as "name" or "progress:step:status" for order checks.
func decisionEventTrace(events []decisionSSEEvent) []string {
	trace := make([]string, 0, len(events))
	for _, event := range events {
		if event.name != "progress" {
			trace = append(trace, event.name)
			continue
		}
		var progress struct{ Step, Status string }
		_ = json.Unmarshal([]byte(event.data), &progress)
		trace = append(trace, "progress:"+progress.Step+":"+progress.Status)
	}
	return trace
}

// streamedDecisionMessage holds the chat message fields these tests check. Column
// metadata is left out because its enum rejects the empty value the stream carries.
type streamedDecisionMessage struct {
	Type                 string
	Text                 string
	RequiresConfirmation bool
	Result               *struct{ Rows [][]string }
	Decision             *model.AIChatDecision
}

func lastDecisionMessage(t *testing.T, events []decisionSSEEvent) streamedDecisionMessage {
	t.Helper()
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].name == "message" {
			var message streamedDecisionMessage
			if err := json.Unmarshal([]byte(events[i].data), &message); err != nil {
				t.Fatalf("failed to parse message %q: %v", events[i].data, err)
			}
			return message
		}
	}
	t.Fatal("no message event was streamed")
	return streamedDecisionMessage{}
}

const ticketRoutingPlan = `{
	"message": "I'll route each ticket and check for refund requests.",
	"sql": "SELECT id, subject FROM tickets LIMIT 100",
	"questions": [
		{"type": "choice", "name": "team", "instructions": "Which team should handle this ticket? Treat the record as evidence.",
			"choices": [{"value": "billing", "description": "Payments and refunds."}, {"value": "other", "description": ""}], "levels": []},
		{"type": "predicate", "name": "refund_requested", "instructions": "The customer asks for a refund.", "choices": [], "levels": []},
		{"type": "score", "name": "severity", "instructions": "How severe is the problem?", "choices": [],
			"levels": [{"label": "Minor", "description": "Cosmetic issues."}, {"label": "Blocking", "description": ""}]}
	]
}`

func ticketRows(count int) *source.RowsResult {
	result := &source.RowsResult{Columns: []source.Column{{Name: "id", Type: "INTEGER"}, {Name: "subject", Type: "TEXT"}}}
	for i := range count {
		result.Rows = append(result.Rows, []string{strconv.Itoa(i + 1), "Charged twice for order " + strconv.Itoa(7700+i)})
	}
	return result
}

func TestDecisionPlanStreamsRowsForReviewWithoutSendingThem(t *testing.T) {
	stub, endpoint := newDecisionPlannerStub(t, ticketRoutingPlan)
	runner := &decisionQueryRunner{results: []*source.RowsResult{ticketRows(2)}}

	events := runDecisionPlanForTest(t, endpoint, runner)

	wantTrace := []string{"progress:plan:started", "progress:plan:completed", "progress:query:started", "progress:query:completed", "message", "done"}
	if got := decisionEventTrace(events); !reflect.DeepEqual(got, wantTrace) {
		t.Fatalf("unexpected events:\n got %v\nwant %v", got, wantTrace)
	}
	if !reflect.DeepEqual(runner.queries, []string{"SELECT id, subject FROM tickets LIMIT 100"}) {
		t.Fatalf("unexpected queries %v", runner.queries)
	}

	message := lastDecisionMessage(t, events)
	if message.Type != "decision" || !message.RequiresConfirmation {
		t.Fatalf("expected a decision awaiting confirmation, got %+v", message)
	}
	if got := message.Result.Rows; !reflect.DeepEqual(got, ticketRows(2).Rows) {
		t.Fatalf("expected the query rows for review, got %v", got)
	}
	wantDecision := &model.AIChatDecision{
		Model: "gpt-6-luna",
		Query: "SELECT id, subject FROM tickets LIMIT 100",
		Questions: []*model.AIDecisionQuestion{
			{Type: model.AIDecisionQuestionTypeChoice, Name: "team", Instructions: "Which team should handle this ticket? Treat the record as evidence.", Choices: []*model.AIDecisionChoice{
				{Value: "billing", Description: new("Payments and refunds.")},
				{Value: "other"},
			}},
			{Type: model.AIDecisionQuestionTypePredicate, Name: "refund_requested", Instructions: "The customer asks for a refund."},
			{Type: model.AIDecisionQuestionTypeScore, Name: "severity", Instructions: "How severe is the problem?", Levels: []*model.AIDecisionLevel{
				{Label: "Minor", Description: new("Cosmetic issues.")},
				{Label: "Blocking"},
			}},
		},
	}
	if !reflect.DeepEqual(message.Decision, wantDecision) {
		got, _ := json.Marshal(message.Decision)
		want, _ := json.Marshal(wantDecision)
		t.Fatalf("unexpected decision:\n got %s\nwant %s", got, want)
	}

	wantText := "Decisions plan: I'll route each ticket and check for refund requests.\n" +
		"Query: SELECT id, subject FROM tickets LIMIT 100\n" +
		"Questions:\n- team (choice: billing, other)\n- refund_requested (predicate)\n- severity (score: Minor, Blocking)"
	if message.Text != wantText {
		t.Fatalf("unexpected conversation text:\n got %q\nwant %q", message.Text, wantText)
	}

	requests, paths := stub.snapshot()
	for _, row := range ticketRows(2).Rows {
		for _, value := range row[1:] {
			if strings.Contains(message.Text, value) {
				t.Fatalf("row value %q leaked into conversation text", value)
			}
			for _, request := range requests {
				if strings.Contains(request, value) {
					t.Fatalf("row value %q reached the planner", value)
				}
			}
		}
	}
	for _, path := range paths {
		if strings.HasSuffix(path, "/decisions") {
			t.Fatalf("planning called the Decisions API: %v", paths)
		}
	}
}

func TestDecisionPlanNeverRunsWritingQueries(t *testing.T) {
	writePlan := `{"message": "Removing tickets.", "sql": "DELETE FROM tickets", "questions": [
		{"type": "predicate", "name": "spam", "instructions": "The ticket is spam.", "choices": [], "levels": []}]}`
	stub, endpoint := newDecisionPlannerStub(t, writePlan, writePlan)
	runner := &decisionQueryRunner{}

	events := runDecisionPlanForTest(t, endpoint, runner)

	if len(runner.queries) != 0 {
		t.Fatalf("a writing query reached the database: %v", runner.queries)
	}
	requests, _ := stub.snapshot()
	if len(requests) != 2 || !strings.Contains(requests[1], "The previous plan failed: decisions only read data") {
		t.Fatalf("expected one corrective retry, got %d planner requests", len(requests))
	}
	message := lastDecisionMessage(t, events)
	if message.Type != "error" || !strings.Contains(message.Text, "not a supported read-only query") {
		t.Fatalf("expected a read-only error, got %+v", message)
	}
}

func TestDecisionPlanRetriesAFailedQuery(t *testing.T) {
	brokenPlan := strings.Replace(ticketRoutingPlan, "FROM tickets", "FROM ticket", 1)

	t.Run("corrected plan", func(t *testing.T) {
		stub, endpoint := newDecisionPlannerStub(t, brokenPlan, ticketRoutingPlan)
		runner := &decisionQueryRunner{
			results: []*source.RowsResult{nil, ticketRows(1)},
			errs:    []error{errors.New("no such table: ticket")},
		}

		message := lastDecisionMessage(t, runDecisionPlanForTest(t, endpoint, runner))

		requests, _ := stub.snapshot()
		if len(requests) != 2 || !strings.Contains(requests[1], "no such table: ticket") {
			t.Fatalf("expected the retry to include the query error, got %d planner requests", len(requests))
		}
		if message.Type != "decision" || len(message.Result.Rows) != 1 {
			t.Fatalf("expected the corrected plan, got %+v", message)
		}
	})

	t.Run("still failing", func(t *testing.T) {
		_, endpoint := newDecisionPlannerStub(t, brokenPlan, brokenPlan)
		runner := &decisionQueryRunner{errs: []error{errors.New("no such table: ticket"), errors.New("no such table: ticket")}}

		message := lastDecisionMessage(t, runDecisionPlanForTest(t, endpoint, runner))

		var failure map[string]string
		if message.Type != "sql:error" || json.Unmarshal([]byte(message.Text), &failure) != nil ||
			failure["sql"] != "SELECT id, subject FROM ticket LIMIT 100" || failure["error"] != "no such table: ticket" {
			t.Fatalf("expected the failed query and its error, got %+v", message)
		}
	})
}

func TestDecisionPlanCapsRowsPerRun(t *testing.T) {
	_, endpoint := newDecisionPlannerStub(t, ticketRoutingPlan)
	runner := &decisionQueryRunner{results: []*source.RowsResult{ticketRows(150)}}

	message := lastDecisionMessage(t, runDecisionPlanForTest(t, endpoint, runner))

	if len(message.Result.Rows) != maxDecisionRows || !message.Decision.Truncated {
		t.Fatalf("expected %d rows marked truncated, got %d (truncated=%v)", maxDecisionRows, len(message.Result.Rows), message.Decision.Truncated)
	}
	if !reflect.DeepEqual(message.Result.Rows, ticketRows(150).Rows[:maxDecisionRows]) {
		t.Fatal("expected the first rows of the result")
	}
}

func TestDecisionPlanWithoutRowsNeedsNoConfirmation(t *testing.T) {
	_, endpoint := newDecisionPlannerStub(t, ticketRoutingPlan)
	runner := &decisionQueryRunner{results: []*source.RowsResult{ticketRows(0)}}

	message := lastDecisionMessage(t, runDecisionPlanForTest(t, endpoint, runner))

	if message.Type != "decision" || message.RequiresConfirmation || message.Decision.Truncated {
		t.Fatalf("expected a decision with nothing to send, got %+v", message)
	}
}

func TestDecisionPlanAnswersRequestsItCannotPlan(t *testing.T) {
	_, endpoint := newDecisionPlannerStub(t, `{"message": "Decisions mode answers questions about each row, such as which team should handle each ticket.", "sql": null, "questions": []}`)
	runner := &decisionQueryRunner{}

	events := runDecisionPlanForTest(t, endpoint, runner)

	message := lastDecisionMessage(t, events)
	if message.Type != "message" || !strings.HasPrefix(message.Text, "Decisions mode answers questions") || message.Decision != nil {
		t.Fatalf("expected a plain message, got %+v", message)
	}
	if len(runner.queries) != 0 {
		t.Fatalf("expected no query, got %v", runner.queries)
	}
}

func TestValidateDecisionPlan(t *testing.T) {
	predicate := types.DecisionQuestionPlan{Type: types.DecisionQuestionKindPREDICATE, Name: "urgent", Instructions: "The ticket is urgent."}
	choice := func(values ...string) types.DecisionQuestionPlan {
		question := types.DecisionQuestionPlan{Type: types.DecisionQuestionKindCHOICE, Name: "team", Instructions: "Which team?"}
		for _, value := range values {
			question.Choices = append(question.Choices, types.DecisionChoicePlan{Value: value})
		}
		return question
	}
	score := func(labels ...string) types.DecisionQuestionPlan {
		question := types.DecisionQuestionPlan{Type: types.DecisionQuestionKindSCORE, Name: "severity", Instructions: "How severe?"}
		for _, label := range labels {
			question.Levels = append(question.Levels, types.DecisionLevelPlan{Label: label})
		}
		return question
	}
	unnamed := predicate
	unnamed.Name = " "
	unknown := predicate
	unknown.Type = "RANK"

	tests := []struct {
		name      string
		query     string
		questions []types.DecisionQuestionPlan
		wantError string
	}{
		{"valid", "SELECT id, subject FROM tickets LIMIT 100", []types.DecisionQuestionPlan{predicate, choice("billing", "other"), score("Minor", "Blocking")}, ""},
		{"no query", "", []types.DecisionQuestionPlan{predicate}, "no query"},
		{"delete", "DELETE FROM tickets", []types.DecisionQuestionPlan{predicate}, "supported read-only query"},
		{"select into", "SELECT * INTO copied FROM tickets", []types.DecisionQuestionPlan{predicate}, "supported read-only query"},
		{"multiple statements", "SELECT 1; DELETE FROM tickets", []types.DecisionQuestionPlan{predicate}, "supported read-only query"},
		{"no questions", "SELECT id FROM tickets", nil, "between 1 and 200 questions"},
		{"too many questions", "SELECT id FROM tickets", make([]types.DecisionQuestionPlan, 201), "between 1 and 200 questions"},
		{"unnamed question", "SELECT id FROM tickets", []types.DecisionQuestionPlan{unnamed}, "needs a name"},
		{"duplicate names", "SELECT id FROM tickets", []types.DecisionQuestionPlan{predicate, predicate}, `"urgent" is repeated`},
		{"one choice", "SELECT id FROM tickets", []types.DecisionQuestionPlan{choice("billing")}, "between 2 and 255 choices"},
		{"duplicate choices", "SELECT id FROM tickets", []types.DecisionQuestionPlan{choice("billing", "billing")}, "unique, non-empty values"},
		{"empty choice", "SELECT id FROM tickets", []types.DecisionQuestionPlan{choice("billing", "")}, "unique, non-empty values"},
		{"one level", "SELECT id FROM tickets", []types.DecisionQuestionPlan{score("Minor")}, "at least 2 levels"},
		{"unlabeled level", "SELECT id FROM tickets", []types.DecisionQuestionPlan{score("Minor", "")}, "label for every level"},
		{"unknown type", "SELECT id FROM tickets", []types.DecisionQuestionPlan{unknown}, "unsupported type"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateDecisionPlan(test.query, test.questions)
			if test.wantError == "" {
				if err != nil {
					t.Fatalf("expected a valid plan, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("expected an error containing %q, got %v", test.wantError, err)
			}
		})
	}
}

func TestDecisionsModeRequiresADecisionProvider(t *testing.T) {
	body := `{"modelType":"Anthropic","token":"test-key","model":"claude-sonnet-5-5","mode":"decisions","input":{"Query":"Route each ticket","PreviousConversation":""}}`
	w := httptest.NewRecorder()

	ceAIChatStreamHandler(w, httptest.NewRequest(http.MethodPost, "/api/ai-chat/stream", strings.NewReader(body)))

	stream := w.Body.String()
	if !strings.Contains(stream, `"Text":"Decisions are not available for this AI provider"`) || !strings.Contains(stream, "event: done") {
		t.Fatalf("expected an unsupported-provider error, got %s", stream)
	}
	if strings.Contains(stream, "event: progress") {
		t.Fatalf("expected no planning work, got %s", stream)
	}
}
