/*
 * Copyright 2026 Clidey, Inc.
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

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/source"
)

// LoginSource persists source credentials when needed and, in browser/server
// mode, issues an opaque HttpOnly session cookie backed by the encrypted session
// store. It returns a successful login status response.
func LoginSource(ctx context.Context, credentials *source.Credentials) (*model.StatusResponse, error) {
	values := credentials.CloneValues()
	log.Debugf("[LoginSource] sourceType=%s, values=%d", credentials.SourceType, len(values))

	if credentials.ID != nil && *credentials.ID != "" {
		storedCredentials := &source.Credentials{
			ID:          credentials.ID,
			SourceType:  credentials.SourceType,
			Values:      values,
			AccessToken: credentials.AccessToken,
			IsProfile:   false,
		}
		if err := SaveCredentials(*credentials.ID, storedCredentials); err != nil {
			warnKeyringUnavailableOnce(err)
		}
	}

	// Desktop/webview clients keep the Authorization-header credential flow; the
	// server-side session cookie is only issued for browser clients.
	if !env.GetIsDesktopMode() && SessionStoreEnabled() {
		issueSessionCookie(ctx, credentials)
	}

	return &model.StatusResponse{Status: true}, nil
}

// issueSessionCookie mints an encrypted session for the given credentials and
// writes the session + CSRF cookies onto the current response. Failures are
// logged but non-fatal — login still succeeds and the client can retry.
func issueSessionCookie(ctx context.Context, credentials *source.Credentials) {
	w, ok := ctx.Value(common.RouterKey_ResponseWriter).(http.ResponseWriter)
	if !ok {
		return
	}
	r, _ := ctx.Value(common.RouterKey_Request).(*http.Request)

	ttl := sessionTTL()
	token, csrfToken, expiresAt, err := CreateSession(credentials, ttl)
	if err != nil {
		log.Warnf("Failed to create session: %v", err)
		return
	}
	setSessionCookie(w, r, token, expiresAt)
	setCSRFCookie(w, r, csrfToken, expiresAt)
}
