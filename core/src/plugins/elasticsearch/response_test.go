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

package elasticsearch

import "testing"

func TestResponseHits(t *testing.T) {
	valid := map[string]any{
		"hits": map[string]any{
			"hits": []any{map[string]any{"_id": "1"}},
		},
	}
	hits, err := responseHits(valid)
	if err != nil {
		t.Fatalf("expected valid hits, got error: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}

	// Malformed responses that previously caused a panic must return an error.
	malformed := []map[string]any{
		{},                                   // no "hits"
		{"hits": "not-an-object"},            // wrong type for hits
		{"hits": map[string]any{}},           // no inner "hits"
		{"hits": map[string]any{"hits": 42}}, // inner hits wrong type
		{"error": "index_not_found"},         // error-shaped response
	}
	for i, m := range malformed {
		if _, err := responseHits(m); err == nil {
			t.Errorf("case %d: expected error for malformed response, got nil", i)
		}
	}
}

func TestHitSource(t *testing.T) {
	source, ok := hitSource(map[string]any{"_source": map[string]any{"a": 1}})
	if !ok || source["a"] != 1 {
		t.Fatalf("expected valid source, got ok=%v source=%v", ok, source)
	}

	for _, bad := range []any{"string", 42, map[string]any{}, map[string]any{"_source": "x"}, nil} {
		if _, ok := hitSource(bad); ok {
			t.Errorf("expected hitSource(%v) to report not-ok", bad)
		}
	}
}

func TestIndicesStats(t *testing.T) {
	if _, err := indicesStats(map[string]any{"indices": map[string]any{"a": 1}}); err != nil {
		t.Fatalf("expected valid indices, got error: %v", err)
	}
	if _, err := indicesStats(map[string]any{}); err == nil {
		t.Error("expected error when indices missing")
	}
	if _, err := indicesStats(map[string]any{"indices": "x"}); err == nil {
		t.Error("expected error when indices wrong type")
	}
}
