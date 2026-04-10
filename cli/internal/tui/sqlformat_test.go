/*
 * Copyright 2025 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tui

import (
	"strings"
	"testing"
)

func TestFormatSQL_Basic(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string // lines the output should contain
	}{
		{
			name:  "simple select",
			input: "select * from users where id = 1",
			want:  []string{"SELECT", "FROM", "WHERE"},
		},
		{
			name:  "keywords uppercased",
			input: "select name from users order by name limit 10",
			want:  []string{"SELECT", "FROM", "ORDER BY", "LIMIT"},
		},
		{
			name:  "join formatted",
			input: "select u.name from users u left join orders o on u.id = o.user_id",
			want:  []string{"SELECT", "FROM", "LEFT JOIN", "ON"},
		},
		{
			name:  "where with and",
			input: "select * from users where active = true and age > 18",
			want:  []string{"WHERE", "AND"},
		},
		{
			name:  "preserves string literals",
			input: "select * from users where name = 'hello world'",
			want:  []string{"'hello world'"},
		},
		{
			name:  "preserves quoted identifiers",
			input: `select "user name" from "my table"`,
			want:  []string{`"user name"`, `"my table"`},
		},
		{
			name:  "empty input",
			input: "",
			want:  []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSQL(tt.input)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("formatSQL(%q) missing %q\ngot: %s", tt.input, want, got)
				}
			}
		})
	}
}

func TestFormatSQL_NewlinesForClauses(t *testing.T) {
	input := "select id, name from users where id > 1 order by name limit 10"
	got := formatSQL(input)
	lines := strings.Split(got, "\n")

	if len(lines) < 4 {
		t.Errorf("Expected at least 4 lines, got %d:\n%s", len(lines), got)
	}

	// First line should start with SELECT
	if !strings.HasPrefix(strings.TrimSpace(lines[0]), "SELECT") {
		t.Errorf("First line should start with SELECT, got: %s", lines[0])
	}
}

func TestFormatSQL_PreservesTableAndColumnNames(t *testing.T) {
	input := "select runId from bike_settings where active = true"
	got := formatSQL(input)

	if !strings.Contains(got, "runId") {
		t.Errorf("Should preserve column name 'runId', got: %s", got)
	}
	if !strings.Contains(got, "bike_settings") {
		t.Errorf("Should preserve table name 'bike_settings', got: %s", got)
	}
}

func TestTokenizeSQL(t *testing.T) {
	tokens := tokenizeSQL("SELECT 'hello' FROM \"users\"")

	kinds := []tokenKind{}
	for _, tok := range tokens {
		if tok.kind != tokenWhitespace {
			kinds = append(kinds, tok.kind)
		}
	}

	// SELECT (word), 'hello' (string), FROM (word), "users" (quoted)
	expected := []tokenKind{tokenWord, tokenString, tokenWord, tokenQuoted}
	if len(kinds) != len(expected) {
		t.Fatalf("Expected %d non-whitespace tokens, got %d", len(expected), len(kinds))
	}
	for i, k := range kinds {
		if k != expected[i] {
			t.Errorf("Token %d: got kind %d, want %d", i, k, expected[i])
		}
	}
}
