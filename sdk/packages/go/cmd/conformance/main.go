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
	case whodb.RowsPage:
		return map[string]any{"rows": canonical(typed.Rows), "totalCount": canonical(typed.TotalCount)}
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
		value, callErr, ok := runOtherDomain(ctx, client, f.Call)
		if !ok {
			return result{Name: f.Name, Pass: false, Reason: fmt.Sprintf("unsupported fixture call: %s.%s", f.Call.Domain, f.Call.Method)}
		}
		return compare(f, value, callErr)
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
	case "capabilities":
		var recordKey any
		if len(f.Call.Args) > 0 {
			recordKey = f.Call.Args[0]
		}
		value, callErr = handle.Capabilities(ctx, recordKey)
	case "previewAction":
		values, _ := f.Call.Args[2].(map[string]any)
		value, callErr = handle.PreviewAction(ctx, fmt.Sprint(f.Call.Args[0]), f.Call.Args[1], values)
	case "action":
		values, _ := f.Call.Args[2].(map[string]any)
		options := optionsFrom(f.Call.Args, 3)
		actionOptions := whodb.ActionOptions{}
		if version, ok := options["expectedVersion"].(float64); ok {
			converted := int(version)
			actionOptions.ExpectedVersion = &converted
		}
		actionOptions.IdempotencyKey, _ = options["idempotencyKey"].(string)
		value, callErr = handle.Action(ctx, fmt.Sprint(f.Call.Args[0]), f.Call.Args[1], values, actionOptions)
	case "actionExecutions":
		limit := 0
		if len(f.Call.Args) > 1 {
			if rawLimit, ok := f.Call.Args[1].(float64); ok {
				limit = int(rawLimit)
			}
		}
		value, callErr = handle.ActionExecutions(ctx, f.Call.Args[0], limit)
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
	case "followLink":
		link, _ := f.Call.Args[1].(string)
		options := optionsFrom(f.Call.Args, 2)
		value, callErr = handle.FollowLink(ctx, f.Call.Args[0], link, intOption(options, "pageSize", 0), intOption(options, "offset", 0))
	default:
		return result{Name: f.Name, Pass: false, Reason: "unsupported fixture method: " + f.Call.Method}
	}

	return compare(f, value, callErr)
}

// compare checks a call outcome against the fixture's expectation.
func compare(f fixture, value any, callErr error) result {
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

func stringArg(args []any, index int) string {
	if len(args) > index {
		if text, ok := args[index].(string); ok {
			return text
		}
	}
	return ""
}

func mapArg(args []any, index int) map[string]any {
	if len(args) > index {
		object, _ := args[index].(map[string]any)
		return object
	}
	return nil
}

func stringListArg(args []any, index int) []string {
	if len(args) <= index {
		return nil
	}
	raw, ok := args[index].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, entry := range raw {
		out = append(out, fmt.Sprint(entry))
	}
	return out
}

func waitOptions(options map[string]any) whodb.WaitOptions {
	out := whodb.WaitOptions{}
	if timeout, ok := options["timeoutMs"].(float64); ok {
		out.Timeout = time.Duration(timeout) * time.Millisecond
		if timeout == 0 {
			// A zero timeout must still be honoured as "no wait"; use the smallest positive duration.
			out.Timeout = time.Nanosecond
		}
	}
	if poll, ok := options["pollMs"].(float64); ok {
		out = out.WithPoll(time.Duration(poll) * time.Millisecond)
	}
	return out
}

// runOtherDomain dispatches non-ontology fixture calls. ok is false when the
// domain/method pair is not supported by the Go SDK.
func runOtherDomain(ctx context.Context, client *whodb.Client, call fixtureCall) (value any, err error, ok bool) {
	args := call.Args
	switch call.Domain {
	case "dataset":
		handle := client.Dataset(call.Handle)
		switch call.Method {
		case "meta":
			value, err = handle.Meta(ctx)
		case "rows":
			options := optionsFrom(args, 0)
			value, err = handle.Rows(ctx, whodb.DatasetRowsOptions{PageSize: intOption(options, "pageSize", 0), PageOffset: intOption(options, "pageOffset", 0)})
		default:
			return nil, nil, false
		}
	case "source":
		handle := client.Source(call.Handle)
		switch call.Method {
		case "objects":
			options := optionsFrom(args, 0)
			parent, _ := options["parent"].(map[string]any)
			value, err = handle.Objects(ctx, whodb.SourceObjectsOptions{Parent: parent, PageSize: intOption(options, "pageSize", 0), PageOffset: intOption(options, "pageOffset", 0)})
		case "columns":
			value, err = handle.Columns(ctx, mapArg(args, 0))
		case "rows":
			options := optionsFrom(args, 1)
			rowOptions := whodb.SourceRowsOptions{PageSize: intOption(options, "pageSize", 0), Offset: intOption(options, "offset", 0)}
			rowOptions.Where, _ = options["where"].(map[string]any)
			value, err = handle.Rows(ctx, mapArg(args, 0), rowOptions)
		default:
			return nil, nil, false
		}
	case "transforms":
		handle := client.Transforms()
		switch call.Method {
		case "list":
			value, err = handle.List(ctx)
		case "create":
			value, err = handle.Create(ctx, mapArg(args, 0))
		case "preview":
			value, err = handle.Preview(ctx, stringArg(args, 0))
		default:
			return nil, nil, false
		}
	case "transform":
		handle := client.Transform(call.Handle)
		switch call.Method {
		case "meta":
			value, err = handle.Meta(ctx)
		case "runs":
			value, err = handle.Runs(ctx, intOption(optionsFrom(args, 0), "limit", 0))
		case "run":
			value, err = handle.Run(ctx)
		case "cancelRun":
			err = handle.CancelRun(ctx, stringArg(args, 0))
		case "wait":
			value, err = handle.Wait(ctx, stringArg(args, 0), waitOptions(optionsFrom(args, 1)))
		case "update":
			value, err = handle.Update(ctx, mapArg(args, 0))
		case "delete":
			err = handle.Delete(ctx)
		default:
			return nil, nil, false
		}
	case "functions":
		handle := client.Functions()
		switch call.Method {
		case "list":
			value, err = handle.List(ctx)
		case "create":
			value, err = handle.Create(ctx, mapArg(args, 0))
		default:
			return nil, nil, false
		}
	case "function":
		handle := client.Function(call.Handle)
		switch call.Method {
		case "meta":
			value, err = handle.Meta(ctx)
		case "execute":
			value, err = handle.Execute(ctx, args[0], stringListArg(args, 1))
		case "test":
			options := optionsFrom(args, 1)
			testOptions := whodb.FunctionTestOptions{}
			if files, ok := options["files"].([]any); ok {
				for _, file := range files {
					object, _ := file.(map[string]any)
					testOptions.Files = append(testOptions.Files, object)
				}
			}
			if ids, ok := options["inputFileIds"].([]any); ok {
				for _, id := range ids {
					testOptions.InputFileIDs = append(testOptions.InputFileIDs, fmt.Sprint(id))
				}
			}
			value, err = handle.Test(ctx, args[0], testOptions)
		case "startRun":
			options := optionsFrom(args, 1)
			var useActive *bool
			if flag, ok := options["useActiveVersion"].(bool); ok {
				useActive = &flag
			}
			value, err = handle.StartRun(ctx, args[0], useActive)
		case "run":
			value, err = handle.Run(ctx, stringArg(args, 0))
		case "runs":
			value, err = handle.Runs(ctx, intOption(optionsFrom(args, 0), "limit", 0))
		case "wait":
			value, err = handle.Wait(ctx, stringArg(args, 0), waitOptions(optionsFrom(args, 1)))
		case "cancelRun":
			value, err = handle.CancelRun(ctx, stringArg(args, 0))
		case "deploy":
			err = handle.Deploy(ctx)
		case "redeploy":
			err = handle.Redeploy(ctx)
		case "update":
			value, err = handle.Update(ctx, mapArg(args, 0))
		case "delete":
			err = handle.Delete(ctx)
		default:
			return nil, nil, false
		}
	case "serviceAccounts":
		handle := client.ServiceAccounts()
		switch call.Method {
		case "list":
			value, err = handle.List(ctx)
		case "create":
			value, err = handle.Create(ctx, stringArg(args, 0), stringArg(args, 1))
		case "createKey":
			value, err = handle.CreateKey(ctx, stringArg(args, 0), stringArg(args, 1))
		case "revokeKey":
			err = handle.RevokeKey(ctx, stringArg(args, 0))
		case "delete":
			err = handle.Delete(ctx, stringArg(args, 0))
		default:
			return nil, nil, false
		}
	case "lineageProviders":
		handle := client.LineageProviders()
		switch call.Method {
		case "list":
			value, err = handle.List(ctx)
		case "save":
			value, err = handle.Save(ctx, mapArg(args, 0))
		case "delete":
			err = handle.Delete(ctx, stringArg(args, 0))
		default:
			return nil, nil, false
		}
	case "sharing":
		handle := client.Sharing()
		switch call.Method {
		case "shared":
			value, err = handle.Shared(ctx)
		case "discover":
			value, err = handle.Discover(ctx)
		case "references":
			value, err = handle.References(ctx)
		case "referenceImpact":
			value, err = handle.ReferenceImpact(ctx, stringArg(args, 0))
		case "share":
			value, err = handle.Share(ctx, stringArg(args, 0), stringArg(args, 1), stringListArg(args, 2))
		case "unshare":
			err = handle.Unshare(ctx, stringArg(args, 0))
		case "createReference":
			value, err = handle.CreateReference(ctx, stringArg(args, 0), stringArg(args, 1), stringArg(args, 2))
		case "removeReference":
			err = handle.RemoveReference(ctx, stringArg(args, 0))
		default:
			return nil, nil, false
		}
	default:
		return nil, nil, false
	}
	return value, err, true
}
