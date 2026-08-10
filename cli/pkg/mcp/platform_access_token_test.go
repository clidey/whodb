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

package mcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clidey/whodb/cli/internal/config"
)

// refreshStub serves the OIDC configuration, discovery, and token endpoints
// platformapi.RefreshToken needs.
type refreshStub struct {
	server    *httptest.Server
	calls     atomic.Int64
	expiresIn int
	status    int
}

func newRefreshStub(t *testing.T, expiresIn int) *refreshStub {
	t.Helper()
	stub := &refreshStub{expiresIn: expiresIn, status: http.StatusOK}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth-config", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"version":2,"issuer":%q,"clientId":"whodb-cli","flows":{"authorizationCodePkce":true,"deviceAuthorization":true}}`, stub.server.URL+"/realms/mothergate")
	})
	mux.HandleFunc("/realms/mothergate/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"issuer":%q,"authorization_endpoint":%q,"token_endpoint":%q}`, stub.server.URL+"/realms/mothergate", stub.server.URL+"/authorize", stub.server.URL+"/token")
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		n := stub.calls.Add(1)
		if stub.status != http.StatusOK {
			w.WriteHeader(stub.status)
			fmt.Fprint(w, `{"error":"invalid_grant"}`)
			return
		}
		// A distinct access token per call makes cache hits observable.
		fmt.Fprintf(w, `{"access_token":"access-%d","expires_in":%d}`, n, stub.expiresIn)
	})
	stub.server = httptest.NewServer(mux)
	t.Cleanup(stub.server.Close)
	return stub
}

// resetPlatformAccessTokenCache clears cache state so each test starts clean.
func resetPlatformAccessTokenCache(t *testing.T) {
	t.Helper()
	reset := func() {
		platformAccessTokenMutex.Lock()
		defer platformAccessTokenMutex.Unlock()
		platformAccessTokens = map[string]platformAccessTokenEntry{}
	}
	reset()
	t.Cleanup(reset)
}

func TestPlatformAccessTokenCachesWithinExpiry(t *testing.T) {
	resetPlatformAccessTokenCache(t)
	stub := newRefreshStub(t, 900)
	cfg := &config.Config{}

	first, err := platformAccessToken(context.Background(), cfg, stub.server.URL, "acct-1", "refresh-1")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if first != "access-1" {
		t.Fatalf("expected access-1, got %q", first)
	}

	// 20 further loads stand in for a long MCP session's tool calls.
	for i := 0; i < 20; i++ {
		got, err := platformAccessToken(context.Background(), cfg, stub.server.URL, "acct-1", "refresh-1")
		if err != nil {
			t.Fatalf("cached call %d: %v", i, err)
		}
		if got != first {
			t.Fatalf("cached call %d returned %q, want %q", i, got, first)
		}
	}

	if got := stub.calls.Load(); got != 1 {
		t.Fatalf("expected exactly 1 network refresh, got %d", got)
	}
}

func TestPlatformAccessTokenRefreshesWhenWithinSkew(t *testing.T) {
	resetPlatformAccessTokenCache(t)
	// Expiry inside the skew window means the cached token must not be reused.
	stub := newRefreshStub(t, 30)
	cfg := &config.Config{}

	first, err := platformAccessToken(context.Background(), cfg, stub.server.URL, "acct-1", "refresh-1")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	second, err := platformAccessToken(context.Background(), cfg, stub.server.URL, "acct-1", "refresh-1")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if first == second {
		t.Fatalf("expected a fresh token inside the skew window, got %q twice", first)
	}
	if got := stub.calls.Load(); got != 2 {
		t.Fatalf("expected 2 refreshes, got %d", got)
	}
}

func TestPlatformAccessTokenDoesNotCacheWithoutExpiresIn(t *testing.T) {
	resetPlatformAccessTokenCache(t)
	// expires_in omitted: lifetime is unknown, so every call must refresh.
	stub := newRefreshStub(t, 0)
	cfg := &config.Config{}

	for i := 0; i < 3; i++ {
		if _, err := platformAccessToken(context.Background(), cfg, stub.server.URL, "acct-1", "refresh-1"); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if got := stub.calls.Load(); got != 3 {
		t.Fatalf("expected 3 refreshes without expires_in, got %d", got)
	}
}

func TestPlatformAccessTokenSeparatesAccounts(t *testing.T) {
	resetPlatformAccessTokenCache(t)
	stub := newRefreshStub(t, 900)
	cfg := &config.Config{}

	a, err := platformAccessToken(context.Background(), cfg, stub.server.URL, "acct-1", "refresh-1")
	if err != nil {
		t.Fatalf("acct-1: %v", err)
	}
	b, err := platformAccessToken(context.Background(), cfg, stub.server.URL, "acct-2", "refresh-2")
	if err != nil {
		t.Fatalf("acct-2: %v", err)
	}
	if a == b {
		t.Fatal("expected per-account tokens, got the same token for both accounts")
	}
	if got := stub.calls.Load(); got != 2 {
		t.Fatalf("expected 2 refreshes for 2 accounts, got %d", got)
	}
}

func TestPlatformAccessTokenEvictsOnRefreshFailure(t *testing.T) {
	resetPlatformAccessTokenCache(t)
	stub := newRefreshStub(t, 900)
	cfg := &config.Config{}

	if _, err := platformAccessToken(context.Background(), cfg, stub.server.URL, "acct-1", "refresh-1"); err != nil {
		t.Fatalf("seed call: %v", err)
	}

	// Force the cached entry stale, then make the refresh fail.
	platformAccessTokenMutex.Lock()
	platformAccessTokens[stub.server.URL+"\x00acct-1"] = platformAccessTokenEntry{
		accessToken: "stale",
		expiresAt:   time.Now().Add(-time.Hour),
	}
	platformAccessTokenMutex.Unlock()
	stub.status = http.StatusBadRequest

	if _, err := platformAccessToken(context.Background(), cfg, stub.server.URL, "acct-1", "refresh-1"); err == nil {
		t.Fatal("expected an error when refresh fails")
	}

	// The stale token must be gone rather than served to later callers.
	platformAccessTokenMutex.Lock()
	_, cached := platformAccessTokens[stub.server.URL+"\x00acct-1"]
	platformAccessTokenMutex.Unlock()
	if cached {
		t.Fatal("expected the cache entry to be evicted after a failed refresh")
	}
}

// TestPlatformAccessTokenConcurrentCallersCoalesce covers the race the MCP SDK
// creates by dispatching tool calls through jsonrpc2.Async: concurrent loads
// must produce exactly one refresh, not one per caller.
func TestPlatformAccessTokenConcurrentCallersCoalesce(t *testing.T) {
	resetPlatformAccessTokenCache(t)
	stub := newRefreshStub(t, 900)
	cfg := &config.Config{}

	const callers = 25
	var wg sync.WaitGroup
	tokens := make([]string, callers)
	errs := make([]error, callers)
	start := make(chan struct{})

	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			tokens[idx], errs[idx] = platformAccessToken(context.Background(), cfg, stub.server.URL, "acct-1", "refresh-1")
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("caller %d: %v", i, err)
		}
	}
	for i, tok := range tokens {
		if tok != tokens[0] {
			t.Fatalf("caller %d got %q, want %q — callers did not coalesce", i, tok, tokens[0])
		}
	}
	if got := stub.calls.Load(); got != 1 {
		t.Fatalf("expected 1 refresh across %d concurrent callers, got %d", callers, got)
	}
}
