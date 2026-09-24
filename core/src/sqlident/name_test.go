package sqlident

import (
	"errors"
	"testing"
)

func TestNameSQLQuotesComponentsIndependently(t *testing.T) {
	name, err := New("tenant.with.dot", "orders; DROP TABLE secrets;--")
	if err != nil {
		t.Fatal(err)
	}

	got, err := name.SQL(DoubleQuote)
	if err != nil {
		t.Fatal(err)
	}
	want := `"tenant.with.dot"."orders; DROP TABLE secrets;--"`
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNameSQLEscapesClosingDelimiter(t *testing.T) {
	tests := []struct {
		name  string
		style QuoteStyle
		input string
		want  string
	}{
		{name: "double quote", style: DoubleQuote, input: `a"b`, want: `"a""b"`},
		{name: "backtick", style: Backtick, input: "a`b", want: "`a``b`"},
		{name: "bracket", style: Bracket, input: "a]b", want: "[a]]b]"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name, err := New("", test.input)
			if err != nil {
				t.Fatal(err)
			}
			got, err := name.SQL(test.style)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}

func TestNameSQLRejectsInvalidNamesAndStrictDelimiters(t *testing.T) {
	if _, err := New("", ""); !errors.Is(err, ErrInvalidIdentifier) {
		t.Fatalf("expected empty object name to fail, got %v", err)
	}
	if _, err := New("", "bad\x00name"); !errors.Is(err, ErrInvalidIdentifier) {
		t.Fatalf("expected NUL to fail, got %v", err)
	}

	name, err := New("", "a`b")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := name.SQL(BacktickStrict); !errors.Is(err, ErrInvalidIdentifier) {
		t.Fatalf("expected strict delimiter rejection, got %v", err)
	}
}

func TestIsSimple(t *testing.T) {
	for _, identifier := range []string{"", "orders", "tenant_42", "$internal", "café"} {
		if !IsSimple(identifier) {
			t.Fatalf("expected %q to be simple", identifier)
		}
	}
	for _, identifier := range []string{"order items", "schema.table", "quoted`name", "semi;colon"} {
		if IsSimple(identifier) {
			t.Fatalf("expected %q to require connector-native metadata lookup", identifier)
		}
	}
}
