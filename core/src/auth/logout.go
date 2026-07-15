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
	"net/http"
	"time"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

func Logout(ctx context.Context) (*model.StatusResponse, error) {
	if creds := GetSourceCredentials(ctx); creds != nil {
		if spec, ok := sourcecatalog.Find(creds.SourceType); ok {
			_ = source.Invalidate(ctx, spec, creds)
		}

		// Best-effort: remove stored credentials from keyring
		if creds.ID != nil {
			if err := DeleteCredentials(*creds.ID); err != nil {
				warnKeyringUnavailableOnce(err)
			}
		}
	}
	// Delete the server-side session (if this request carried a session cookie).
	if r, ok := ctx.Value(common.RouterKey_Request).(*http.Request); ok {
		if sessionToken, found := sessionTokenFromRequest(r); found {
			if err := DeleteSession(sessionToken); err != nil {
				log.Debugf("[Logout] failed to delete session: %v", err)
			}
		}
	}
	// Clear the auth cookies. Passing a nil cookie emits no valid deletion header,
	// so an explicit expired cookie with a matching name and path is required for
	// the browser to actually remove it.
	if w, ok := ctx.Value(common.RouterKey_ResponseWriter).(http.ResponseWriter); ok {
		http.SetCookie(w, &http.Cookie{
			Name:     string(AuthKey_Token),
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		clearSessionCookies(w)
	}
	return &model.StatusResponse{
		Status: true,
	}, nil
}
