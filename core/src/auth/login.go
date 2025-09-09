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
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common"
)

func Login(ctx context.Context, input *model.LoginCredentials) (*model.StatusResponse, error) {
	loginInfoJSON, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	cookieValue := base64.StdEncoding.EncodeToString(loginInfoJSON)

	cookie := &http.Cookie{
		Name:     string(AuthKey_Token),
		Value:    cookieValue,
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(ctx.Value(common.RouterKey_ResponseWriter).(http.ResponseWriter), cookie)

	return &model.StatusResponse{
		Status: true,
	}, nil
}
