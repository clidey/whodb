package audit

import (
	"context"
	"testing"
)

type testAuditService struct {
	event AuditEvent
}

func (s *testAuditService) Record(event AuditEvent) {
	s.event = event
}

func TestRecordWithContextUsesActorProvider(t *testing.T) {
	service := &testAuditService{}
	SetAuditService(service)
	SetActorProvider(func(context.Context) Actor {
		return Actor{ID: "user-1", Type: "user"}
	})
	t.Cleanup(func() {
		SetAuditService(nil)
		SetActorProvider(nil)
	})

	RecordWithContext(context.Background(), AuditEvent{Action: "query.execute"})

	if service.event.Actor.ID != "user-1" {
		t.Fatalf("expected actor id to be enriched, got %q", service.event.Actor.ID)
	}
	if service.event.Actor.Type != "user" {
		t.Fatalf("expected actor type to be enriched, got %q", service.event.Actor.Type)
	}
}

func TestRecordWithContextPreservesExplicitActor(t *testing.T) {
	service := &testAuditService{}
	SetAuditService(service)
	SetActorProvider(func(context.Context) Actor {
		return Actor{ID: "user-1", Type: "user"}
	})
	t.Cleanup(func() {
		SetAuditService(nil)
		SetActorProvider(nil)
	})

	RecordWithContext(context.Background(), AuditEvent{
		Action: "query.execute",
		Actor:  Actor{ID: "system", Type: "system"},
	})

	if service.event.Actor.ID != "system" {
		t.Fatalf("expected explicit actor id to be preserved, got %q", service.event.Actor.ID)
	}
	if service.event.Actor.Type != "system" {
		t.Fatalf("expected explicit actor type to be preserved, got %q", service.event.Actor.Type)
	}
}
