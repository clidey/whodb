package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

func TestAutomationMCPRejectsScopeChangesBeforeSendingCredential(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests.Add(1) }))
	defer server.Close()
	t.Setenv(platformapi.APIKeyEnv, "test-process-credential")
	t.Setenv(platformapi.SessionHostEnv, server.URL)
	t.Setenv(platformapi.SessionOrgEnv, "11111111-1111-4111-8111-111111111111")
	t.Setenv(platformapi.SessionProjectEnv, "22222222-2222-4222-8222-222222222222")
	for _, scope := range []platformapi.SessionScope{
		{Host: "https://different.example"},
		{Host: server.URL, Org: "33333333-3333-4333-8333-333333333333"},
		{Host: server.URL, Project: "33333333-3333-4333-8333-333333333333"},
	} {
		if _, err := loadAutomationPlatformToolSession(context.Background(), scope); err == nil {
			t.Fatalf("scope change accepted: %+v", scope)
		}
	}
	if requests.Load() != 0 {
		t.Fatal("credential sent before scope validation")
	}
	t.Setenv(platformapi.APIKeyEnv, "")
	if _, err := loadAutomationPlatformToolSession(context.Background(), platformapi.SessionScope{Host: server.URL}); err == nil {
		t.Fatal("empty credential accepted")
	}
	if requests.Load() != 0 {
		t.Fatal("empty credential sent")
	}
}
