package whodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// recordedRequest captures what the stub server saw for one request.
type recordedRequest struct {
	headers   http.Header
	operation string
}

// scriptedServer replays canned HTTP responses in order and records requests.
type scriptedServer struct {
	mu        sync.Mutex
	responses []func(w http.ResponseWriter)
	requests  []recordedRequest
	server    *httptest.Server
}

func newScriptedServer(t *testing.T, responses ...func(w http.ResponseWriter)) *scriptedServer {
	t.Helper()
	s := &scriptedServer{responses: responses}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query string `json:"query"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		operation := ""
		if fields := strings.Fields(body.Query); len(fields) >= 2 {
			operation = fields[1]
			if index := strings.IndexAny(operation, "({"); index > 0 {
				operation = operation[:index]
			}
		}
		s.mu.Lock()
		s.requests = append(s.requests, recordedRequest{headers: r.Header.Clone(), operation: operation})
		index := len(s.requests) - 1
		s.mu.Unlock()
		if index < len(s.responses) {
			s.responses[index](w)
			return
		}
		t.Errorf("unexpected request #%d", index+1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(s.server.Close)
	return s
}

func jsonResponse(status int, body string) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

// countingCredentials counts Refresh calls and rotates its token afterwards.
type countingCredentials struct {
	mu       sync.Mutex
	token    string
	refreshs int
}

func (c *countingCredentials) Token(context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.token, nil
}

func (c *countingCredentials) Refresh() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.refreshs++
	c.token = fmt.Sprintf("token-%d", c.refreshs)
}

func TestHTTPTransportSendsHeaders(t *testing.T) {
	stub := newScriptedServer(t, jsonResponse(200, `{"data":{"Op":true}}`))
	transport := newHTTPTransport(stub.server.URL, staticCredentials{value: "tok"})
	transport.setWorkspace("org-1", "proj-1")
	if _, err := transport.Execute(context.Background(), "Op", "query Op { Op }", nil); err != nil {
		t.Fatal(err)
	}
	headers := stub.requests[0].headers
	if got := headers.Get("Authorization"); got != "Bearer tok" {
		t.Errorf("Authorization: got %q", got)
	}
	if got := headers.Get("User-Agent"); got != "clidey-whodb-go/"+resolvedVersion() {
		t.Errorf("User-Agent: got %q", got)
	}
	if headers.Get("X-Whodb-Org-Id") != "org-1" || headers.Get("X-Whodb-Project-Id") != "proj-1" {
		t.Errorf("workspace headers: got %q / %q", headers.Get("X-Whodb-Org-Id"), headers.Get("X-Whodb-Project-Id"))
	}
}

func TestHTTPTransportRetriesOn5xx(t *testing.T) {
	stub := newScriptedServer(t,
		jsonResponse(503, `oops`),
		jsonResponse(200, `{"data":{"Op":42}}`),
	)
	transport := newHTTPTransport(stub.server.URL, staticCredentials{value: "tok"})
	data, err := transport.Execute(context.Background(), "Op", "query Op { Op }", nil)
	if err != nil {
		t.Fatal(err)
	}
	if data["Op"] != float64(42) {
		t.Errorf("result: got %v", data["Op"])
	}
	if len(stub.requests) != 2 {
		t.Errorf("expected exactly one retry, saw %d requests", len(stub.requests))
	}
}

func TestHTTPTransportRefreshesOn401(t *testing.T) {
	stub := newScriptedServer(t,
		jsonResponse(401, `{}`),
		jsonResponse(200, `{"data":{"Op":true}}`),
	)
	credentials := &countingCredentials{token: "stale"}
	transport := newHTTPTransport(stub.server.URL, credentials)
	if _, err := transport.Execute(context.Background(), "Op", "query Op { Op }", nil); err != nil {
		t.Fatal(err)
	}
	if credentials.refreshs != 1 {
		t.Errorf("Refresh calls: got %d, want 1", credentials.refreshs)
	}
	if got := stub.requests[1].headers.Get("Authorization"); got != "Bearer token-1" {
		t.Errorf("retried request must carry the refreshed token, got %q", got)
	}
}

func TestHTTPTransportPersistent401IsAuthError(t *testing.T) {
	stub := newScriptedServer(t, jsonResponse(401, `{}`), jsonResponse(401, `{}`))
	transport := newHTTPTransport(stub.server.URL, &countingCredentials{token: "bad"})
	_, err := transport.Execute(context.Background(), "Op", "query Op { Op }", nil)
	if !errors.Is(err, ErrAuth) {
		t.Errorf("persistent 401 must map to ErrAuth, got %v", err)
	}
}

func TestHTTPTransportMapsGraphQLErrors(t *testing.T) {
	stub := newScriptedServer(t,
		jsonResponse(200, `{"errors":[{"message":"nope","extensions":{"code":"NOT_FOUND"}}]}`),
	)
	transport := newHTTPTransport(stub.server.URL, staticCredentials{value: "tok"})
	_, err := transport.Execute(context.Background(), "Op", "query Op { Op }", nil)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("NOT_FOUND must map to ErrNotFound, got %v", err)
	}
}

func clearClientEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{"WHODB_API_KEY", "WHODB_ORG", "WHODB_PROJECT", "WHODB_HOST", "WHODB_IPC_TOKEN"} {
		t.Setenv(name, "")
	}
}

func TestClientResolvesWorkspaceSlugs(t *testing.T) {
	clearClientEnv(t)
	stub := newScriptedServer(t,
		jsonResponse(200, `{"data":{"MyOrganizations":[{"id":"11111111-1111-1111-1111-111111111111","slug":"acme"}]}}`),
		jsonResponse(200, `{"data":{"Projects":[{"id":"22222222-2222-2222-2222-222222222222","slug":"analytics"}]}}`),
		jsonResponse(200, `{"data":{"OntologyEntities":[]}}`),
	)
	client, err := New(Config{Token: "tok", Org: "acme", Project: "analytics", Host: stub.server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.OntologyEntities(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := []string{stub.requests[0].operation, stub.requests[1].operation, stub.requests[2].operation}; got[0] != "MyOrganizations" || got[1] != "Projects" || got[2] != "OntologyEntities" {
		t.Errorf("operation order: got %v", got)
	}
	final := stub.requests[2].headers
	if final.Get("X-Whodb-Org-Id") != "11111111-1111-1111-1111-111111111111" ||
		final.Get("X-Whodb-Project-Id") != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("resolved workspace headers: got %q / %q", final.Get("X-Whodb-Org-Id"), final.Get("X-Whodb-Project-Id"))
	}
}

func TestClientDiscoversWorkspaceViaMyWorkspace(t *testing.T) {
	clearClientEnv(t)
	stub := newScriptedServer(t,
		jsonResponse(200, `{"data":{"MyWorkspace":{"orgId":"33333333-3333-3333-3333-333333333333","projectId":"44444444-4444-4444-4444-444444444444"}}}`),
		jsonResponse(200, `{"data":{"OntologyEntities":[]}}`),
	)
	client, err := New(Config{APIKey: "whodb_sk_test", Host: stub.server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.OntologyEntities(context.Background()); err != nil {
		t.Fatal(err)
	}
	if stub.requests[0].operation != "MyWorkspace" {
		t.Errorf("first operation must be MyWorkspace, got %q", stub.requests[0].operation)
	}
	final := stub.requests[1].headers
	if final.Get("X-Whodb-Project-Id") != "44444444-4444-4444-4444-444444444444" {
		t.Errorf("discovered project header: got %q", final.Get("X-Whodb-Project-Id"))
	}
}

func TestClientMultiGrantKeyRequiresProject(t *testing.T) {
	clearClientEnv(t)
	stub := newScriptedServer(t,
		jsonResponse(200, `{"data":{"MyWorkspace":{"orgId":"33333333-3333-3333-3333-333333333333","projectId":null}}}`),
	)
	client, err := New(Config{APIKey: "whodb_sk_test", Host: stub.server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.OntologyEntities(context.Background())
	if !errors.Is(err, ErrValidation) {
		t.Errorf("null projectId must map to ErrValidation, got %v", err)
	}
}
