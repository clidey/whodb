package auth

import (
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

const maxRequestBodySize = 10 * 1024 * 1024 // Limit request body size to 10 MB

func GetCredentials(ctx context.Context) *engine.Credentials {
	credentials := ctx.Value(AuthKey_Credentials)
	if credentials == nil {
		return nil
	}
	return credentials.(*engine.Credentials)
}

func isPublicRoute(r *http.Request) bool {
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

		if isAllowed(r, body) {
			next.ServeHTTP(w, r)
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
	case "Login", "Logout", "GetProfiles", "UpdateSettings", "SettingsConfig":
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
