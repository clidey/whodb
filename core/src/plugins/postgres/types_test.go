package postgres

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

func TestNormalizeType(t *testing.T) {
	if got := NormalizeType("int"); got != "INTEGER" {
		t.Fatalf("expected INTEGER, got %q", got)
	}

	if got := NormalizeType("varchar(25)"); got != "CHARACTER VARYING(25)" {
		t.Fatalf("expected CHARACTER VARYING(25), got %q", got)
	}
}

func TestGetDatabaseMetadataIncludesAliasMap(t *testing.T) {
	p := &PostgresPlugin{}
	p.Type = engine.DatabaseType_Postgres

	meta := p.GetDatabaseMetadata()
	if meta == nil {
		t.Fatalf("expected metadata, got nil")
	}
	if meta.DatabaseType != engine.DatabaseType_Postgres {
		t.Fatalf("expected DatabaseType %q, got %q", engine.DatabaseType_Postgres, meta.DatabaseType)
	}
	if meta.AliasMap["INT"] != "INTEGER" {
		t.Fatalf("expected INT alias to be INTEGER, got %q", meta.AliasMap["INT"])
	}
}
