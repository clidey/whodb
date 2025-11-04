// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
    "context"
    "net/http"

    "github.com/clidey/whodb/core/graph/model"
    "github.com/clidey/whodb/core/src/common"
)

func Logout(ctx context.Context) (*model.StatusResponse, error) {
    // Best-effort: remove stored credentials for current profile from keyring
    if creds := GetCredentials(ctx); creds != nil && creds.Id != nil {
        if err := DeleteCredentials(*creds.Id); err != nil {
            warnKeyringUnavailableOnce(err)
        }
    }
    http.SetCookie(ctx.Value(common.RouterKey_ResponseWriter).(http.ResponseWriter), nil)
    return &model.StatusResponse{
        Status: true,
    }, nil
}
