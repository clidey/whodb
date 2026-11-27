package sqlite3

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

func TestNormalizeType(t *testing.T) {
	plugin := NewSqlite3Plugin().PluginFunctions.(*Sqlite3Plugin)

	if got := plugin.NormalizeType("int"); got != "INTEGER" {
		t.Fatalf("expected INTEGER, got %q", got)
	}

	if got := plugin.NormalizeType("varchar(100)"); got != "TEXT(100)" {
		t.Fatalf("expected TEXT(100), got %q", got)
	}
}

func TestSourceSessionMetadataIncludesAliasMap(t *testing.T) {
	meta, ok := sourcecatalog.ResolveSessionMetadata(string(engine.DatabaseType_Sqlite3))
	if !ok || meta == nil {
		t.Fatalf("expected metadata, got nil")
	}
	if meta.AliasMap["INT"] != "INTEGER" {
		t.Fatalf("expected INT alias to be INTEGER, got %q", meta.AliasMap["INT"])
	}
}

func TestSQLiteColumnTypesAreNotTrustedForExport(t *testing.T) {
	meta, ok := sourcecatalog.ResolveSessionMetadata(string(engine.DatabaseType_Sqlite3))
	if !ok {
		t.Fatal("expected SQLite session metadata")
	}
	// A non-STRICT INTEGER column can hold '=cmd()', so even numeric columns
	// must be guarded as free text.
	if text := source.FreeTextColumns(meta, []source.Column{{Name: "id", Type: "INTEGER"}}); !text[0] {
		t.Fatal("expected SQLite columns to count as free text whatever their declared type")
	}
}
