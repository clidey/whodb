package graph

import (
	"context"
	"testing"
	"time"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/internal/testutil"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/session"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateStandaloneSessionCreatesServerSession(t *testing.T) {
	var captured *engine.PluginConfig
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.IsAvailableFunc = func(_ context.Context, config *engine.PluginConfig) bool {
		captured = config
		return true
	}
	setEngineMock(t, mock)

	t.Setenv("WHODB_SESSION_ENCRYPTION_KEY", "12345678901234567890123456789012")
	var service *session.Service
	origFactory := session.DefaultServiceFactory
	session.DefaultServiceFactory = func() (*session.Service, error) {
		db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
		if err != nil {
			return nil, err
		}
		service, err = session.NewService(db, env.GetSessionEncryptionKey(), 24*time.Hour)
		return service, err
	}
	t.Cleanup(func() {
		session.DefaultServiceFactory = origFactory
		session.ResetDefaultService()
	})

	resolver := &Resolver{}
	payload, err := resolver.Mutation().CreateStandaloneSession(context.Background(), model.LoginCredentials{
		Type:     "Postgres",
		Hostname: "localhost",
		Username: "postgres",
		Password: "secret",
		Database: "postgres",
		Advanced: []*model.RecordInput{{Key: "Port", Value: "5432"}},
	})
	if err != nil {
		t.Fatalf("expected standalone session to succeed, got %v", err)
	}
	if payload == nil || payload.SessionToken == "" {
		t.Fatalf("expected session token payload, got %#v", payload)
	}
	if payload.DisplayName != "Postgres @ localhost/postgres" {
		t.Fatalf("expected generated display name, got %q", payload.DisplayName)
	}
	if payload.Type != "Postgres" || payload.Hostname != "localhost" || payload.Port != "5432" || payload.Database != "postgres" {
		t.Fatalf("expected connection summary to roundtrip, got %#v", payload)
	}
	if captured == nil || captured.Credentials == nil || captured.Credentials.Password != "secret" {
		t.Fatalf("expected plugin connectivity check to receive credentials, got %#v", captured)
	}

	credentials, record, err := service.ResolveToken(context.Background(), payload.SessionToken)
	if err != nil {
		t.Fatalf("expected returned token to resolve, got %v", err)
	}
	if record.Source != "standalone" {
		t.Fatalf("expected standalone source, got %q", record.Source)
	}
	if credentials.Password != "secret" || credentials.Database != "postgres" {
		t.Fatalf("expected encrypted credentials to roundtrip, got %#v", credentials)
	}
}

func TestCreateStandaloneSessionDisplayNameOmitsEmptyDatabase(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Redis"))
	mock.IsAvailableFunc = func(context.Context, *engine.PluginConfig) bool { return true }
	setEngineMock(t, mock)
	getService := newStandaloneSessionServiceForTest(t)

	resolver := &Resolver{}
	payload, err := resolver.Mutation().CreateStandaloneSession(context.Background(), model.LoginCredentials{
		Type:     "Redis",
		Hostname: "localhost",
		Username: "",
		Password: "secret",
		Database: "",
		Advanced: []*model.RecordInput{{Key: "Port", Value: "6379"}},
	})
	if err != nil {
		t.Fatalf("expected standalone session to succeed, got %v", err)
	}
	if payload.DisplayName != "Redis @ localhost" {
		t.Fatalf("expected display name without database, got %q", payload.DisplayName)
	}

	_, record, err := getService().ResolveToken(context.Background(), payload.SessionToken)
	if err != nil {
		t.Fatalf("expected returned token to resolve, got %v", err)
	}
	if record.Source != "standalone" || record.DatabaseName != "" {
		t.Fatalf("expected standalone Redis session without database, got %#v", record)
	}
}

func TestCreateStandaloneSessionRejectsFailedConnectivity(t *testing.T) {
	mock := testutil.NewPluginMock(engine.DatabaseType("Postgres"))
	mock.IsAvailableFunc = func(context.Context, *engine.PluginConfig) bool { return false }
	setEngineMock(t, mock)

	origFactory := session.DefaultServiceFactory
	serviceRequested := false
	session.DefaultServiceFactory = func() (*session.Service, error) {
		serviceRequested = true
		return nil, nil
	}
	t.Cleanup(func() {
		session.DefaultServiceFactory = origFactory
		session.ResetDefaultService()
	})

	resolver := &Resolver{}
	payload, err := resolver.Mutation().CreateStandaloneSession(context.Background(), model.LoginCredentials{
		Type:     "Postgres",
		Hostname: "localhost",
		Username: "postgres",
		Password: "wrong",
		Database: "postgres",
		Advanced: []*model.RecordInput{{Key: "Port", Value: "5432"}},
	})
	if err == nil {
		t.Fatalf("expected standalone session to fail when connectivity fails")
	}
	if payload != nil {
		t.Fatalf("expected no payload on failed connectivity, got %#v", payload)
	}
	if serviceRequested {
		t.Fatalf("expected failed connectivity to avoid session creation")
	}
}

func newStandaloneSessionServiceForTest(t *testing.T) func() *session.Service {
	t.Helper()

	t.Setenv("WHODB_SESSION_ENCRYPTION_KEY", "12345678901234567890123456789012")
	var service *session.Service
	origFactory := session.DefaultServiceFactory
	session.DefaultServiceFactory = func() (*session.Service, error) {
		db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
		if err != nil {
			return nil, err
		}
		service, err = session.NewService(db, env.GetSessionEncryptionKey(), 24*time.Hour)
		return service, err
	}
	t.Cleanup(func() {
		session.DefaultServiceFactory = origFactory
		session.ResetDefaultService()
	})
	return func() *session.Service {
		return service
	}
}
