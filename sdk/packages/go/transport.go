package whodb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Transport executes one platform operation and returns the GraphQL data
// object. Implementations: the default HTTP transport and IpcTransport
// (inside the WhoDB Functions runtime). The generated core and facades never
// speak HTTP directly.
type Transport interface {
	Execute(ctx context.Context, operation, document string, variables map[string]any) (map[string]any, error)
}

var retryableStatus = map[int]bool{502: true, 503: true, 504: true}

// httpTransport is the default GraphQL-over-HTTP transport (POST /api/query).
type httpTransport struct {
	endpoint    string
	credentials CredentialProvider
	client      *http.Client

	mu        sync.RWMutex
	orgID     string
	projectID string
}

func newHTTPTransport(host string, credentials CredentialProvider) *httpTransport {
	return &httpTransport{
		endpoint:    stripTrailingSlash(host) + "/api/query",
		credentials: credentials,
		client:      http.DefaultClient,
	}
}

func stripTrailingSlash(host string) string {
	for len(host) > 0 && host[len(host)-1] == '/' {
		host = host[:len(host)-1]
	}
	return host
}

// setWorkspace sets the workspace scope headers used on subsequent requests.
func (t *httpTransport) setWorkspace(orgID, projectID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.orgID = orgID
	t.projectID = projectID
}

func (t *httpTransport) post(ctx context.Context, body []byte) (*http.Response, error) {
	token, err := t.credentials.Token(ctx)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("User-Agent", "clidey-whodb-go/"+SDKVersion)
	t.mu.RLock()
	if t.orgID != "" {
		request.Header.Set("X-Whodb-Org-Id", t.orgID)
	}
	if t.projectID != "" {
		request.Header.Set("X-Whodb-Project-Id", t.projectID)
	}
	t.mu.RUnlock()
	return t.client.Do(request)
}

// Execute performs one operation with a single 401-refresh and 5xx retry.
func (t *httpTransport) Execute(ctx context.Context, operation, document string, variables map[string]any) (map[string]any, error) {
	body, err := json.Marshal(map[string]any{"query": document, "variables": variables})
	if err != nil {
		return nil, err
	}
	response, err := t.post(ctx, body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode == http.StatusUnauthorized {
		_ = response.Body.Close()
		t.credentials.Refresh()
		if response, err = t.post(ctx, body); err != nil {
			return nil, err
		}
	} else if retryableStatus[response.StatusCode] {
		_ = response.Body.Close()
		if response, err = t.post(ctx, body); err != nil {
			return nil, err
		}
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("%w — check your API key or run: whodb login", ErrAuth)
	}
	if response.StatusCode >= 400 {
		return nil, &PlatformError{Message: fmt.Sprintf("platform request failed with HTTP %d", response.StatusCode), Code: fmt.Sprintf("HTTP_%d", response.StatusCode)}
	}
	var payload struct {
		Data   map[string]any `json:"data"`
		Errors []graphQLError `json:"errors"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, &PlatformError{Message: "invalid response for " + operation, Code: "INVALID_RESPONSE"}
	}
	if len(payload.Errors) > 0 {
		return nil, mapGraphQLErrors(payload.Errors)
	}
	if payload.Data == nil {
		return nil, &PlatformError{Message: "empty response for " + operation, Code: "EMPTY_RESPONSE"}
	}
	return payload.Data, nil
}
