package clickhouse

import "testing"

func TestIsMultiStatement(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  bool
	}{
		{name: "no semicolons", query: "SELECT 1", want: false},
		{name: "single statement", query: "SELECT 1;", want: false},
		{name: "two statements", query: "SELECT 1; SELECT 2;", want: true},
		{name: "line comment ignored", query: "SELECT 1; -- comment;\n", want: false},
		{name: "hash comment ignored", query: "SELECT 1; # comment;\n", want: false},
		{name: "block comment ignored", query: "SELECT 1; /* ; */ SELECT 2;", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isMultiStatement(tc.query); got != tc.want {
				t.Fatalf("isMultiStatement(%q) = %v, want %v", tc.query, got, tc.want)
			}
		})
	}
}
