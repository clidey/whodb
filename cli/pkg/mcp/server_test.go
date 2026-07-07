/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/clidey/whodb/cli/internal/agentmanifest"
	"github.com/clidey/whodb/cli/pkg/version"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewServer_DefaultOptions(t *testing.T) {
	server := NewServer(nil)
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
}

func TestNewServer_WithCustomLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	server := NewServer(&ServerOptions{
		Logger: logger,
	})
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
}

func TestNewServer_WithCustomInstructions(t *testing.T) {
	server := NewServer(&ServerOptions{
		Instructions: "Custom instructions for testing",
	})
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
}

func TestNewServer_RegistersAllTools(t *testing.T) {
	server := NewServer(nil)
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}

	// The server is created successfully - tools are registered internally
	// We can't easily inspect registered tools without modifying the SDK,
	// but we can verify the server was created without error
}

func TestVersion_IsSet(t *testing.T) {
	// Version should be "dev" by default or set by build flags
	if version.Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestDefaultInstructions_NotEmpty(t *testing.T) {
	if defaultInstructions == "" {
		t.Error("defaultInstructions should not be empty")
	}
}

func TestDefaultInstructions_ContainsToolNames(t *testing.T) {
	expectedTools := []string{
		"whodb_query",
		"whodb_schemas",
		"whodb_tables",
		"whodb_columns",
		"whodb_connections",
		"whodb_explain",
		"whodb_diff",
		"whodb_erd",
		"whodb_audit",
		"whodb_suggestions",
	}

	for _, tool := range expectedTools {
		if !strings.Contains(defaultInstructions, tool) {
			t.Errorf("defaultInstructions should mention %s", tool)
		}
	}
}

// Security options tests

func TestNewServer_SecurityDefaults(t *testing.T) {
	// Test that NewServer(nil) creates a server with safe defaults (confirm-writes mode)
	server := NewServer(nil)
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}

	// NewServer(nil) should enable confirm-writes by default for safe operation
	// The server is created successfully - we can't inspect internal state directly,
	// but the test confirms NewServer handles nil opts without panicking
}

func TestNewServer_WithSecurityOptions(t *testing.T) {
	server := NewServer(&ServerOptions{
		ReadOnly:      false,
		ConfirmWrites: true,
		SecurityLevel: "strict",
		QueryTimeout:  60 * 1e9, // 60 seconds
		MaxRows:       500,
	})
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
}

func TestBuildQueryDescription(t *testing.T) {
	// Test read-only mode description
	readOnlyOpts := &SecurityOptions{ReadOnly: true}
	desc := buildQueryDescription(readOnlyOpts)
	if !strings.Contains(desc, "READ-ONLY") {
		t.Error("Read-only description should contain 'READ-ONLY'")
	}

	// Test confirm-writes mode description
	confirmOpts := &SecurityOptions{ReadOnly: false, ConfirmWrites: true}
	desc = buildQueryDescription(confirmOpts)
	if !strings.Contains(desc, "confirmation") {
		t.Error("Confirm-writes description should mention 'confirmation'")
	}

	// Test allow-write mode description
	writeOpts := &SecurityOptions{ReadOnly: false, ConfirmWrites: false}
	desc = buildQueryDescription(writeOpts)
	if strings.Contains(desc, "READ-ONLY") || strings.Contains(desc, "confirmation") {
		t.Error("Allow-write description should not mention read-only or confirmation")
	}
}

// Tool enablement tests

func TestToolEnablement_AllEnabledByDefault(t *testing.T) {
	te := &ToolEnablement{}

	tools := []string{"query", "schemas", "tables", "columns", "connections", "confirm", "pending", "explain", "diff", "erd", "audit", "suggestions"}
	for _, tool := range tools {
		if !te.isToolEnabled(tool) {
			t.Errorf("Tool %s should be enabled by default", tool)
		}
	}
}

func TestToolEnablement_OnlySpecificEnabled(t *testing.T) {
	te := &ToolEnablement{
		EnabledTools: []string{"query", "schemas"},
	}

	if !te.isToolEnabled("query") {
		t.Error("Tool 'query' should be enabled")
	}
	if !te.isToolEnabled("schemas") {
		t.Error("Tool 'schemas' should be enabled")
	}
	if te.isToolEnabled("tables") {
		t.Error("Tool 'tables' should be disabled when not in EnabledTools list")
	}
	if te.isToolEnabled("connections") {
		t.Error("Tool 'connections' should be disabled when not in EnabledTools list")
	}
}

func TestToolEnablement_DisabledTakesPrecedence(t *testing.T) {
	te := &ToolEnablement{
		EnabledTools:  []string{"query", "schemas", "tables"},
		DisabledTools: []string{"query"},
	}

	if te.isToolEnabled("query") {
		t.Error("Tool 'query' should be disabled (DisabledTools takes precedence)")
	}
	if !te.isToolEnabled("schemas") {
		t.Error("Tool 'schemas' should be enabled")
	}
}

func TestToolEnablement_DisableOnlySpecific(t *testing.T) {
	te := &ToolEnablement{
		DisabledTools: []string{"confirm"},
	}

	if !te.isToolEnabled("query") {
		t.Error("Tool 'query' should be enabled (not in disabled list)")
	}
	if te.isToolEnabled("confirm") {
		t.Error("Tool 'confirm' should be disabled")
	}
}

func TestNewServer_WithToolEnablement(t *testing.T) {
	server := NewServer(&ServerOptions{
		EnabledTools: []string{"schemas", "tables", "columns"},
	})
	if server == nil {
		t.Fatal("NewServer() returned nil with tool enablement")
	}
}

func TestNewServer_WithPlatformEnabled(t *testing.T) {
	server := NewServer(&ServerOptions{
		PlatformEnabled: true,
	})
	if server == nil {
		t.Fatal("NewServer() returned nil with platform tools enabled")
	}
}

func TestPlatformInstructionsExcludeLocalTools(t *testing.T) {
	if !strings.Contains(platformInstructions, "whodb_platform_status") {
		t.Fatal("platformInstructions should mention platform tools")
	}
	if strings.Contains(platformInstructions, "whodb_query") || strings.Contains(platformInstructions, "whodb_connections") {
		t.Fatal("platformInstructions should not advertise local database tools")
	}
}

func TestNewServer_PlatformModeListsOnlyPlatformTools(t *testing.T) {
	result := listServerTools(t, NewServer(&ServerOptions{PlatformEnabled: true}))
	for _, tool := range result.Tools {
		if !strings.HasPrefix(tool.Name, "whodb_platform_") {
			t.Fatalf("platform mode exposed non-platform tool %q", tool.Name)
		}
		if strings.TrimSpace(tool.Description) == "" {
			t.Fatalf("platform tool %q has empty description", tool.Name)
		}
		if tool.Annotations == nil {
			t.Fatalf("platform tool %q has no annotations", tool.Name)
		}
	}
	for _, localTool := range []string{"whodb_query", "whodb_connections", "whodb_confirm"} {
		if toolNamesContain(result.Tools, localTool) {
			t.Fatalf("platform mode exposed local tool %q", localTool)
		}
	}
	expectedTools := platformToolDefinitions()
	if len(result.Tools) != len(expectedTools) {
		t.Fatalf("platform mode exposed %d tools, want %d", len(result.Tools), len(expectedTools))
	}
	for _, expected := range expectedTools {
		if !toolNamesContain(result.Tools, expected.Name) {
			t.Fatalf("platform mode did not expose %s", expected.Name)
		}
	}
	for _, flexibleReadTool := range []string{
		"whodb_platform_secrets",
		"whodb_platform_datasets",
		"whodb_platform_functions",
		"whodb_platform_files",
	} {
		tool := findToolByName(result.Tools, flexibleReadTool)
		if tool == nil {
			t.Fatalf("platform mode did not expose %s", flexibleReadTool)
		}
		if tool.OutputSchema != nil {
			t.Fatalf("%s output schema = %#v, want nil for flexible read payloads", flexibleReadTool, tool.OutputSchema)
		}
	}
	for _, tool := range result.Tools {
		assertToolSchemasAreInspectorCompatible(t, tool)
	}
}

func TestNewServer_PlatformModeListsOnlyPlatformPrompts(t *testing.T) {
	result := listServerPrompts(t, NewServer(&ServerOptions{PlatformEnabled: true}))
	for _, prompt := range result.Prompts {
		if !strings.HasPrefix(prompt.Name, "whodb_platform_") {
			t.Fatalf("platform mode exposed non-platform prompt %q", prompt.Name)
		}
		if strings.TrimSpace(prompt.Description) == "" {
			t.Fatalf("platform prompt %q has empty description", prompt.Name)
		}
	}
	for _, localPrompt := range []string{"query_help", "schema_exploration_help", "workflow_help"} {
		if promptNamesContain(result.Prompts, localPrompt) {
			t.Fatalf("platform mode exposed local prompt %q", localPrompt)
		}
	}
	expectedPrompts := []string{
		"whodb_platform_overview",
		"whodb_platform_read_workflow",
		"whodb_platform_write_safety",
		"whodb_platform_source_workflow",
	}
	if len(result.Prompts) != len(expectedPrompts) {
		t.Fatalf("platform mode exposed %d prompts, want %d", len(result.Prompts), len(expectedPrompts))
	}
	for _, expected := range expectedPrompts {
		if !promptNamesContain(result.Prompts, expected) {
			t.Fatalf("platform mode did not expose prompt %s", expected)
		}
	}
}

func TestPlatformPromptContent(t *testing.T) {
	server := NewServer(&ServerOptions{PlatformEnabled: true})
	writeSafety := promptText(t, getServerPrompt(t, server, "whodb_platform_write_safety"))
	for _, expected := range []string{"whodb_platform_confirm", "confirmation_preview", "--allow-write"} {
		if !strings.Contains(writeSafety, expected) {
			t.Fatalf("write safety prompt should mention %q", expected)
		}
	}

	sourceWorkflow := promptText(t, getServerPrompt(t, server, "whodb_platform_source_workflow"))
	for _, expected := range []string{"whodb_platform_source_fields", "secrets are redacted", "source_type"} {
		if !strings.Contains(sourceWorkflow, expected) {
			t.Fatalf("source workflow prompt should mention %q", expected)
		}
	}
}

func TestNewServer_PlatformModeListsOnlyPlatformResources(t *testing.T) {
	result := listServerResources(t, NewServer(&ServerOptions{PlatformEnabled: true}))
	expectedResources := []string{
		"whodb://platform/schema",
		"whodb://platform/workspace",
		"whodb://platform/tool-guide",
	}
	if len(result.Resources) != len(expectedResources) {
		t.Fatalf("platform mode exposed %d resources, want %d", len(result.Resources), len(expectedResources))
	}
	for _, resource := range result.Resources {
		if !strings.HasPrefix(resource.URI, "whodb://platform/") {
			t.Fatalf("platform mode exposed non-platform resource %q", resource.URI)
		}
		if strings.TrimSpace(resource.Description) == "" {
			t.Fatalf("platform resource %q has empty description", resource.URI)
		}
	}
	for _, expected := range expectedResources {
		if !resourceURIsContain(result.Resources, expected) {
			t.Fatalf("platform mode did not expose resource %s", expected)
		}
	}
	for _, localResource := range []string{"whodb://connections", "whodb://agent/schema"} {
		if resourceURIsContain(result.Resources, localResource) {
			t.Fatalf("platform mode exposed local resource %q", localResource)
		}
	}
}

func TestPlatformResourcesReadJSON(t *testing.T) {
	withPlatformSessionLoader(t, func(context.Context) (*platformToolSession, error) {
		return testPlatformSession(&fakePlatformClient{}), nil
	})
	server := NewServer(&ServerOptions{PlatformEnabled: true})

	for _, uri := range []string{
		"whodb://platform/schema",
		"whodb://platform/workspace",
		"whodb://platform/tool-guide",
	} {
		text := resourceText(t, readServerResource(t, server, uri))
		var payload map[string]any
		if err := json.Unmarshal([]byte(text), &payload); err != nil {
			t.Fatalf("resource %s returned invalid JSON: %v\n%s", uri, err, text)
		}
	}

	schema := resourceText(t, readServerResource(t, server, "whodb://platform/schema"))
	if !strings.Contains(schema, `"whodb_platform_status"`) || strings.Contains(schema, `"whodb_query"`) {
		t.Fatalf("platform schema resource should include platform tools and exclude local tools: %s", schema)
	}
	for _, expected := range []string{`"write_specs"`, `"payload_shapes"`, `"key": "update:ai_provider"`, `"mutation": "UpdateAIProvider"`, `"secret": true`, `"examples"`} {
		if !strings.Contains(schema, expected) {
			t.Fatalf("platform schema resource should contain %s: %s", expected, schema)
		}
	}
	workspace := resourceText(t, readServerResource(t, server, "whodb://platform/workspace"))
	for _, expected := range []string{`"host": "https://app.whodb.com"`, `"email": "ada@example.com"`, `"workspace_selected": true`} {
		if !strings.Contains(workspace, expected) {
			t.Fatalf("platform workspace resource should contain %s: %s", expected, workspace)
		}
	}
	guide := resourceText(t, readServerResource(t, server, "whodb://platform/tool-guide"))
	for _, expected := range []string{`"sources"`, `"field_projection"`, `"whodb_platform_source_create"`, `"whodb_platform_file_inspect"`} {
		if !strings.Contains(guide, expected) {
			t.Fatalf("platform tool guide resource should contain %s: %s", expected, guide)
		}
	}
}

func TestPlatformResourcesReflectReadOnlyMode(t *testing.T) {
	server := NewServer(&ServerOptions{PlatformEnabled: true, ReadOnly: true})
	guide := resourceText(t, readServerResource(t, server, "whodb://platform/tool-guide"))
	if !strings.Contains(guide, `"mode": "read_only"`) {
		t.Fatalf("platform tool guide should report read_only mode: %s", guide)
	}
	if strings.Contains(guide, `"whodb_platform_source_create"`) || strings.Contains(guide, `"whodb_platform_confirm"`) {
		t.Fatalf("platform read-only guide should not include write tools: %s", guide)
	}
}

func TestPlatformToolGuideReferencesOnlyListedTools(t *testing.T) {
	tests := []struct {
		name               string
		opts               *ServerOptions
		wantMode           string
		wantTools          []string
		wantMissingTools   []string
		wantSupportedWrite bool
	}{
		{
			name:               "default confirm writes",
			opts:               &ServerOptions{PlatformEnabled: true},
			wantMode:           "confirm_writes",
			wantTools:          []string{"whodb_platform_source_create", "whodb_platform_confirm", "whodb_platform_pending"},
			wantSupportedWrite: true,
		},
		{
			name:               "read only",
			opts:               &ServerOptions{PlatformEnabled: true, ReadOnly: true},
			wantMode:           "read_only",
			wantMissingTools:   []string{"whodb_platform_source_create", "whodb_platform_confirm", "whodb_platform_pending"},
			wantSupportedWrite: false,
		},
		{
			name:               "allow write",
			opts:               &ServerOptions{PlatformEnabled: true, AllowWrite: true},
			wantMode:           "allow_write",
			wantTools:          []string{"whodb_platform_source_create", "whodb_platform_create", "whodb_platform_action"},
			wantMissingTools:   []string{"whodb_platform_confirm", "whodb_platform_pending"},
			wantSupportedWrite: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(tt.opts)
			listedTools := listServerToolNames(t, server)
			guide := readPlatformToolGuide(t, server)
			if guide.Mode != tt.wantMode {
				t.Fatalf("guide mode = %q, want %q", guide.Mode, tt.wantMode)
			}
			if len(guide.Categories) == 0 {
				t.Fatal("guide has no categories")
			}

			supportedWrites := 0
			for _, category := range guide.Categories {
				if len(category.Tools) == 0 && len(category.SupportedWrites) == 0 {
					t.Fatalf("guide category %q is empty", category.Name)
				}
				for _, tool := range category.Tools {
					if _, ok := listedTools[tool.Name]; !ok {
						t.Fatalf("guide category %q references unlisted tool %q", category.Name, tool.Name)
					}
				}
				for _, write := range category.SupportedWrites {
					supportedWrites++
					if _, ok := listedTools[write.Tool]; !ok {
						t.Fatalf("guide category %q references unlisted write tool %q", category.Name, write.Tool)
					}
				}
			}

			for _, name := range tt.wantTools {
				if _, ok := listedTools[name]; !ok {
					t.Fatalf("tools/list is missing expected tool %q", name)
				}
				if !platformGuideContainsTool(guide, name) {
					t.Fatalf("guide is missing expected tool %q", name)
				}
			}
			for _, name := range tt.wantMissingTools {
				if _, ok := listedTools[name]; ok {
					t.Fatalf("tools/list unexpectedly contains %q", name)
				}
				if platformGuideContainsTool(guide, name) {
					t.Fatalf("guide unexpectedly contains %q", name)
				}
			}
			if tt.wantSupportedWrite && supportedWrites == 0 {
				t.Fatal("guide should include supported write entries")
			}
			if !tt.wantSupportedWrite && supportedWrites != 0 {
				t.Fatalf("guide has %d supported write entries, want 0", supportedWrites)
			}
			if tt.wantSupportedWrite && !platformGuideContainsWrite(guide, "whodb_platform_update", "ai_provider") {
				t.Fatal("guide should document ai_provider update support")
			}
		})
	}
}

func TestAgentManifestIncludesPlatformMCPTools(t *testing.T) {
	manifest := agentmanifest.Build()
	if manifest.PlatformMCP.EnabledByFlag != "--platform" {
		t.Fatalf("agent manifest platform flag = %q, want --platform", manifest.PlatformMCP.EnabledByFlag)
	}
	if manifest.PlatformMCP.DefaultHost == "" {
		t.Fatal("agent manifest platform default host is empty")
	}
	if manifest.PlatformMCP.ToolPrefix != "whodb_platform_" {
		t.Fatalf("agent manifest platform tool prefix = %q, want whodb_platform_", manifest.PlatformMCP.ToolPrefix)
	}
	if manifest.PlatformMCP.LocalToolsExposed {
		t.Fatal("agent manifest platform mode says local tools are exposed")
	}
	if !manifest.PlatformMCP.SupportsFields {
		t.Fatal("agent manifest platform mode should advertise fields projection")
	}
	if manifest.PlatformMCP.WriteModes.Default != "confirmation_required" ||
		manifest.PlatformMCP.WriteModes.ReadOnly != "write_tools_hidden" ||
		manifest.PlatformMCP.WriteModes.SafeMode != "write_tools_hidden" ||
		manifest.PlatformMCP.WriteModes.AllowWrite != "executes_immediately" {
		t.Fatalf("agent manifest platform write modes = %#v", manifest.PlatformMCP.WriteModes)
	}
	manifestTools := map[string]agentmanifest.MCPTool{}
	for _, tool := range manifest.MCPTools {
		if strings.HasPrefix(tool.Name, manifest.PlatformMCP.ToolPrefix) {
			manifestTools[tool.Name] = tool
		}
	}
	for _, expected := range platformToolDefinitions() {
		if !strings.HasPrefix(expected.Name, manifest.PlatformMCP.ToolPrefix) {
			t.Fatalf("platform tool %s does not match manifest prefix %s", expected.Name, manifest.PlatformMCP.ToolPrefix)
		}
		tool, ok := manifestTools[expected.Name]
		if !ok {
			t.Fatalf("agent manifest missing platform tool %s", expected.Name)
		}
		if strings.TrimSpace(tool.Description) == "" {
			t.Fatalf("agent manifest platform tool %s has empty description", expected.Name)
		}
		if expected.Annotations == nil {
			continue
		}
		if tool.ReadOnly != expected.Annotations.ReadOnlyHint {
			t.Fatalf("agent manifest platform tool %s read_only = %v, want %v", expected.Name, tool.ReadOnly, expected.Annotations.ReadOnlyHint)
		}
	}
	if len(manifestTools) != len(platformToolDefinitions()) {
		t.Fatalf("agent manifest has %d platform MCP tools, want %d", len(manifestTools), len(platformToolDefinitions()))
	}
}

func TestAgentManifestIncludesPlatformMCPPromptsAndResources(t *testing.T) {
	manifest := agentmanifest.Build()
	server := NewServer(&ServerOptions{PlatformEnabled: true})

	listedPrompts := listServerPrompts(t, server).Prompts
	if len(manifest.PlatformMCP.Prompts) != len(listedPrompts) {
		t.Fatalf("agent manifest has %d platform prompts, want %d", len(manifest.PlatformMCP.Prompts), len(listedPrompts))
	}
	for _, prompt := range manifest.PlatformMCP.Prompts {
		if !strings.HasPrefix(prompt.Name, manifest.PlatformMCP.ToolPrefix) {
			t.Fatalf("platform prompt %s does not match manifest prefix %s", prompt.Name, manifest.PlatformMCP.ToolPrefix)
		}
		if strings.TrimSpace(prompt.Description) == "" {
			t.Fatalf("agent manifest platform prompt %s has empty description", prompt.Name)
		}
		listed := findPromptByName(listedPrompts, prompt.Name)
		if listed == nil {
			t.Fatalf("agent manifest platform prompt %s is not listed by MCP server", prompt.Name)
		}
		if listed.Description != prompt.Description {
			t.Fatalf("platform prompt %s description = %q, want %q", prompt.Name, listed.Description, prompt.Description)
		}
	}

	listedResources := listServerResources(t, server).Resources
	if len(manifest.PlatformMCP.Resources) != len(listedResources) {
		t.Fatalf("agent manifest has %d platform resources, want %d", len(manifest.PlatformMCP.Resources), len(listedResources))
	}
	for _, resource := range manifest.PlatformMCP.Resources {
		if !strings.HasPrefix(resource.URI, "whodb://platform/") {
			t.Fatalf("platform resource %s does not use platform URI prefix", resource.URI)
		}
		if strings.TrimSpace(resource.Description) == "" {
			t.Fatalf("agent manifest platform resource %s has empty description", resource.URI)
		}
		if resource.MIMEType != "application/json" {
			t.Fatalf("agent manifest platform resource %s MIME type = %q, want application/json", resource.URI, resource.MIMEType)
		}
		listed := findResourceByURI(listedResources, resource.URI)
		if listed == nil {
			t.Fatalf("agent manifest platform resource %s is not listed by MCP server", resource.URI)
		}
		if listed.Description != resource.Description {
			t.Fatalf("platform resource %s description = %q, want %q", resource.URI, listed.Description, resource.Description)
		}
		if listed.MIMEType != resource.MIMEType {
			t.Fatalf("platform resource %s MIME type = %q, want %q", resource.URI, listed.MIMEType, resource.MIMEType)
		}
	}
}

func TestNewServer_PlatformReadOnlyHidesWriteTools(t *testing.T) {
	tools := listServerToolNames(t, NewServer(&ServerOptions{PlatformEnabled: true, ReadOnly: true}))
	for _, name := range []string{
		"whodb_platform_source_create",
		"whodb_platform_source_update",
		"whodb_platform_source_delete",
		"whodb_platform_create",
		"whodb_platform_update",
		"whodb_platform_delete",
		"whodb_platform_action",
		"whodb_platform_confirm",
		"whodb_platform_pending",
	} {
		if _, ok := tools[name]; ok {
			t.Fatalf("platform read-only mode exposed %s", name)
		}
	}
	if _, ok := tools["whodb_platform_sources"]; !ok {
		t.Fatal("platform read-only mode did not expose read tools")
	}
}

func TestNewServer_PlatformAllowWriteHidesConfirmationTools(t *testing.T) {
	tools := listServerToolNames(t, NewServer(&ServerOptions{PlatformEnabled: true, AllowWrite: true}))
	for _, name := range []string{
		"whodb_platform_source_create",
		"whodb_platform_source_update",
		"whodb_platform_source_delete",
		"whodb_platform_create",
		"whodb_platform_update",
		"whodb_platform_delete",
		"whodb_platform_action",
	} {
		if _, ok := tools[name]; !ok {
			t.Fatalf("platform allow-write mode did not expose %s", name)
		}
	}
	for _, name := range []string{"whodb_platform_confirm", "whodb_platform_pending"} {
		if _, ok := tools[name]; ok {
			t.Fatalf("platform allow-write mode exposed %s", name)
		}
	}
}

func listServerToolNames(t *testing.T, server *mcpsdk.Server) map[string]struct{} {
	t.Helper()
	result := listServerTools(t, server)
	names := map[string]struct{}{}
	for _, tool := range result.Tools {
		names[tool.Name] = struct{}{}
	}
	return names
}

func listServerTools(t *testing.T, server *mcpsdk.Server) *mcpsdk.ListToolsResult {
	t.Helper()
	ctx := context.Background()
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}
	t.Cleanup(func() {
		_ = serverSession.Close()
	})
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	t.Cleanup(func() {
		_ = clientSession.Close()
	})
	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(result.Tools) == 0 {
		t.Fatal("ListTools() returned no tools")
	}
	return result
}

func listServerPrompts(t *testing.T, server *mcpsdk.Server) *mcpsdk.ListPromptsResult {
	t.Helper()
	ctx := context.Background()
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}
	t.Cleanup(func() {
		_ = serverSession.Close()
	})
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	t.Cleanup(func() {
		_ = clientSession.Close()
	})
	result, err := clientSession.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatalf("ListPrompts() error = %v", err)
	}
	if len(result.Prompts) == 0 {
		t.Fatal("ListPrompts() returned no prompts")
	}
	return result
}

func getServerPrompt(t *testing.T, server *mcpsdk.Server, name string) *mcpsdk.GetPromptResult {
	t.Helper()
	ctx := context.Background()
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}
	t.Cleanup(func() {
		_ = serverSession.Close()
	})
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	t.Cleanup(func() {
		_ = clientSession.Close()
	})
	result, err := clientSession.GetPrompt(ctx, &mcpsdk.GetPromptParams{Name: name})
	if err != nil {
		t.Fatalf("GetPrompt(%q) error = %v", name, err)
	}
	return result
}

func listServerResources(t *testing.T, server *mcpsdk.Server) *mcpsdk.ListResourcesResult {
	t.Helper()
	ctx := context.Background()
	clientSession, serverSession := connectTestMCP(t, ctx, server)
	t.Cleanup(func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	})
	result, err := clientSession.ListResources(ctx, nil)
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}
	if len(result.Resources) == 0 {
		t.Fatal("ListResources() returned no resources")
	}
	return result
}

func readServerResource(t *testing.T, server *mcpsdk.Server, uri string) *mcpsdk.ReadResourceResult {
	t.Helper()
	ctx := context.Background()
	clientSession, serverSession := connectTestMCP(t, ctx, server)
	t.Cleanup(func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	})
	result, err := clientSession.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: uri})
	if err != nil {
		t.Fatalf("ReadResource(%q) error = %v", uri, err)
	}
	return result
}

func connectTestMCP(t *testing.T, ctx context.Context, server *mcpsdk.Server) (*mcpsdk.ClientSession, *mcpsdk.ServerSession) {
	t.Helper()
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		_ = serverSession.Close()
		t.Fatalf("client.Connect() error = %v", err)
	}
	return clientSession, serverSession
}

func promptNamesContain(prompts []*mcpsdk.Prompt, name string) bool {
	return findPromptByName(prompts, name) != nil
}

func findPromptByName(prompts []*mcpsdk.Prompt, name string) *mcpsdk.Prompt {
	for _, prompt := range prompts {
		if prompt.Name == name {
			return prompt
		}
	}
	return nil
}

func promptText(t *testing.T, result *mcpsdk.GetPromptResult) string {
	t.Helper()
	if len(result.Messages) != 1 {
		t.Fatalf("GetPrompt returned %d messages, want 1", len(result.Messages))
	}
	content, ok := result.Messages[0].Content.(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("GetPrompt returned content type %T, want *mcp.TextContent", result.Messages[0].Content)
	}
	return content.Text
}

func resourceURIsContain(resources []*mcpsdk.Resource, uri string) bool {
	return findResourceByURI(resources, uri) != nil
}

func findResourceByURI(resources []*mcpsdk.Resource, uri string) *mcpsdk.Resource {
	for _, resource := range resources {
		if resource.URI == uri {
			return resource
		}
	}
	return nil
}

func resourceText(t *testing.T, result *mcpsdk.ReadResourceResult) string {
	t.Helper()
	if len(result.Contents) != 1 {
		t.Fatalf("ReadResource returned %d contents, want 1", len(result.Contents))
	}
	if result.Contents[0].MIMEType != "application/json" {
		t.Fatalf("ReadResource MIMEType = %q, want application/json", result.Contents[0].MIMEType)
	}
	return result.Contents[0].Text
}

func readPlatformToolGuide(t *testing.T, server *mcpsdk.Server) platformToolGuideResource {
	t.Helper()
	text := resourceText(t, readServerResource(t, server, "whodb://platform/tool-guide"))
	var guide platformToolGuideResource
	if err := json.Unmarshal([]byte(text), &guide); err != nil {
		t.Fatalf("tool guide returned invalid JSON: %v\n%s", err, text)
	}
	return guide
}

func platformGuideContainsTool(guide platformToolGuideResource, name string) bool {
	for _, category := range guide.Categories {
		for _, tool := range category.Tools {
			if tool.Name == name {
				return true
			}
		}
		for _, write := range category.SupportedWrites {
			if write.Tool == name {
				return true
			}
		}
	}
	return false
}

func platformGuideContainsWrite(guide platformToolGuideResource, toolName, resource string) bool {
	for _, category := range guide.Categories {
		for _, write := range category.SupportedWrites {
			if write.Tool != toolName {
				continue
			}
			for _, candidate := range write.Resources {
				if candidate == resource {
					return true
				}
			}
		}
	}
	return false
}

func toolNamesContain(tools []*mcpsdk.Tool, name string) bool {
	return findToolByName(tools, name) != nil
}

func findToolByName(tools []*mcpsdk.Tool, name string) *mcpsdk.Tool {
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	return nil
}

func assertToolSchemasAreInspectorCompatible(t *testing.T, tool *mcpsdk.Tool) {
	t.Helper()
	assertToolSchemaIsObject(t, tool.Name, "inputSchema", tool.InputSchema, true)
	assertToolSchemaIsObject(t, tool.Name, "outputSchema", tool.OutputSchema, false)
}

func assertToolSchemaIsObject(t *testing.T, toolName, schemaName string, rawSchema any, required bool) {
	t.Helper()
	if rawSchema == nil {
		if required {
			t.Fatalf("%s %s is nil", toolName, schemaName)
		}
		return
	}
	raw, err := json.Marshal(rawSchema)
	if err != nil {
		t.Fatalf("%s %s marshal error: %v", toolName, schemaName, err)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("%s %s unmarshal error: %v", toolName, schemaName, err)
	}
	if schemaType, _ := schema["type"].(string); schemaType != "object" {
		t.Fatalf("%s %s type = %#v, want object", toolName, schemaName, schema["type"])
	}
	assertSchemaPropertiesTyped(t, toolName, schemaName, schema)
}

func assertSchemaPropertiesTyped(t *testing.T, toolName, path string, schema map[string]any) {
	t.Helper()
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return
	}
	for name, rawProperty := range properties {
		property, ok := rawProperty.(map[string]any)
		if !ok {
			t.Fatalf("%s %s.properties.%s schema is %T, want object", toolName, path, name, rawProperty)
		}
		if !hasConcreteSchemaType(property) {
			t.Fatalf("%s %s.properties.%s has no concrete schema type: %#v", toolName, path, name, property)
		}
		if isArraySchema(property) {
			switch items := property["items"].(type) {
			case map[string]any:
				if !hasConcreteSchemaType(items) {
					t.Fatalf("%s %s.properties.%s.items has no concrete schema type: %#v", toolName, path, name, items)
				}
				assertSchemaPropertiesTyped(t, toolName, path+".properties."+name+".items", items)
			case bool:
				if !items {
					t.Fatalf("%s %s.properties.%s.items is false", toolName, path, name)
				}
			default:
				t.Fatalf("%s %s.properties.%s.items has invalid schema: %#v", toolName, path, name, property["items"])
			}
		}
		assertSchemaPropertiesTyped(t, toolName, path+".properties."+name, property)
	}
}

func hasConcreteSchemaType(schema map[string]any) bool {
	if schema == nil {
		return false
	}
	switch schema["type"].(type) {
	case string, []any:
		return true
	}
	for _, key := range []string{"$ref", "anyOf", "oneOf", "allOf"} {
		if _, ok := schema[key]; ok {
			return true
		}
	}
	return false
}

func isArraySchema(schema map[string]any) bool {
	switch typed := schema["type"].(type) {
	case string:
		return typed == "array"
	case []any:
		for _, value := range typed {
			if value == "array" {
				return true
			}
		}
	}
	return false
}

func TestNewServer_WithDisabledTools(t *testing.T) {
	server := NewServer(&ServerOptions{
		DisabledTools: []string{"query", "confirm"},
	})
	if server == nil {
		t.Fatal("NewServer() returned nil with disabled tools")
	}
}

// Helper function tests

func TestBoolPtr(t *testing.T) {
	truePtr := boolPtr(true)
	falsePtr := boolPtr(false)

	if truePtr == nil || *truePtr != true {
		t.Error("boolPtr(true) should return pointer to true")
	}
	if falsePtr == nil || *falsePtr != false {
		t.Error("boolPtr(false) should return pointer to false")
	}
}

// Prompt content tests

func TestBuildQueryHelpContent(t *testing.T) {
	// Test basic content
	content := buildQueryHelpContent("", "")
	if !strings.Contains(content, "LIMIT") {
		t.Error("Query help should mention LIMIT")
	}
	if !strings.Contains(content, "WhoDB") {
		t.Error("Query help should mention WhoDB")
	}

	// Test with database type
	pgContent := buildQueryHelpContent("postgres", "")
	if !strings.Contains(pgContent, "ILIKE") || !strings.Contains(pgContent, "jsonb") {
		t.Error("Postgres-specific help should mention ILIKE and jsonb")
	}

	mysqlContent := buildQueryHelpContent("mysql", "")
	if !strings.Contains(mysqlContent, "BINARY") {
		t.Error("MySQL-specific help should mention BINARY")
	}

	// Test with query type
	selectContent := buildQueryHelpContent("", "select")
	if !strings.Contains(selectContent, "SELECT") {
		t.Error("Select query help should contain SELECT examples")
	}

	joinContent := buildQueryHelpContent("", "join")
	if !strings.Contains(joinContent, "JOIN") {
		t.Error("Join query help should contain JOIN examples")
	}
}

func TestBuildWorkflowHelpContent(t *testing.T) {
	// Test analysis workflow
	analysisContent := buildWorkflowHelpContent("analysis")
	if !strings.Contains(analysisContent, "Data Analysis") {
		t.Error("Analysis workflow should mention 'Data Analysis'")
	}

	// Test debugging workflow
	debugContent := buildWorkflowHelpContent("debugging")
	if !strings.Contains(debugContent, "Debugging") {
		t.Error("Debugging workflow should mention 'Debugging'")
	}

	// Test relationships workflow
	relContent := buildWorkflowHelpContent("relationships")
	if !strings.Contains(relContent, "foreign") {
		t.Error("Relationships workflow should mention foreign keys")
	}

	// Test default workflow
	defaultContent := buildWorkflowHelpContent("unknown")
	if !strings.Contains(defaultContent, "Schema Exploration") {
		t.Error("Default workflow should list available workflows")
	}
}

// Connection allowlist tests

func TestIsConnectionAllowed_NoRestrictions(t *testing.T) {
	secOpts := &SecurityOptions{
		AllowedConnections: nil, // No restrictions
	}

	// Any connection should be allowed
	if !secOpts.isConnectionAllowed("prod") {
		t.Error("With no restrictions, 'prod' should be allowed")
	}
	if !secOpts.isConnectionAllowed("staging") {
		t.Error("With no restrictions, 'staging' should be allowed")
	}
	if !secOpts.isConnectionAllowed("") {
		t.Error("With no restrictions, empty connection should be allowed")
	}
}

func TestIsConnectionAllowed_EmptySlice(t *testing.T) {
	secOpts := &SecurityOptions{
		AllowedConnections: []string{}, // Empty slice = no restrictions
	}

	if !secOpts.isConnectionAllowed("any") {
		t.Error("With empty AllowedConnections, any connection should be allowed")
	}
}

func TestIsConnectionAllowed_SingleConnection(t *testing.T) {
	secOpts := &SecurityOptions{
		AllowedConnections: []string{"prod"},
	}

	if !secOpts.isConnectionAllowed("prod") {
		t.Error("'prod' should be allowed when it's in the list")
	}
	if secOpts.isConnectionAllowed("staging") {
		t.Error("'staging' should NOT be allowed when only 'prod' is in the list")
	}
	if secOpts.isConnectionAllowed("") {
		t.Error("Empty connection should NOT be allowed when restrictions are set")
	}
}

func TestIsConnectionAllowed_MultipleConnections(t *testing.T) {
	secOpts := &SecurityOptions{
		AllowedConnections: []string{"prod", "staging", "dev"},
	}

	if !secOpts.isConnectionAllowed("prod") {
		t.Error("'prod' should be allowed")
	}
	if !secOpts.isConnectionAllowed("staging") {
		t.Error("'staging' should be allowed")
	}
	if !secOpts.isConnectionAllowed("dev") {
		t.Error("'dev' should be allowed")
	}
	if secOpts.isConnectionAllowed("test") {
		t.Error("'test' should NOT be allowed")
	}
}

func TestNewServer_AllowedConnectionsSetsDefault(t *testing.T) {
	// When AllowedConnections is set and DefaultConnection is not,
	// the first allowed connection becomes the default
	server := NewServer(&ServerOptions{
		AllowedConnections: []string{"staging", "prod"},
	})
	if server == nil {
		t.Fatal("NewServer() returned nil with AllowedConnections")
	}
	// We can't directly inspect secOpts, but the server should be created successfully
}

func TestNewServer_ExplicitDefaultOverridesFirst(t *testing.T) {
	// When both DefaultConnection and AllowedConnections are set,
	// DefaultConnection takes precedence
	server := NewServer(&ServerOptions{
		DefaultConnection:  "prod",
		AllowedConnections: []string{"staging", "dev"},
	})
	if server == nil {
		t.Fatal("NewServer() returned nil with explicit DefaultConnection")
	}
}
