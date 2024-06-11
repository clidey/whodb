package auth

import (
	"net/http"
)

const DBTokenCookieKey = "DB_TOKEN"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
