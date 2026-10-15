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
	"testing"

	"github.com/vektah/gqlparser/v2"
)

// Mutation documents are validated against the EE schema the same way query
// documents are, so a renamed argument or result field fails here instead of
// at runtime in an agent session.
func TestPlatformMutationSpecsMatchEESchema(t *testing.T) {
	schema := loadPlatformSchema(t)
	for name, spec := range platformMutationSpecs {
		t.Run(name, func(t *testing.T) {
			if spec.Operation != name {
				t.Fatalf("spec operation %q does not match key %q", spec.Operation, name)
			}
			if _, err := gqlparser.LoadQuery(schema, spec.Query); err != nil {
				t.Fatalf("mutation no longer matches platform schema: %v", err)
			}
		})
	}
}
