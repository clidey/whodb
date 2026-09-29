package duckdb

import (
	"testing"

	"gorm.io/gorm"
)

func TestQuotedColumnTypesRetainsDuckDBTypeParameters(t *testing.T) {
	db, err := gorm.Open(Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open DuckDB: %v", err)
	}
	const table = `identifier"; DROP TABLE guard;--`
	if err := db.Exec(`CREATE TABLE "identifier""; DROP TABLE guard;--" (id INTEGER PRIMARY KEY, name VARCHAR(50), amount DECIMAL(12,3))`).Error; err != nil {
		t.Fatalf("create quoted DuckDB table: %v", err)
	}

	plugin := NewDuckDBPlugin().PluginFunctions.(*DuckDBPlugin)
	columns, err := plugin.QuotedColumnTypes(db, "main", table)
	if err != nil {
		t.Fatalf("read quoted DuckDB metadata: %v", err)
	}
	if len(columns) != 3 {
		t.Fatalf("expected three DuckDB columns, got %#v", columns)
	}
	if columns[1].DatabaseTypeName() != "VARCHAR" {
		t.Fatalf("expected DuckDB VARCHAR metadata, got %q", columns[1].DatabaseTypeName())
	}
	if precision, scale, ok := columns[2].DecimalSize(); !ok || precision != 12 || scale != 3 {
		t.Fatalf("expected DECIMAL(12,3) metadata, got precision=%d scale=%d ok=%t", precision, scale, ok)
	}
}
