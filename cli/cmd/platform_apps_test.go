/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/clidey/whodb/cli/internal/platform"
)

func TestRemapAppOntologyIDsByAPIName(t *testing.T) {
	warnings := []string{}
	got := remapAppOntologyIDs(
		[]string{"source-product", "missing"},
		[]platform.Ontology{{ID: "source-product", APIName: "product"}},
		[]platform.Ontology{{ID: "legacy-product", APIName: "Product"}, {ID: "target-product", APIName: "product"}},
		&warnings,
	)
	if len(got) != 1 || got[0] != "target-product" {
		t.Fatalf("remapped ids = %#v, want target-product", got)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %#v, want one unresolved reference", warnings)
	}
}

func TestMergeAppBrandFilesPreservesIntentAndOverlaysTheme(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.json"), []byte(`{"theme":"system"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "brand.json"), []byte(`{"name":"Christy"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := mergeAppBrandFiles([]platformAppFile{{Path: "app.json", Content: `{"theme":"light","intent":{"pages":["main"]}}`}}, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("file count = %d, want 2", len(files))
	}
	var app map[string]any
	for _, file := range files {
		if file.Path == "app.json" {
			if err := json.Unmarshal([]byte(file.Content), &app); err != nil {
				t.Fatal(err)
			}
		}
	}
	if app["theme"] != "system" || app["intent"] == nil {
		t.Fatalf("merged app = %#v, want system theme and preserved intent", app)
	}
}
