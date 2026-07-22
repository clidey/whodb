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
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/clidey/whodb/core/graph/model"
)

const goldenSourceTypesFile = "testdata/source_types.json"

// TestSourceTypesGoldenFile ensures the serialized SourceTypes GraphQL response
// matches the checked-in golden file. When a connector change intentionally
// shifts the contract, regenerate the golden file with:
//
//	go test -run TestGenerateSourceTypesGoldenFile ./graph/
func TestSourceTypesGoldenFile(t *testing.T) {
	// Not parallel: SourceTypes reads the shared global source registry, and
	// other tests in this package (e.g. TestQuerySourceContentReadsRegisteredContentSource)
	// register test-only types into it with no cleanup. Running in parallel
	// made this test's outcome depend on unrelated test ordering.

	types, err := (&Resolver{}).Query().SourceTypes(context.Background())
	if err != nil {
		t.Fatalf("expected source types query to succeed, got %v", err)
	}

	got := serializeSourceTypes(t, types)

	want, err := os.ReadFile(filepath.Join("testdata", "source_types.json"))
	if err != nil {
		t.Fatalf("golden file missing — run TestGenerateSourceTypesGoldenFile to create it: %v", err)
	}

	if wantStr := normalizeGolden(string(want)); got != wantStr {
		t.Fatalf(
			"SourceTypes golden file mismatch.\n\nRun to regenerate:\n\tgo test -run TestGenerateSourceTypesGoldenFile ./graph/\n\nDiff (-want +got):\n%s",
			unifiedDiff("source_types.json", wantStr, got),
		)
	}
}

// TestGenerateSourceTypesGoldenFile regenerates the golden file on demand.
func TestGenerateSourceTypesGoldenFile(t *testing.T) {
	types, err := (&Resolver{}).Query().SourceTypes(context.Background())
	if err != nil {
		t.Fatalf("expected source types query to succeed, got %v", err)
	}

	got := serializeSourceTypes(t, types)

	dir := filepath.Join("testdata")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create testdata dir: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(dir, "source_types.json"),
		[]byte(got+"\n"),
		0644,
	); err != nil {
		t.Fatalf("failed to write golden file: %v", err)
	}

	t.Logf("golden file written: %s (%d bytes)", goldenSourceTypesFile, len(got))
}

func serializeSourceTypes(t *testing.T, types []*model.SourceType) string {
	t.Helper()

	sliced := make([]*model.SourceType, len(types))
	copy(sliced, types)
	slices.SortFunc(sliced, func(a, b *model.SourceType) int {
		return strings.Compare(a.ID, b.ID)
	})

	// Stable-sort nested slices so JSON output is deterministic
	for _, st := range sliced {
		sortSourceType(st)
	}

	b, err := json.MarshalIndent(sliced, "", "  ")
	if err != nil {
		t.Fatalf("failed to serialize source types: %v", err)
	}
	return string(b)
}

func sortSourceType(st *model.SourceType) {
	slices.SortFunc(st.ConnectionFields, func(a, b *model.SourceConnectionField) int {
		return strings.Compare(a.Key, b.Key)
	})
	slices.SortFunc(st.SSLModes, func(a, b *model.SourceSSLMode) int {
		return strings.Compare(a.Value, b.Value)
	})

	if st.Contract == nil {
		return
	}

	slices.SortFunc(st.Contract.Surfaces, func(a, b model.SourceSurface) int {
		return strings.Compare(string(a), string(b))
	})
	slices.SortFunc(st.Contract.RootActions, func(a, b model.SourceAction) int {
		return strings.Compare(string(a), string(b))
	})
	slices.SortFunc(st.Contract.BrowsePath, func(a, b model.SourceObjectKind) int {
		return strings.Compare(string(a), string(b))
	})
	slices.SortFunc(st.Contract.ObjectTypes, func(a, b *model.SourceObjectType) int {
		return strings.Compare(string(a.Kind), string(b.Kind))
	})

	for _, ot := range st.Contract.ObjectTypes {
		if ot == nil {
			continue
		}
		slices.SortFunc(ot.Actions, func(a, b model.SourceAction) int {
			return strings.Compare(string(a), string(b))
		})
		slices.SortFunc(ot.Views, func(a, b model.SourceView) int {
			return strings.Compare(string(a), string(b))
		})
	}

	if st.DiscoveryPrefill != nil {
		slices.SortFunc(st.DiscoveryPrefill.AdvancedDefaults, func(a, b *model.SourceDiscoveryAdvancedDefault) int {
			return strings.Compare(a.Key, b.Key)
		})
		for _, ad := range st.DiscoveryPrefill.AdvancedDefaults {
			if ad == nil {
				continue
			}
			slices.SortFunc(ad.ProviderTypes, func(a, b string) int {
				return strings.Compare(a, b)
			})
			slices.SortFunc(ad.Conditions, func(a, b *model.SourceDiscoveryMetadataCondition) int {
				return strings.Compare(a.Key, b.Key)
			})
		}
	}
}

// normalizeGolden trims trailing whitespace per-line so diffs are resistant to
// editor whitespace changes.
func normalizeGolden(s string) string {
	s = strings.TrimRight(s, "\n ")
	b := strings.Builder{}
	for _, line := range strings.SplitAfter(s, "\n") {
		b.WriteString(strings.TrimRight(line, " \t\r"))
	}
	return b.String()
}

// unifiedDiff returns a minimal + prefix diff between old and new.
func unifiedDiff(label, old, new string) string {
	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	var b strings.Builder
	for i := 0; i < len(oldLines) || i < len(newLines); i++ {
		oldLine := ""
		newLine := ""
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if oldLine != newLine {
			if oldLine != "" {
				b.WriteString("- " + oldLine + "\n")
			}
			if newLine != "" {
				b.WriteString("+ " + newLine + "\n")
			}
		}
	}
	return b.String()
}
