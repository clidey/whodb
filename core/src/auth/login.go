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
    "github.com/clidey/whodb/core/src/engine"
    "github.com/clidey/whodb/core/src/env"
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
    // Ensure cookies are HTTPS-only in production
    cookie.Secure = !env.IsDevelopment

    http.SetCookie(ctx.Value(common.RouterKey_ResponseWriter).(http.ResponseWriter), cookie)

    // Persist credentials in OS keychain when an Id is provided
    if input.ID != nil && *input.ID != "" {
        advanced := make([]engine.Record, 0, len(input.Advanced))
        for _, rec := range input.Advanced {
            advanced = append(advanced, engine.Record{Key: rec.Key, Value: rec.Value})
        }
        if err := SaveCredentials(*input.ID, &engine.Credentials{
            Id:        input.ID,
            Type:      input.Type,
            Hostname:  input.Hostname,
            Username:  input.Username,
            Password:  input.Password,
            Database:  input.Database,
            Advanced:  advanced,
            IsProfile: false,
        }); err != nil {
            warnKeyringUnavailableOnce(err)
        }
    }

    return &model.StatusResponse{
        Status: true,
    }, nil
}
