package clickhouse

import (
	"reflect"
	"strings"
	"testing"

	"gorm.io/gorm/clause"
)

func TestSplitTopLevelRespectsQuotesAndNesting(t *testing.T) {
	input := "'a,b', {'k1': 1, 'k2': 2}, (1, 2, 3)"
	parts := splitTopLevel(input, ',')
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d: %#v", len(parts), parts)
	}
}

func TestConvertArrayLiteralTypedSlice(t *testing.T) {
	val, err := convertArrayLiteral("[1, 2, 3]", "Array(Int32)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := val.([]int32)
	if !ok {
		t.Fatalf("expected []int32, got %T", val)
	}
	if !reflect.DeepEqual(arr, []int32{1, 2, 3}) {
		t.Fatalf("unexpected array: %#v", arr)
	}
}

func TestConvertMapLiteralTypedMap(t *testing.T) {
	val, err := convertMapLiteral("{'key1': 10, 'key2': 20}", "Map(String, Int32)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := val.(map[string]int32)
	if !ok {
		t.Fatalf("expected map[string]int32, got %T", val)
	}
	if m["key1"] != 10 || m["key2"] != 20 {
		t.Fatalf("unexpected map: %#v", m)
	}
}

func TestFormatLiteralEscapesQuotesAndBackslashes(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"plain", "'plain'"},
		{"O'Brien", `'O\'Brien'`},
		{`back\slash`, `'back\\slash'`},
		{`'); DROP TABLE users; --`, `'\'); DROP TABLE users; --'`},
	}
	for _, tc := range cases {
		if got := formatLiteral(tc.in); got != tc.want {
			t.Errorf("formatLiteral(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestArrayToExprEscapesMaliciousValues verifies the raw-SQL sink for Array-column
// UPDATEs cannot be broken out of via a crafted string element.
func TestArrayToExprEscapesMaliciousValues(t *testing.T) {
	expr, ok := arrayToExpr([]string{"ok", "'); DROP TABLE users; --"}).(clause.Expr)
	if !ok {
		t.Fatalf("expected clause.Expr, got %T", arrayToExpr([]string{}))
	}
	// The malicious quote must be escaped, so no unescaped closing quote can
	// terminate the literal early. The dangerous SQL stays inside the string.
	if strings.Contains(expr.SQL, "'); DROP") && !strings.Contains(expr.SQL, `\'); DROP`) {
		t.Fatalf("array element quote not escaped: %s", expr.SQL)
	}
	if !strings.Contains(expr.SQL, `\'`) {
		t.Fatalf("expected escaped single quote in %s", expr.SQL)
	}
}

func TestConvertTupleLiteralReturnsTypedValues(t *testing.T) {
	val, err := convertTupleLiteral("('hello', 42, 3.14)", "Tuple(String, Int32, Float64)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	items, ok := val.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", val)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0] != "hello" {
		t.Fatalf("expected first element to be hello, got %#v", items[0])
	}
	if _, ok := items[1].(int32); !ok {
		t.Fatalf("expected second element to be int32, got %T", items[1])
	}
	if _, ok := items[2].(float64); !ok {
		t.Fatalf("expected third element to be float64, got %T", items[2])
	}
}
