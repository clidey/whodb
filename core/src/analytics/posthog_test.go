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

package analytics

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/posthog/posthog-go"
)

type fakePosthogClient struct {
	messages []posthog.Message
}

func (f *fakePosthogClient) Enqueue(message posthog.Message) error {
	f.messages = append(f.messages, message)
	return nil
}

func (f *fakePosthogClient) Close() error                           { return nil }
func (f *fakePosthogClient) CloseWithContext(context.Context) error { return nil }

func (f *fakePosthogClient) IsFeatureEnabled(posthog.FeatureFlagPayload) (any, error) {
	return nil, nil
}

func (f *fakePosthogClient) GetFeatureFlag(posthog.FeatureFlagPayload) (any, error) {
	return nil, nil
}

func (f *fakePosthogClient) GetFeatureFlagResult(posthog.FeatureFlagPayload) (*posthog.FeatureFlagResult, error) {
	return nil, nil
}

func (f *fakePosthogClient) GetFeatureFlagPayload(posthog.FeatureFlagPayload) (string, error) {
	return "", nil
}

func (f *fakePosthogClient) GetRemoteConfigPayload(string) (string, error) {
	return "", nil
}

func (f *fakePosthogClient) GetAllFlags(posthog.FeatureFlagPayloadNoKey) (map[string]any, error) {
	return nil, nil
}

func (f *fakePosthogClient) ReloadFeatureFlags() error {
	return nil
}

func (f *fakePosthogClient) EvaluateFlags(posthog.EvaluateFlagsPayload) (*posthog.FeatureFlagEvaluations, error) {
	return nil, nil
}

func (f *fakePosthogClient) GetFeatureFlags() ([]posthog.FeatureFlag, error) {
	return nil, nil
}

func (f *fakePosthogClient) GetLastCapturedEvent() *posthog.Capture {
	return nil
}

func resetAnalyticsState() {
	storeClient(nil)
	enabled.Store(false)
	initOnce = sync.Once{}
	distinctIDResolver = nil
	cfg = Config{}
}

func TestBuildMetadataPrefersOriginAndStripsPort(t *testing.T) {
	t.Cleanup(resetAnalyticsState)

	req := httptest.NewRequest(http.MethodPost, "http://api.example.com:8080/api/query", nil)
	req.Header.Set("Origin", "https://frontend.local:3000")
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Referer", "https://frontend.local/dashboard")
	req.Header.Set("X-Request-Id", "req-123")

	metadata := BuildMetadata(req)

	if metadata.Domain != "frontend.local" {
		t.Fatalf("expected origin host to be used as domain, got %s", metadata.Domain)
	}
	if metadata.Path != "/api/query" {
		t.Fatalf("expected path to be set, got %s", metadata.Path)
	}
	if metadata.Method != http.MethodPost {
		t.Fatalf("expected method %s, got %s", http.MethodPost, metadata.Method)
	}
	if metadata.UserAgent != "test-agent" || metadata.Referer != "https://frontend.local/dashboard" {
		t.Fatalf("expected user agent and referer to be preserved")
	}
	if metadata.RequestID != "req-123" {
		t.Fatalf("expected request id to be derived from header, got %s", metadata.RequestID)
	}
}

func TestHashIdentifierHandlesEmptyInput(t *testing.T) {
	if got := HashIdentifier("   "); got != "" {
		t.Fatalf("expected empty string to return empty hash, got %s", got)
	}
	value := "example"
	if HashIdentifier(value) != HashIdentifier(value) {
		t.Fatalf("expected hash function to be deterministic")
	}
}

func TestCaptureDoesNothingWhenDisabled(t *testing.T) {
	t.Cleanup(resetAnalyticsState)
	client := &fakePosthogClient{}
	storeClient(client)
	enabled.Store(false)

	Capture(context.Background(), "event", map[string]any{"a": 1})
	if len(client.messages) != 0 {
		t.Fatalf("expected no messages when analytics disabled")
	}
}

func TestBuildPropertiesMergesMetadataAndConfig(t *testing.T) {
	t.Cleanup(resetAnalyticsState)

	cfg = Config{
		Environment: "prod",
		AppVersion:  "1.2.3",
		Edition:     "ce",
		Source:      "backend",
	}
	ctx := WithMetadata(context.Background(), Metadata{
		Domain:    "frontend.local",
		Path:      "/api",
		Method:    http.MethodPost,
		UserAgent: "agent",
		Referer:   "ref",
		RequestID: "req-1",
	})

	props := buildProperties(ctx, map[string]any{
		"custom": "value",
		"path":   "override",
	})

	if props["domain"] != "frontend.local" {
		t.Fatalf("expected domain to be added from metadata, got %v", props["domain"])
	}
	if props["$host"] != "frontend.local" {
		t.Fatalf("expected $host to be added from metadata domain, got %v", props["$host"])
	}
	if props["path"] != "override" {
		t.Fatalf("expected explicit path property to be preserved, got %v", props["path"])
	}
	if props["build_environment"] != "prod" || props["app_version"] != "1.2.3" {
		t.Fatalf("expected config metadata to be included")
	}
	if props["build_edition"] != "ce" || props["source"] != "backend" {
		t.Fatalf("expected edition and source to be stamped, got %v / %v", props["build_edition"], props["source"])
	}
	if props["$lib"] != libraryName {
		t.Fatalf("expected $lib to be set to %s", libraryName)
	}
	if props["custom"] != "value" {
		t.Fatalf("expected custom properties to be retained")
	}
}

func TestCaptureErrorEmitsErrorCodeNotMessage(t *testing.T) {
	t.Cleanup(resetAnalyticsState)

	client := &fakePosthogClient{}
	storeClient(client)
	enabled.Store(true)

	CaptureError(context.Background(), "login.execute", errors.New("connection refused: host db.internal:5432"), nil)

	if len(client.messages) != 1 {
		t.Fatalf("expected one event to be enqueued, got %d", len(client.messages))
	}
	capture, ok := client.messages[0].(posthog.Capture)
	if !ok {
		t.Fatalf("expected capture message, got %T", client.messages[0])
	}
	if capture.Properties["error_code"] != "connection_failed" {
		t.Fatalf("expected error_code connection_failed, got %v", capture.Properties["error_code"])
	}
	if _, exists := capture.Properties["error_message"]; exists {
		t.Fatalf("expected raw error message to be omitted")
	}
	if capture.Properties["operation"] != "login.execute" {
		t.Fatalf("expected operation to be preserved, got %v", capture.Properties["operation"])
	}
}

func TestDistinctIDResolverTakesPrecedence(t *testing.T) {
	t.Cleanup(resetAnalyticsState)

	ctx := WithMetadata(context.Background(), Metadata{DistinctID: "header-id"})

	if got := distinctIDFromContext(ctx); got != "header-id" {
		t.Fatalf("expected header id without resolver, got %s", got)
	}

	SetDistinctIDResolver(func(context.Context) string { return "user-42" })
	if got := distinctIDFromContext(ctx); got != "user-42" {
		t.Fatalf("expected resolver id to take precedence, got %s", got)
	}

	SetDistinctIDResolver(func(context.Context) string { return "  " })
	if got := distinctIDFromContext(ctx); got != "header-id" {
		t.Fatalf("expected blank resolver result to fall through to header, got %s", got)
	}

	if got := distinctIDFromContext(context.Background()); got != anonymousID {
		t.Fatalf("expected anonymous fallback, got %s", got)
	}
}

func TestErrorCodeTaxonomy(t *testing.T) {
	cases := map[string]string{
		"connection refused":       "connection_failed",
		"quota exceeded for org":   "quota_exceeded",
		"request timeout":          "timeout",
		"invalid input for column": "invalid_input",
		"something exploded":       "internal_error",
	}
	for message, want := range cases {
		if got := ErrorCode(errors.New(message)); got != want {
			t.Fatalf("ErrorCode(%q) = %s, want %s", message, got, want)
		}
	}
	if got := ErrorCode(nil); got != "" {
		t.Fatalf("expected empty code for nil error, got %s", got)
	}
}

func TestCaptureWithDistinctIDRequiresEnabledAndConfigured(t *testing.T) {
	t.Cleanup(resetAnalyticsState)

	client := &fakePosthogClient{}
	storeClient(client)

	// Disabled: should drop events
	enabled.Store(false)
	CaptureWithDistinctID(context.Background(), "abc123", "event-disabled", map[string]any{"a": 1})
	if len(client.messages) != 0 {
		t.Fatalf("expected no events to be enqueued when disabled")
	}

	// Enabled and configured: should enqueue
	enabled.Store(true)
	CaptureWithDistinctID(context.Background(), "abc123", "event-enabled", map[string]any{"a": 1})
	if len(client.messages) != 1 {
		t.Fatalf("expected one event to be enqueued, got %d", len(client.messages))
	}

	var capture posthog.Capture
	switch msg := client.messages[0].(type) {
	case posthog.Capture:
		capture = msg
	case *posthog.Capture:
		capture = *msg
	default:
		t.Fatalf("expected capture message, got %T", client.messages[0])
	}
	if capture.Event != "event-enabled" || capture.DistinctId != "abc123" {
		t.Fatalf("capture payload did not match expectations: %#v", capture)
	}
}

func TestCaptureBuildsGroupsFromProperties(t *testing.T) {
	t.Cleanup(resetAnalyticsState)

	client := &fakePosthogClient{}
	storeClient(client)
	enabled.Store(true)

	CaptureWithDistinctID(context.Background(), "abc123", "event-with-groups", map[string]any{
		"$groups": map[string]any{
			"organization": "org_1",
			"project":      "project_1",
		},
	})

	if len(client.messages) != 1 {
		t.Fatalf("expected one event to be enqueued, got %d", len(client.messages))
	}

	capture, ok := client.messages[0].(posthog.Capture)
	if !ok {
		t.Fatalf("expected capture message, got %T", client.messages[0])
	}
	if capture.Groups["organization"] != "org_1" || capture.Groups["project"] != "project_1" {
		t.Fatalf("expected groups to be attached to capture, got %#v", capture.Groups)
	}
}

func TestIdentifyGroupEnqueuesGroupIdentify(t *testing.T) {
	t.Cleanup(resetAnalyticsState)

	client := &fakePosthogClient{}
	storeClient(client)
	enabled.Store(true)

	IdentifyGroup(context.Background(), GroupIdentity{
		Type: "organization",
		Key:  "org_1",
		Properties: map[string]any{
			"plan": "team",
		},
	})

	if len(client.messages) != 1 {
		t.Fatalf("expected one event to be enqueued, got %d", len(client.messages))
	}

	groupIdentify, ok := client.messages[0].(posthog.GroupIdentify)
	if !ok {
		t.Fatalf("expected group identify message, got %T", client.messages[0])
	}
	if groupIdentify.Type != "organization" || groupIdentify.Key != "org_1" {
		t.Fatalf("group identify payload did not match expectations: %#v", groupIdentify)
	}
	if groupIdentify.Properties["plan"] != "team" {
		t.Fatalf("expected group properties to be preserved, got %#v", groupIdentify.Properties)
	}
}
