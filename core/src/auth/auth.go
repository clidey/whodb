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
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"strings"
	"sync"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/common/ssl"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

type AuthKey string

type authenticatedSourceKey struct{}
type csrfFailedKey struct{}
type authenticationUnavailableKey struct{}

const (
	AuthKey_Token  AuthKey = "Token"
	AuthKey_Source AuthKey = "SourceCredentials"
)

const maxRequestBodySize = 1024 * 1024      // Limit request body size to 1MB
const maxUploadBodySize = 250 * 1024 * 1024 // Limit multipart upload body size to 250MB

// GetSourceCredentials returns the source-first credentials from the current request context.
func GetSourceCredentials(ctx context.Context) *source.Credentials {
	credentials := ctx.Value(AuthKey_Source)
	if credentials == nil {
		return nil
	}
	return credentials.(*source.Credentials)
}

// GetAuthenticatedSourceCredentials returns credentials only after the request
// passed authentication and CSRF checks, excluding public-operation session hints.
func GetAuthenticatedSourceCredentials(ctx context.Context) *source.Credentials {
	if authenticated, _ := ctx.Value(authenticatedSourceKey{}).(bool); !authenticated {
		return nil
	}
	return GetSourceCredentials(ctx)
}

func isPublicRoute(r *http.Request) bool {
	// Paths not under /api/ are always public — SPA routes, auth proxy endpoints, static assets.
	return !strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api"
}

func isGraphQLRequest(r *http.Request) bool {
	return r.URL.Path == "/api/query"
}

func AuthMiddleware(next http.Handler) http.Handler {
	var onceHeader sync.Once
	var onceCookie sync.Once
	var onceSession sync.Once
	var onceKeyring sync.Once
	var onceInline sync.Once
	var onceProfile sync.Once
	const maxAuthHeaderLen = 16 * 1024
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicRoute(r) {
			next.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
			// Multipart uploads (file uploads) use a higher body limit.
			r.Body = http.MaxBytesReader(w, r.Body, maxUploadBodySize)
		} else {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		}

		if authBypassFn != nil && authBypassFn(r) {
			next.ServeHTTP(w, r)
			return
		}

		var token string
		// Prefer Authorization header if present to support desktop/webview environments
		if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
			if len(token) > maxAuthHeaderLen {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			onceHeader.Do(func() { log.Info("Auth: using Authorization header") })
		}

		// When no Authorization header is present, prefer the opaque session
		// cookie (browser clients). It maps to server-side encrypted credentials
		// and returns early — it never flows through the base64 credential path.
		if token == "" {
			if sessionToken, ok := sessionTokenFromRequest(r); ok {
				onceSession.Do(func() { log.Info("Auth: using session cookie") })
				if serveWithSessionCookie(w, r, next, sessionToken, isGraphQLRequest(r)) {
					return
				}
				if isGraphQLRequest(r) {
					next.ServeHTTP(w, r)
					return
				}
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		// Fallback to the legacy base64-credentials cookie when no header/session.
		if token == "" {
			if dbCookie, err := r.Cookie(string(AuthKey_Token)); err == nil {
				token = dbCookie.Value
				onceCookie.Do(func() { log.Info("Auth: using cookie-based auth") })
			} else {
				log.Debugf("[Auth] Cookie not found: %v", err)
			}
		}

		if token == "" && isGraphQLRequest(r) {
			next.ServeHTTP(w, r)
			return
		}
		if token == "" {
			log.Debug("[Auth] No token found (no cookie or header), returning 401")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		decodedValue, err := base64.StdEncoding.DecodeString(token)
		if err != nil {
			log.Debugf("[Auth] Failed to decode base64 token: %v (token len=%d)", err, len(token))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		credentials := &source.Credentials{}
		err = json.Unmarshal(decodedValue, credentials)
		if err != nil {
			log.Debugf("[Auth] Failed to unmarshal credentials JSON: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Profile trust comes only from server-side resolution, never the token.
		credentials.IsProfile = false
		inline := true
		isSavedProfileReference := credentials.ID != nil && credentials.SourceType == ""

		if isSavedProfileReference {
			// Client sent a saved-profile reference. Resolve the stored credentials and
			// apply any field overrides carried in the request payload.
			matched := false
			_, storedProfile, ok := src.FindSourceProfile(*credentials.ID)
			if ok {
				storedProfile.ID = credentials.ID
				storedProfile.Values, err = MergeSourceProfileValues(storedProfile.Values, credentials.Values)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				credentials = storedProfile
				matched = true
				inline = false
				onceProfile.Do(func() { log.Info("Auth: credentials resolved via saved profile") })
			}
			if !matched {
				if stored, err := LoadCredentials(*credentials.ID); err == nil && stored != nil {
					stored.ID = credentials.ID
					stored.Values, err = MergeSourceProfileValues(stored.Values, credentials.Values)
					if err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
					credentials = stored
					inline = false
					onceKeyring.Do(func() { log.Info("Auth: credentials resolved via OS keyring") })
				} else {
					// ID-only request but no stored credentials found
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			}
		} else if credentials.ID != nil {
			// Client sent full credentials with ID - validate or store for future use
			// This is the initial login case for desktop apps
			onceInline.Do(func() { log.Info("Auth: credentials supplied inline with ID") })
		}

		if inline {
			onceInline.Do(func() { log.Info("Auth: credentials supplied inline") })
		}

		if _, ok := sourcecatalog.Find(credentials.SourceType); !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, AuthKey_Source, credentials)
		ctx = context.WithValue(ctx, authenticatedSourceKey{}, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// serveWithSessionCookie resolves an opaque session token and injects its
// credentials into the request context. GraphQL CSRF enforcement is deferred
// until gqlgen has selected the operation; other HTTP requests are rejected
// immediately when their CSRF token is invalid.
func serveWithSessionCookie(w http.ResponseWriter, r *http.Request, next http.Handler, sessionToken string, deferGraphQLCSRF bool) bool {
	ttl := sessionTTL()
	credentials, csrfHash, needsRefresh, err := LookupSession(sessionToken, ttl)
	if err != nil {
		// Only clear the cookies when the session is genuinely gone (expired,
		// never existed, or undecryptable). Any other error (e.g. a transient
		// SQLite busy/IO error from the session store) leaves the cookie intact
		// so a subsequent request with the same still-valid session can succeed.
		if errors.Is(err, errSessionNotFound) || errors.Is(err, errSessionInvalid) {
			clearSessionCookies(w)
		}
		if deferGraphQLCSRF && !errors.Is(err, errSessionNotFound) && !errors.Is(err, errSessionInvalid) {
			ctx := context.WithValue(r.Context(), authenticationUnavailableKey{}, true)
			next.ServeHTTP(w, r.WithContext(ctx))
			return true
		}
		return false
	}

	csrfFailed := false
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
		if !validateCSRF(r, csrfHash) {
			if !deferGraphQLCSRF {
				log.Debug("[Auth] CSRF token missing or invalid for session request")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return true
			}
			csrfFailed = true
		}
	}

	if needsRefresh {
		if expiresAt, rerr := RefreshSession(sessionToken, ttl); rerr == nil {
			setSessionCookie(w, r, sessionToken, expiresAt)
		}
	}

	ctx := context.WithValue(r.Context(), AuthKey_Source, credentials)
	ctx = context.WithValue(ctx, authenticatedSourceKey{}, true)
	ctx = context.WithValue(ctx, csrfFailedKey{}, csrfFailed)
	next.ServeHTTP(w, r.WithContext(ctx))
	return true
}

// authBypassFn, if set, is called before CE credential authentication.
// When it returns true the CE credential check is skipped entirely.
// Extensions use this to provide alternative authentication mechanisms.
var authBypassFn func(*http.Request) bool

// RegisterAuthBypass registers a bypass function for CE credential authentication.
// It must be called before the HTTP server starts. When the function returns true,
// the request passes through without requiring CE database credentials.
func RegisterAuthBypass(fn func(*http.Request) bool) {
	authBypassFn = fn
}

// MergeSourceProfileValues merges client overrides while keeping server-side
// TLS file paths under the control of the stored profile.
func MergeSourceProfileValues(base map[string]string, overrides map[string]string) (map[string]string, error) {
	for _, key := range []string{ssl.KeySSLCACertPath, ssl.KeySSLClientCertPath, ssl.KeySSLClientKeyPath} {
		if value, ok := overrides[key]; ok && value != base[key] {
			return nil, errors.New("SSL file paths cannot be overridden by clients")
		}
	}
	merged := map[string]string{}
	maps.Copy(merged, base)
	maps.Copy(merged, overrides)
	return merged, nil
}
