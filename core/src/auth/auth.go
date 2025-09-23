/*
 * Copyright 2025 Clidey, Inc.
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

	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
)

type AuthKey string

const (
	AuthKey_Token       AuthKey = "Token"
	AuthKey_Credentials AuthKey = "Credentials"
)

const maxRequestBodySize = 1024 * 1024 // Limit request body size to 1MB

func GetCredentials(ctx context.Context) *engine.Credentials {
	credentials := ctx.Value(AuthKey_Credentials)
	if credentials == nil {
		return nil
	}
	return credentials.(*engine.Credentials)
}

func isPublicRoute(r *http.Request) bool {
	if env.IsDevelopment {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return false
		}

		r.Body = io.NopCloser(bytes.NewReader(body))
		if r.Method != http.MethodPost {
			return false
		}

		var query map[string]interface{}
		if err := json.Unmarshal(body, &query); err != nil {
			return false
		}

		if q, ok := query["query"].(string); ok && strings.Contains(q, "IntrospectionQuery") {
			return true
		}
	}

	return (!strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api")
}

func AuthMiddleware(next http.Handler) http.Handler {
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

		// Check if this is a Tauri desktop app request
		origin := r.Header.Get("Origin")
		if origin == "https://tauri.localhost" || origin == "tauri://localhost" {
			println("[DEBUG] Tauri desktop app detected, origin:", origin)
			println("[DEBUG] Request path:", r.URL.Path)

			// First try to get credentials from custom header (for Windows compatibility)
			desktopCreds := r.Header.Get("X-Desktop-Credentials")
			if desktopCreds != "" {
				println("[DEBUG] Found X-Desktop-Credentials header")
				decodedValue, err := base64.StdEncoding.DecodeString(desktopCreds)
				if err == nil {
					credentials := &engine.Credentials{}
					err = json.Unmarshal(decodedValue, credentials)
					if err == nil {
						println("[DEBUG] Successfully decoded credentials from header")
						println("  Has Username:", credentials.Username != "")
						println("  Has Hostname:", credentials.Hostname != "")
						ctx := r.Context()
						ctx = context.WithValue(ctx, AuthKey_Credentials, credentials)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					} else {
						println("[DEBUG] Failed to unmarshal credentials from header:", err.Error())
					}
				} else {
					println("[DEBUG] Failed to decode header credentials:", err.Error())
				}
			}

			// Fallback to cookie-based auth (works on macOS)
			dbCookie, err := r.Cookie(string(AuthKey_Token))
			if err == nil && dbCookie.Value != "" {
				println("[DEBUG] Found token cookie for Tauri app")
				decodedValue, err := base64.StdEncoding.DecodeString(dbCookie.Value)
				if err == nil {
					credentials := &engine.Credentials{}
					err = json.Unmarshal(decodedValue, credentials)
					if err == nil {
						println("[DEBUG] Successfully decoded credentials from cookie")
						println("  Has Username:", credentials.Username != "")
						println("  Has Hostname:", credentials.Hostname != "")
						ctx := r.Context()
						ctx = context.WithValue(ctx, AuthKey_Credentials, credentials)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					} else {
						println("[DEBUG] Failed to unmarshal credentials:", err.Error())
					}
				} else {
					println("[DEBUG] Failed to decode cookie:", err.Error())
				}
			} else {
				if err != nil {
					println("[DEBUG] No token cookie found for Tauri app, err:", err.Error())
				} else {
					println("[DEBUG] No token cookie found for Tauri app")
				}
			}
			// Allow access even without credentials for Tauri (for login page)
			println("[DEBUG] Allowing Tauri access without credentials")
			ctx := r.Context()
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		var token string
		if env.IsAPIGatewayEnabled {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		} else {
			dbCookie, err := r.Cookie(string(AuthKey_Token))
			if err == nil {
				token = dbCookie.Value
			}
		}

		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		decodedValue, err := base64.StdEncoding.DecodeString(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		credentials := &engine.Credentials{}
		err = json.Unmarshal(decodedValue, credentials)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if env.IsAPIGatewayEnabled && (credentials.AccessToken == nil || (credentials.AccessToken != nil && !isTokenValid(*credentials.AccessToken))) {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if credentials.Id != nil {
			profiles := src.GetLoginProfiles()
			for i, loginProfile := range profiles {
				profileId := src.GetLoginProfileId(i, loginProfile)
				if *credentials.Id == profileId {
					profile := *src.GetLoginCredentials(loginProfile)
					if credentials.Database != "" {
						profile.Database = credentials.Database
					}
					credentials = &profile
					break
				}
			}
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, AuthKey_Credentials, credentials)
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
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

func isAllowed(r *http.Request, body []byte) bool {
	if r.Method != http.MethodPost {
		return false
	}

	query := GraphQLRequest{}
	if err := json.Unmarshal(body, &query); err != nil {
		return false
	}

	if query.OperationName == "GetDatabase" {
		return query.Variables["type"] == string(engine.DatabaseType_Sqlite3)
	}

	switch query.OperationName {
	case "Login", "LoginWithProfile", "Logout", "GetProfiles", "UpdateSettings", "SettingsConfig":
		return true
	}
	return false
}

func isTokenValid(token string) bool {
	for _, t := range env.Tokens {
		if t == token {
			return true
		}
	}
	return false
}
