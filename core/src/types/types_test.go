package types

import (
	"encoding/json"
	"testing"
)

func TestDatabaseCredentials_UnmarshalJSON_AdvancedKey(t *testing.T) {
	input := `{"host":"db.local","user":"alice","password":"secret","database":"app","advanced":{"SSL Mode":"require","Connection Timeout":"30"}}`

	var creds DatabaseCredentials
	if err := json.Unmarshal([]byte(input), &creds); err != nil {
		t.Fatalf("failed to unmarshal with advanced key: %v", err)
	}

	if creds.Hostname != "db.local" {
		t.Errorf("expected host db.local, got %s", creds.Hostname)
	}
	if len(creds.Advanced) != 2 {
		t.Fatalf("expected 2 advanced entries, got %d", len(creds.Advanced))
	}
	if creds.Advanced["SSL Mode"] != "require" {
		t.Errorf("expected SSL Mode=require, got %s", creds.Advanced["SSL Mode"])
	}
	if creds.Advanced["Connection Timeout"] != "30" {
		t.Errorf("expected Connection Timeout=30, got %s", creds.Advanced["Connection Timeout"])
	}
}

func TestDatabaseCredentials_UnmarshalJSON_LegacyConfigKey(t *testing.T) {
	input := `{"host":"db.local","user":"alice","password":"secret","database":"app","config":{"SSL Mode":"verify-ca","Port":"5433"}}`

	var creds DatabaseCredentials
	if err := json.Unmarshal([]byte(input), &creds); err != nil {
		t.Fatalf("failed to unmarshal with legacy config key: %v", err)
	}

	if len(creds.Advanced) != 2 {
		t.Fatalf("expected 2 advanced entries from legacy config, got %d", len(creds.Advanced))
	}
	if creds.Advanced["SSL Mode"] != "verify-ca" {
		t.Errorf("expected SSL Mode=verify-ca, got %s", creds.Advanced["SSL Mode"])
	}
	if creds.Advanced["Port"] != "5433" {
		t.Errorf("expected Port=5433, got %s", creds.Advanced["Port"])
	}
}

func TestDatabaseCredentials_UnmarshalJSON_AdvancedTakesPrecedence(t *testing.T) {
	// If both "advanced" and "config" are present, "advanced" wins
	input := `{"host":"db.local","user":"alice","password":"secret","database":"app","advanced":{"SSL Mode":"require"},"config":{"SSL Mode":"verify-full"}}`

	var creds DatabaseCredentials
	if err := json.Unmarshal([]byte(input), &creds); err != nil {
		t.Fatalf("failed to unmarshal with both keys: %v", err)
	}

	if creds.Advanced["SSL Mode"] != "require" {
		t.Errorf("expected advanced to take precedence, got SSL Mode=%s", creds.Advanced["SSL Mode"])
	}
}

func TestDatabaseCredentials_UnmarshalJSON_NoAdvancedOrConfig(t *testing.T) {
	input := `{"host":"db.local","user":"alice","password":"secret","database":"app"}`

	var creds DatabaseCredentials
	if err := json.Unmarshal([]byte(input), &creds); err != nil {
		t.Fatalf("failed to unmarshal without advanced or config: %v", err)
	}

	if creds.Advanced != nil {
		t.Errorf("expected nil Advanced when neither key is present, got %v", creds.Advanced)
	}
}
