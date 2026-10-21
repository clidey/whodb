package platform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAutomationClientHostBindingAndRevocation(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer whodb_sk_test" || r.Header.Get(workspaceOrgHeader) != "org" || r.Header.Get(workspaceProjectHeader) != "project" {
			t.Error("credential or workspace missing")
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	t.Setenv(APIKeyEnv, "whodb_sk_test")
	t.Setenv(SessionHostEnv, server.URL)
	t.Setenv(SessionOrgEnv, "org")
	t.Setenv(SessionProjectEnv, "project")
	if _, _, err := AutomationClient("https://other.example"); err == nil {
		t.Fatal("cross-host forwarding accepted")
	}
	client, _, err := AutomationClient("")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Me(context.Background())
	if err == nil || strings.Contains(err.Error(), "whodb_sk_test") {
		t.Fatalf("unsafe or missing error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("revocation attempted refresh/retry: %d", calls)
	}
	t.Setenv(SessionProjectEnv, "")
	if _, _, err := AutomationClient(""); err == nil {
		t.Fatal("missing scope accepted")
	}
	t.Setenv(APIKeyEnv, "")
	if _, _, err := AutomationClient(""); err == nil {
		t.Fatal("empty key fell back to personal login")
	}
}

func TestAutomationClientDoesNotFollowRedirect(t *testing.T) {
	forwarded := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { forwarded = true }))
	defer target.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer origin.Close()
	t.Setenv(APIKeyEnv, "whodb_sk_test")
	t.Setenv(SessionHostEnv, origin.URL)
	t.Setenv(SessionOrgEnv, "org")
	t.Setenv(SessionProjectEnv, "project")
	client, _, err := AutomationClient("")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Me(context.Background()); err == nil {
		t.Fatal("expected redirect rejection")
	}
	if forwarded {
		t.Fatal("forwarded service-account credential to redirect target")
	}
}
