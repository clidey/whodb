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
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRequireBearerToken(t *testing.T) {
	var reached bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	handler := requireBearerToken("s3cret", next)

	cases := []struct {
		name       string
		header     string
		wantStatus int
		wantReach  bool
	}{
		{"valid token", "Bearer s3cret", http.StatusOK, true},
		{"missing header", "", http.StatusUnauthorized, false},
		{"wrong token", "Bearer nope", http.StatusUnauthorized, false},
		{"missing bearer prefix", "s3cret", http.StatusUnauthorized, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reached = false
			req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if reached != tc.wantReach {
				t.Errorf("handler reached = %v, want %v", reached, tc.wantReach)
			}
		})
	}
}

func TestHTTPBindRequiresNetworkAuthentication(t *testing.T) {
	for _, host := range []string{"0.0.0.0", "::", "192.168.1.2", "example.test"} {
		if err := validateHTTPBind(host, ""); err == nil {
			t.Errorf("accepted unauthenticated %s", host)
		}
		if err := validateHTTPBind(host, " "); err == nil {
			t.Errorf("accepted blank token for %s", host)
		}
		if err := validateHTTPBind(host, "test-secret"); err != nil {
			t.Fatal(err)
		}
	}
	for _, host := range []string{"localhost", "127.0.0.1", "127.0.0.2", "::1"} {
		if err := validateHTTPBind(host, ""); err != nil {
			t.Errorf("loopback %s: %v", host, err)
		}
	}
	if err := RunHTTP(context.Background(), nil, &HTTPOptions{Host: "0.0.0.0"}, nil); err == nil {
		t.Fatal("public listener started without authentication")
	}
}

func TestHTTPTransportAuthenticatesRealRequests(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "test", Version: "1"}, nil)
	go func() {
		done <- RunHTTP(ctx, server, &HTTPOptions{Host: "127.0.0.1", Port: port, AuthToken: "secret"}, nil)
	}()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	})
	endpoint := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := &http.Client{Timeout: time.Second}
	for deadline := time.Now().Add(5 * time.Second); ; {
		res, err := client.Get(endpoint + "/health")
		if err == nil {
			res.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	for _, token := range []string{"", "wrong", "secret"} {
		req, err := http.NewRequest(http.MethodPost, endpoint+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		want := http.StatusUnauthorized
		if token == "secret" {
			want = http.StatusOK
		}
		if res.StatusCode != want {
			t.Errorf("token %q status=%d, want %d", token, res.StatusCode, want)
		}
	}
}
