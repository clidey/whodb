/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/platform"
)

// appCloneTestSession serves the queries apps clone and undo-clone make and
// records each mutation's variables by operation.
func appCloneTestSession(t *testing.T, cloneResponse string) (*platformSession, map[string]map[string]any) {
	t.Helper()
	mutations := map[string]map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(request.Query, "CloneApp"):
			mutations["CloneApp"] = request.Variables
			_, _ = w.Write([]byte(cloneResponse))
		case strings.Contains(request.Query, "RestoreAppVersionToDraft"):
			mutations["RestoreAppVersionToDraft"] = request.Variables
			_, _ = w.Write([]byte(`{"data":{"RestoreAppVersionToDraft":{"app":{"id":"app-2"},"files":[]}}}`))
		case strings.Contains(request.Query, "DeleteApp"):
			mutations["DeleteApp"] = request.Variables
			_, _ = w.Write([]byte(`{"data":{"DeleteApp":{"Status":true}}}`))
		case strings.Contains(request.Query, "ProjectApps"):
			apps := `[{"id":"app-1","name":"Ledger"}]`
			if request.Variables["projectId"] == "proj-2" {
				apps = `[{"id":"app-2","name":"Acme Ledger"}]`
			}
			_, _ = w.Write([]byte(`{"data":{"ProjectApps":` + apps + `}}`))
		case strings.Contains(request.Query, "AppView"):
			_, _ = w.Write([]byte(`{"data":{"AppView":{"pages":["main"],"files":[{"path":"app.json","content":"{\"theme\":\"light\",\"intent\":{}}"}]}}}`))
		case strings.Contains(request.Query, "MyOrganizations"):
			_, _ = w.Write([]byte(`{"data":{"MyOrganizations":[{"id":"org-1","name":"Clidey","slug":"clidey"}]}}`))
		case strings.Contains(request.Query, "Projects"):
			_, _ = w.Write([]byte(`{"data":{"Projects":[{"id":"proj-1","orgId":"org-1","name":"Source","slug":"source"},{"id":"proj-2","orgId":"org-1","name":"Target","slug":"target"}]}}`))
		default:
			t.Errorf("unexpected query: %s", request.Query)
		}
	}))
	t.Cleanup(server.Close)
	client, err := platform.NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	operations := []platform.PlatformManifestOperation{}
	for _, name := range []string{"MyOrganizations", "Projects", "ProjectApps", "AppView"} {
		operations = append(operations, platform.PlatformManifestOperation{Name: name, Kind: "Query"})
	}
	for _, name := range []string{"CloneApp", "DeleteApp", "RestoreAppVersionToDraft"} {
		operations = append(operations, platform.PlatformManifestOperation{Name: name, Kind: "Mutation"})
	}
	client.SetPlatformManifest(&platform.PlatformManifest{Operations: operations})
	return &platformSession{
		Host:   config.PlatformHost{URL: server.URL, DefaultOrgID: "org-1", DefaultOrgName: "Clidey", DefaultProjectID: "proj-1", DefaultProjectName: "Source"},
		Client: client,
	}, mutations
}

func TestClonePlatformAppWithoutYesIsAServerDryRun(t *testing.T) {
	session, mutations := appCloneTestSession(t, `{"data":{"CloneApp":{"app":null,"action":"update","name":"Acme Ledger","targetAppId":"app-2","fileCount":3000,"ontologyIds":[],"readOnlyOntologyIds":[],"functionIds":[],"warnings":["target project has no ontology named \"missing\"; it is not bound"],"backupVersion":null,"dryRun":true}}}`)

	result, err := clonePlatformApp(context.Background(), session, "Ledger", appCloneOptions{TargetProject: "Target", NamePrefix: "Acme ", Overwrite: true})
	if err != nil {
		t.Fatalf("clonePlatformApp() error = %v", err)
	}
	variables := mutations["CloneApp"]
	if variables["dryRun"] != true || variables["projectId"] != "proj-1" || variables["appId"] != "app-1" || variables["targetProjectId"] != "proj-2" || variables["name"] != "Acme Ledger" || variables["overwrite"] != true {
		t.Fatalf("CloneApp variables = %#v", variables)
	}
	if _, ok := variables["targetOrgId"]; ok {
		t.Fatalf("same-org clone sent targetOrgId: %#v", variables)
	}
	if !result.DryRun || result.Action != "update" || result.Files != 3000 || result.Undo != "" || len(result.Warnings) != 1 {
		t.Fatalf("dry run result = %#v", result)
	}
}

func TestClonePlatformAppSendsBrandFilesInTheCloneRequest(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{"app.json": `{"theme":"dark"}`, "brand.json": `{}`, ".DS_Store": "junk"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	session, mutations := appCloneTestSession(t, `{"data":{"CloneApp":{"app":{"id":"app-2"},"action":"update","name":"Acme Ledger","targetAppId":"app-2","fileCount":3001,"ontologyIds":[],"readOnlyOntologyIds":[],"functionIds":[],"warnings":[],"backupVersion":4,"dryRun":false}}}`)

	result, err := clonePlatformApp(context.Background(), session, "app-1", appCloneOptions{TargetProject: "proj-2", NamePrefix: "Acme ", Overwrite: true, BrandDir: dir, Execute: true})
	if err != nil {
		t.Fatalf("clonePlatformApp() error = %v", err)
	}
	variables := mutations["CloneApp"]
	files, _ := variables["files"].([]any)
	if variables["dryRun"] != false || len(files) != 2 {
		t.Fatalf("CloneApp variables = %#v, want a real run with two brand files", variables)
	}
	appJSON, _ := files[0].(map[string]any)
	var merged map[string]any
	if err := json.Unmarshal([]byte(appJSON["content"].(string)), &merged); err != nil || appJSON["path"] != "app.json" || merged["theme"] != "dark" || merged["intent"] == nil {
		t.Fatalf("brand app.json = %#v (%v), want dark theme merged over the source", appJSON, err)
	}
	if result.Action != "updated" || result.BackupVersion == nil || *result.BackupVersion != 4 || result.BrandFiles != 2 {
		t.Fatalf("result = %#v", result)
	}
	if result.Undo != "whodb apps undo-clone app-2 --org org-1 --project proj-2 --backup-version 4" {
		t.Fatalf("undo = %q", result.Undo)
	}
}

func TestUndoPlatformAppClone(t *testing.T) {
	session, mutations := appCloneTestSession(t, "")

	plan, err := undoPlatformAppClone(context.Background(), session, "Acme Ledger", "", "proj-2", 4, false)
	if err != nil || plan.Action != "restore" || !plan.DryRun || len(mutations) != 0 {
		t.Fatalf("plan = %#v, err = %v, mutations = %#v; want a restore plan without writes", plan, err, mutations)
	}
	restored, err := undoPlatformAppClone(context.Background(), session, "Acme Ledger", "", "proj-2", 4, true)
	if err != nil || restored.Action != "restored" {
		t.Fatalf("restore = %#v, err = %v", restored, err)
	}
	if variables := mutations["RestoreAppVersionToDraft"]; variables["appId"] != "app-2" || variables["projectId"] != "proj-2" || variables["version"] != float64(4) {
		t.Fatalf("restore variables = %#v", variables)
	}
	deleted, err := undoPlatformAppClone(context.Background(), session, "app-2", "", "proj-2", 0, true)
	if err != nil || deleted.Action != "deleted" {
		t.Fatalf("delete = %#v, err = %v", deleted, err)
	}
	if variables := mutations["DeleteApp"]; variables["id"] != "app-2" || variables["projectId"] != "proj-2" {
		t.Fatalf("delete variables = %#v", variables)
	}
}

func TestReadAppBrandFilesMergesAppJSONOverSource(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.json"), []byte(`{"theme":"system"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "brand.json"), []byte(`{"name":"Christy"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := readAppBrandFiles(dir, func() (string, error) {
		return `{"theme":"light","intent":{"pages":["main"]}}`, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].Path != "app.json" || files[1].Path != "brand.json" {
		t.Fatalf("files = %#v, want only the two brand files", files)
	}
	var app map[string]any
	if err := json.Unmarshal([]byte(files[0].Content), &app); err != nil {
		t.Fatal(err)
	}
	if app["theme"] != "system" || app["intent"] == nil {
		t.Fatalf("merged app = %#v, want system theme and preserved intent", app)
	}
}

func TestReadAppBrandFilesSkipsHiddenFilesAndSourceWithoutAppJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "brand.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".DS_Store"), []byte("junk"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "config"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := readAppBrandFiles(dir, func() (string, error) {
		t.Fatal("source app.json loaded without a brand app.json")
		return "", nil
	})
	if err != nil || len(files) != 1 || files[0].Path != "brand.json" {
		t.Fatalf("files = %#v, err = %v, want only brand.json", files, err)
	}
	if files, err := readAppBrandFiles("", nil); err != nil || files != nil {
		t.Fatalf("no brand dir = %#v, %v, want nothing", files, err)
	}
}

func TestReadAppBrandFilesRejectsSymlinks(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "linked.json")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := readAppBrandFiles(dir, nil); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("error = %v, want a symlink refusal", err)
	}
}
