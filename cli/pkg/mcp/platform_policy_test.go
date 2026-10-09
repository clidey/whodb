/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestPlatformPolicyRoutingAndConfirmation(t *testing.T) {
	setupTestEnv(t)
	uat, prod := newScopeTestHost(t, "uat"), newScopeTestHost(t, "prod")
	cfg, err := config.LoadConfigWithoutSecrets()
	if err != nil {
		t.Fatal(err)
	}
	for name, host := range map[string]*scopeTestHost{"uat": uat, "prod": prod} {
		cfg.UpsertPlatformHost(config.PlatformHost{URL: host.server.URL, AccountID: "account-" + name, DefaultOrgID: "org-1", DefaultProjectID: "proj-1"})
		if err := cfg.SavePlatformRefreshToken(host.server.URL, "account-"+name, "refresh"); err != nil {
			t.Fatal(err)
		}
	}
	cfg.SetDefaultPlatformHost(prod.server.URL)
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	policy := &PlatformPolicy{AllowedHosts: []string{uat.server.URL, prod.server.URL}, AllowedWorkspaces: []PlatformWorkspaceTarget{{Host: uat.server.URL, Org: "org-1", Project: "proj-1"}, {Host: prod.server.URL, Org: "org-1", Project: "proj-1"}}, ReadOnlyHosts: []string{prod.server.URL}}
	client := scopeTestClient(t, &ServerOptions{PlatformEnabled: true, ConfirmWrites: true, PlatformPolicy: policy})
	target := func(host, project string) map[string]any {
		return map[string]any{"workspace": map[string]any{"host": host, "org": "acme", "project": project}}
	}
	if out := scopeCall(t, client, "whodb_platform_sources", target(uat.server.URL, "analysis")); out["error"] != nil {
		t.Fatal(out)
	}
	if out := scopeCall(t, client, "whodb_platform_sources", target(uat.server.URL, "operations")); out["error"] == nil {
		t.Fatal("disallowed project succeeded")
	}
	if out := scopeCall(t, client, "whodb_platform_sources", target("https://blocked.example.test", "analysis")); !strings.Contains(out["error"].(string), "policy") {
		t.Fatal(out)
	}
	projects := scopeCall(t, client, "whodb_platform_projects", map[string]any{"workspace": map[string]any{"host": uat.server.URL, "org": "acme"}})
	if len(projects["projects"].([]any)) != 1 {
		t.Fatal(projects)
	}
	conflict := target(uat.server.URL, "analysis")
	conflict["project"] = "operations"
	if out := scopeCall(t, client, "whodb_platform_project_delete", conflict); out["error"] == nil {
		t.Fatal("legacy selector escaped workspace policy")
	}
	write := target(prod.server.URL, "analysis")
	write["source"] = "prod-proj-1"
	if out := scopeCall(t, client, "whodb_platform_source_delete", write); out["error"] == nil {
		t.Fatal("production write preview allowed")
	}
	token, _ := storePendingPlatformAction(&PendingPlatformAction{AccountID: "account-prod", Host: prod.server.URL, OrgID: "org-1", ProjectID: "proj-1", Operation: "delete_source", SourceID: "prod-proj-1"})
	if out := scopeCall(t, client, "whodb_platform_confirm", map[string]any{"token": token}); out["error"] == nil {
		t.Fatal("pinned production confirmation allowed")
	}
	write = target(uat.server.URL, "analysis")
	write["source"] = "uat-proj-1"
	preview := scopeCall(t, client, "whodb_platform_source_delete", write)
	if preview["error"] != nil {
		t.Fatal(preview)
	}
	confirmed := scopeCall(t, client, "whodb_platform_confirm", map[string]any{"token": preview["confirmation_token"]})
	if confirmed["error"] != nil {
		t.Fatal(confirmed)
	}
	readonlyClient := scopeTestClient(t, &ServerOptions{PlatformEnabled: true, AllowWrite: true, PlatformPolicy: &PlatformPolicy{ReadOnlyWorkspaces: []PlatformWorkspaceTarget{{Host: uat.server.URL, Org: "org-1", Project: "proj-1"}}}})
	if out := scopeCall(t, readonlyClient, "whodb_platform_source_delete", write); out["error"] == nil {
		t.Fatal("allow-write bypassed workspace read-only policy")
	}
	restrictedClient := scopeTestClient(t, &ServerOptions{PlatformEnabled: true, ConfirmWrites: true, PlatformPolicy: &PlatformPolicy{AllowedHosts: []string{uat.server.URL}}})
	hosts := scopeCall(t, restrictedClient, "whodb_platform_hosts", map[string]any{})
	if len(hosts["hosts"].([]any)) != 1 {
		t.Fatal(hosts)
	}
	pending := scopeCall(t, restrictedClient, "whodb_platform_pending", map[string]any{})
	for _, item := range pending["pending"].([]any) {
		if item.(map[string]any)["token"] == token {
			t.Fatal("disallowed host pending action leaked")
		}
	}
	prod.mu.Lock()
	defer prod.mu.Unlock()
	if len(prod.deleted) != 0 {
		t.Fatal("production mutated")
	}
}

func TestPlatformToolFiltering(t *testing.T) {
	client := scopeTestClient(t, &ServerOptions{PlatformEnabled: true, ConfirmWrites: true, EnabledTools: []string{"whodb_platform_hosts", "whodb_platform_status"}, DisabledTools: []string{"whodb_platform_status"}})
	list, err := client.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Tools) != 1 || list.Tools[0].Name != "whodb_platform_hosts" {
		t.Fatal(list)
	}
	result, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "whodb_platform_status", Arguments: map[string]any{}})
	if err == nil && !result.IsError {
		t.Fatal("disabled tool callable")
	}
	resource, err := client.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "whodb://platform/schema"})
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal([]byte(resource.Contents[0].Text), &schema); err != nil {
		t.Fatal(err)
	}
	if len(schema.Tools) != 1 || schema.Tools[0].Name != "whodb_platform_hosts" {
		t.Fatal(schema)
	}
}

func TestLoadPlatformPolicyValidation(t *testing.T) {
	for _, data := range []string{`null`, `{"allowed_host":[]}`, `{"allowed_workspaces":[{"host":"https://example.test"}]}`, `{} {}`} {
		path := filepath.Join(t.TempDir(), "policy.json")
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadPlatformPolicy(path); err == nil {
			t.Fatalf("accepted %s", data)
		}
	}
}

func TestLoadPlatformPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(path, []byte(`{"allowed_hosts":["https://example.test/"],"read_only_workspaces":[{"host":"https://example.test/","org":"org-1","project":"project-1"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	p, err := LoadPlatformPolicy(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.AllowedHosts[0] != "https://example.test" || p.ReadOnlyWorkspaces[0].Host != "https://example.test" {
		t.Fatal(p)
	}
}
