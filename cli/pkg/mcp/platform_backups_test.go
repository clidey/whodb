/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestBackupToolCatalogAndWriteBoundary(t *testing.T) {
	for _, readOnly := range []bool{false, true} {
		server := NewPlatformServer(readOnly, "test")
		clientTransport, serverTransport := mcp.NewInMemoryTransports()
		ctx := context.Background()
		serverSession, err := server.Connect(ctx, serverTransport, nil)
		if err != nil {
			t.Fatal(err)
		}
		client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
		clientSession, err := client.Connect(ctx, clientTransport, nil)
		if err != nil {
			_ = serverSession.Close()
			t.Fatal(err)
		}
		result, err := clientSession.ListTools(ctx, nil)
		_ = clientSession.Close()
		_ = serverSession.Close()
		if err != nil {
			t.Fatal(err)
		}
		found := map[string]bool{}
		for _, tool := range result.Tools {
			if strings.HasPrefix(tool.Name, "whodb_platform_org_backup_") {
				found[tool.Name] = true
				if strings.Contains(tool.Name, "restore") || strings.Contains(tool.Name, "key") || strings.Contains(tool.Name, "download") {
					t.Fatalf("MCP exposed recovery material tool %s", tool.Name)
				}
			}
		}
		for _, name := range []string{"whodb_platform_org_backup_status", "whodb_platform_org_backup_scope", "whodb_platform_org_backup_notifications"} {
			if !found[name] {
				t.Errorf("read_only=%v missing %s", readOnly, name)
			}
		}
		for _, name := range []string{"whodb_platform_org_backup_create", "whodb_platform_org_backup_policy_set"} {
			if found[name] == readOnly {
				t.Errorf("read_only=%v write visibility for %s is wrong", readOnly, name)
			}
			if !PlatformToolRequiresWrite(name) {
				t.Errorf("%s must require write scope", name)
			}
		}
	}
}
