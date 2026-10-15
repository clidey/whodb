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

package whodb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// RowsPage is one hydrated page of rows plus the total the platform reported
// (nil when the operation does not report one).
type RowsPage struct {
	Rows       []Row
	TotalCount any
}

// WaitOptions bound how long run-polling methods wait and how often they poll.
type WaitOptions struct {
	// Timeout caps the wait (default 60s).
	Timeout time.Duration
	// Poll is the delay between reads (default 2s). Zero polls without delay.
	Poll time.Duration
	// pollSet distinguishes an explicit zero poll from the default.
	pollSet bool
}

// WithPoll returns options with an explicit poll interval; zero means no delay.
func (o WaitOptions) WithPoll(poll time.Duration) WaitOptions {
	o.Poll = poll
	o.pollSet = true
	return o
}

const (
	defaultWaitTimeout = 60 * time.Second
	defaultWaitPoll    = 2 * time.Second
)

// isTerminalRunStatus reports statuses after which a function or transform run
// will not change.
func isTerminalRunStatus(status any) bool {
	switch strings.ToLower(strings.TrimSpace(fmt.Sprint(status))) {
	case "success", "succeeded", "completed", "complete", "failed", "failure", "cancelled", "canceled", "error":
		return true
	}
	return false
}

// pollUntil reads until done or the timeout elapses (PlatformError RUN_TIMEOUT).
func pollUntil(ctx context.Context, read func() (map[string]any, error), done func(map[string]any) bool, options WaitOptions, describe string) (map[string]any, error) {
	timeout := options.Timeout
	if timeout == 0 {
		timeout = defaultWaitTimeout
	}
	poll := options.Poll
	if poll == 0 && !options.pollSet {
		poll = defaultWaitPoll
	}
	deadline := time.Now().Add(timeout)
	for {
		value, err := read()
		if err != nil {
			return nil, err
		}
		if done(value) {
			return value, nil
		}
		if !time.Now().Before(deadline) {
			return nil, &PlatformError{Message: fmt.Sprintf("%s did not finish within %s", describe, timeout), Code: "RUN_TIMEOUT"}
		}
		if poll > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(poll):
			}
		}
	}
}

// encodeInput renders a function input as the compact JSON string the platform expects.
func encodeInput(input any) (string, error) {
	if text, ok := input.(string); ok {
		return text, nil
	}
	if input == nil {
		return "{}", nil
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("%w: encode function input: %v", ErrValidation, err)
	}
	return string(encoded), nil
}

func asMap(result any) map[string]any {
	object, _ := result.(map[string]any)
	return object
}

func asList(result any) []map[string]any {
	raw, _ := result.([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, entry := range raw {
		if object, ok := entry.(map[string]any); ok {
			out = append(out, object)
		}
	}
	return out
}

// findByIDOrName resolves a project resource addressed by id or name.
func findByIDOrName(items []map[string]any, value string) map[string]any {
	for _, item := range items {
		if item["id"] == value || item["name"] == value {
			return item
		}
	}
	return nil
}

// mergeInput copies definition fields and injects the given values.
func mergeInput(definition map[string]any, injected map[string]any) map[string]any {
	input := make(map[string]any, len(definition)+len(injected))
	for key, value := range definition {
		input[key] = value
	}
	for key, value := range injected {
		input[key] = value
	}
	return input
}
