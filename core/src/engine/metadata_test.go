package engine

import (
	"errors"
	"testing"

	"github.com/clidey/whodb/core/src/source"
)

func TestValidateColumnTypeAllowsWhenMetadataMissing(t *testing.T) {
	if err := ValidateColumnType("NOT_A_REAL_TYPE", "TestDB", nil); err != nil {
		t.Fatalf("expected nil error when metadata is nil, got %v", err)
	}

	if err := ValidateColumnType("NOT_A_REAL_TYPE", "TestDB", &source.TypeSessionMetadata{}); err != nil {
		t.Fatalf("expected nil error when metadata has no type definitions, got %v", err)
	}
}

func TestValidateColumnTypeRejectsUnsupported(t *testing.T) {
	meta := &source.TypeSessionMetadata{
		TypeDefinitions: []TypeDefinition{
			{ID: "INTEGER"},
		},
		AliasMap: map[string]string{},
	}

	err := ValidateColumnType("NOT_A_REAL_TYPE", "TestDB", meta)
	var unsupported *UnsupportedTypeError
	if err == nil || !errors.As(err, &unsupported) {
		t.Fatalf("expected UnsupportedTypeError, got %v", err)
	}
	if unsupported.TypeName != "NOT_A_REAL_TYPE" {
		t.Fatalf("expected TypeName to be preserved, got %q", unsupported.TypeName)
	}
	if unsupported.DatabaseType != "TestDB" {
		t.Fatalf("expected DatabaseType to be preserved, got %q", unsupported.DatabaseType)
	}
}

func TestValidateColumnTypeResolvesAliases(t *testing.T) {
	meta := &source.TypeSessionMetadata{
		TypeDefinitions: []TypeDefinition{
			{ID: "INTEGER"},
			{ID: "VARCHAR", HasLength: true},
		},
		AliasMap: map[string]string{
			"INT": "INTEGER",
		},
	}

	if err := ValidateColumnType("int", "TestDB", meta); err != nil {
		t.Fatalf("expected alias INT -> INTEGER to validate, got %v", err)
	}

	if err := ValidateColumnType("VARCHAR(255)", "TestDB", meta); err != nil {
		t.Fatalf("expected parametrized type to validate, got %v", err)
	}
}

func TestValidateColumnTypeParams(t *testing.T) {
	meta := &source.TypeSessionMetadata{
		TypeDefinitions: []TypeDefinition{
			{ID: "VARCHAR", HasLength: true},
			{ID: "DECIMAL", HasPrecision: true},
			{ID: "DATETIME64"},
			{ID: "ENUM8"},
		},
		AliasMap: map[string]string{},
	}

	valid := []string{
		"VARCHAR(255)",
		"DECIMAL(10, 2)",
		"DATETIME64(3, 'UTC')",
		"ENUM8('active' = 1, 'inactive' = 2)",
	}
	for _, tn := range valid {
		if err := ValidateColumnType(tn, "TestDB", meta); err != nil {
			t.Errorf("expected %q to validate, got %v", tn, err)
		}
	}

	// Injection attempts: valid base type, malicious parameter section.
	invalid := []string{
		"VARCHAR(255); DROP TABLE users; --",
		"VARCHAR(255) DEFAULT (SELECT 1)",
		"DECIMAL(10)) ; DROP TABLE users; --",
		"VARCHAR(255) /* comment */",
		"ENUM8('a'=1) UNION SELECT",
	}
	for _, tn := range invalid {
		err := ValidateColumnType(tn, "TestDB", meta)
		var unsupported *UnsupportedTypeError
		if err == nil || !errors.As(err, &unsupported) {
			t.Errorf("expected %q to be rejected, got %v", tn, err)
		}
	}
}
