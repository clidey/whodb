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
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMixedModulesShareOneServer(t *testing.T) {
	setupTestEnv(t)
	cfg, err := config.LoadConfigWithoutSecrets()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Connections = []config.Connection{{Name: "allowed", Type: "Postgres"}, {Name: "hidden", Type: "Postgres"}}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	client := scopeTestClient(t, &ServerOptions{PlatformEnabled: true, DatabaseEnabled: true, ConfirmWrites: true, AllowedConnections: []string{"allowed"}})
	listed, err := client.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, tool := range listed.Tools {
		names[tool.Name] = true
	}
	for _, name := range []string{"whodb_platform_hosts", "whodb_platform_confirm", "whodb_query", "whodb_confirm", "whodb_connections"} {
		if !names[name] {
			t.Errorf("missing module tool %s", name)
		}
	}
	resource, err := client.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "whodb://connections"})
	if err != nil {
		t.Fatal(err)
	}
	var connections []string
	if err := json.Unmarshal([]byte(resource.Contents[0].Text), &connections); err != nil {
		t.Fatal(err)
	}
	if len(connections) != 1 || connections[0] != "allowed" {
		t.Fatal(connections)
	}
	resource, err = client.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: "whodb://mcp/configuration"})
	if err != nil {
		t.Fatal(err)
	}
	var active struct {
		Modules []string `json:"modules"`
	}
	if err := json.Unmarshal([]byte(resource.Contents[0].Text), &active); err != nil {
		t.Fatal(err)
	}
	if len(active.Modules) != 2 {
		t.Fatal(active)
	}
}

func TestMixedModuleToolFilters(t *testing.T) {
	client := scopeTestClient(t, &ServerOptions{PlatformEnabled: true, DatabaseEnabled: true, EnabledTools: []string{"whodb_platform_hosts", "whodb_connections", "whodb_query"}, DisabledTools: []string{"query"}})
	listed, err := client.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Tools) != 2 {
		t.Fatal(listed)
	}
	for _, tool := range listed.Tools {
		if tool.Name != "whodb_platform_hosts" && tool.Name != "whodb_connections" {
			t.Fatal(tool.Name)
		}
	}
}
