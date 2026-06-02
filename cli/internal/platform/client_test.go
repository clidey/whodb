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

package platform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"default", "", DefaultHost, false},
		{"adds https", "app.whodb.com", "https://app.whodb.com", false},
		{"trims trailing slash", "https://staging.whodb.com/", "https://staging.whodb.com", false},
		{"keeps path prefix", "https://example.com/whodb/", "https://example.com/whodb", false},
		{"allows localhost http", "http://localhost:8080/", "http://localhost:8080", false},
		{"allows ipv4 loopback http", "http://127.0.0.1:8080/", "http://127.0.0.1:8080", false},
		{"allows ipv6 loopback http", "http://[::1]:8080/", "http://[::1]:8080", false},
		{"rejects non-loopback http", "http://staging.whodb.com", "", true},
		{"rejects unsupported scheme", "ftp://example.com", "", true},
		{"rejects missing host", "https://", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeHost(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeHost(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("NormalizeHost(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveAuthHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth-config" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"mothergateUrl":"http://localhost:4000/"}`))
	}))
	defer server.Close()

	got, err := ResolveAuthHost(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("ResolveAuthHost() error = %v", err)
	}
	if got != "http://localhost:4000" {
		t.Fatalf("ResolveAuthHost() = %q, want %q", got, "http://localhost:4000")
	}
}

func TestResolveAuthHostRequiresMothergateURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	if _, err := ResolveAuthHost(context.Background(), server.URL); err == nil {
		t.Fatalf("ResolveAuthHost() error = nil, want error")
	}
}
