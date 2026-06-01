package duckdb

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	duckdbDriver "github.com/duckdb/duckdb-go/v2"
	"gorm.io/gorm/clause"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
)

func TestDuckDBColumnCodec(t *testing.T) {
	plugin := &DuckDBPlugin{}

	testCases := []struct {
		name       string
		columnType string
		value      any
		want       string
	}{
		{
			name:       "uuid bytes",
			columnType: "UUID",
			value:      []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
			want:       "12345678-9abc-def0-1122-334455667788",
		},
		{
			name:       "blob bytes",
			columnType: "BLOB",
			value:      []byte{0x01, 0x02, 0x0a},
			want:       "0x01020A",
		},
		{
			name:       "date",
			columnType: "DATE",
			value:      time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
			want:       "2026-01-02",
		},
		{
			name:       "time",
			columnType: "TIME",
			value:      time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
			want:       "03:04:05",
		},
		{
			name:       "timestamp",
			columnType: "TIMESTAMP",
			value:      time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
			want:       "2026-01-02 03:04:05",
		},
		{
			name:       "interval",
			columnType: "INTERVAL",
			value:      duckdbDriver.Interval{Months: 14, Days: 3, Micros: int64((4*time.Hour + 5*time.Minute + 6*time.Second) / time.Microsecond)},
			want:       "1 year 2 months 3 days 4 hours 5 minutes 6 seconds",
		},
		{
			name:       "json value",
			columnType: "JSON",
			value:      map[string]any{"name": "duck"},
			want:       `{"name":"duck"}`,
		},
		{
			name:       "default formatting",
			columnType: "INTEGER",
			value:      42,
			want:       "42",
		},
	}

	for _, tc := range testCases {
		codec := plugin.GetColumnCodec(tc.columnType)
		if codec == nil {
			t.Fatalf("%s: expected DuckDB codec", tc.name)
		}
		scanner := any(tc.value)
		got, err := codec.Format(&scanner)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
		if got != tc.want {
			t.Fatalf("%s: expected %q, got %q", tc.name, tc.want, got)
		}
	}

	if got, err := plugin.GetColumnCodec("TEXT").Format(new(any)); err != nil || got != "" {
		t.Fatalf("expected nil scanner to format as empty string, got %q err=%v", got, err)
	}
}

func TestDuckDBHandleCustomDataType(t *testing.T) {
	plugin := &DuckDBPlugin{}

	exprValue, handled, err := plugin.HandleCustomDataType("", "UUID", true)
	if err != nil {
		t.Fatalf("expected empty nullable value to succeed, got %v", err)
	}
	if !handled || exprValue != nil {
		t.Fatalf("expected empty nullable value to become nil, got handled=%t value=%#v", handled, exprValue)
	}

	testCases := []struct {
		columnType string
		value      string
		wantSQL    string
		wantHandle bool
	}{
		{columnType: "INTERVAL", value: "1 day", wantSQL: "CAST('1 day' AS INTERVAL)", wantHandle: true},
		{columnType: "JSON", value: `{"k":"v"}`, wantSQL: `CAST('{"k":"v"}' AS JSON)`, wantHandle: true},
		{columnType: "UUID", value: "550e8400-e29b-41d4-a716-446655440000", wantSQL: "CAST('550e8400-e29b-41d4-a716-446655440000' AS UUID)", wantHandle: true},
		{columnType: "BLOB", value: "ABCD", wantSQL: "'ABCD'::BLOB", wantHandle: true},
		{columnType: "DECIMAL(10,2)", value: "12.34", wantSQL: "CAST('12.34' AS DECIMAL(10,2))", wantHandle: true},
		{columnType: "TEXT", value: "hello", wantHandle: false},
	}

	for _, tc := range testCases {
		value, handled, err := plugin.HandleCustomDataType(tc.value, tc.columnType, false)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.columnType, err)
		}
		if handled != tc.wantHandle {
			t.Fatalf("%s: expected handled=%t, got %t", tc.columnType, tc.wantHandle, handled)
		}
		if !handled {
			if value != nil {
				t.Fatalf("%s: expected nil value for unhandled type, got %#v", tc.columnType, value)
			}
			continue
		}
		expr, ok := value.(clause.Expr)
		if !ok {
			t.Fatalf("%s: expected gorm clause.Expr, got %#v", tc.columnType, value)
		}
		if expr.SQL != tc.wantSQL {
			t.Fatalf("%s: expected SQL %q, got %q", tc.columnType, tc.wantSQL, expr.SQL)
		}
	}
}

func TestDuckDBHelpers(t *testing.T) {
	pluginDef := NewDuckDBPlugin()
	plugin := pluginDef.PluginFunctions.(*DuckDBPlugin)

	if got := plugin.FormTableName("", "orders"); got != "orders" {
		t.Fatalf("expected bare table name, got %q", got)
	}
	if got := plugin.FormTableName("public", "orders"); got != "public.orders" {
		t.Fatalf("expected schema-qualified table name, got %q", got)
	}
	if got := plugin.ResolveGraphSchema(nil, "analytics"); got != "analytics" {
		t.Fatalf("expected schema to pass through, got %q", got)
	}
	if !plugin.ShouldCheckRowsAffected() {
		t.Fatal("expected DuckDB plugin to check rows affected")
	}
	if pluginDef.Type != engine.DatabaseType_DuckDB {
		t.Fatalf("expected DuckDB plugin type, got %q", pluginDef.Type)
	}
}

func TestDuckDBGetDatabases(t *testing.T) {
	plugin := &DuckDBPlugin{}
	originalDesktop := os.Getenv("WHODB_DESKTOP")
	originalCLI := os.Getenv("WHODB_CLI")
	originalIsDevelopment := env.IsDevelopment
	t.Cleanup(func() {
		_ = os.Setenv("WHODB_DESKTOP", originalDesktop)
		_ = os.Setenv("WHODB_CLI", originalCLI)
		env.IsDevelopment = originalIsDevelopment
	})

	if err := os.Setenv("WHODB_DESKTOP", "true"); err != nil {
		t.Fatalf("failed to set desktop env: %v", err)
	}
	if err := os.Setenv("WHODB_CLI", ""); err != nil {
		t.Fatalf("failed to clear cli env: %v", err)
	}
	if got, err := plugin.GetDatabases(&engine.PluginConfig{}); err != nil || len(got) != 0 {
		t.Fatalf("expected local mode to skip database discovery, got %#v err=%v", got, err)
	}

	if err := os.Setenv("WHODB_DESKTOP", ""); err != nil {
		t.Fatalf("failed to clear desktop env: %v", err)
	}
	env.IsDevelopment = true
	if err := os.MkdirAll("tmp", 0o755); err != nil {
		t.Fatalf("failed to create tmp directory: %v", err)
	}

	fileA := filepath.Join("tmp", "duckdb-test-a.db")
	fileB := filepath.Join("tmp", "duckdb-test-b.db")
	for _, path := range []string{fileA, fileB} {
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatalf("failed to create %s: %v", path, err)
		}
		t.Cleanup(func() {
			_ = os.Remove(path)
		})
	}

	got, err := plugin.GetDatabases(&engine.PluginConfig{})
	if err != nil {
		t.Fatalf("expected server mode database discovery to succeed, got %v", err)
	}
	if !slices.Contains(got, filepath.Base(fileA)) || !slices.Contains(got, filepath.Base(fileB)) {
		t.Fatalf("expected tmp database files to be discovered, got %#v", got)
	}
}
