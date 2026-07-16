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
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/clidey/whodb/core/src/env"
)

const (
	// sessionCookieName is the session cookie used over plain HTTP (dev/local).
	sessionCookieName = "whodb_ce_session"
	// secureSessionCookieName is used over HTTPS; the __Host- prefix requires
	// Secure, no Domain, and Path=/, hardening it against subdomain/scheme attacks.
	secureSessionCookieName = "__Host-whodb_ce_session"
	// csrfCookieName holds the CSRF token. It is intentionally readable by JS
	// (not HttpOnly) so the frontend can echo it back in the X-CSRF-Token header
	// (double-submit pattern).
	csrfCookieName = "whodb_csrf"
	// csrfHeaderName is the request header carrying the CSRF token on unsafe
	// cookie-authenticated requests. HTTP header names are case-insensitive; the
	// canonical casing is used so it matches the "X-CSRF-Token" sent by clients.
	csrfHeaderName = "X-Csrf-Token"

	// defaultSessionTTL is the sliding idle timeout when WHODB_SESSION_TTL is unset.
	defaultSessionTTL = 168 * time.Hour
)

// cookieSiteMode is Strict because CE has no cross-site flow that needs
// the session cookie on a top-level navigation (no OAuth callback; external
// deep-links land on the pre-auth login page). Strict maximizes CSRF protection.
const cookieSiteMode = http.SameSiteStrictMode

// sessionTTL returns the configured sliding idle timeout for sessions, parsed
// from WHODB_SESSION_TTL, falling back to defaultSessionTTL when unset or invalid.
func sessionTTL() time.Duration {
	raw := strings.TrimSpace(env.SessionTTL)
	if raw == "" {
		return defaultSessionTTL
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return defaultSessionTTL
	}
	return d
}

// isSecureRequest reports whether the request should be treated as HTTPS, so
// cookies can be marked Secure and use the __Host- prefix.
//
// It intentionally does NOT trust X-Forwarded-Proto: a proxy (e.g. the dev vite
// server) can forward "https" while the actual browser↔server transport is plain
// HTTP, which would make the browser silently drop the Secure cookie and break
// auth. Deployments that terminate TLS at a proxy set WHODB_SECURE=true.
func isSecureRequest(r *http.Request) bool {
	if r != nil && r.TLS != nil {
		return true
	}
	return env.Secure
}

// sessionTokenFromRequest extracts the opaque session token from the request
// cookies, preferring the secure (__Host-) cookie. The bool reports whether a
// token was found.
func sessionTokenFromRequest(r *http.Request) (string, bool) {
	if cookie, err := r.Cookie(secureSessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, true
	}
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, true
	}
	return "", false
}

// setSessionCookie writes the session cookie (HttpOnly) and clears the opposite
// variant so a scheme change does not leave a stale cookie behind.
func setSessionCookie(w http.ResponseWriter, r *http.Request, token string, expiresAt time.Time) {
	secure := isSecureRequest(r)
	name := sessionCookieName
	alternate := secureSessionCookieName
	if secure {
		name = secureSessionCookieName
		alternate = sessionCookieName
	}
	expireCookie(w, alternate)

	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge <= 0 {
		maxAge = -1
	}
	// Secure is set only over TLS so local HTTP dev still works; Strict SameSite
	// + HttpOnly protect the session cookie regardless.
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- Secure is TLS-conditional so local HTTP remains supported; HttpOnly and SameSite are set.
		Name:     name,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: cookieSiteMode,
	})
}

// setCSRFCookie writes the readable CSRF cookie the frontend echoes back in the
// X-CSRF-Token header.
func setCSRFCookie(w http.ResponseWriter, r *http.Request, csrfToken string, expiresAt time.Time) {
	secure := isSecureRequest(r)
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge <= 0 {
		maxAge = -1
	}
	// This cookie is deliberately readable by JS (not HttpOnly) so the frontend
	// can echo it back in the X-Csrf-Token header (double-submit CSRF pattern).
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- double-submit CSRF cookie must be JS-readable; Secure and SameSite are set.
		Name:     csrfCookieName,
		Value:    csrfToken,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: false,
		Secure:   secure,
		SameSite: cookieSiteMode,
	})
}

// clearSessionCookies expires every session-related cookie (both session cookie
// variants and the CSRF cookie).
func clearSessionCookies(w http.ResponseWriter) {
	expireCookie(w, sessionCookieName)
	expireCookie(w, secureSessionCookieName)
	expireCookie(w, csrfCookieName)
}

// expireCookie writes a deletion cookie for the given name. The __Host- prefixed
// cookie must be Secure for the browser to accept (and thus delete) it; the CSRF
// cookie is the only non-HttpOnly one.
func expireCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- deletion cookie mirrors the security attributes of the cookie being removed.
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: name != csrfCookieName,
		Secure:   name == secureSessionCookieName,
		SameSite: cookieSiteMode,
	})
}

// validateCSRF reports whether the request carries an X-CSRF-Token header whose
// SHA-256 hash matches the stored csrfHash, using a constant-time comparison.
func validateCSRF(r *http.Request, csrfHash string) bool {
	provided := strings.TrimSpace(r.Header.Get(csrfHeaderName))
	if provided == "" || csrfHash == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(hashToken(provided)), []byte(csrfHash)) == 1
}

// hashToken returns the hex-encoded SHA-256 of a token. Session and CSRF tokens
// are stored only as hashes so a store leak does not reveal usable values.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
