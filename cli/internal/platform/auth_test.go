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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestDeviceAuthorizationUsesPKCE(t *testing.T) {
	var challenge string
	var authServer *httptest.Server
	authServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/mothergate/.well-known/openid-configuration":
			_, _ = w.Write([]byte(`{"issuer":"` + authServer.URL + `/realms/mothergate","authorization_endpoint":"` + authServer.URL + `/authorize","token_endpoint":"` + authServer.URL + `/token","device_authorization_endpoint":"` + authServer.URL + `/device"}`))
		case "/device":
			_ = r.ParseForm()
			challenge = r.Form.Get("code_challenge")
			if challenge == "" || r.Form.Get("code_challenge_method") != "S256" {
				t.Fatal("device authorization request did not include S256 PKCE")
			}
			_, _ = w.Write([]byte(`{"device_code":"device-code","user_code":"ABCD-EFGH","verification_uri":"` + authServer.URL + `/verify","expires_in":30,"interval":1}`))
		case "/token":
			_ = r.ParseForm()
			if got := oauth2.S256ChallengeFromVerifier(r.Form.Get("code_verifier")); got != challenge {
				t.Fatalf("token code_verifier challenge = %q, want %q", got, challenge)
			}
			_, _ = w.Write([]byte(`{"access_token":"access-token","refresh_token":"refresh-token"}`))
		default:
			t.Fatalf("unexpected auth path %q", r.URL.Path)
		}
	}))
	defer authServer.Close()

	platformServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":2,"issuer":"` + authServer.URL + `/realms/mothergate","clientId":"whodb","flows":{"authorizationCodePkce":true,"deviceAuthorization":true}}`))
	}))
	defer platformServer.Close()

	tokens, err := Login(context.Background(), LoginOptions{Host: platformServer.URL, UseDeviceAuthorization: true, Timeout: 3 * time.Second})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if tokens.AccessToken != "access-token" || tokens.RefreshToken != "refresh-token" {
		t.Fatalf("Login() tokens = %#v", tokens)
	}
}

func TestLogoutPostsBearerTokenToAuthHost(t *testing.T) {
	var gotToken string
	var authServer *httptest.Server
	authServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/mothergate/.well-known/openid-configuration":
			_, _ = w.Write([]byte(`{"issuer":"` + authServer.URL + `/realms/mothergate","authorization_endpoint":"` + authServer.URL + `/authorize","token_endpoint":"` + authServer.URL + `/token","revocation_endpoint":"` + authServer.URL + `/revoke"}`))
		case "/revoke":
			_ = r.ParseForm()
			gotToken = r.Form.Get("token")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected auth path %q", r.URL.Path)
		}
	}))
	defer authServer.Close()

	platformServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth-config" {
			t.Fatalf("unexpected platform path %q", r.URL.Path)
		}
		authURL, err := json.Marshal(authServer.URL + "/realms/mothergate")
		if err != nil {
			t.Fatalf("marshal auth URL: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":2,"issuer":` + string(authURL) + `,"clientId":"whodb","flows":{"authorizationCodePkce":true,"deviceAuthorization":true}}`))
	}))
	defer platformServer.Close()

	if err := Logout(context.Background(), platformServer.URL, "access-token"); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if gotToken != "access-token" {
		t.Fatalf("token = %q, want access-token", gotToken)
	}
}

func TestIsInvalidGrant(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"refresh token expired"}`))
	}))
	defer authServer.Close()

	_, err := requestTokens(context.Background(), authServer.URL, nil)
	if err == nil {
		t.Fatal("postAuth() error = nil, want auth error")
	}
	if !IsInvalidGrant(err) {
		t.Fatalf("IsInvalidGrant(%v) = false, want true", err)
	}
	if strings.Contains(err.Error(), "refresh token expired") {
		t.Fatalf("auth error leaked server description: %q", err.Error())
	}
}

func TestMCPConnectedURL(t *testing.T) {
	for _, tc := range []struct {
		host string
		want string
	}{
		{host: "https://app.example.com/", want: "https://app.example.com/mcp-connected"},
		{host: "http://localhost:8080", want: "http://localhost:3000/mcp-connected"},
		{host: "http://127.0.0.1:8080", want: "http://127.0.0.1:3000/mcp-connected"},
	} {
		if got := mcpConnectedURL(tc.host); got != tc.want {
			t.Fatalf("mcpConnectedURL(%q) = %q, want %q", tc.host, got, tc.want)
		}
	}
}
