package mysql

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

func TestNormalizeType(t *testing.T) {
	if got := NormalizeType("integer"); got != "INT" {
		t.Fatalf("expected INT, got %q", got)
	}

	if got := NormalizeType("character varying(50)"); got != "VARCHAR(50)" {
		t.Fatalf("expected VARCHAR(50), got %q", got)
	}
}

func TestGetDatabaseMetadataUsesPluginType(t *testing.T) {
	p := &MySQLPlugin{}
	p.Type = engine.DatabaseType_MySQL

	meta := p.GetDatabaseMetadata()
	if meta == nil {
		t.Fatalf("expected metadata, got nil")
	}
	if meta.DatabaseType != engine.DatabaseType_MySQL {
		t.Fatalf("expected DatabaseType %q, got %q", engine.DatabaseType_MySQL, meta.DatabaseType)
	}
	if meta.AliasMap["INTEGER"] != "INT" {
		t.Fatalf("expected INTEGER alias to be INT, got %q", meta.AliasMap["INTEGER"])
	}
}
