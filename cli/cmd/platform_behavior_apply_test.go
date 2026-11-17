package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/clidey/whodb/cli/internal/platform"
)

type behaviorManifestFake struct {
	queries  []string
	writes   int
	revision int
}

func (f *behaviorManifestFake) PlatformQuery(_ context.Context, operation string, _ map[string]any) (any, error) {
	f.queries = append(f.queries, operation)
	if operation == "OntologyBehavior" {
		return map[string]any{"revision": f.revision}, nil
	}
	return map[string]any{"preview": true}, nil
}
func (f *behaviorManifestFake) PlatformMutation(_ context.Context, operation string, _ map[string]any) (*platform.PlatformMutationResult, error) {
	f.writes++
	if operation != "ApplyOntologyBehaviors" {
		return nil, fmt.Errorf("unexpected mutation %s", operation)
	}
	return &platform.PlatformMutationResult{Data: json.RawMessage(`{"preview":false}`)}, nil
}

func TestBehaviorManifestStrictSourceAndPrecision(t *testing.T) {
	valid := `{"version":1,"behaviors":[{"api_name":"stock","document":{"actions":{"reserve":{}},"defaults":{"amount":{"kind":"literal","value":9007199254740993}}}}]}`
	manifest, err := decodeBehaviorManifest([]byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := behaviorManifestMap(manifest)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(encoded)
	if !strings.Contains(string(raw), "9007199254740993") {
		t.Fatal("integer precision lost")
	}
	for _, source := range []string{`{"version":2,"behaviors":[]}`, `{"version":1,"behaviors":[]}`, valid + ` {}`, strings.Replace(valid, `"version":1`, `"unknown":true,"version":1`, 1), `{"behaviors":[{"api_name":"stock","document":null}]}`, `{"behaviors":[{"api_name":"stock","document":{"actions":{}}},{"api_name":"stock","document":{"actions":{}}}]}`} {
		if _, err := decodeBehaviorManifest([]byte(source)); err == nil {
			t.Errorf("invalid source accepted: %s", source)
		}
	}
}

func TestBehaviorManifestPreviewNeverWritesAndApplyIsOneMutation(t *testing.T) {
	fake := &behaviorManifestFake{revision: 3}
	manifest := behaviorApplyManifest{Version: 1, Behaviors: []behaviorApplyEntry{{APIName: "stock", Document: map[string]any{"actions": map[string]any{}}}}}
	entities := []platform.Ontology{{ID: "target", APIName: "stock"}}
	if err := prepareBehaviorManifest(context.Background(), fake, "project", &manifest, entities); err != nil {
		t.Fatal(err)
	}
	if *manifest.Behaviors[0].ExpectedRevision != 3 {
		t.Fatal("revision was not pinned")
	}
	if _, err := executeBehaviorManifest(context.Background(), fake, "project", manifest, false); err != nil {
		t.Fatal(err)
	}
	if fake.writes != 0 || fake.queries[len(fake.queries)-1] != "ValidateOntologyBehaviors" {
		t.Fatal("preview wrote")
	}
	if _, err := executeBehaviorManifest(context.Background(), fake, "project", manifest, true); err != nil {
		t.Fatal(err)
	}
	if fake.writes != 1 {
		t.Fatal("manifest was not applied as one mutation")
	}
	fake.revision = 4
	if err := prepareBehaviorManifest(context.Background(), fake, "project", &manifest, entities); err == nil {
		t.Fatal("stale source accepted")
	}
	if err := prepareBehaviorManifest(context.Background(), fake, "project", &manifest, nil); err == nil {
		t.Fatal("missing ontology skipped")
	}
	if fake.writes != 1 {
		t.Fatal("failed preparation wrote")
	}
}
