package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type platformIdempotencyResult struct {
	PayloadHash string
	ResultJSON  string
}

// Reserve and compare under one lock so concurrent local calls cannot execute
// the same key twice. Hosted checkpoints preserve the claim across restarts.
func reservePlatformIdempotency(ctx context.Context, action *PendingPlatformAction, confirm bool, requestID string) (*PlatformGenericWriteOutput, error) {
	raw, err := json.Marshal(struct {
		Mutation  string
		Variables map[string]any
	}{action.Mutation, action.Variables})
	if err != nil {
		return nil, err
	}
	action.PayloadHash = fmt.Sprintf("%x", sha256.Sum256(raw))
	state := platformStateFor(ctx)
	state.mu.Lock()
	defer state.mu.Unlock()
	key := action.IdempotencyKey
	conflict := func() (*PlatformGenericWriteOutput, error) {
		return nil, errors.New("idempotency key conflict: this key belongs to a different or unverifiable payload; use a new key for a new operation")
	}
	if done, ok := state.IdempotencyDone[key]; ok && time.Since(done) < 30*24*time.Hour {
		result := state.IdempotencyResults[key]
		if result.PayloadHash != action.PayloadHash {
			return conflict()
		}
		return &PlatformGenericWriteOutput{Status: "already_applied", IdempotencyReplayed: true, ResultJSON: result.ResultJSON, RequestID: requestID}, nil
	}
	if existing := state.Pending[state.IdempotencyPending[key]]; existing != nil && (existing.InFlight || time.Now().Before(existing.ExpiresAt)) {
		if existing.PayloadHash != action.PayloadHash {
			return conflict()
		}
		if existing.InFlight {
			return nil, errors.New("idempotency key conflict: the operation is in progress or its result is uncertain; inspect its status before retrying")
		}
		if confirm {
			out := platformGenericConfirmationOutput(ctx, requestID, existing.Token, existing.ExpiresAt, existing.Mutation, existing.Preview())
			return &out, nil
		}
		// A previously approved preview must still be confirmed explicitly.
		return nil, errors.New("idempotency key conflict: confirm the existing pending preview")
	}
	action.Token = generateConfirmationToken()
	action.ExpiresAt = time.Now().Add(5 * time.Minute)
	action.InFlight = !confirm
	state.Pending[action.Token] = action
	state.IdempotencyPending[key] = action.Token
	if state.Checkpoint != nil {
		if err := state.Checkpoint(); err != nil {
			delete(state.Pending, action.Token)
			delete(state.IdempotencyPending, key)
			return nil, err
		}
	}
	return nil, nil
}

// Replay stores only bounded identifiers and status, never credentials, function
// output, source data, or once-only secrets from a successful mutation response.
func safePlatformReplayResult(raw string) string {
	var object map[string]json.RawMessage
	if json.Unmarshal([]byte(raw), &object) != nil {
		return ""
	}
	safe := map[string]any{}
	for _, key := range []string{"id", "runId", "status", "success"} { //nolint:goconst // Explicit allowlist of replayable wire fields.
		value, ok := object[key]
		if !ok || len(value) > 1024 {
			continue
		}
		var primitive any
		if json.Unmarshal(value, &primitive) != nil {
			continue
		}
		switch primitive.(type) {
		case string, bool, float64:
			safe[key] = primitive
		}
	}
	if len(safe) == 0 {
		return ""
	}
	data, _ := json.Marshal(safe)
	return string(data)
}
