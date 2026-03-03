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

package envconfig

import (
	"testing"

	"github.com/clidey/whodb/core/src/env"
)

func TestGetDefaultDatabaseCredentialsParsesEnv(t *testing.T) {
	t.Setenv("WHODB_POSTGRES", `[{"host":"db.local","user":"alice","password":"secret","database":"app"}]`)

	creds := GetDefaultDatabaseCredentials("postgres")
	if len(creds) != 1 {
		t.Fatalf("expected one credential parsed from env, got %d", len(creds))
	}

	if creds[0].Hostname != "db.local" || creds[0].Username != "alice" || creds[0].Database != "app" {
		t.Fatalf("unexpected credentials parsed: %+v", creds[0])
	}
}

func TestFindAllDatabaseCredentialsFallback(t *testing.T) {
	t.Setenv("WHODB_MYSQL", "")
	t.Setenv("WHODB_MYSQL_1", `{"host":"mysql.local","user":"bob","password":"pw","database":"northwind"}`)

	creds := GetDefaultDatabaseCredentials("mysql")
	if len(creds) != 1 {
		t.Fatalf("expected fallback credentials to be discovered, got %d", len(creds))
	}
	if creds[0].Hostname != "mysql.local" || creds[0].Username != "bob" {
		t.Fatalf("unexpected fallback credential: %+v", creds[0])
	}
}

func TestParseGoogleAIProvider_WithAPIKeyAndModels(t *testing.T) {
	orig := env.GoogleAIAPIKey
	origModels := env.GoogleAIModels
	defer func() { env.GoogleAIAPIKey = orig; env.GoogleAIModels = origModels }()

	env.GoogleAIAPIKey = "test-key-123"
	env.GoogleAIModels = "gemini-2.0-flash,gemini-1.5-pro"

	p := ParseGoogleAIProvider()
	if p == nil {
		t.Fatal("expected non-nil provider config")
	}
	if p.ProviderId != "google-ai-1" {
		t.Fatalf("expected provider ID 'google-ai-1', got %q", p.ProviderId)
	}
	if p.ClientType != "google-ai" {
		t.Fatalf("expected client type 'google-ai', got %q", p.ClientType)
	}
	if p.Name != "Google AI" {
		t.Fatalf("expected default name 'Google AI', got %q", p.Name)
	}
	if p.APIKey != "test-key-123" {
		t.Fatalf("expected API key 'test-key-123', got %q", p.APIKey)
	}
	if len(p.Models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(p.Models))
	}
	if p.BaseURL != "https://generativelanguage.googleapis.com/v1beta" {
		t.Fatalf("expected default base URL, got %q", p.BaseURL)
	}
}

func TestParseGoogleAIProvider_FallbackGoogleAPIKey(t *testing.T) {
	orig := env.GoogleAIAPIKey
	origModels := env.GoogleAIModels
	defer func() { env.GoogleAIAPIKey = orig; env.GoogleAIModels = origModels }()

	env.GoogleAIAPIKey = ""
	env.GoogleAIModels = "gemini-2.0-flash"
	t.Setenv("GOOGLE_API_KEY", "fallback-key")

	p := ParseGoogleAIProvider()
	if p == nil {
		t.Fatal("expected non-nil provider config from GOOGLE_API_KEY fallback")
	}
	if p.APIKey != "fallback-key" {
		t.Fatalf("expected fallback API key, got %q", p.APIKey)
	}
}

func TestParseGoogleAIProvider_MissingAPIKey(t *testing.T) {
	orig := env.GoogleAIAPIKey
	origModels := env.GoogleAIModels
	defer func() { env.GoogleAIAPIKey = orig; env.GoogleAIModels = origModels }()

	env.GoogleAIAPIKey = ""
	env.GoogleAIModels = "gemini-2.0-flash"
	t.Setenv("GOOGLE_API_KEY", "")

	p := ParseGoogleAIProvider()
	if p != nil {
		t.Fatal("expected nil when no API key is set")
	}
}

func TestParseGoogleAIProvider_MissingModels(t *testing.T) {
	orig := env.GoogleAIAPIKey
	origModels := env.GoogleAIModels
	defer func() { env.GoogleAIAPIKey = orig; env.GoogleAIModels = origModels }()

	env.GoogleAIAPIKey = "test-key"
	env.GoogleAIModels = ""

	p := ParseGoogleAIProvider()
	if p != nil {
		t.Fatal("expected nil when no models are set")
	}
}

func TestParseGoogleAIProvider_CustomNameAndBaseURL(t *testing.T) {
	orig := env.GoogleAIAPIKey
	origModels := env.GoogleAIModels
	origName := env.GoogleAIName
	origURL := env.GoogleAIBaseURL
	defer func() {
		env.GoogleAIAPIKey = orig
		env.GoogleAIModels = origModels
		env.GoogleAIName = origName
		env.GoogleAIBaseURL = origURL
	}()

	env.GoogleAIAPIKey = "key"
	env.GoogleAIModels = "model-1"
	env.GoogleAIName = "My Gemini"
	env.GoogleAIBaseURL = "https://custom.endpoint.com/v1"

	p := ParseGoogleAIProvider()
	if p == nil {
		t.Fatal("expected non-nil provider config")
	}
	if p.Name != "My Gemini" {
		t.Fatalf("expected name 'My Gemini', got %q", p.Name)
	}
	if p.BaseURL != "https://custom.endpoint.com/v1" {
		t.Fatalf("expected custom base URL, got %q", p.BaseURL)
	}
}
