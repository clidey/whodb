/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package cmd

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2"
	gqlast "github.com/vektah/gqlparser/v2/ast"
)

func TestBackupCommandNamesAndSelection(t *testing.T) {
	root := newBackupsCommand()
	for _, path := range [][]string{{"status"}, {"scope"}, {"create"}, {"policy", "set"}, {"key", "generate"}, {"download"}, {"upload"}, {"restore", "preview"}, {"restore", "start"}, {"delete"}} {
		if _, _, err := root.Find(path); err != nil {
			t.Fatalf("missing command %v: %v", path, err)
		}
	}
	all, err := backupSelection("")
	if err != nil || len(all["projects"].([]string)) != 0 {
		t.Fatalf("all selection: %v, %v", all, err)
	}
	path := filepath.Join(t.TempDir(), "selection.json")
	if err := os.WriteFile(path, []byte(`{"projects":["project-a"],"resources":[{"projectId":"project-b","type":"function","id":"function-c"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	selected, err := backupSelection(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected["projects"].([]any)) != 1 || len(selected["resources"].([]any)) != 1 {
		t.Fatalf("selection lost resources: %v", selected)
	}
}

func TestBackupContentDigest(t *testing.T) {
	const emptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	for _, test := range []struct {
		header string
		valid  bool
	}{
		{"sha-256=:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=:", true},
		{"sha-512=:AA==:, sha-256=:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=:", true},
		{"sha-256=:bad:", false},
		{"sha-256=:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=:, sha-256=:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=:", false},
		{"sha-512=:AA==:", false},
	} {
		digest, err := backupContentDigest(test.header)
		if test.valid && (err != nil || digest != emptySHA256) {
			t.Errorf("valid Content-Digest %q: digest=%q err=%v", test.header, digest, err)
		}
		if !test.valid && err == nil {
			t.Errorf("invalid Content-Digest %q was accepted", test.header)
		}
	}
}

// TestBackupDocumentsMatchPlatformSchema validates every backup document the CLI
// and MCP send against the hosted schema.
func TestBackupDocumentsMatchPlatformSchema(t *testing.T) {
	root := filepath.Join("..", "..")
	// The hosted schema extension lives in a private submodule that CE CI does not check out.
	extension := filepath.Join(root, "ee", "core", "graph", "schema.extension.graphqls")
	if _, err := os.Stat(extension); os.IsNotExist(err) {
		t.Skip("ee submodule not present; skipping platform schema check")
	}
	var sources []*gqlast.Source
	for _, path := range []string{filepath.Join(root, "core", "graph", "schema.graphqls"), extension} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		sources = append(sources, &gqlast.Source{Name: path, Input: string(body)})
	}
	schema, err := gqlparser.LoadSchema(sources...)
	if err != nil {
		t.Fatalf("load platform schema: %v", err)
	}
	for _, path := range []string{"platform_backups.go", filepath.Join("..", "pkg", "mcp", "platform_backups.go")} {
		file, err := goparser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			document, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatal(err)
			}
			// A document, not the "mutation " prefix the command itself tests for.
			if (!strings.HasPrefix(document, "query ") && !strings.HasPrefix(document, "mutation ")) || !strings.Contains(document, "{") {
				return true
			}
			count++
			if _, errs := gqlparser.LoadQuery(schema, document); len(errs) > 0 {
				t.Errorf("%s: %v", path, errs)
			}
			return true
		})
		if count < 5 {
			t.Errorf("%s: only %d documents checked", path, count)
		}
	}
}
