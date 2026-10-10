package mcp

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

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
