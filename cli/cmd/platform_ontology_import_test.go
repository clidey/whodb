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

package cmd

import (
	"reflect"
	"testing"
)

func TestOntologyImportTransitionPathSupportsCompiledLiterals(t *testing.T) {
	behavior := ontologyImportBehaviorDocument{
		StateProperty: "status",
		InitialState:  "Draft",
		Actions: map[string]ontologyImportActionDefinition{
			"approve": {From: []string{"Draft"}, Set: map[string]any{"status": map[string]any{"kind": "literal", "value": "Approved"}}},
			"release": {From: []string{"Approved"}, Set: map[string]any{"status": map[string]any{"kind": "literal", "value": "Active"}}},
		},
	}
	path, err := ontologyImportTransitionPath(behavior, "Active")
	if err != nil {
		t.Fatalf("ontologyImportTransitionPath() error = %v", err)
	}
	if !reflect.DeepEqual(path, []string{"approve", "release"}) {
		t.Fatalf("path = %#v", path)
	}
}

func TestSelectOntologyActionValuesUsesContractAndExclusions(t *testing.T) {
	row := map[string]any{"id": "1", "name": "Ada", "status": "Draft", "count": int64(3)}
	inputs := map[string]any{"name": map[string]any{}, "count": map[string]any{}, "status": map[string]any{}}
	values := selectOntologyActionValues(row, inputs, []string{"status"})
	want := map[string]any{"name": "Ada", "count": int64(3)}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("values = %#v, want %#v", values, want)
	}
}

func TestOntologyImportLiteralString(t *testing.T) {
	if got := ontologyImportLiteralString("Approved"); got != "Approved" {
		t.Fatalf("plain literal = %q", got)
	}
	if got := ontologyImportLiteralString(map[string]any{"kind": "literal", "value": "Active"}); got != "Active" {
		t.Fatalf("compiled literal = %q", got)
	}
	if got := ontologyImportLiteralString(map[string]any{"kind": "path", "value": "input.status"}); got != "" {
		t.Fatalf("dynamic expression = %q", got)
	}
}
