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

package sqlexport

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/clidey/whodb/core/src/engine"
)

func testDialect() Dialect {
	return GenericDialect{QuoteIdentifierFunc: func(identifier string) string {
		return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
	}}
}

func TestWriteInsertFormatsLiteralsAndIdentifiers(t *testing.T) {
	columns := []engine.Column{
		{Name: "id", Type: "integer", IsPrimary: true},
		{Name: "name", Type: "text"},
		{Name: "price", Type: "numeric"},
		{Name: "active", Type: "boolean"},
		{Name: "created_at", Type: "timestamp"},
		{Name: "payload", Type: "jsonb"},
		{Name: "data", Type: "bytea"},
	}
	rows := []Row{{
		"id":         []byte("7"),
		"name":       "O'Malley",
		"price":      []byte("12.50"),
		"active":     []byte("true"),
		"created_at": time.Date(2026, 6, 26, 12, 30, 0, 0, time.UTC),
		"payload":    []byte(`{"ok":true}`),
		"data":       []byte{0xde, 0xad},
	}}

	var out bytes.Buffer
	if err := WriteInsert(&out, testDialect(), Table{Schema: "public", Name: "order"}, columns, rows); err != nil {
		t.Fatalf("WriteInsert returned error: %v", err)
	}

	want := "INSERT INTO \"public\".\"order\" (\"id\", \"name\", \"price\", \"active\", \"created_at\", \"payload\", \"data\") VALUES\n" +
		"  (7, 'O''Malley', 12.50, TRUE, '2026-06-26 12:30:00', '{\"ok\":true}', '\\xDEAD');\n"
	if out.String() != want {
		t.Fatalf("unexpected INSERT output:\n%s", out.String())
	}
}

func TestWriteUpdateUsesCompositePrimaryKeyOnlyInWhere(t *testing.T) {
	setColumns := []engine.Column{
		{Name: "name", Type: "text"},
	}
	primaryKeyColumns := []engine.Column{
		{Name: "tenant_id", Type: "integer", IsPrimary: true},
		{Name: "id", Type: "integer", IsPrimary: true},
	}
	row := Row{
		"tenant_id": 42,
		"id":        7,
		"name":      "Alice",
	}

	var out bytes.Buffer
	if err := WriteUpdate(&out, testDialect(), Table{Name: "users"}, setColumns, primaryKeyColumns, row); err != nil {
		t.Fatalf("WriteUpdate returned error: %v", err)
	}

	want := "UPDATE \"users\" SET \"name\" = 'Alice' WHERE \"tenant_id\" = 42 AND \"id\" = 7;\n"
	if out.String() != want {
		t.Fatalf("unexpected UPDATE output:\n%s", out.String())
	}
}

func TestWriteUpdateRejectsMissingPrimaryKey(t *testing.T) {
	err := WriteUpdate(&bytes.Buffer{}, testDialect(), Table{Name: "users"}, []engine.Column{{Name: "name", Type: "text"}}, nil, Row{})
	if err == nil {
		t.Fatal("expected missing primary key error")
	}
}
