package auth

import (
	"context"
	"net/http"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common"
)

func Logout(ctx context.Context) (*model.AuthResponse, error) {
	http.SetCookie(ctx.Value(common.RouterKey_ResponseWriter).(http.ResponseWriter), nil)
	return &model.AuthResponse{
		Status: true,
	}, nil
}
