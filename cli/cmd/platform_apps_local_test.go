package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/clidey/whodb/cli/internal/appfolder"
	"github.com/clidey/whodb/cli/internal/platform"
	"github.com/spf13/cobra"
)

func TestPublishAppFolderSyncsBindingsFilesAndPromotesLast(t *testing.T) {
	operations := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var call struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(request.Body).Decode(&call); err != nil {
			t.Error(err)
			return
		}
		respond := func(name, value string) {
			operations = append(operations, name)
			_, _ = w.Write([]byte(`{"data":{"` + name + `":` + value + `}}`))
		}
		switch {
		case strings.Contains(call.Query, "ProjectApps"):
			respond("ProjectApps", `[{"id":"app-1","name":"Orders","ontologyIds":[],"readOnlyOntologyIds":[],"functionIds":[]}]`)
		case strings.Contains(call.Query, "OntologyEntities"):
			respond("OntologyEntities", `[{"id":"ont-1","apiName":"Order"}]`)
		case strings.Contains(call.Query, "ProjectFunctions"):
			respond("ProjectFunctions", `[{"id":"fn-1","name":"submit_order"}]`)
		case strings.Contains(call.Query, "AppFiles"):
			respond("AppFiles", `[{"path":"app.json","content":"old"},{"path":"pages/old/tree.json","content":"old"}]`)
		case strings.Contains(call.Query, "UpdateApp"):
			input := call.Variables["input"].(map[string]any)
			if !reflect.DeepEqual(input["ontologyIds"], []any{"ont-1"}) || !reflect.DeepEqual(input["functionIds"], []any{"fn-1"}) {
				t.Errorf("binding IDs = %#v", input)
			}
			respond("UpdateApp", `{"id":"app-1"}`)
		case strings.Contains(call.Query, "UpsertAppFile"):
			respond("UpsertAppFile", `{"path":"app.json","content":"new"}`)
		case strings.Contains(call.Query, "DeleteAppFile"):
			respond("DeleteAppFile", `{"Status":true}`)
		case strings.Contains(call.Query, "PromoteObject"):
			respond("PromoteObject", `{"id":"v-2","objectId":"app-1","objectType":"app","version":2}`)
		default:
			t.Errorf("unexpected operation %s", call.Query)
		}
	}))
	t.Cleanup(server.Close)
	client, err := platform.NewClient(server.URL, "token")
	if err != nil {
		t.Fatal(err)
	}
	manifest := &platform.PlatformManifest{}
	for _, name := range []string{"ProjectApps", "OntologyEntities", "ProjectFunctions", "AppFiles"} {
		manifest.Operations = append(manifest.Operations, platform.PlatformManifestOperation{Name: name, Kind: "Query"})
	}
	for _, name := range []string{"UpdateApp", "UpsertAppFile", "DeleteAppFile", "PromoteObject"} {
		manifest.Operations = append(manifest.Operations, platform.PlatformManifestOperation{Name: name, Kind: "Mutation"})
	}
	client.SetPlatformManifest(manifest)
	client.SetWorkspaceContext("org-1", "proj-1")
	workspace := &appfolder.Workspace{
		Manifest: appfolder.Manifest{Name: "Orders", Ontologies: []string{"Order"}, Functions: []string{"submit_order"}},
		Files:    []appfolder.File{{Path: "app.json", Content: "new"}},
	}
	plan, err := publishAppFolder(context.Background(), &platformSession{Client: client}, "proj-1", workspace, "", false, false, "ship")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Version != 2 || !reflect.DeepEqual(plan.UpsertPaths, []string{"app.json"}) || !reflect.DeepEqual(plan.DeletePaths, []string{"pages/old/tree.json"}) {
		t.Fatalf("plan = %#v", plan)
	}
	want := []string{"ProjectApps", "OntologyEntities", "ProjectFunctions", "AppFiles", "UpsertAppFile", "DeleteAppFile", "UpdateApp", "PromoteObject"}
	if !reflect.DeepEqual(operations, want) {
		t.Fatalf("operations = %v, want %v", operations, want)
	}
}

func TestInitAppFromHostedPreservesFilesAndBindings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var call struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(request.Body).Decode(&call); err != nil {
			t.Error(err)
			return
		}
		switch {
		case strings.Contains(call.Query, "ProjectApps"):
			_, _ = w.Write([]byte(`{"data":{"ProjectApps":[{"id":"app-1","name":"Orders","description":"Order work","ontologyIds":["ont-1"],"readOnlyOntologyIds":["ont-2"],"functionIds":["fn-1"]}]}}`))
		case strings.Contains(call.Query, "AppFiles"):
			files := []platformAppFile{
				{Path: "app.json", Content: `{"defaultPage":"detail","theme":"dark"}`},
				{Path: "pages/main/tree.json", Content: `{"version":1,"root":{"id":"root","type":"Col"}}`},
				{Path: "pages/detail/tree.json", Content: `{"version":1,"root":{"id":"detail","type":"Col"}}`},
				{Path: "pages/detail/state.json", Content: `{}`},
				{Path: "shared/state.json", Content: `{}`},
			}
			encoded, _ := json.Marshal(map[string]any{"data": map[string]any{"AppFiles": files}})
			_, _ = w.Write(encoded)
		case strings.Contains(call.Query, "OntologyEntities"):
			_, _ = w.Write([]byte(`{"data":{"OntologyEntities":[{"id":"ont-1","apiName":"Order"},{"id":"ont-2","apiName":"Customer"}]}}`))
		case strings.Contains(call.Query, "ProjectFunctions"):
			_, _ = w.Write([]byte(`{"data":{"ProjectFunctions":[{"id":"fn-1","name":"submit_order"}]}}`))
		default:
			t.Errorf("unexpected operation %s", call.Query)
		}
	}))
	defer server.Close()
	client, err := platform.NewClient(server.URL, "token")
	if err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(t.TempDir(), "orders")
	app, count, err := initAppFromHosted(context.Background(), &platformSession{Client: client}, "proj-1", "app-1", folder)
	if err != nil {
		t.Fatal(err)
	}
	if app.ID != "app-1" || count != 5 {
		t.Fatalf("app = %#v, file count = %d", app, count)
	}
	workspace, err := appfolder.Read(folder)
	if err != nil {
		t.Fatal(err)
	}
	if workspace.Manifest.Name != "Orders" || workspace.Manifest.DefaultPage != "detail" || !reflect.DeepEqual(workspace.Manifest.Ontologies, []string{"Order"}) || !reflect.DeepEqual(workspace.Manifest.ReadOnlyOntologies, []string{"Customer"}) || !reflect.DeepEqual(workspace.Manifest.Functions, []string{"submit_order"}) {
		t.Fatalf("manifest = %#v", workspace.Manifest)
	}
	var appJSON map[string]any
	content, err := os.ReadFile(filepath.Join(folder, "app.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, &appJSON); err != nil || appJSON["theme"] != "dark" {
		t.Fatalf("app.json = %s, err = %v", content, err)
	}
	if _, _, err := initAppFromHosted(context.Background(), &platformSession{Client: client}, "proj-1", "app-1", folder); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("second import error = %v", err)
	}
}

func TestWriteImportedAppFolderRejectsInvalidPathsWithoutCreatingFolder(t *testing.T) {
	for _, path := range []string{"../outside", "pages/../app.json", "/app.json", "pages\\main\\tree.json", "pages/main/.hidden", "assets/logo.svg"} {
		if localAppFilePath(path) {
			t.Errorf("accepted path %q", path)
		}
	}
	folder := filepath.Join(t.TempDir(), "broken")
	if err := writeImportedAppFolder(folder, map[string]string{"app.json": `{"name":"Broken","defaultPage":"main"}`}); err == nil {
		t.Fatal("expected validation error")
	}
	if _, err := os.Stat(folder); !os.IsNotExist(err) {
		t.Fatalf("folder should not exist after validation failure: %v", err)
	}
}

// TestLocalAppFunctionEnvironment ensures managed previews can select active
// functions without changing the default environment of hosted screenshots.
func TestLocalAppFunctionEnvironment(t *testing.T) {
	for _, command := range []*cobra.Command{appsDevCmd, appsVerifyCmd} {
		flag := command.Flags().Lookup("env")
		if flag == nil || flag.DefValue != "dev" {
			t.Fatalf("%s env default = %v, want dev", command.Name(), flag)
		}
		prior := appLocalEnv
		if err := flag.Value.Set("published"); err != nil {
			t.Fatal(err)
		}
		if appLocalEnv != "published" {
			t.Fatalf("local environment = %q, want published", appLocalEnv)
		}
		if appCaptureEnv != "published" {
			t.Fatalf("hosted screenshot environment changed to %q", appCaptureEnv)
		}
		_ = flag.Value.Set(prior)
	}
}
