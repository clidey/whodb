package auth

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/zalando/go-keyring"
)

func TestSaveLoadDeleteCredentialsUsesKeyringWhenDesktopMode(t *testing.T) {
	keyring.MockInit()
	t.Setenv("WHODB_DESKTOP", "true")

	creds := &engine.Credentials{
		Id:       strPtr("profile1"),
		Type:     "Postgres",
		Hostname: "db.local",
		Username: "alice",
		Password: "pw",
		Database: "app",
	}

	if err := SaveCredentials(*creds.Id, creds); err != nil {
		t.Fatalf("expected save to succeed with mocked keyring, got %v", err)
	}

	loaded, err := LoadCredentials(*creds.Id)
	if err != nil {
		t.Fatalf("expected load to succeed, got %v", err)
	}
	if loaded == nil || loaded.Username != "alice" || loaded.Password != "pw" {
		t.Fatalf("loaded credentials mismatch: %#v", loaded)
	}

	if err := DeleteCredentials(*creds.Id); err != nil {
		t.Fatalf("expected delete to succeed, got %v", err)
	}
	if _, err := LoadCredentials(*creds.Id); err == nil {
		t.Fatalf("expected subsequent load to fail after deletion")
	}
}

func TestSaveCredentialsNoopWhenNotDesktop(t *testing.T) {
	t.Setenv("WHODB_DESKTOP", "false")
	if err := SaveCredentials("id", &engine.Credentials{}); err != nil {
		t.Fatalf("expected noop save to succeed without error")
	}
}

func strPtr(s string) *string { return &s }
