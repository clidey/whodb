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
	"net/http/httptest"
	"testing"

	"github.com/clidey/whodb/core/src/common"
)

func TestLogoutClearsAuthCookie(t *testing.T) {
	rr := httptest.NewRecorder()
	ctx := context.WithValue(context.Background(), common.RouterKey_ResponseWriter, rr)

	if _, err := Logout(ctx); err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}

	var found bool
	for _, c := range rr.Result().Cookies() {
		if c.Name == string(AuthKey_Token) {
			found = true
			if c.MaxAge >= 0 {
				t.Errorf("expected auth cookie to be expired (MaxAge < 0), got MaxAge=%d", c.MaxAge)
			}
			if c.Value != "" {
				t.Errorf("expected empty cookie value, got %q", c.Value)
			}
			if c.Path != "/" {
				t.Errorf("expected cookie path '/', got %q", c.Path)
			}
		}
	}
	if !found {
		t.Fatalf("expected a %q deletion cookie in the response, got none", AuthKey_Token)
	}
}

func TestLogoutWithoutResponseWriterDoesNotPanic(t *testing.T) {
	// Logout must not panic when invoked without an HTTP ResponseWriter in context
	// (e.g. from a non-HTTP caller); the cookie clearing is simply skipped.
	if _, err := Logout(context.Background()); err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}
}
