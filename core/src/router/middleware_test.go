package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clidey/whodb/core/src/analytics"
	coreaudit "github.com/clidey/whodb/core/src/audit"
)

func TestContextMiddlewareAddsMetadata(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://api.local/data", nil)
	req.Host = "api.local:8080"
	req.Header.Set("User-Agent", "tester")
	req.Header.Set("X-Whodb-Analytics-Id", "user-123")
	req.Header.Set("X-Request-Id", "req-1")
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	rr := httptest.NewRecorder()
	var captured analytics.Metadata
	var request coreaudit.Request

	handler := contextMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = analytics.MetadataFromContext(r.Context())
		request = coreaudit.RequestFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected middleware to allow request, got status %d", rr.Code)
	}

	if captured.Domain != "api.local" {
		t.Fatalf("expected domain to be derived from host, got %s", captured.Domain)
	}
	if captured.DistinctID != "user-123" {
		t.Fatalf("expected distinct id to be captured from header, got %s", captured.DistinctID)
	}
	if captured.RequestID != "req-1" {
		t.Fatalf("expected request id to be captured from header, got %s", captured.RequestID)
	}
	if request.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("expected trace id to be captured from traceparent, got %s", request.TraceID)
	}
	if request.SpanID != "00f067aa0ba902b7" {
		t.Fatalf("expected span id to be captured from traceparent, got %s", request.SpanID)
	}
}
