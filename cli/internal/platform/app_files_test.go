/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package platform

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

type fakeAppFileQuerier struct {
	results map[string]any
	pages   map[string][]any
	calls   []string
}

func (f *fakeAppFileQuerier) PlatformQuery(_ context.Context, operation string, variables map[string]any) (any, error) {
	if operation != "AppDetail" && variables["env"] != "dev" {
		return nil, nil
	}
	if operation == "AppPageFiles" {
		page, _ := variables["page"].(string)
		f.calls = append(f.calls, operation+":"+page)
		return f.pages[page], nil
	}
	f.calls = append(f.calls, operation)
	return f.results[operation], nil
}

func TestReadAppFilesByPageReadsSharedFilesThenEachPage(t *testing.T) {
	client := &fakeAppFileQuerier{
		results: map[string]any{
			"AppDetail": map[string]any{"id": "app-1", "sourceHidden": false},
			"AppView": map[string]any{
				"pages": []any{"main", "ledger"},
				"files": []any{
					map[string]any{"path": "shared/state.json", "content": "{}"},
					map[string]any{"path": "app.json", "content": `{"defaultPage":"main"}`},
					map[string]any{"path": "pages/main/tree.json", "content": "from-view"},
				},
			},
		},
		pages: map[string][]any{
			"main":   {map[string]any{"path": "pages/main/tree.json", "content": "main-tree"}},
			"ledger": {map[string]any{"path": "pages/ledger/tree.json", "content": "ledger-tree"}, map[string]any{"path": "pages/ledger/state.json", "content": "{}"}},
		},
	}

	files, err := ReadAppFilesByPage(context.Background(), client, "proj-1", "app-1")
	if err != nil {
		t.Fatalf("ReadAppFilesByPage: %v", err)
	}
	paths := make([]string, len(files))
	for i, file := range files {
		paths[i] = file.Path
	}
	want := []string{"app.json", "pages/ledger/state.json", "pages/ledger/tree.json", "pages/main/tree.json", "shared/state.json"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
	if files[3].Content != "main-tree" {
		t.Fatalf("main page content = %q, want the AppPageFiles copy", files[3].Content)
	}
	if got := strings.Join(client.calls, ","); got != "AppDetail,AppView,AppPageFiles:main,AppPageFiles:ledger" {
		t.Fatalf("calls = %s", got)
	}
}

func TestReadAppFilesByPageReturnsNothingForHiddenSource(t *testing.T) {
	client := &fakeAppFileQuerier{results: map[string]any{"AppDetail": map[string]any{"sourceHidden": true}}}

	files, err := ReadAppFilesByPage(context.Background(), client, "proj-1", "app-1")
	if err != nil {
		t.Fatalf("ReadAppFilesByPage: %v", err)
	}
	if files == nil || len(files) != 0 {
		t.Fatalf("files = %#v, want an empty list", files)
	}
	if !reflect.DeepEqual(client.calls, []string{"AppDetail"}) {
		t.Fatalf("calls = %v, want only AppDetail", client.calls)
	}
}

func TestCloneAppMutationSpec(t *testing.T) {
	spec, ok := platformMutationSpecs["CloneApp"]
	if !ok {
		t.Fatal("CloneApp mutation spec is missing")
	}
	for _, argument := range []string{"projectId: $projectId", "appId: $appId", "targetProjectId: $targetProjectId", "targetOrgId: $targetOrgId", "name: $name", "overwrite: $overwrite", "files: $files", "dryRun: $dryRun", "backupVersion"} {
		if !strings.Contains(spec.Query, argument) {
			t.Fatalf("CloneApp query is missing %q:\n%s", argument, spec.Query)
		}
	}
	generic, ok := GenericWriteSpecs["action:clone:app"]
	if !ok || generic.Mutation != "CloneApp" || generic.Mode != GenericWriteModeDirect || !generic.InjectProjectID {
		t.Fatalf("generic clone spec = %+v, want a direct CloneApp write with the source project injected", generic)
	}
	if _, ok := PayloadShapes["action:clone:app"]; !ok {
		t.Fatal("action:clone:app payload shape is missing")
	}
}
