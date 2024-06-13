package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common"
)

func Login(ctx context.Context, input *model.LoginCredentials) (*model.LoginResponse, error) {
	loginInfoJSON, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	cookieValue := base64.StdEncoding.EncodeToString(loginInfoJSON)

	cookie := &http.Cookie{
		Name:     string(Authkey_Token),
		Value:    cookieValue,
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(24 * time.Hour),
	}

	http.SetCookie(ctx.Value(common.RouterKey_ResponseWriter).(http.ResponseWriter), cookie)

	return &model.LoginResponse{
		Status: true,
	}, nil
}
