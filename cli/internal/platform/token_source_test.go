/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package platform

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type testRefreshTokenStore struct {
	mu    sync.Mutex
	token string
}

func (s *testRefreshTokenStore) GetPlatformRefreshToken(string, string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token, nil
}

func (s *testRefreshTokenStore) SavePlatformRefreshToken(_, _, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = token
	return nil
}

func resetAccessTokenCache(t *testing.T) {
	t.Helper()
	accessTokenCacheMu.Lock()
	accessTokenCache = map[string]accessTokenEntry{}
	accessTokenCacheMu.Unlock()
	t.Cleanup(func() {
		accessTokenCacheMu.Lock()
		accessTokenCache = map[string]accessTokenEntry{}
		accessTokenCacheMu.Unlock()
	})
}

func newOIDCTestServer(t *testing.T, expiresIn int) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var server *httptest.Server
	var refreshes atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth-config", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"version":2,"issuer":%q,"clientId":"whodb","flows":{"authorizationCodePkce":true,"deviceAuthorization":true}}`, server.URL+"/realms/mothergate")
	})
	mux.HandleFunc("/realms/mothergate/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"issuer":%q,"authorization_endpoint":%q,"token_endpoint":%q,"device_authorization_endpoint":%q,"revocation_endpoint":%q}`, server.URL+"/realms/mothergate", server.URL+"/authorize", server.URL+"/token", server.URL+"/device", server.URL+"/revoke")
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		count := refreshes.Add(1)
		_, _ = fmt.Fprintf(w, `{"access_token":"access-%d","refresh_token":"rotated-%d","expires_in":%d}`, count, count, expiresIn)
	})
	server = httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, &refreshes
}

func TestOIDCTokenSourceCachesAndPersistsRotation(t *testing.T) {
	resetAccessTokenCache(t)
	server, refreshes := newOIDCTestServer(t, 900)
	store := &testRefreshTokenStore{token: "refresh-1"}
	source := NewOIDCTokenSource(server.URL, "account-1", store)

	first, err := source.Token(context.Background())
	if err != nil {
		t.Fatalf("first Token() error = %v", err)
	}
	second, err := source.Token(context.Background())
	if err != nil {
		t.Fatalf("second Token() error = %v", err)
	}
	if first != "access-1" || second != first {
		t.Fatalf("tokens = %q, %q; want one cached access token", first, second)
	}
	if got := refreshes.Load(); got != 1 {
		t.Fatalf("refreshes = %d, want 1", got)
	}
	if store.token != "rotated-1" {
		t.Fatalf("stored refresh token = %q, want rotated-1", store.token)
	}
}

func TestOIDCTokenSourceExpiresAndRefreshes(t *testing.T) {
	resetAccessTokenCache(t)
	server, refreshes := newOIDCTestServer(t, 1)
	store := &testRefreshTokenStore{token: "refresh-1"}
	source := NewOIDCTokenSource(server.URL, "account-2", store)

	if _, err := source.Token(context.Background()); err != nil {
		t.Fatalf("first Token() error = %v", err)
	}
	time.Sleep(1100 * time.Millisecond)
	if _, err := source.Token(context.Background()); err != nil {
		t.Fatalf("second Token() error = %v", err)
	}
	if got := refreshes.Load(); got != 2 {
		t.Fatalf("refreshes = %d, want 2 after expiry", got)
	}
}
