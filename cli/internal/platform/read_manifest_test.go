package platform

import (
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

func TestReadDocumentRespectsNestedHostFields(t *testing.T) {
	client := &Client{manifest: &PlatformManifest{
		Operations: []PlatformManifestOperation{{Name: "AppView", Kind: "Query", Returns: "AppViewResult"}},
		Types: []PlatformManifestType{
			{Name: "AppViewResult", Fields: []PlatformManifestField{{Name: "app", Type: "App"}, {Name: "pages", Type: "String", List: true}}},
			{Name: "App", Fields: []PlatformManifestField{{Name: "id", Type: "ID"}, {Name: "name", Type: "String"}}},
		},
	}}
	document := client.platformReadDocument("AppView", platformQuerySpecs["AppView"])
	for _, unsupported := range []string{"offlineCapable", "recoveryBindings", "sourceHidden", "version", "files"} {
		if strings.Contains(document, unsupported) {
			t.Errorf("queried unsupported %s: %s", unsupported, document)
		}
	}
	if !strings.Contains(document, "name") {
		t.Fatal("supported fields lost")
	}
	if _, err := parser.ParseQuery(&ast.Source{Input: document}); err != nil {
		t.Fatal(err)
	}
	// A refreshed manifest must affect the next projection, rather than retaining
	// the first host's selection in a cached or modified shared document.
	client.manifest.Types[1].Fields = append(client.manifest.Types[1].Fields, PlatformManifestField{Name: "offlineCapable", Type: "Boolean"})
	if !strings.Contains(client.platformReadDocument("AppView", platformQuerySpecs["AppView"]), "offlineCapable") {
		t.Fatal("refreshed fields not selected")
	}
}
