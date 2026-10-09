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
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zalando/go-keyring"
)

// Unit tests must never use the developer's OS keyring.
func TestMain(m *testing.M) { keyring.MockInit(); os.Exit(m.Run()) }

type scopeTestHost struct {
	server  *httptest.Server
	mu      sync.Mutex
	deleted []string
	deny    bool
}

func newScopeTestHost(t *testing.T, name string) *scopeTestHost {
	t.Helper()
	host := &scopeTestHost{}
	host.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		issuer := host.server.URL + "/oidc"
		var out any
		switch r.URL.Path {
		case "/api/auth-config":
			out = map[string]any{"version": 2, "issuer": issuer, "clientId": "cli"}
		case "/oidc/.well-known/openid-configuration":
			out = map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token"}
		case "/oidc/token":
			out = map[string]any{"access_token": name, "expires_in": 3600}
		case "/api/query":
			if r.Header.Get("Authorization") != "Bearer "+name {
				http.Error(w, "wrong host credentials", http.StatusUnauthorized)
				return
			}
			var req struct {
				Query     string         `json:"query"`
				Variables map[string]any `json:"variables"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Error(err)
				return
			}
			data := map[string]any{}
			switch {
			case strings.Contains(req.Query, "PlatformManifest"):
				ops := []platformapi.PlatformManifestOperation{}
				for _, op := range []string{"MyOrganizations", "Projects", "ProjectSources", "Me"} {
					ops = append(ops, platformapi.PlatformManifestOperation{Name: op, Kind: "Query"})
				}
				ops = append(ops, platformapi.PlatformManifestOperation{Name: "DeleteSource", Kind: "Mutation"})
				data["PlatformManifest"] = platformapi.PlatformManifest{PlatformVersion: "test", ManifestProtocolVersion: "1", Operations: ops}
			case strings.Contains(req.Query, "MyOrganizations"):
				data["MyOrganizations"] = []map[string]any{{"id": "org-1", "name": "Acme", "slug": "acme"}}
			case strings.Contains(req.Query, "Projects"):
				data["Projects"] = []map[string]any{{"id": "proj-1", "orgId": "org-1", "name": "Analysis", "slug": "analysis"}, {"id": "proj-2", "orgId": "org-1", "name": "Operations", "slug": "operations"}, {"id": "proj-3", "orgId": "org-1", "name": "Duplicate", "slug": "dup-a"}, {"id": "proj-4", "orgId": "org-1", "name": "Duplicate", "slug": "dup-b"}}
			case strings.Contains(req.Query, "ProjectSources"):
				project, _ := req.Variables["projectId"].(string)
				host.mu.Lock()
				denied := host.deny
				host.mu.Unlock()
				if denied {
					out = map[string]any{"errors": []map[string]string{{"message": "permission denied"}}}
					break
				}
				data["ProjectSources"] = []map[string]any{{"id": name + "-" + project, "name": name + "-" + project, "projectId": project, "sourceType": "Postgres"}}
			case strings.Contains(req.Query, "DeleteSource"):
				host.mu.Lock()
				host.deleted = append(host.deleted, fmt.Sprint(req.Variables["projectId"]))
				host.mu.Unlock()
				data["DeleteSource"] = map[string]any{"Status": true}
			case strings.Contains(req.Query, "Me"):
				data["Me"] = map[string]any{"id": "account-" + name, "email": name + "@example.test"}
			default:
				t.Errorf("unexpected GraphQL query %s", req.Query)
			}
			if out == nil {
				out = map[string]any{"data": data}
			}
		default:
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(out); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(host.server.Close)
	return host
}

func scopeTestClient(t *testing.T, options ...*ServerOptions) *mcp.ClientSession {
	t.Helper()
	ct, st := mcp.NewInMemoryTransports()
	opts := &ServerOptions{PlatformEnabled: true, ConfirmWrites: true}
	if len(options) > 0 {
		opts = options[0]
	}
	server := NewServer(opts)
	ss, err := server.Connect(context.Background(), st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "scope-test", Version: "1"}, nil)
	cs, err := client.Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func scopeCall(t *testing.T, cs *mcp.ClientSession, tool string, args map[string]any) map[string]any {
	t.Helper()
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	message, _ := out["error"].(string)
	if result.IsError != (message != "") {
		t.Fatalf("%s error flag does not match payload: %+v", tool, result)
	}
	return out
}

func TestPlatformMCPMultiHostWorkspaceRouting(t *testing.T) {
	setupTestEnv(t)
	uat, prod := newScopeTestHost(t, "uat"), newScopeTestHost(t, "prod")
	cfg, err := config.LoadConfigWithoutSecrets()
	if err != nil {
		t.Fatal(err)
	}
	for name, host := range map[string]*scopeTestHost{"uat": uat, "prod": prod} {
		cfg.UpsertPlatformHost(config.PlatformHost{URL: host.server.URL, AccountID: "account-" + name, Email: name + "@example.test", DefaultOrgID: "org-1", DefaultProjectID: "proj-1"})
		if err := cfg.SavePlatformRefreshToken(host.server.URL, "account-"+name, "refresh-"+name); err != nil {
			t.Fatal(err)
		}
	}
	cfg.SetDefaultPlatformHost(prod.server.URL)
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	path, _ := config.GetConfigPath()
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Process defaults deliberately disagree with explicit calls.
	t.Setenv(platformapi.SessionHostEnv, prod.server.URL)
	t.Setenv(platformapi.SessionOrgEnv, "acme")
	t.Setenv(platformapi.SessionProjectEnv, "analysis")
	client := scopeTestClient(t)
	hosts := scopeCall(t, client, "whodb_platform_hosts", map[string]any{})
	if len(hosts["hosts"].([]any)) != 2 {
		t.Fatalf("hosts = %#v", hosts)
	}
	target := func(host, project string) map[string]any {
		return map[string]any{"workspace": map[string]any{"host": host, "org": "acme", "project": project}}
	}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			host, name, project, id := uat, "uat", "operations", "proj-2"
			if i%2 == 0 {
				host, name, project, id = prod, "prod", "analysis", "proj-1"
			}
			out := scopeCall(t, client, "whodb_platform_sources", target(host.server.URL, project))
			scope := out["scope"].(map[string]any)
			if scope["host"] != host.server.URL || scope["project_id"] != id {
				t.Errorf("wrong scope: %#v", out)
			}
			sources := out["sources"].([]any)
			if sources[0].(map[string]any)["id"] != name+"-"+id {
				t.Errorf("wrong source: %#v", out)
			}
		}(i)
	}
	wg.Wait()
	resolved := scopeCall(t, client, "whodb_platform_workspace_resolve", target(uat.server.URL, "operations"))
	if resolved["default_project_id"] != "proj-2" {
		t.Fatalf("resolve: %#v", resolved)
	}
	// Org-only discovery must clear the process's project selection.
	projects := scopeCall(t, client, "whodb_platform_projects", map[string]any{"workspace": map[string]any{"host": uat.server.URL, "org": "acme"}})
	if projects["scope"].(map[string]any)["project_id"] != nil {
		t.Fatalf("inherited unrelated project: %#v", projects)
	}
	bad := scopeCall(t, client, "whodb_platform_sources", target(uat.server.URL, "missing"))
	if !strings.Contains(fmt.Sprint(bad["error"]), "not found") {
		t.Fatalf("missing project: %#v", bad)
	}
	invalid, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: "whodb_platform_sources", Arguments: map[string]any{"workspace": map[string]any{"project": "analysis"}}})
	if err != nil || !invalid.IsError {
		t.Fatalf("partial target accepted: %#v %v", invalid, err)
	}

	ambiguous := scopeCall(t, client, "whodb_platform_sources", target(uat.server.URL, "Duplicate"))
	if !strings.Contains(fmt.Sprint(ambiguous["error"]), "ambiguous") {
		t.Fatalf("ambiguous target accepted: %#v", ambiguous)
	}
	missingLogin := scopeCall(t, client, "whodb_platform_sources", target("https://missing.example.test", "analysis"))
	if !strings.Contains(fmt.Sprint(missingLogin["error"]), "not logged in") {
		t.Fatalf("missing login: %#v", missingLogin)
	}
	uat.mu.Lock()
	uat.deny = true
	uat.mu.Unlock()
	denied := scopeCall(t, client, "whodb_platform_sources", target(uat.server.URL, "analysis"))
	if !strings.Contains(fmt.Sprint(denied["error"]), "permission denied") {
		t.Fatalf("permission error: %#v", denied)
	}
	uat.mu.Lock()
	uat.deny = false
	uat.mu.Unlock()
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("explicit MCP reads changed saved configuration")
	}
	// A confirmed action remains on UAT even though the process defaults to prod.

	previewArgs := target(uat.server.URL, "operations")
	previewArgs["source"] = "uat-proj-2"
	preview := scopeCall(t, client, "whodb_platform_source_delete", previewArgs)
	if preview["error"] != nil {
		t.Fatalf("preview: %#v", preview)
	}
	details := preview["confirmation_preview"].(map[string]any)
	if details["account_id"] != "account-uat" || details["host"] != uat.server.URL {
		t.Fatalf("preview target: %#v", details)
	}
	token := preview["confirmation_token"].(string)
	wrongTarget := target(prod.server.URL, "analysis")
	wrongTarget["token"] = token
	wrongConfirmation := scopeCall(t, client, "whodb_platform_confirm", wrongTarget)
	if !strings.Contains(fmt.Sprint(wrongConfirmation["error"]), "does not match") {
		t.Fatalf("conflicting confirmation target accepted: %#v", wrongConfirmation)
	}
	confirmed := scopeCall(t, client, "whodb_platform_confirm", map[string]any{"token": token})
	if confirmed["error"] != nil {
		t.Fatalf("confirm: %#v", confirmed)
	}
	if confirmed["scope"].(map[string]any)["host"] != uat.server.URL {
		t.Fatalf("confirmation scope: %#v", confirmed)
	}
	uat.mu.Lock()
	deletions := append([]string(nil), uat.deleted...)
	uat.mu.Unlock()
	if len(deletions) != 1 || deletions[0] != "proj-2" {
		t.Fatalf("wrong confirmation destination: %v", deletions)
	}
	prod.mu.Lock()
	n := len(prod.deleted)
	prod.mu.Unlock()
	if n != 0 {
		t.Fatal("production received mutation")
	}
	token, _ = storePendingPlatformAction(&PendingPlatformAction{AccountID: "previous-account", Host: uat.server.URL, OrgID: "org-1", ProjectID: "proj-2", Operation: "delete_source", SourceID: "source-2"})
	rejected := scopeCall(t, client, "whodb_platform_confirm", map[string]any{"token": token})
	if !strings.Contains(fmt.Sprint(rejected["error"]), "account changed") {
		t.Fatalf("changed account accepted: %#v", rejected)
	}
	uat.server.Close()
	unreachable := scopeCall(t, client, "whodb_platform_sources", target(uat.server.URL, "analysis"))
	if unreachable["error"] == nil {
		t.Fatalf("unreachable host unexpectedly succeeded: %#v", unreachable)
	}
}

func TestPlatformToolSchemasExposeWorkspace(t *testing.T) {
	tools := listServerTools(t, NewServer(&ServerOptions{PlatformEnabled: true}))
	for _, tool := range tools.Tools {
		raw, _ := json.Marshal(tool.InputSchema)
		var schema map[string]any
		_ = json.Unmarshal(raw, &schema)
		properties, _ := schema["properties"].(map[string]any)
		if properties["workspace"] == nil {
			t.Errorf("%s missing workspace input", tool.Name)
		}
	}
}
