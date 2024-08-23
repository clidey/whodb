package router

import (
	"context"
	"net/http"

	"github.com/clidey/whodb/core/src/common"
)

func contextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), common.RouterKey_ResponseWriter, w)
		ctx = context.WithValue(ctx, common.RouterKey_Request, r)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
