package auth

import (
	"testing"

	"github.com/clidey/whodb/core/src/source"
	"github.com/zalando/go-keyring"
)

func TestSaveLoadDeleteCredentialsUsesKeyringWhenDesktopMode(t *testing.T) {
	keyring.MockInit()
	t.Setenv("WHODB_DESKTOP", "true")

	creds := &source.Credentials{
		ID:         new("profile1"),
		SourceType: "Postgres",
		Values: map[string]string{
			"Hostname": "db.local",
			"Username": "alice",
			"Password": "pw",
			"Database": "app",
		},
	}

	if err := SaveCredentials(*creds.ID, creds); err != nil {
		t.Fatalf("expected save to succeed with mocked keyring, got %v", err)
	}

	loaded, err := LoadCredentials(*creds.ID)
	if err != nil {
		t.Fatalf("expected load to succeed, got %v", err)
	}
	if loaded == nil || loaded.Values["Username"] != "alice" || loaded.Values["Password"] != "pw" {
		t.Fatalf("loaded credentials mismatch: %#v", loaded)
	}

	if err := DeleteCredentials(*creds.ID); err != nil {
		t.Fatalf("expected delete to succeed, got %v", err)
	}
	if _, err := LoadCredentials(*creds.ID); err == nil {
		t.Fatalf("expected subsequent load to fail after deletion")
	}
}

func TestSaveCredentialsNoopWhenNotDesktop(t *testing.T) {
	t.Setenv("WHODB_DESKTOP", "false")
	if err := SaveCredentials("id", &source.Credentials{}); err != nil {
		t.Fatalf("expected noop save to succeed without error")
	}
}

//go:fix inline
func strPtr(s string) *string { return new(s) }
