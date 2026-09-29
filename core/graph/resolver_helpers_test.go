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

package graph

import (
	"context"
	"maps"
	"testing"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/source"
)

func testSourceContext(sourceType string, values map[string]string) context.Context {
	return context.WithValue(context.Background(), auth.AuthKey_Source, &source.Credentials{
		SourceType: sourceType,
		Values:     cloneStringMap(values),
	})
}

func testSourceRef(kind model.SourceObjectKind, path ...string) model.SourceObjectRefInput {
	return model.SourceObjectRefInput{
		Kind: kind,
		Path: path,
	}
}

func testSourceRefPtr(kind model.SourceObjectKind, path ...string) *model.SourceObjectRefInput {
	ref := testSourceRef(kind, path...)
	return &ref
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return map[string]string{}
	}

	cloned := make(map[string]string, len(values))
	maps.Copy(cloned, values)
	return cloned
}

func TestMapColumnsToModelPreservesMetadata(t *testing.T) {
	refTable := "users"
	refColumn := "id"
	length := 255
	precision := 10
	scale := 2
	columns := []engine.Column{
		{
			Name:             "id",
			Type:             "INTEGER",
			IsPrimary:        true,
			Length:           &length,
			Precision:        &precision,
			Scale:            &scale,
			IsForeignKey:     true,
			ReferencedTable:  &refTable,
			ReferencedColumn: &refColumn,
		},
	}

	mapped := MapColumnsToModel(columns)
	if len(mapped) != 1 {
		t.Fatalf("expected one mapped column, got %#v", mapped)
	}
	if mapped[0].Name != "id" || !mapped[0].IsPrimary || !mapped[0].IsForeignKey {
		t.Fatalf("expected mapped column metadata to be preserved, got %#v", mapped[0])
	}
	if mapped[0].ReferencedTable == nil || *mapped[0].ReferencedTable != "users" {
		t.Fatalf("expected referenced table metadata to be preserved, got %#v", mapped[0])
	}
}
