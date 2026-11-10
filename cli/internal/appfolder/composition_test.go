package appfolder

import (
	"strings"
	"testing"
)

func TestReadSharedLayoutAndHandlerModules(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "app.json", `{"name":"Warehouse","defaultPage":"main"}`)
	writeTestFile(t, root, "shared/trees/shell.json", `{"id":"shell","type":"Row","children":[{"id":"nav","type":"Button","on":{"click":"shared:navigation.open"}},{"id":"content","type":"Slot"}]}`)
	writeTestFile(t, root, "shared/scripts/navigation.js", `({open: (event,ctx)=>ctx.navigate('main')})`)
	writeTestFile(t, root, "pages/main/scripts/actions.js", `({save: (event,ctx)=>ctx.set('saved',true)})`)
	original := `{"version":1,"layout":"shell","root":{"id":"save","type":"Button","on":{"click":"actions.save"}}}`
	writeTestFile(t, root, "pages/main/tree.json", original)
	workspace, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range workspace.Files {
		if file.Path == "pages/main/tree.json" && file.Content != original {
			t.Fatal("validation expanded the stored page")
		}
	}
	writeTestFile(t, root, "shared/trees/shell.json", `{"id":"save","type":"Row","children":[{"id":"content","type":"Slot"}]}`)
	if _, err := Read(root); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expanded collision: %v", err)
	}
	writeTestFile(t, root, "shared/trees/shell.json", `{"id":"loop","type":"Include","props":{"ref":"shell"}}`)
	if _, err := Read(root); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("circular layout: %v", err)
	}
}

func TestReadSharedReferenceRequiresAnExistingScript(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "app.json", `{"name":"Warehouse","defaultPage":"main"}`)
	writeTestFile(t, root, "pages/main/tree.json", `{"version":1,"root":{"id":"button","type":"Button","on":{"click":"shared:missing.save"}}}`)
	if _, err := Read(root); err == nil {
		t.Fatal("missing shared script was accepted")
	}
}
