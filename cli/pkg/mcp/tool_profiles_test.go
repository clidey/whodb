package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestTaskProfilesCatalogAndSafety(t *testing.T) {
	full := scopeTestClient(t, &ServerOptions{PlatformEnabled: true, ConfirmWrites: true})
	all, err := full.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	allBytes, _ := json.Marshal(all)
	for profile, required := range map[string][]string{
		"data":      {"sources", "source_objects", "source_rows", "ontology_query"},
		"functions": {"functions", "function", "create", "action", "function_runs", "function_wait"},
		"workspace": {"workspace_summary", "project_health", "project_create", "service_accounts", "security_audit"},
	} {
		t.Run(profile, func(t *testing.T) {
			client := scopeTestClient(t, &ServerOptions{PlatformEnabled: true, ConfirmWrites: true, ToolProfile: profile})
			catalog, err := client.ListTools(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			names := map[string]bool{}
			for _, tool := range catalog.Tools {
				names[tool.Name] = true
			}
			for _, name := range append(required, "confirm", "pending", "my_grants") {
				if !names["whodb_platform_"+name] {
					t.Errorf("journey lacks %s", name)
				}
			}
			data, _ := json.Marshal(catalog)
			t.Logf("%s: %d tools, %d bytes; all: %d tools, %d bytes", profile, len(catalog.Tools), len(data), len(all.Tools), len(allBytes))
			if len(data) >= len(allBytes)/2 {
				t.Error("profile did not halve catalog size")
			}
			filters := &ToolEnablement{ToolProfile: profile, DisabledTools: []string{"whodb_platform_confirm"}}
			if filters.isToolEnabled("whodb_platform_confirm") {
				t.Fatal("profile bypasses disabled tools")
			}
			if platformToolEnabledForMode("whodb_platform_create", &SecurityOptions{ReadOnly: true, ToolEnablement: filters}) {
				t.Fatal("profile bypasses read-only mode")
			}
		})
	}
	if ValidateToolProfile("typo") == nil {
		t.Fatal("unknown profile accepted")
	}
}
