package analytics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
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

func (f *fakePosthogClient) Close() error { return nil }

func (f *fakePosthogClient) IsFeatureEnabled(posthog.FeatureFlagPayload) (interface{}, error) {
	return nil, nil
}

func (f *fakePosthogClient) GetFeatureFlag(posthog.FeatureFlagPayload) (interface{}, error) {
	return nil, nil
}

func (f *fakePosthogClient) GetFeatureFlagPayload(posthog.FeatureFlagPayload) (string, error) {
	return "", nil
}

func (f *fakePosthogClient) GetRemoteConfigPayload(string) (string, error) {
	return "", nil
}

func (f *fakePosthogClient) GetAllFlags(posthog.FeatureFlagPayloadNoKey) (map[string]interface{}, error) {
	return nil, nil
}

func (f *fakePosthogClient) ReloadFeatureFlags() error {
	return nil
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
	fallbackID = atomic.Value{}
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

func TestBuildPropertiesMergesMetadataAndConfig(t *testing.T) {
	t.Cleanup(resetAnalyticsState)

	cfg = Config{
		Environment: "prod",
		AppVersion:  "1.2.3",
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
	if props["path"] != "override" {
		t.Fatalf("expected explicit path property to be preserved, got %v", props["path"])
	}
	if props["environment"] != "prod" || props["app_version"] != "1.2.3" {
		t.Fatalf("expected config metadata to be included")
	}
	if props["$lib"] != libraryName {
		t.Fatalf("expected $lib to be set to %s", libraryName)
	}
	if props["custom"] != "value" {
		t.Fatalf("expected custom properties to be retained")
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
