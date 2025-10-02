// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

    var token string
        // Prefer Authorization header if present to support desktop/webview environments
        if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
            token = strings.TrimPrefix(authHeader, "Bearer ")
            if len(token) > maxAuthHeaderLen {
                http.Error(w, "Bad Request", http.StatusBadRequest)
                return
            }
            onceHeader.Do(func() { log.Logger.Info("Auth: using Authorization header") })
        }
        // Fallback to cookie when header is not provided
        if token == "" {
            if dbCookie, err := r.Cookie(string(AuthKey_Token)); err == nil {
                token = dbCookie.Value
                onceCookie.Do(func() { log.Logger.Info("Auth: using cookie-based auth") })
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

        inline := true
        isIdOnly := credentials.Id != nil && credentials.Type == "" && credentials.Hostname == ""

        if credentials.Id != nil && isIdOnly {
            // Client sent only ID - must match a saved profile or keyring entry
            matched := false
            profiles := src.GetLoginProfiles()
            for i, loginProfile := range profiles {
                profileId := src.GetLoginProfileId(i, loginProfile)
                if *credentials.Id == profileId {
                    profile := *src.GetLoginCredentials(loginProfile)
                    profile.Id = credentials.Id
                    if credentials.Database != "" {
                        profile.Database = credentials.Database
                    }
                    credentials = &profile
                    matched = true
                    inline = false
                    onceProfile.Do(func() { log.Logger.Info("Auth: credentials resolved via saved profile") })
                    break
                }
            }
            if !matched {
                if stored, err := LoadCredentials(*credentials.Id); err == nil && stored != nil {
                    if credentials.Database != "" {
                        stored.Database = credentials.Database
                    }
                    stored.Id = credentials.Id
                    credentials = stored
                    inline = false
                    onceKeyring.Do(func() { log.Logger.Info("Auth: credentials resolved via OS keyring") })
                } else {
                    // ID-only request but no stored credentials found
                    http.Error(w, "Unauthorized", http.StatusUnauthorized)
                    return
                }
            }
        } else if credentials.Id != nil && !isIdOnly {
            // Client sent full credentials with ID - validate or store for future use
            // This is the initial login case for desktop apps
            onceInline.Do(func() { log.Logger.Info("Auth: credentials supplied inline with ID") })
        }

        if inline {
            onceInline.Do(func() { log.Logger.Info("Auth: credentials supplied inline") })
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
    case "Login", "LoginWithProfile", "GetProfiles", "UpdateSettings", "SettingsConfig":
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
