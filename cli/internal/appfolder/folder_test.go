package appfolder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, root, path, content string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReadValidatesEventScriptsAndIgnoresProjectFiles(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "app.json", `{"name":"Orders","defaultPage":"main","ontologies":["Order"],"functions":["submit_order"]}`)
	writeTestFile(t, root, "pages/main/tree.json", `{"version":1,"root":{"id":"root","type":"Col","children":[{"id":"save","type":"Button","on":{"click":"save_order"}}]}}`)
	writeTestFile(t, root, "pages/main/state.json", `{}`)
	writeTestFile(t, root, "pages/main/scripts/save_order.js", `(event, ctx) => ctx.whodb.call('submit_order', {})`)
	writeTestFile(t, root, "README.md", `Do not upload me`)
	workspace, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(workspace.Pages) != 1 || workspace.Pages[0] != "main" || len(workspace.Files) != 4 {
		t.Fatalf("workspace = %#v", workspace)
	}
	if workspace.Manifest.Functions[0] != "submit_order" {
		t.Fatalf("manifest = %#v", workspace.Manifest)
	}
}

func TestReadReportsMissingHandlerAtNode(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "app.json", `{"name":"Orders","defaultPage":"main"}`)
	writeTestFile(t, root, "pages/main/tree.json", `{"version":1,"root":{"id":"save","type":"Button","on":{"click":"missing"}}}`)
	_, err := Read(root)
	if err == nil || !strings.Contains(err.Error(), `node "save" event "click"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestReadReportsDuplicateNodeID(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "app.json", `{"name":"Orders","defaultPage":"main"}`)
	writeTestFile(t, root, "pages/main/tree.json", `{"version":1,"root":{"id":"same","type":"Col","children":[{"id":"same","type":"Button"}]}}`)
	_, err := Read(root)
	if err == nil || !strings.Contains(err.Error(), `duplicate node id "same"`) {
		t.Fatalf("error = %v", err)
	}
}
