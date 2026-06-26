/*
 * Copyright 2026 Clidey, Inc.
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

package redis

import "testing"

func TestTokenizeRedisCommand(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "simple", input: "GET mykey", want: []string{"GET", "mykey"}},
		{name: "double quoted", input: `SET mykey "hello world"`, want: []string{"SET", "mykey", "hello world"}},
		{name: "single quoted", input: "SET mykey 'hello world'", want: []string{"SET", "mykey", "hello world"}},
		{name: "multiline", input: "SET\n  mykey\n  myvalue", want: []string{"SET", "mykey", "myvalue"}},
		{name: "chained command becomes arguments", input: "GET mykey\nFLUSHDB", want: []string{"GET", "mykey", "FLUSHDB"}},
		{name: "empty", input: "", want: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := tokenizeRedisCommand(test.input)
			if len(got) != len(test.want) {
				t.Fatalf("expected %d tokens, got %d: %#v", len(test.want), len(got), got)
			}
			for i := range test.want {
				if got[i] != test.want[i] {
					t.Fatalf("expected token %d to be %q, got %#v", i, test.want[i], got)
				}
			}
		})
	}
}

func TestFormatRedisRawResult(t *testing.T) {
	stringResult := formatRedisRawResult("GET", "hello")
	if len(stringResult.Columns) != 1 || stringResult.Columns[0].Name != "value" {
		t.Fatalf("expected value column, got %#v", stringResult.Columns)
	}
	if stringResult.Rows[0][0] != "hello" {
		t.Fatalf("expected string value, got %#v", stringResult.Rows)
	}

	intResult := formatRedisRawResult("DBSIZE", int64(42))
	if intResult.Rows[0][0] != "42" {
		t.Fatalf("expected integer value, got %#v", intResult.Rows)
	}

	nilResult := formatRedisRawResult("GET", nil)
	if nilResult.Rows[0][0] != "(nil)" {
		t.Fatalf("expected nil marker, got %#v", nilResult.Rows)
	}
}

func TestFormatRedisRawResultCollections(t *testing.T) {
	hashResult := formatRedisRawResult("HGETALL", []any{"name", "Ada", "age", "37"})
	if len(hashResult.Columns) != 2 || hashResult.Columns[0].Name != "field" || hashResult.Columns[1].Name != "value" {
		t.Fatalf("expected field/value columns, got %#v", hashResult.Columns)
	}
	if len(hashResult.Rows) != 2 || hashResult.Rows[0][0] != "name" || hashResult.Rows[0][1] != "Ada" {
		t.Fatalf("unexpected hash rows: %#v", hashResult.Rows)
	}

	listResult := formatRedisRawResult("LRANGE", []any{"a", "b", "c"})
	if len(listResult.Columns) != 2 || listResult.Columns[0].Name != "index" || listResult.Columns[1].Name != "value" {
		t.Fatalf("expected index/value columns, got %#v", listResult.Columns)
	}
	if len(listResult.Rows) != 3 || listResult.Rows[2][0] != "2" || listResult.Rows[2][1] != "c" {
		t.Fatalf("unexpected list rows: %#v", listResult.Rows)
	}

	emptyResult := formatRedisRawResult("KEYS", []any{})
	if len(emptyResult.Rows) != 0 || emptyResult.TotalCount != 0 {
		t.Fatalf("expected empty row result, got %#v", emptyResult)
	}
}
