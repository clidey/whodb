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

package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestPlatformOperationsMatchEESchema(t *testing.T) {
	schema := loadPlatformSchema(t)
	for name, operation := range platformOperations {
		t.Run(name, func(t *testing.T) {
			if _, err := gqlparser.LoadQuery(schema, operation); err != nil {
				t.Fatalf("operation no longer matches platform schema: %v", err)
			}
		})
	}
}

func loadPlatformSchema(t *testing.T) *ast.Schema {
	t.Helper()
	sources := []*ast.Source{
		readSchemaSource(t, "core/graph/schema.graphqls"),
		readSchemaSource(t, "ee/core/graph/schema.extension.graphqls"),
	}
	schema, err := gqlparser.LoadSchema(sources...)
	if err != nil {
		t.Fatalf("load platform schema: %v", err)
	}
	return schema
}

func readSchemaSource(t *testing.T, relativePath string) *ast.Source {
	t.Helper()
	path := filepath.Join("..", "..", "..", relativePath)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", relativePath, err)
	}
	return &ast.Source{
		Name:  relativePath,
		Input: string(body),
	}
}
