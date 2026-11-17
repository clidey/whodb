package appfolder

import "testing"

func TestRuntimeAssetsValidateCompleteExplicitDependencies(t *testing.T) {
	files := map[string]string{"shared/state.json": "{}", "shared/widgets/Grid.js": "()=>null"}
	workspace := &Workspace{Pages: []string{"main", "stock"}, Manifest: Manifest{Runtime: &RuntimeMetadata{SharedAssets: map[string][]string{"*": {"shared/state.json"}, "main": {}, "stock": {"shared/widgets/Grid.js"}}}}}
	if err := checkRuntimeAssets(workspace, files); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		assets map[string][]string
	}{
		{"missing page", map[string][]string{"main": {}}},
		{"unknown page", map[string][]string{"main": {}, "stock": {}, "other": {}}},
		{"missing asset", map[string][]string{"main": {}, "stock": {"shared/widgets/Missing.js"}}},
		{"traversal", map[string][]string{"main": {}, "stock": {"shared/../app.json"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspace.Manifest.Runtime.SharedAssets = test.assets
			if err := checkRuntimeAssets(workspace, files); err == nil {
				t.Fatal("invalid dependencies accepted")
			}
		})
	}
	workspace.Manifest.Runtime = nil
	if err := checkRuntimeAssets(workspace, files); err != nil {
		t.Fatalf("legacy app rejected: %v", err)
	}
}
