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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/source/adapters"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

type AuthKey string

const (
	AuthKey_Token       AuthKey = "Token"
	AuthKey_Credentials AuthKey = "Credentials"
	AuthKey_Source      AuthKey = "SourceCredentials"
)

const maxRequestBodySize = 1024 * 1024 // Limit request body size to 1MB

func GetCredentials(ctx context.Context) *engine.Credentials {
	credentials := ctx.Value(AuthKey_Credentials)
	if credentials == nil {
		return nil
	}
	return credentials.(*engine.Credentials)
}

// GetSourceCredentials returns the source-first credentials from the current request context.
func GetSourceCredentials(ctx context.Context) *source.Credentials {
	credentials := ctx.Value(AuthKey_Source)
	if credentials == nil {
		return nil
	}
	return credentials.(*source.Credentials)
}

func isPublicRoute(r *http.Request) bool {
	// Paths not under /api/ are always public — SPA routes, auth proxy endpoints, static assets.
	if !strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api" {
		return true
	}

	// In dev mode, also allow GraphQL introspection requests without credentials.
	if env.IsDevelopment && r.Method == http.MethodPost {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return false
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		var query map[string]any
		if err := json.Unmarshal(body, &query); err == nil {
			if q, ok := query["query"].(string); ok && strings.Contains(q, "IntrospectionQuery") {
				return true
			}
		}
	}

	return false
}

func AuthMiddleware(next http.Handler) http.Handler {
	var onceHeader sync.Once
	var onceCookie sync.Once
	var onceKeyring sync.Once
	var onceInline sync.Once
	var onceProfile sync.Once
	const maxAuthHeaderLen = 16 * 1024
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicRoute(r) {
			next.ServeHTTP(w, r)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		body, err := readRequestBody(r)
		if err != nil {
			if err.Error() == "http: request body too large" {
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			} else {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		// this is to ensure that it can be re-read by the GraphQL layer
		r.Body = io.NopCloser(bytes.NewBuffer(body))
		if isAllowed(r, body) {
			next.ServeHTTP(w, r)
			return
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
		// Fallback to cookie when header is not provided
		if token == "" {
			if dbCookie, err := r.Cookie(string(AuthKey_Token)); err == nil {
				token = dbCookie.Value
				onceCookie.Do(func() { log.Info("Auth: using cookie-based auth") })
			} else {
				log.Debugf("[Auth] Cookie not found: %v", err)
			}
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

		if env.IsAPIGatewayEnabled && (credentials.AccessToken == nil || (credentials.AccessToken != nil && !isTokenValid(*credentials.AccessToken))) {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		inline := true
		isSavedProfileReference := credentials.ID != nil && credentials.SourceType == ""

		if isSavedProfileReference {
			// Client sent a saved-profile reference. Resolve the stored credentials and
			// apply any field overrides carried in the request payload.
			matched := false
			_, storedProfile, ok := src.FindSourceProfile(*credentials.ID)
			if ok {
				storedProfile.ID = credentials.ID
				storedProfile.Values = mergeCredentialValues(storedProfile.Values, credentials.Values)
				credentials = storedProfile
				matched = true
				inline = false
				onceProfile.Do(func() { log.Info("Auth: credentials resolved via saved profile") })
			}
			if !matched {
				if stored, err := LoadCredentials(*credentials.ID); err == nil && stored != nil {
					stored.ID = credentials.ID
					stored.Values = mergeCredentialValues(stored.Values, credentials.Values)
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

		spec, ok := sourcecatalog.Find(credentials.SourceType)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		engineCredentials := adapters.EngineCredentials(spec, credentials)
		ctx := r.Context()
		ctx = context.WithValue(ctx, AuthKey_Source, credentials)
		ctx = context.WithValue(ctx, AuthKey_Credentials, engineCredentials)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func readRequestBody(r *http.Request) ([]byte, error) {
	buf := &strings.Builder{}
	_, err := io.Copy(buf, r.Body)
	if err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

type GraphQLRequest struct {
	OperationName string         `json:"operationName"`
	Variables     map[string]any `json:"variables"`
}

// additionalAllowedOps holds operation names registered by extensions (e.g. EE)
// that should be allowed without authentication.
var additionalAllowedOps []string

// RegisterAllowedOperation adds a GraphQL operation name to the unauthenticated allowlist.
// It must be called during init(), before the HTTP server starts.
func RegisterAllowedOperation(opName string) {
	additionalAllowedOps = append(additionalAllowedOps, opName)
}

// authBypassFn, if set, is called for requests that are not in the public allowlist.
// When it returns true the CE credential check is skipped entirely.
// Extensions use this to provide alternative authentication mechanisms.
var authBypassFn func(*http.Request) bool

// RegisterAuthBypass registers a bypass function for CE credential authentication.
// It must be called before the HTTP server starts. When the function returns true,
// the request passes through without requiring CE database credentials.
func RegisterAuthBypass(fn func(*http.Request) bool) {
	authBypassFn = fn
}

func isAllowed(r *http.Request, body []byte) bool {
	if r.Method != http.MethodPost {
		return false
	}

	query := GraphQLRequest{}
	if err := json.Unmarshal(body, &query); err != nil {
		return false
	}

	if query.OperationName == "SourceFieldOptions" {
		sourceType, _ := query.Variables["sourceType"].(string)
		return sourceType == string(engine.DatabaseType_Sqlite3) || sourceType == string(engine.DatabaseType_DuckDB)
	}

	switch query.OperationName {
	case "LoginSource",
		"LoginWithSourceProfile",
		"SourceProfiles",
		"SettingsConfig", "GetVersion",
		"GetAWSProviders", "GetCloudProviders", "GetCloudProvider",
		"GetDiscoveredConnections", "GetProviderConnections", "SourceTypes",
		"GetLocalAWSProfiles", "GetAWSRegions",
		"AddAWSProvider", "TestAWSCredentials", "TestCloudProvider",
		"RefreshCloudProvider", "RemoveCloudProvider", "UpdateAWSProvider",
		"GenerateRDSAuthToken",
		"GetAzureProviders", "GetAzureProvider",
		"GetAzureSubscriptions", "GetAzureRegions",
		"AddAzureProvider", "UpdateAzureProvider", "TestAzureCredentials",
		"RefreshAzureProvider", "GenerateAzureADToken",
		"GetLocalGCPProjects", "GetGCPRegions",
		"GetGCPProviders", "GetGCPProvider",
		"AddGCPProvider", "UpdateGCPProvider", "TestGCPCredentials",
		"RefreshGCPProvider", "GenerateCloudSQLIAMAuthToken":
		return true
	}
	for _, op := range additionalAllowedOps {
		if query.OperationName == op {
			return true
		}
	}
	return false
}

func mergeCredentialValues(base map[string]string, overrides map[string]string) map[string]string {
	merged := map[string]string{}
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overrides {
		merged[key] = value
	}
	return merged
}

func isTokenValid(token string) bool {
	for _, t := range env.Tokens {
		if t == token {
			return true
		}
	}
	return false
}
