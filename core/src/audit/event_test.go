package audit

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestPrepareEventUsesSpanContextForTraceFields(t *testing.T) {
	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatalf("trace.TraceIDFromHex() error = %v", err)
	}
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		t.Fatalf("trace.SpanIDFromHex() error = %v", err)
	}

	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
		Remote:  true,
	}))

	event := prepareEvent(ctx, AuditEvent{Action: "http.request"}, func(context.Context) Actor {
		return Actor{}
	}, noOpEventEnricher)

	if event.Request.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("expected trace id from span context, got %q", event.Request.TraceID)
	}
	if event.Request.SpanID != "00f067aa0ba902b7" {
		t.Fatalf("expected span id from span context, got %q", event.Request.SpanID)
	}
}

func TestPrepareEventUsesContextScopeForOrgAndProject(t *testing.T) {
	ctx := WithScope(context.Background(), Scope{
		OrgID:     "org-123",
		ProjectID: "project-456",
	})

	event := prepareEvent(ctx, AuditEvent{Action: "graphql.query.ProjectTransforms"}, func(context.Context) Actor {
		return Actor{}
	}, noOpEventEnricher)

	if event.OrgID != "org-123" {
		t.Fatalf("expected org id from context scope, got %q", event.OrgID)
	}
	if event.ProjectID != "project-456" {
		t.Fatalf("expected project id from context scope, got %q", event.ProjectID)
	}
}
