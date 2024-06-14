package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	"github.com/clidey/whodb/core/src/engine"
)

type AuthKey string

const (
	AuthKey_Token       AuthKey = "Token"
	AuthKey_Credentials AuthKey = "Credentials"
)

func GetCredentials(ctx context.Context) *engine.Credentials {
	return ctx.Value(AuthKey_Credentials).(*engine.Credentials)
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		r.Body = io.NopCloser(bytes.NewReader(body))
		if isLoginMutation(r, body) {
			next.ServeHTTP(w, r)
			return
		}

		dbCookie, err := r.Cookie(string(AuthKey_Token))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		decodedValue, err := base64.StdEncoding.DecodeString(dbCookie.Value)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		credentials := &engine.Credentials{}
		err = json.Unmarshal(decodedValue, credentials)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, AuthKey_Credentials, credentials)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type GraphQLRequest struct {
	OperationName string `json:"operationName"`
}

func isLoginMutation(r *http.Request, body []byte) bool {
	if r.Method != http.MethodPost {
		return false
	}

	query := GraphQLRequest{}

	if err := json.Unmarshal(body, &query); err != nil {
		return false
	}

	return query.OperationName == "Login"
}
