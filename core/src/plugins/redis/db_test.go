package redis

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

func TestDBRejectsInvalidPortBeforeDialing(t *testing.T) {
	cfg := &engine.PluginConfig{
		Credentials: &engine.Credentials{
			Type:     string(engine.DatabaseType_Redis),
			Hostname: "localhost",
			Advanced: []engine.Record{
				{Key: "Port", Value: "not-a-number"},
			},
		},
	}

	_, err := DB(cfg)
	if err == nil {
		t.Fatalf("expected error for invalid port")
	}
}

func TestDBRejectsInvalidDatabaseBeforeDialing(t *testing.T) {
	cfg := &engine.PluginConfig{
		Credentials: &engine.Credentials{
			Type:     string(engine.DatabaseType_Redis),
			Hostname: "localhost",
			Database: "not-a-number",
		},
	}

	_, err := DB(cfg)
	if err == nil {
		t.Fatalf("expected error for invalid database")
	}
}
