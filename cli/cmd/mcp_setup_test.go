/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
	whodbmcp "github.com/clidey/whodb/cli/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPSetupWithoutSkills(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	ctx := context.Background()
	active := config.DefaultMCPSettings()
	server := whodbmcp.NewServer(&whodbmcp.ServerOptions{PlatformEnabled: true, SetupHandler: mcpSetupHandler(active)})
	ct, st := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ss.Close() }()
	client := mcp.NewClient(&mcp.Implementation{Name: "no-skills", Version: "1"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cs.Close() }()
	if !strings.Contains(cs.InitializeResult().Instructions, "without installed skills") {
		t.Fatal("missing bootstrap guidance")
	}
	list, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tool := range list.Tools {
		found = found || tool.Name == "whodb_mcp_setup"
	}
	if !found {
		t.Fatal("setup not discoverable")
	}
	call := func(args map[string]any, wantError bool) map[string]any {
		t.Helper()
		result, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "whodb_mcp_setup", Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		if result.IsError != wantError {
			t.Fatalf("error=%v: %+v", result.IsError, result)
		}
		raw, err := json.Marshal(result.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		var out map[string]any
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	inspect := call(map[string]any{"action": "inspect"}, false)
	data := inspect["result"].(map[string]any)["data"].(map[string]any)
	if data["settings_schema"] == nil || data["module_choices"] == nil || inspect["guidance"] == nil {
		t.Fatal(inspect)
	}
	patch := map[string]any{"modules": []string{"platform", "database"}, "write_mode": "read-only"}
	preview := call(map[string]any{"action": "preview", "patch": patch}, false)
	before, _ := config.LoadMCPSettings()
	if before.WriteMode != "confirm" {
		t.Fatal("preview saved settings")
	}
	token := preview["confirmation_token"]
	call(map[string]any{"action": "apply", "confirmation_token": token}, true)
	call(map[string]any{"action": "apply", "confirmation_token": token, "approved": true, "patch": patch}, true)
	applied := call(map[string]any{"action": "apply", "confirmation_token": token, "approved": true}, false)
	if applied["result"].(map[string]any)["data"].(map[string]any)["restart_required"] != true {
		t.Fatal(applied)
	}
	saved, _ := config.LoadMCPSettings()
	if saved.WriteMode != "read-only" || !saved.HasModule("database") {
		t.Fatal(saved)
	}
	if len(applied["active"].(map[string]any)["modules"].([]any)) != 1 {
		t.Fatal("changed live modules before restart")
	}
	call(map[string]any{"action": "apply", "confirmation_token": token, "approved": true}, true)
	// Another client/local CLI change invalidates an already approved snapshot.
	preview = call(map[string]any{"action": "preview", "patch": map[string]any{"write_mode": "confirm"}}, false)
	saved.DefaultConnection = "changed-elsewhere"
	if err := config.SaveMCPSettings(saved); err != nil {
		t.Fatal(err)
	}
	call(map[string]any{"action": "apply", "confirmation_token": preview["confirmation_token"], "approved": true}, true)
	after, _ := config.LoadMCPSettings()
	if after.DefaultConnection != "changed-elsewhere" || after.WriteMode != "read-only" {
		t.Fatal(after)
	}
	call(map[string]any{"action": "preview", "patch": map[string]any{"password": "not-a-setting"}}, true)
	// Verification uses the live platform-only scope, not the saved database module.
	verified := call(map[string]any{"action": "verify"}, true)
	checks := verified["result"].(map[string]any)["checks"].([]any)
	if len(checks) != 1 || checks[0].(map[string]any)["module"] != "platform" {
		t.Fatal(verified)
	}
}

func TestMCPSetupInspectRespectsActiveTargets(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	cfg, err := config.LoadConfigWithoutSecrets()
	if err != nil {
		t.Fatal(err)
	}
	cfg.UpsertPlatformHost(config.PlatformHost{URL: "https://allowed.test", DefaultOrgID: "hidden-org", DefaultProjectID: "hidden-project"})
	cfg.UpsertPlatformHost(config.PlatformHost{URL: "https://hidden.test"})
	cfg.Connections = []config.Connection{{Name: "allowed", Type: "Postgres"}, {Name: "hidden", Type: "Postgres"}}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	active := config.MCPSettings{Modules: []string{"platform", "database"}, WriteMode: "confirm", AllowedConnections: []string{"allowed"}, PlatformPolicy: &config.MCPPlatformPolicy{AllowedWorkspaces: []config.MCPWorkspace{{Host: "https://allowed.test", Org: "org", Project: "project"}}}}
	raw, err := mcpSetupHandler(active)(context.Background(), "inspect", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "hidden") {
		t.Fatalf("restricted metadata leaked: %s", raw)
	}
}
