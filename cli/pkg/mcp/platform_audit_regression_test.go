package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

func TestWaitReturnsLastStatusBeforeTransportDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	polls := 0
	_, out, err := waitForPlatformRun(ctx, nil, "wait", "request", nil, time.Minute, time.Millisecond, func(ctx context.Context) (*platformapi.FunctionRun, bool, error) {
		polls++
		if polls == 1 {
			return &platformapi.FunctionRun{ID: "run", Status: "running"}, false, nil
		}
		<-ctx.Done()
		return nil, false, ctx.Err()
	})
	if err != nil || out.Error != "" || ctx.Err() != nil || len(out.Warnings) != 1 || out.Data.(*platformapi.FunctionRun).ID != "run" {
		t.Fatalf("out=%+v err=%v parent=%v", out, err, ctx.Err())
	}
}

func TestPlatformErrorCategories(t *testing.T) {
	for _, tc := range []struct {
		err   error
		code  string
		retry bool
	}{
		{errors.New("organization access denied"), "permission_denied", false},
		{errors.New("confirmation token has expired"), "confirmation_expired", false},
		{fmt.Errorf("wrapped: %w", context.DeadlineExceeded), "timeout", true},
		{platformapi.GraphQLError{Message: "private", Code: "RESOURCE_EXHAUSTED"}, "quota_exceeded", false},
		{platformapi.GraphQLError{Message: "organization required", Code: "FORBIDDEN"}, "permission_denied", false},
		{platformapi.GraphQLError{Message: "private", Code: "SERVICE_UNAVAILABLE"}, "service_unavailable", true},
	} {
		out := platformReadErrorOutput(tc.err, "trace")
		if out.ErrorCode != tc.code || out.Retryable != tc.retry || out.RequestID != "trace" || out.Recovery == nil {
			t.Fatalf("%v: %+v", tc.err, out)
		}
	}
}

func TestIdempotencyAtomicPayloadAndRestart(t *testing.T) {
	state := NewPlatformState()
	ctx := WithPlatformRuntime(context.Background(), &PlatformRuntime{State: state})
	action := func(value string) *PendingPlatformAction {
		return &PendingPlatformAction{IdempotencyKey: "key", Mutation: "CreateFunction", Variables: map[string]any{"name": value}}
	}
	var wg sync.WaitGroup
	results := make(chan *PendingPlatformAction, 20)
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a := action("first")
			out, err := reservePlatformIdempotency(ctx, a, false, "req")
			if err == nil && out == nil {
				results <- a
			}
		}()
	}
	wg.Wait()
	close(results)
	if len(results) != 1 {
		t.Fatalf("executions=%d", len(results))
	}
	a := <-results
	if _, err := reservePlatformIdempotency(ctx, action("changed"), false, "req"); err == nil {
		t.Fatal("different payload accepted")
	}
	consumePendingPlatformActionResult(a.Token, `{"id":"created-id","status":"ready","password":"DO-NOT-PERSIST","output":"sensitive"}`, ctx)
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "DO-NOT-PERSIST") || strings.Contains(string(raw), "sensitive") {
		t.Fatal("secret in persisted replay")
	}
	restored := NewPlatformState()
	if err := json.Unmarshal(raw, restored); err != nil {
		t.Fatal(err)
	}
	ctx = WithPlatformRuntime(context.Background(), &PlatformRuntime{State: restored})
	out, err := reservePlatformIdempotency(ctx, action("first"), false, "next")
	if err != nil || !out.IdempotencyReplayed || !strings.Contains(out.ResultJSON, "created-id") {
		t.Fatalf("replay=%+v err=%v", out, err)
	}
	if _, err := reservePlatformIdempotency(ctx, action("changed"), false, "req"); err == nil {
		t.Fatal("completed key accepted changed payload")
	}
}

func TestIdempotencyRejectsOversizeBeforeSession(t *testing.T) {
	_, out, err := handlePlatformGenericWrite(context.Background(), "create", PlatformGenericWriteInput{IdempotencyKey: strings.Repeat("a", 201)}, "create", true)
	if err != nil || out.ErrorCode != "invalid_input" {
		t.Fatalf("%+v %v", out, err)
	}
}

func TestPlatformStateRequiredForPlansAndWritesOnly(t *testing.T) {
	for name, want := range map[string]bool{"whodb_platform_function_wait": false, "whodb_platform_transform_wait": false, "whodb_platform_sources": false, "whodb_platform_workflow_plan": true, "whodb_platform_workflow_get": true, "whodb_platform_pending": true, "whodb_platform_confirm": true, "unknown": true} {
		if got := PlatformToolRequiresState(name); got != want {
			t.Fatalf("%s: %v", name, got)
		}
	}
}

func TestPendingListPreservesExpiredInFlightReservation(t *testing.T) {
	state := NewPlatformState()
	ctx := WithPlatformRuntime(context.Background(), &PlatformRuntime{State: state})
	action := &PendingPlatformAction{IdempotencyKey: "key", Mutation: "CreateFunction", Variables: map[string]any{"name": "first"}}
	if _, err := reservePlatformIdempotency(ctx, action, false, "req"); err != nil {
		t.Fatal(err)
	}
	action.ExpiresAt = time.Now().Add(-time.Minute)
	if pending := listPendingPlatformActions(ctx); len(pending) != 0 {
		t.Fatal("expired approval still displayed")
	}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	restored := NewPlatformState()
	if err := json.Unmarshal(raw, restored); err != nil {
		t.Fatal(err)
	}
	ctx = WithPlatformRuntime(context.Background(), &PlatformRuntime{State: restored})
	retry := &PendingPlatformAction{IdempotencyKey: "key", Mutation: "CreateFunction", Variables: map[string]any{"name": "first"}}
	if _, err := reservePlatformIdempotency(ctx, retry, false, "retry"); err == nil {
		t.Fatal("uncertain operation was allowed to execute again")
	}
}
