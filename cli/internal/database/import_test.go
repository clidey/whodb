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

package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadCSV_WithHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	os.WriteFile(path, []byte("name,age,active\nalice,30,true\nbob,25,false\n"), 0600)

	headers, rows, err := ReadCSV(path, ',', true)
	if err != nil {
		t.Fatalf("ReadCSV: %v", err)
	}

	if len(headers) != 3 {
		t.Errorf("headers=%d, want 3", len(headers))
	}
	if headers[0] != "name" || headers[1] != "age" || headers[2] != "active" {
		t.Errorf("headers=%v", headers)
	}
	if len(rows) != 2 {
		t.Errorf("rows=%d, want 2", len(rows))
	}
}

func TestReadCSV_WithoutHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	os.WriteFile(path, []byte("alice,30\nbob,25\n"), 0600)

	headers, rows, err := ReadCSV(path, ',', false)
	if err != nil {
		t.Fatalf("ReadCSV: %v", err)
	}

	if headers[0] != "col1" || headers[1] != "col2" {
		t.Errorf("headers=%v, want [col1 col2]", headers)
	}
	if len(rows) != 2 {
		t.Errorf("rows=%d, want 2", len(rows))
	}
}

func TestDetectDelimiter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    rune
	}{
		{"comma", "a,b,c\n1,2,3", ','},
		{"tab", "a\tb\tc\n1\t2\t3", '\t'},
		{"semicolon", "a;b;c\n1;2;3", ';'},
		{"pipe", "a|b|c\n1|2|3", '|'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "test.csv")
			os.WriteFile(path, []byte(tt.content), 0600)

			got := DetectDelimiter(path)
			if got != tt.want {
				t.Errorf("DetectDelimiter=%c, want %c", got, tt.want)
			}
		})
	}
}

func TestDetectFormat(t *testing.T) {
	if DetectFormat("data.csv") != "csv" {
		t.Error("expected csv")
	}
	if DetectFormat("data.xlsx") != "excel" {
		t.Error("expected excel")
	}
	if DetectFormat("data.tsv") != "csv" {
		t.Error("expected csv for .tsv")
	}
}

func TestInferColumnTypes(t *testing.T) {
	headers := []string{"id", "name", "score", "active"}
	rows := [][]string{
		{"1", "alice", "3.14", "true"},
		{"2", "bob", "2.71", "false"},
		{"3", "charlie", "1.0", "true"},
	}

	types := InferColumnTypes(headers, rows)

	if types[0] != "INTEGER" {
		t.Errorf("id type=%s, want INTEGER", types[0])
	}
	if types[1] != "TEXT" {
		t.Errorf("name type=%s, want TEXT", types[1])
	}
	if types[2] != "REAL" {
		t.Errorf("score type=%s, want REAL", types[2])
	}
	if types[3] != "BOOLEAN" {
		t.Errorf("active type=%s, want BOOLEAN", types[3])
	}
}

func TestInferColumnTypes_Empty(t *testing.T) {
	types := InferColumnTypes([]string{"a"}, [][]string{{""}})
	if types[0] != "TEXT" {
		t.Errorf("empty column type=%s, want TEXT", types[0])
	}
}

func TestPreviewImport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.csv")

	var content string
	content = "id,name\n"
	for i := 0; i < 100; i++ {
		content += "1,alice\n"
	}
	os.WriteFile(path, []byte(content), 0600)

	headers, rows, err := PreviewImport(path, ImportOptions{HasHeader: true}, 5)
	if err != nil {
		t.Fatalf("PreviewImport: %v", err)
	}

	if len(headers) != 2 {
		t.Errorf("headers=%d, want 2", len(headers))
	}
	if len(rows) != 5 {
		t.Errorf("rows=%d, want 5 (preview limit)", len(rows))
	}
}
