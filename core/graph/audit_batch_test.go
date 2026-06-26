package graph

import (
	"context"
	"testing"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/audit"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/settings"
	"github.com/clidey/whodb/core/src/source"
)

type capturingAuditService struct {
	events []audit.AuditEvent
}

func (s *capturingAuditService) Record(event audit.AuditEvent) {
	s.events = append(s.events, event)
}

func TestUpdateSettingsEmitsDedicatedAuditEvent(t *testing.T) {
	originalSettings := settings.Get()
	settings.UpdateSettings(settings.MetricsEnabledField(false))
	t.Cleanup(func() {
		settings.UpdateSettings(settings.MetricsEnabledField(originalSettings.MetricsEnabled))
	})

	service := &capturingAuditService{}
	audit.SetAuditService(service)
	audit.SetActorProvider(nil)
	t.Cleanup(func() {
		audit.SetAuditService(nil)
		audit.SetActorProvider(nil)
	})

	status, err := (&Resolver{}).Mutation().UpdateSettings(context.Background(), model.SettingsConfigInput{
		MetricsEnabled: strPtr("true"),
	})
	if err != nil {
		t.Fatalf("expected settings update to succeed, got %v", err)
	}
	if status == nil || !status.Status {
		t.Fatalf("expected settings update status true, got %#v", status)
	}
	if len(service.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(service.events))
	}

	event := service.events[0]
	if event.Action != "settings.update" {
		t.Fatalf("expected settings.update action, got %s", event.Action)
	}
	if event.Outcome != audit.OutcomeSuccess {
		t.Fatalf("expected success outcome, got %s", event.Outcome)
	}
	if before, ok := event.Details["metrics_enabled_before"].(bool); !ok || before {
		t.Fatalf("expected metrics_enabled_before false, got %#v", event.Details["metrics_enabled_before"])
	}
	if after, ok := event.Details["metrics_enabled_after"].(bool); !ok || !after {
		t.Fatalf("expected metrics_enabled_after true, got %#v", event.Details["metrics_enabled_after"])
	}
}

func TestPerformSourceLoginDisabledCredentialsEmitsDeniedAudit(t *testing.T) {
	originalDisableCredentialForm := env.DisableCredentialForm
	env.DisableCredentialForm = true
	t.Cleanup(func() {
		env.DisableCredentialForm = originalDisableCredentialForm
	})

	service := &capturingAuditService{}
	audit.SetAuditService(service)
	audit.SetActorProvider(nil)
	t.Cleanup(func() {
		audit.SetAuditService(nil)
		audit.SetActorProvider(nil)
	})

	_, err := performSourceLogin(context.Background(), &source.Credentials{SourceType: "Postgres"}, "")
	if err == nil {
		t.Fatal("expected login to fail when credential form is disabled")
	}
	if len(service.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(service.events))
	}
	if service.events[0].Action != "login.source" {
		t.Fatalf("expected login.source action, got %s", service.events[0].Action)
	}
	if service.events[0].Outcome != audit.OutcomeDenied {
		t.Fatalf("expected denied audit outcome, got %s", service.events[0].Outcome)
	}
}

func TestPerformSourceLoginPreconfiguredProfileBypassesDisabledCredentials(t *testing.T) {
	originalDisableCredentialForm := env.DisableCredentialForm
	env.DisableCredentialForm = true
	t.Cleanup(func() {
		env.DisableCredentialForm = originalDisableCredentialForm
	})

	audit.SetAuditService(nil)
	audit.SetActorProvider(nil)
	t.Cleanup(func() {
		audit.SetAuditService(nil)
		audit.SetActorProvider(nil)
	})

	// A login originating from a preconfigured profile carries a non-empty
	// profileSource and must not be rejected by the disabled credential form
	// gate. An unknown source type is used so execution stops at the catalog
	// lookup (returning "unauthorized") rather than attempting a real
	// connection; the key assertion is that the gate did not reject it.
	_, err := performSourceLogin(context.Background(), &source.Credentials{SourceType: "nonexistent-source"}, "environment")
	if err == nil {
		t.Fatal("expected an error for an unknown source type")
	}
	if err.Error() == "login with credentials is disabled; use preconfigured connections" {
		t.Fatal("preconfigured profile login should bypass the disabled credential form gate")
	}
}

func strPtr(value string) *string {
	return &value
}
