package sqlite3

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

func TestNormalizeType(t *testing.T) {
	if got := NormalizeType("int"); got != "INTEGER" {
		t.Fatalf("expected INTEGER, got %q", got)
	}

	if got := NormalizeType("varchar(100)"); got != "TEXT(100)" {
		t.Fatalf("expected TEXT(100), got %q", got)
	}
}

func TestGetDatabaseMetadataIncludesAliasMap(t *testing.T) {
	p := &Sqlite3Plugin{}
	p.Type = engine.DatabaseType_Sqlite3

	meta := p.GetDatabaseMetadata()
	if meta == nil {
		t.Fatalf("expected metadata, got nil")
	}
	if meta.DatabaseType != engine.DatabaseType_Sqlite3 {
		t.Fatalf("expected DatabaseType %q, got %q", engine.DatabaseType_Sqlite3, meta.DatabaseType)
	}
	if meta.AliasMap["INT"] != "INTEGER" {
		t.Fatalf("expected INT alias to be INTEGER, got %q", meta.AliasMap["INT"])
	}
}
