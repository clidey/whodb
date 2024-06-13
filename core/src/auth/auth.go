package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/clidey/whodb/core/graph/model"
)

type AuthKey string

const (
	Authkey_Token       AuthKey = "Token"
	Authkey_Credentials AuthKey = "Credentials"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dbCookie, err := r.Cookie(string(Authkey_Token))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		decodedValue, err := base64.StdEncoding.DecodeString(dbCookie.Value)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		credentials := &model.LoginCredentials{}
		err = json.Unmarshal(decodedValue, credentials)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, Authkey_Credentials, credentials)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
