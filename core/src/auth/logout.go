package auth

import (
	"context"
	"net/http"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common"
)

func Logout(ctx context.Context) (*model.StatusResponse, error) {
	http.SetCookie(ctx.Value(common.RouterKey_ResponseWriter).(http.ResponseWriter), nil)
	return &model.StatusResponse{
		Status: true,
	}, nil
}
