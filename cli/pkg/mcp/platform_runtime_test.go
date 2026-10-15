package mcp

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func runtimeTestClient(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	ct, st := mcp.NewInMemoryTransports()
	ss, err := server.Connect(t.Context(), st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "runtime-test", Version: "1"}, nil)
	cs, err := client.Connect(t.Context(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func TestEmbeddedPlatformCatalogMatchesCLI(t *testing.T) {
	for _, readOnly := range []bool{false, true} {
		cli := runtimeTestClient(t, NewServer(&ServerOptions{PlatformEnabled: true, ReadOnly: readOnly, ConfirmWrites: true, MaxRows: 100, DisabledTools: []string{"whodb_platform_app_screenshot"}}))
		embedded := runtimeTestClient(t, NewPlatformServer(readOnly, "runtime-test"))
		catalog := func(client *mcp.ClientSession) map[string]string {
			out := map[string]string{}
			for tool, err := range client.Tools(t.Context(), nil) {
				if err != nil {
					t.Fatal(err)
				}
				if !strings.HasPrefix(tool.Name, "whodb_platform_") {
					continue
				}
				data, err := json.Marshal(tool)
				if err != nil {
					t.Fatal(err)
				}
				out[tool.Name] = string(data)
			}
			return out
		}
		a, b := catalog(cli), catalog(embedded)
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("catalog drift: CLI=%d embedded=%d readOnly=%v", len(a), len(b), readOnly)
		}
		t.Logf("readOnly=%v: %d identical platform tools", readOnly, len(a))
	}
}

func TestEmbeddedStateIsolationAndPersistence(t *testing.T) {
	a, b := NewPlatformState(), NewPlatformState()
	ctxA := WithPlatformRuntime(context.Background(), &PlatformRuntime{State: a})
	ctxB := WithPlatformRuntime(context.Background(), &PlatformRuntime{State: b})
	token, _ := storePendingPlatformAction(&PendingPlatformAction{AccountID: "alice", IdempotencyKey: "same-key"}, ctxA)
	if _, err := getPendingPlatformAction(token, ctxB); err == nil {
		t.Fatal("cross-user confirmation")
	}
	if _, ok := pendingPlatformIdempotencyToken("same-key", ctxB); ok {
		t.Fatal("cross-user idempotency")
	}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	restored := NewPlatformState()
	if err := json.Unmarshal(data, restored); err != nil {
		t.Fatal(err)
	}
	ctxA = WithPlatformRuntime(context.Background(), &PlatformRuntime{State: restored})
	checkpointed := false
	restored.Checkpoint = func() error { checkpointed = true; return nil }
	if _, err := getPendingPlatformAction(token, ctxA); err != nil {
		t.Fatal(err)
	}
	if !checkpointed {
		t.Fatal("execution claim was not persisted")
	}
	data, err = json.Marshal(restored)
	if err != nil {
		t.Fatal(err)
	}
	restarted := NewPlatformState()
	if err := json.Unmarshal(data, restarted); err != nil {
		t.Fatal(err)
	}
	if _, err := getPendingPlatformAction(token, WithPlatformRuntime(context.Background(), &PlatformRuntime{State: restarted})); err == nil {
		t.Fatal("in-flight action replayed after restart")
	}
	consumePendingPlatformAction(token, ctxA)
	if !platformIdempotencyCompleted("same-key", ctxA) || platformIdempotencyCompleted("same-key", ctxB) {
		t.Fatal("completion isolation")
	}
}

func TestHostedApprovalOwnershipAndExpiry(t *testing.T) {
	if PlatformToolRequiresWrite("whodb_platform_orgs") || !PlatformToolRequiresWrite("whodb_platform_project_create") || !PlatformToolRequiresWrite("whodb_platform_pending") || !PlatformToolRequiresWrite("unrecognized_tool") {
		t.Fatal("hosted write scope classification failed")
	}
	now := time.Now()
	state := NewPlatformState()
	state.Pending["active"] = &PendingPlatformAction{AccountID: "alice", ExpiresAt: now.Add(time.Minute)}
	state.Pending["expired"] = &PendingPlatformAction{AccountID: "alice", ExpiresAt: now.Add(-time.Minute)}
	state.IdempotencyPending["expired-key"] = "expired"
	state.IdempotencyDone["old-key"] = now.Add(-31 * 24 * time.Hour)
	state.Workflows = []platformWorkflowPlan{{ID: "old", UpdatedAt: now.Add(-31 * 24 * time.Hour)}}
	if state.DecidePendingApproval("bob", "active", true) || state.DecidePendingApproval("alice", "expired", true) {
		t.Fatal("cross-account or expired approval accepted")
	}
	if !state.DecidePendingApproval("alice", "active", true) || !state.Pending["active"].BrowserApproved {
		t.Fatal("owner approval not recorded")
	}
	state.PruneExpired(now)
	if state.Pending["expired"] != nil || state.IdempotencyPending["expired-key"] != "" || !state.IdempotencyDone["old-key"].IsZero() || len(state.Workflows) != 0 {
		t.Fatal("expired sensitive action retained")
	}
	if state.NextCleanupAt().IsZero() {
		t.Fatal("active action has no cleanup deadline")
	}
	if !state.DecidePendingApproval("alice", "active", false) || state.Pending["active"] != nil || !state.NextCleanupAt().IsZero() {
		t.Fatal("rejected action retained")
	}
}

func TestHostedCheckpointCanCalculateExpiryDuringClaim(t *testing.T) {
	state := NewPlatformState()
	state.Pending["token"] = &PendingPlatformAction{AccountID: "alice", ExpiresAt: time.Now().Add(time.Minute)}
	state.Checkpoint = func() error {
		if state.NextCleanupAt().IsZero() {
			t.Fatal("checkpoint lost cleanup deadline")
		}
		return nil
	}
	ctx := WithPlatformRuntime(context.Background(), &PlatformRuntime{State: state})
	if _, err := getPendingPlatformAction("token", ctx); err != nil {
		t.Fatal(err)
	}
}

func TestEmbeddedRuntimeIgnoresLocalDefaults(t *testing.T) {
	t.Setenv("WHODB_PLATFORM_HOST", "https://elsewhere.example")
	ctx := WithPlatformRuntime(t.Context(), &PlatformRuntime{Host: "https://own.example", State: NewPlatformState()})
	scope, explicit, err := platformRequestScope(ctx)
	if err != nil || !explicit || scope.Host != "https://own.example" || scope.Org != "" || scope.Project != "" {
		t.Fatal(scope, explicit, err)
	}
	ctx = context.WithValue(ctx, platformRequestKey{}, &platformRequest{Target: &PlatformWorkspaceTarget{Host: "https://elsewhere.example"}})
	if _, _, err := platformRequestScope(ctx); err == nil {
		t.Fatal("allowed remote host override")
	}
}
