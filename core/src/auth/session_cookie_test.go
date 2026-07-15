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

package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/clidey/whodb/core/src/env"
)

func TestSetSessionCookieUsesCECookieNames(t *testing.T) {
	originalSecure := env.Secure
	env.Secure = false
	t.Cleanup(func() { env.Secure = originalSecure })

	tests := []struct {
		name       string
		target     string
		cookieName string
		secure     bool
	}{
		{name: "HTTP", target: "http://example.com", cookieName: "whodb_ce_session"},
		{name: "HTTPS", target: "https://example.com", cookieName: "__Host-whodb_ce_session", secure: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tt.target, nil)

			setSessionCookie(recorder, request, "session-token", time.Now().Add(time.Hour))

			var sessionCookie *http.Cookie
			for _, cookie := range recorder.Result().Cookies() {
				if cookie.Value == "session-token" {
					sessionCookie = cookie
					break
				}
			}
			if sessionCookie == nil {
				t.Fatal("expected a session cookie")
			}
			if sessionCookie.Name != tt.cookieName {
				t.Fatalf("expected cookie %q, got %q", tt.cookieName, sessionCookie.Name)
			}
			if sessionCookie.Secure != tt.secure {
				t.Fatalf("expected Secure=%t, got %t", tt.secure, sessionCookie.Secure)
			}
		})
	}
}

func TestSessionTokenFromRequestIgnoresPreviousCookieNames(t *testing.T) {
	for _, name := range []string{"whodb_session", "__Host-whodb_session"} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
			request.AddCookie(&http.Cookie{Name: name, Value: "other-edition-token"})

			if token, found := sessionTokenFromRequest(request); found {
				t.Fatalf("expected cookie %q to be ignored, got token %q", name, token)
			}
		})
	}
}

func TestClearSessionCookiesLeavesPreviousCookieNamesAlone(t *testing.T) {
	recorder := httptest.NewRecorder()
	clearSessionCookies(recorder)

	cleared := map[string]bool{}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.MaxAge < 0 {
			cleared[cookie.Name] = true
		}
	}

	for _, name := range []string{"whodb_ce_session", "__Host-whodb_ce_session", "whodb_csrf"} {
		if !cleared[name] {
			t.Errorf("expected cookie %q to be cleared", name)
		}
	}
	for _, name := range []string{"whodb_session", "__Host-whodb_session"} {
		if cleared[name] {
			t.Errorf("expected cookie %q to be left untouched", name)
		}
	}
}
