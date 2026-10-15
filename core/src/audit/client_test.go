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

package audit

import (
	"context"
	"strings"
	"testing"
)

func TestSanitizeClient(t *testing.T) {
	cases := map[string]string{
		"whodb-cli/1.2.3":         "whodb-cli/1.2.3",
		"  whodb-cli-mcp/1.2.3  ": "whodb-cli-mcp/1.2.3",
		"":                        "",
		"bad\x00client":           "",
		"line\nbreak":             "",
		"unicode-é":               "",
		strings.Repeat("a", 65):   "",
		strings.Repeat("a", 64):   strings.Repeat("a", 64),
	}
	for input, want := range cases {
		if got := SanitizeClient(input); got != want {
			t.Errorf("SanitizeClient(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestWithRequestOverridesClient(t *testing.T) {
	ctx := WithRequest(context.Background(), Request{Path: "/mcp", UserAgent: "claude-code/2.0"})
	ctx = WithRequest(ctx, Request{Client: "mcp"})
	request := RequestFromContext(ctx)
	if request.Client != "mcp" || request.Path != "/mcp" || request.UserAgent != "claude-code/2.0" {
		t.Fatalf("expected client to merge without clobbering other fields, got %+v", request)
	}
	// An empty override keeps the existing label.
	request = RequestFromContext(WithRequest(ctx, Request{Client: ""}))
	if request.Client != "mcp" {
		t.Fatalf("expected empty override to keep client, got %q", request.Client)
	}
}
