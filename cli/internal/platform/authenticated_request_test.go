package platform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthenticatedRequestStaysOnHost(t *testing.T) {
	foreignCalls := 0
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { foreignCalls++; w.WriteHeader(http.StatusOK) }))
	defer foreign.Close()
	var authorization, org string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		org = r.Header.Get("X-Whodb-Org-Id")
		http.Redirect(w, r, foreign.URL, http.StatusFound)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "test-token")
	if err != nil {
		t.Fatal(err)
	}
	client.SetWorkspaceContext("org-id", "")
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/download", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.DoAuthenticated(request); err == nil {
		t.Fatal("cross-host redirect was followed")
	}
	if authorization != "Bearer test-token" || org != "org-id" || foreignCalls != 0 {
		t.Fatalf("auth=%q org=%q foreignCalls=%d", authorization, org, foreignCalls)
	}
	foreignRequest, err := http.NewRequestWithContext(context.Background(), http.MethodGet, foreign.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.DoAuthenticated(foreignRequest); err == nil {
		t.Fatal("cross-host request accepted")
	}
}
