package common

import "testing"

func TestParseAndFormatTypeSpec(t *testing.T) {
	t.Run("varchar length", func(t *testing.T) {
		spec := ParseTypeSpec("VARCHAR(255)")
		if spec.BaseType != "VARCHAR" || spec.Length != 255 || spec.Precision != 0 || spec.Scale != 0 {
			t.Fatalf("unexpected spec: %#v", spec)
		}
		if got := FormatTypeSpec(spec); got != "VARCHAR(255)" {
			t.Fatalf("expected VARCHAR(255), got %q", got)
		}
	})

	t.Run("decimal precision scale", func(t *testing.T) {
		spec := ParseTypeSpec("DECIMAL(10,2)")
		if spec.BaseType != "DECIMAL" || spec.Precision != 10 || spec.Scale != 2 || spec.Length != 0 {
			t.Fatalf("unexpected spec: %#v", spec)
		}
		if got := FormatTypeSpec(spec); got != "DECIMAL(10,2)" {
			t.Fatalf("expected DECIMAL(10,2), got %q", got)
		}
	})

	t.Run("bare type", func(t *testing.T) {
		spec := ParseTypeSpec("integer")
		if spec.BaseType != "INTEGER" || spec.Length != 0 || spec.Precision != 0 || spec.Scale != 0 {
			t.Fatalf("unexpected spec: %#v", spec)
		}
		if got := FormatTypeSpec(spec); got != "INTEGER" {
			t.Fatalf("expected INTEGER, got %q", got)
		}
	})
}

func TestNormalizeTypeWithMapPreservesParams(t *testing.T) {
	aliasMap := map[string]string{
		"INT":     "INTEGER",
		"VARCHAR": "CHARACTER VARYING",
	}

	if got := NormalizeTypeWithMap("int", aliasMap); got != "INTEGER" {
		t.Fatalf("expected INTEGER, got %q", got)
	}

	if got := NormalizeTypeWithMap("varchar(100)", aliasMap); got != "CHARACTER VARYING(100)" {
		t.Fatalf("expected CHARACTER VARYING(100), got %q", got)
	}

	if got := NormalizeTypeWithMap("unknown(3)", aliasMap); got != "UNKNOWN(3)" {
		t.Fatalf("expected UNKNOWN(3), got %q", got)
	}
}
