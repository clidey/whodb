// Command conformance is the Go SDK fixture runner: it reads fixtures as JSON
// lines on stdin, executes each against the SDK with a mock transport, and
// writes one JSON result line per fixture to stdout. Protocol shared with the
// TypeScript and Python runners.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	whodb "github.com/clidey/whodb/sdk/packages/go"
)

type fixture struct {
	Name       string           `json:"name"`
	Call       fixtureCall      `json:"call"`
	Transcript []transcriptStep `json:"transcript"`
	Expect     json.RawMessage  `json:"expectResult"`
	Error      *expectError     `json:"expectError"`
}

type fixtureCall struct {
	Domain  string `json:"domain"`
	Handle  string `json:"handle"`
	Method  string `json:"method"`
	Args    []any  `json:"args"`
	Collect string `json:"collect"`
}

type transcriptStep struct {
	ExpectRequest struct {
		Operation        string         `json:"operation"`
		Variables        map[string]any `json:"variables"`
		VariablesContain map[string]any `json:"variablesContain"`
		InputContains    map[string]any `json:"inputContains"`
	} `json:"expectRequest"`
	Response struct {
		Data   map[string]any `json:"data"`
		Errors []struct {
			Message    string `json:"message"`
			Extensions struct {
				Code string `json:"code"`
			} `json:"extensions"`
		} `json:"errors"`
	} `json:"response"`
}

type expectError struct {
	Type            string `json:"type"`
	MessageContains string `json:"messageContains"`
}

type result struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Reason string `json:"reason,omitempty"`
}

// mockTransport replays a fixture's scripted transcript.
type mockTransport struct {
	steps []transcriptStep
	index int
}

func canonicalJSON(value any) string {
	encoded, _ := json.Marshal(value)
	var normalized any
	_ = json.Unmarshal(encoded, &normalized)
	out, _ := json.Marshal(normalized)
	return string(out)
}

func (m *mockTransport) Execute(_ context.Context, operation, _ string, variables map[string]any) (map[string]any, error) {
	if m.index >= len(m.steps) {
		return nil, fmt.Errorf("unexpected call #%d: %s", m.index+1, operation)
	}
	step := m.steps[m.index]
	m.index++
	expect := step.ExpectRequest
	if expect.Operation != "" && expect.Operation != operation {
		return nil, fmt.Errorf("expected operation %s, got %s", expect.Operation, operation)
	}
	assertContains := func(actual, expected any, path string) error {
		if canonicalJSON(actual) != canonicalJSON(expected) {
			return fmt.Errorf("mismatch at %s: expected %s, got %s", path, canonicalJSON(expected), canonicalJSON(actual))
		}
		return nil
	}
	for key, value := range expect.Variables {
		if err := assertContains(variables[key], value, "variables."+key); err != nil {
			return nil, err
		}
	}
	for key, value := range expect.VariablesContain {
		if err := assertContains(variables[key], value, "variables."+key); err != nil {
			return nil, err
		}
	}
	if len(expect.InputContains) > 0 {
		input, _ := variables["input"].(map[string]any)
		for key, value := range expect.InputContains {
			if err := assertContains(input[key], value, "input."+key); err != nil {
				return nil, err
			}
		}
	}
	if len(step.Response.Errors) > 0 {
		first := step.Response.Errors[0]
		switch first.Extensions.Code {
		case "UNAUTHENTICATED", "FORBIDDEN":
			return nil, fmt.Errorf("whodb: authentication failed: %s", first.Message)
		case "NOT_FOUND":
			return nil, fmt.Errorf("whodb: not found: %s", first.Message)
		case "BAD_USER_INPUT", "GRAPHQL_VALIDATION_FAILED":
			return nil, fmt.Errorf("whodb: invalid request: %s", first.Message)
		default:
			return nil, fmt.Errorf("whodb: platform error %s: %s", first.Extensions.Code, first.Message)
		}
	}
	return step.Response.Data, nil
}

// canonical serializes results for comparison; time.Time becomes
// "@date:<ISO-8601 Z>" per the shared protocol.
func canonical(value any) any {
	switch typed := value.(type) {
	case time.Time:
		return "@date:" + strings.Replace(typed.UTC().Format(time.RFC3339), "+00:00", "Z", 1)
	case []whodb.Row:
		out := make([]any, 0, len(typed))
		for _, row := range typed {
			out = append(out, canonical(row))
		}
		return out
	case [][]whodb.Row:
		out := make([]any, 0, len(typed))
		for _, page := range typed {
			out = append(out, canonical(page))
		}
		return out
	case map[string]any: // includes whodb.Row (type alias)
		out := map[string]any{}
		for key, item := range typed {
			out[key] = canonical(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, canonical(item))
		}
		return out
	default:
		return value
	}
}

// errorType maps a Go SDK error to the shared taxonomy names used in fixtures.
func errorType(err error) string {
	switch {
	case errors.Is(err, whodb.ErrAuth):
		return "AuthError"
	case errors.Is(err, whodb.ErrNotFound):
		return "NotFoundError"
	case errors.Is(err, whodb.ErrValidation):
		return "ValidationError"
	case errors.Is(err, whodb.ErrVersion):
		return "WhoDBVersionError"
	case errors.Is(err, whodb.ErrTransportCapability):
		return "TransportCapabilityError"
	}
	var platformErr *whodb.PlatformError
	if errors.As(err, &platformErr) {
		return "PlatformError"
	}
	// The mock transport surfaces taxonomy via message prefixes.
	message := err.Error()
	switch {
	case strings.Contains(message, "authentication failed"):
		return "AuthError"
	case strings.Contains(message, "not found"):
		return "NotFoundError"
	case strings.Contains(message, "invalid request"):
		return "ValidationError"
	default:
		return "PlatformError"
	}
}

func optionsFrom(args []any, index int) map[string]any {
	if len(args) > index {
		options, _ := args[index].(map[string]any)
		return options
	}
	return nil
}

func intOption(options map[string]any, key string, fallback int) int {
	if value, ok := options[key].(float64); ok {
		return int(value)
	}
	return fallback
}

func runFixture(f fixture) result {
	transport := &mockTransport{steps: f.Transcript}
	client, err := whodb.New(whodb.Config{
		Org:       "00000000-0000-0000-0000-000000000001",
		Project:   "proj-1",
		Transport: transport,
	})
	if err != nil {
		return result{Name: f.Name, Pass: false, Reason: err.Error()}
	}
	ctx := context.Background()
	if f.Call.Domain != "ontology" {
		return result{Name: f.Name, Pass: false, Reason: "unsupported fixture domain: " + f.Call.Domain}
	}
	handle := client.Ontology(f.Call.Handle)

	var value any
	var callErr error
	switch f.Call.Method {
	case "get":
		row, err := handle.Get(ctx, f.Call.Args[0])
		if row == nil {
			value = nil
		} else {
			value = row
		}
		callErr = err
	case "describe":
		value, callErr = handle.Describe(ctx)
	case "list":
		options := optionsFrom(f.Call.Args, 0)
		listOptions := whodb.ListOptions{PageSize: intOption(options, "pageSize", 0)}
		if where, ok := options["where"].(map[string]any); ok {
			listOptions.Where = where
		}
		if f.Call.Collect == "pages" {
			pages := [][]whodb.Row{}
			callErr = handle.Pages(ctx, listOptions, func(rows []whodb.Row) bool {
				pages = append(pages, rows)
				return true
			})
			value = pages
		} else {
			value, callErr = handle.List(ctx, listOptions)
		}
	case "create":
		data, _ := f.Call.Args[0].(map[string]any)
		callErr = handle.Create(ctx, data)
		value = nil
	case "createMany":
		rawRows, _ := f.Call.Args[0].([]any)
		rows := make([]map[string]any, 0, len(rawRows))
		for _, entry := range rawRows {
			row, _ := entry.(map[string]any)
			rows = append(rows, row)
		}
		options := optionsFrom(f.Call.Args, 1)
		idempotencyKey, _ := options["idempotencyKey"].(string)
		value, callErr = handle.CreateMany(ctx, rows, idempotencyKey)
	case "update":
		data, _ := f.Call.Args[1].(map[string]any)
		callErr = handle.Update(ctx, f.Call.Args[0], data)
		value = nil
	case "delete":
		callErr = handle.Delete(ctx, f.Call.Args[0])
		value = nil
	default:
		return result{Name: f.Name, Pass: false, Reason: "unsupported fixture method: " + f.Call.Method}
	}

	if callErr != nil {
		if f.Error != nil {
			gotType := errorType(callErr)
			if gotType == f.Error.Type && strings.Contains(callErr.Error(), f.Error.MessageContains) {
				return result{Name: f.Name, Pass: true}
			}
			return result{Name: f.Name, Pass: false, Reason: fmt.Sprintf("expected %s(%s), got %s: %v", f.Error.Type, f.Error.MessageContains, gotType, callErr)}
		}
		return result{Name: f.Name, Pass: false, Reason: callErr.Error()}
	}
	if f.Error != nil {
		return result{Name: f.Name, Pass: false, Reason: "expected " + f.Error.Type + ", got success"}
	}

	var want any
	if len(f.Expect) > 0 {
		_ = json.Unmarshal(f.Expect, &want)
	}
	got := canonical(value)
	if canonicalJSON(got) != canonicalJSON(want) {
		return result{Name: f.Name, Pass: false, Reason: fmt.Sprintf("result mismatch:\n  want %s\n  got  %s", canonicalJSON(want), canonicalJSON(got))}
	}
	return result{Name: f.Name, Pass: true}
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var f fixture
		if err := json.Unmarshal([]byte(line), &f); err != nil {
			_ = encoder.Encode(result{Name: "(parse error)", Pass: false, Reason: err.Error()})
			continue
		}
		_ = encoder.Encode(runFixture(f))
	}
}
