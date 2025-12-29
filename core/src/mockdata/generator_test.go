package mockdata

import (
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/clidey/whodb/core/src/engine"
)

func TestGenerateRowDataWithConstraintsSkipsSerialAndRespectsCheckValues(t *testing.T) {
	g := NewGenerator()
	g.faker = gofakeit.New(1) // deterministic output

	columns := []engine.Column{
		{Name: "id", Type: "serial"},
		{Name: "status", Type: "varchar(10)"},
		{Name: "created_at", Type: "timestamp"},
	}
	constraints := map[string]map[string]any{
		"status": {"check_values": []string{"open", "closed"}},
	}

	records, err := g.GenerateRowDataWithConstraints(columns, constraints)
	if err != nil {
		t.Fatalf("unexpected error generating row data: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected serial column to be skipped, got %d records", len(records))
	}

	status := findRecord(records, "status")
	if status == nil {
		t.Fatalf("expected status column to be present")
	}
	if status.Value != "open" && status.Value != "closed" {
		t.Fatalf("expected status to respect check_values constraint, got %s", status.Value)
	}
	if status.Extra["Type"] != "varchar(10)" {
		t.Fatalf("expected type to be stored in Extra metadata")
	}

	created := findRecord(records, "created_at")
	if created == nil {
		t.Fatalf("expected created_at column to be present")
	}
	if _, err := time.Parse("2006-01-02 15:04:05", created.Value); err != nil {
		t.Fatalf("expected timestamp to be formatted without timezone, got %s", created.Value)
	}
}

func TestGenerateRowWithDefaultsCoversCommonTypes(t *testing.T) {
	g := NewGenerator()
	g.faker = gofakeit.New(1)

	columns := []engine.Column{
		{Name: "id", Type: "serial"},
		{Name: "price", Type: "decimal(10,2)"},
		{Name: "active", Type: "boolean"},
		{Name: "created", Type: "date"},
		{Name: "note", Type: "varchar(2)"},
		{Name: "payload", Type: "jsonb"},
	}

	records := g.GenerateRowWithDefaults(columns)
	if len(records) != 5 { // serial skipped
		t.Fatalf("expected serial column to be skipped, got %d records", len(records))
	}

	price := findRecord(records, "price")
	if price == nil || price.Value != "0.0" {
		t.Fatalf("expected decimal default to be 0.0, got %#v", price)
	}

	active := findRecord(records, "active")
	if active == nil || (active.Value != "false" && active.Value != "true") {
		t.Fatalf("expected boolean default to be present, got %#v", active)
	}

	note := findRecord(records, "note")
	if note == nil || len(note.Value) != 2 {
		t.Fatalf("expected varchar(2) default to respect length, got %#v", note)
	}

	payload := findRecord(records, "payload")
	if payload == nil || !strings.HasPrefix(payload.Value, "{") {
		t.Fatalf("expected json default to be object, got %#v", payload)
	}
}

func TestGenerateRowDataWithConstraintsHandlesArrays(t *testing.T) {
	g := NewGenerator()
	g.faker = gofakeit.New(2)

	columns := []engine.Column{
		{Name: "tags", Type: "text[]"},
	}

	records, err := g.GenerateRowDataWithConstraints(columns, nil)
	if err != nil {
		t.Fatalf("unexpected error generating array data: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected one record, got %d", len(records))
	}
	if !strings.HasPrefix(records[0].Value, "{") || !strings.HasSuffix(records[0].Value, "}") {
		t.Fatalf("expected array value to be wrapped in braces, got %s", records[0].Value)
	}
}

func findRecord(records []engine.Record, key string) *engine.Record {
	for _, r := range records {
		if r.Key == key {
			return &r
		}
	}
	return nil
}

func TestDetectDatabaseTypeHandlesArrayAndDefaults(t *testing.T) {
	if got := detectDatabaseType("text[]"); got != "array" {
		t.Fatalf("expected array type for text[], got %s", got)
	}
	if got := detectDatabaseType("unknown_type"); got != "text" {
		t.Fatalf("expected unknown types to default to text, got %s", got)
	}
}

func TestParseMaxLenExtractsLength(t *testing.T) {
	if got := parseMaxLen("varchar(42)"); got != 42 {
		t.Fatalf("expected length 42, got %d", got)
	}
	if got := parseMaxLen("varchar"); got != 0 {
		t.Fatalf("expected zero when no length specified, got %d", got)
	}
}
