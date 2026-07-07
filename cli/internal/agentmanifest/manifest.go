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

// Package agentmanifest builds the machine-readable WhoDB capability manifest
// used by agent-facing CLI and MCP surfaces.
package agentmanifest

import (
	"slices"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/clidey/whodb/cli/internal/sourcetypes"
	"github.com/clidey/whodb/cli/pkg/version"
	"github.com/clidey/whodb/core/src/source"
)

// Manifest describes WhoDB's agent-facing command, source, and MCP surface.
type Manifest struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	SourceTypes []SourceType      `json:"source_types"`
	Commands    []Command         `json:"commands"`
	MCPTools    []MCPTool         `json:"mcp_tools"`
	PlatformMCP PlatformMCP       `json:"platform_mcp"`
	SafetyModes []SafetyMode      `json:"safety_modes"`
	Workflows   []WorkflowSummary `json:"workflows"`
}

// SourceType describes one source type in a compact agent-readable form.
type SourceType struct {
	ID               string            `json:"id"`
	Label            string            `json:"label"`
	Category         string            `json:"category"`
	Model            string            `json:"model"`
	Transport        string            `json:"transport"`
	SchemaFidelity   string            `json:"schema_fidelity,omitempty"`
	Surfaces         []string          `json:"surfaces"`
	BrowsePath       []string          `json:"browse_path"`
	ConnectionFields []ConnectionField `json:"connection_fields"`
	ExplainMode      string            `json:"explain_mode,omitempty"`
	SupportsAnalyze  bool              `json:"supports_analyze,omitempty"`
}

// ConnectionField describes one connection field without exposing values.
type ConnectionField struct {
	Key      string `json:"key"`
	Kind     string `json:"kind"`
	Section  string `json:"section"`
	Required bool   `json:"required"`
}

// Command describes a programmatic CLI command that agents can call.
type Command struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	Formats             []string `json:"formats"`
	RequiresConnection  bool     `json:"requires_connection,omitempty"`
	RequiresConnections int      `json:"requires_connections,omitempty"`
}

// MCPTool describes one MCP tool exposed by the WhoDB MCP server.
type MCPTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ReadOnly    bool   `json:"read_only"`
}

// MCPPrompt describes one MCP prompt exposed by the WhoDB MCP server.
type MCPPrompt struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// MCPResource describes one MCP resource exposed by the WhoDB MCP server.
type MCPResource struct {
	URI         string `json:"uri"`
	Description string `json:"description"`
	MIMEType    string `json:"mime_type"`
}

// PlatformMCP describes hosted-platform MCP mode for agent consumers.
type PlatformMCP struct {
	EnabledByFlag           string           `json:"enabled_by_flag"`
	RequiresLogin           bool             `json:"requires_login"`
	RequiresWorkspace       bool             `json:"requires_workspace"`
	DefaultHost             string           `json:"default_host"`
	ToolPrefix              string           `json:"tool_prefix"`
	LocalToolsExposed       bool             `json:"local_tools_exposed"`
	SupportsFields          bool             `json:"supports_fields_projection"`
	LocalToolSelectionFlags []string         `json:"local_tool_selection_flags"`
	SetupCommands           []string         `json:"setup_commands"`
	WriteModes              PlatformMCPModes `json:"write_modes"`
	Prompts                 []MCPPrompt      `json:"prompts"`
	Resources               []MCPResource    `json:"resources"`
}

// PlatformMCPModes describes hosted-platform write behavior by MCP mode.
type PlatformMCPModes struct {
	Default    string `json:"default"`
	ReadOnly   string `json:"read_only"`
	SafeMode   string `json:"safe_mode"`
	AllowWrite string `json:"allow_write"`
}

// SafetyMode describes an MCP query-execution safety mode.
type SafetyMode struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// WorkflowSummary describes an agent-friendly workflow exposed by the CLI.
type WorkflowSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Build returns the current WhoDB capability manifest.
func Build() Manifest {
	return Manifest{
		Name:        "whodb",
		Version:     version.Version,
		SourceTypes: buildSourceTypes(),
		Commands:    buildCommands(),
		MCPTools:    buildMCPTools(),
		PlatformMCP: buildPlatformMCP(),
		SafetyModes: buildSafetyModes(),
		Workflows:   buildWorkflows(),
	}
}

func buildSourceTypes() []SourceType {
	ids := sourcetypes.IDs()
	result := make([]SourceType, 0, len(ids))
	for _, id := range ids {
		spec, ok := sourcetypes.Find(id)
		if !ok {
			continue
		}
		result = append(result, SourceType{
			ID:               spec.ID,
			Label:            spec.Label,
			Category:         string(spec.Category),
			Model:            string(spec.Contract.Model),
			Transport:        string(spec.Traits.Connection.Transport),
			SchemaFidelity:   string(spec.Traits.Presentation.SchemaFidelity),
			Surfaces:         surfaces(spec.Contract.Surfaces),
			BrowsePath:       browsePath(spec.Contract.BrowsePath),
			ConnectionFields: connectionFields(spec.ConnectionFields),
			ExplainMode:      string(spec.Traits.Query.ExplainMode),
			SupportsAnalyze:  spec.Traits.Query.SupportsAnalyze,
		})
	}
	return result
}

func connectionFields(fields []source.ConnectionField) []ConnectionField {
	result := make([]ConnectionField, 0, len(fields))
	for _, field := range fields {
		result = append(result, ConnectionField{
			Key:      field.Key,
			Kind:     string(field.Kind),
			Section:  string(field.Section),
			Required: field.Required,
		})
	}
	return result
}

func surfaces(values []source.Surface) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

func browsePath(values []source.ObjectKind) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

func buildCommands() []Command {
	commands := []Command{
		{Name: "agent schema", Description: "Emit WhoDB's machine-readable agent capability manifest.", Formats: []string{"json"}},
		{Name: "doctor", Description: "Run connection, schema, and metadata diagnostics for one database connection.", Formats: []string{"table", "json"}, RequiresConnection: true},
		{Name: "runbooks list", Description: "List built-in database workflows.", Formats: []string{"table", "json"}},
		{Name: "runbooks describe", Description: "Describe one built-in database workflow.", Formats: []string{"table", "json"}},
		{Name: "runbooks run", Description: "Run one built-in database workflow.", Formats: []string{"table", "json"}},
		{Name: "skills list", Description: "List bundled WhoDB skills and agents.", Formats: []string{"table", "json"}},
		{Name: "skills install", Description: "Install bundled WhoDB skills or MCP integrations into local assistant configuration.", Formats: []string{"table", "json"}},
		{Name: "query", Description: "Execute a SQL query.", Formats: []string{"plain", "json", "ndjson", "csv"}, RequiresConnection: true},
		{Name: "schemas", Description: "List schemas for a connection.", Formats: []string{"plain", "json", "csv"}, RequiresConnection: true},
		{Name: "tables", Description: "List tables or storage units in a schema.", Formats: []string{"plain", "json", "csv"}, RequiresConnection: true},
		{Name: "columns", Description: "Describe columns for one table or storage unit.", Formats: []string{"plain", "json", "csv"}, RequiresConnection: true},
		{Name: "connections list", Description: "List saved and environment-defined connections.", Formats: []string{"table", "plain", "json", "csv"}},
		{Name: "connections test", Description: "Test one connection.", Formats: []string{"table", "json"}, RequiresConnection: true},
		{Name: "diff", Description: "Compare schema metadata between two connections.", Formats: []string{"table", "json"}, RequiresConnections: 2},
		{Name: "erd", Description: "Render graph metadata used by the ERD view.", Formats: []string{"text", "json"}, RequiresConnection: true},
		{Name: "audit", Description: "Run data-quality checks on one schema or table.", Formats: []string{"table", "json"}, RequiresConnection: true},
		{Name: "explain", Description: "Run database-native EXPLAIN for a query.", Formats: []string{"table", "plain", "json", "ndjson", "csv"}, RequiresConnection: true},
		{Name: "suggestions", Description: "Load backend-generated starter queries.", Formats: []string{"table", "json"}, RequiresConnection: true},
		{Name: "export", Description: "Export table or query data.", Formats: []string{"csv"}, RequiresConnection: true},
	}
	return commands
}

func buildMCPTools() []MCPTool {
	tools := []MCPTool{
		{Name: "whodb_connections", Description: "List available database connections.", ReadOnly: true},
		{Name: "whodb_schemas", Description: "List database schemas.", ReadOnly: true},
		{Name: "whodb_tables", Description: "List tables in a schema.", ReadOnly: true},
		{Name: "whodb_columns", Description: "Describe table columns.", ReadOnly: true},
		{Name: "whodb_query", Description: "Execute security-validated SQL queries.", ReadOnly: false},
		{Name: "whodb_confirm", Description: "Confirm pending write operations.", ReadOnly: false},
		{Name: "whodb_pending", Description: "List pending write confirmations.", ReadOnly: true},
		{Name: "whodb_explain", Description: "Run EXPLAIN for a query.", ReadOnly: true},
		{Name: "whodb_diff", Description: "Compare schema metadata between two connections.", ReadOnly: true},
		{Name: "whodb_erd", Description: "Load graph and relationship metadata.", ReadOnly: true},
		{Name: "whodb_audit", Description: "Run data-quality audits.", ReadOnly: true},
		{Name: "whodb_suggestions", Description: "Load backend query suggestions.", ReadOnly: true},
		{Name: "whodb_platform_status", Description: "Show hosted WhoDB login and selected workspace when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_orgs", Description: "List hosted WhoDB organizations visible to the signed-in user when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_projects", Description: "List hosted WhoDB projects for an organization when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_sources", Description: "List hosted WhoDB sources in the selected project when MCP starts with --platform; accepts fields for projection.", ReadOnly: true},
		{Name: "whodb_platform_source_types", Description: "List hosted source types available for creation when MCP starts with --platform; accepts fields for projection.", ReadOnly: true},
		{Name: "whodb_platform_source_fields", Description: "List connection fields for one hosted source type when MCP starts with --platform; accepts fields for projection.", ReadOnly: true},
		{Name: "whodb_platform_source_objects", Description: "Browse hosted source objects when MCP starts with --platform; accepts fields for projection.", ReadOnly: true},
		{Name: "whodb_platform_source_columns", Description: "Inspect hosted source object columns when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_source_rows", Description: "Preview hosted source object rows when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_source_constraints", Description: "Inspect editable hosted source field constraints when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_source_content", Description: "Read hosted source object content when supported and MCP starts with --platform; request content fields only when needed.", ReadOnly: true},
		{Name: "whodb_platform_source_config", Description: "Inspect redacted hosted source config when MCP starts with --platform; secrets are masked.", ReadOnly: true},
		{Name: "whodb_platform_source_test", Description: "Test saved or draft hosted source connections when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_secrets", Description: "List hosted secret metadata without secret values when MCP starts with --platform; accepts fields for projection.", ReadOnly: true},
		{Name: "whodb_platform_ai_providers", Description: "List hosted AI provider metadata without API keys when MCP starts with --platform; accepts fields for projection.", ReadOnly: true},
		{Name: "whodb_platform_ai_provider_models", Description: "List models for one hosted AI provider when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_ontologies", Description: "List hosted ontology object types when MCP starts with --platform; accepts fields for projection.", ReadOnly: true},
		{Name: "whodb_platform_ontology", Description: "Inspect one hosted ontology object type when MCP starts with --platform; accepts fields for projection.", ReadOnly: true},
		{Name: "whodb_platform_ontology_fast_lookups", Description: "List saved fast lookups for one ontology when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_ontology_fast_lookup_suggestions", Description: "List suggested fast lookups for one ontology when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_ontology_rows", Description: "Preview rows for one hosted ontology when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_ontology_follow_link", Description: "Follow one hosted ontology link from a row when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_datasets", Description: "List hosted datasets when MCP starts with --platform; accepts fields for projection.", ReadOnly: true},
		{Name: "whodb_platform_dataset", Description: "Inspect one hosted dataset when MCP starts with --platform; accepts fields for projection.", ReadOnly: true},
		{Name: "whodb_platform_dataset_rows", Description: "Preview hosted dataset rows when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_lineage", Description: "Inspect hosted lineage around one root node when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_lineage_neighbors", Description: "Inspect immediate hosted lineage neighbors when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_project_lineage", Description: "Inspect hosted project-level lineage when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_transforms", Description: "List hosted transforms when MCP starts with --platform; accepts fields for projection.", ReadOnly: true},
		{Name: "whodb_platform_transform_runs", Description: "List recent runs for one hosted transform when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_functions", Description: "List hosted ontology functions when MCP starts with --platform; accepts fields for projection.", ReadOnly: true},
		{Name: "whodb_platform_function", Description: "Inspect one hosted ontology function when MCP starts with --platform; request file content fields only when needed.", ReadOnly: true},
		{Name: "whodb_platform_files", Description: "List hosted project folders and files when MCP starts with --platform; accepts fields for projection.", ReadOnly: true},
		{Name: "whodb_platform_file_preview", Description: "Preview one hosted project file when MCP starts with --platform; request text or tabular payload fields only when needed.", ReadOnly: true},
		{Name: "whodb_platform_file_inspect", Description: "Inspect hosted tabular file columns and inferred promote-to-dataset mappings when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_file_search", Description: "Search hosted project files when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_tabular_files", Description: "List tabular hosted project files when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_storage_usage", Description: "Show hosted project storage usage in bytes when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_project_create", Description: "Create hosted projects when MCP starts with --platform; returns a confirmation token by default, executes immediately with --allow-write, and is hidden in --read-only and --safe-mode.", ReadOnly: false},
		{Name: "whodb_platform_project_rename", Description: "Rename hosted projects when MCP starts with --platform; returns a confirmation token by default, executes immediately with --allow-write, and is hidden in --read-only and --safe-mode.", ReadOnly: false},
		{Name: "whodb_platform_project_delete", Description: "Delete hosted projects when MCP starts with --platform; returns a confirmation token by default, executes immediately with --allow-write, and is hidden in --read-only and --safe-mode.", ReadOnly: false},
		{Name: "whodb_platform_source_create", Description: "Create hosted sources when MCP starts with --platform; returns a confirmation token by default, executes immediately with --allow-write, and is hidden in --read-only and --safe-mode.", ReadOnly: false},
		{Name: "whodb_platform_source_update", Description: "Update hosted sources when MCP starts with --platform; returns a confirmation token by default, executes immediately with --allow-write, and is hidden in --read-only and --safe-mode.", ReadOnly: false},
		{Name: "whodb_platform_source_delete", Description: "Delete hosted sources when MCP starts with --platform; returns a confirmation token by default, executes immediately with --allow-write, and is hidden in --read-only and --safe-mode.", ReadOnly: false},
		{Name: "whodb_platform_create", Description: "Create hosted resources when MCP starts with --platform; returns a confirmation token by default, executes immediately with --allow-write, and is hidden in --read-only and --safe-mode.", ReadOnly: false},
		{Name: "whodb_platform_update", Description: "Update hosted resources when MCP starts with --platform; returns a confirmation token by default, executes immediately with --allow-write, and is hidden in --read-only and --safe-mode.", ReadOnly: false},
		{Name: "whodb_platform_delete", Description: "Delete hosted resources when MCP starts with --platform; returns a confirmation token by default, executes immediately with --allow-write, and is hidden in --read-only and --safe-mode.", ReadOnly: false},
		{Name: "whodb_platform_action", Description: "Run hosted actions such as upload, move, run, or deploy when MCP starts with --platform; returns a confirmation token by default, executes immediately with --allow-write, and is hidden in --read-only and --safe-mode.", ReadOnly: false},
		{Name: "whodb_platform_create_dataset", Description: "Create hosted datasets with typed input when MCP starts with --platform; uses the same confirmation behavior as generic writes.", ReadOnly: false},
		{Name: "whodb_platform_promote_file_to_dataset", Description: "Promote hosted files to datasets with typed input when MCP starts with --platform; use file inspection first for column mappings.", ReadOnly: false},
		{Name: "whodb_platform_add_ontology_record", Description: "Add hosted ontology records with typed input when MCP starts with --platform; uses the same confirmation behavior as generic writes.", ReadOnly: false},
		{Name: "whodb_platform_update_ontology_record", Description: "Update hosted ontology records with typed input when MCP starts with --platform; uses the same confirmation behavior as generic writes.", ReadOnly: false},
		{Name: "whodb_platform_delete_ontology_record", Description: "Delete hosted ontology records with typed input when MCP starts with --platform; uses the same confirmation behavior as generic writes.", ReadOnly: false},
		{Name: "whodb_platform_create_ontology_fast_lookup", Description: "Create hosted ontology fast lookups with typed input when MCP starts with --platform; uses the same confirmation behavior as generic writes.", ReadOnly: false},
		{Name: "whodb_platform_delete_ontology_fast_lookup", Description: "Delete hosted ontology fast lookups with typed input when MCP starts with --platform; uses the same confirmation behavior as generic writes.", ReadOnly: false},
		{Name: "whodb_platform_pending", Description: "List pending hosted platform confirmations when MCP starts with --platform; preview metadata never includes credential or secret values, and the tool is not exposed with --allow-write, --read-only, or --safe-mode.", ReadOnly: true},
		{Name: "whodb_platform_confirm", Description: "Confirm pending hosted platform writes when MCP starts with --platform after user approval; not exposed with --allow-write, --read-only, or --safe-mode.", ReadOnly: false},
		{Name: "whodb_platform_doctor", Description: "Check hosted platform MCP readiness, including login, selected workspace, and manifest availability.", ReadOnly: true},
		{Name: "whodb_platform_bundle_export", Description: "Export selected hosted project metadata as a portable bundle without secret values; uploaded file content is opt-in and capped.", ReadOnly: true},
		{Name: "whodb_platform_bundle_diff", Description: "Compare a project bundle against the selected hosted project and return create or skip actions.", ReadOnly: true},
		{Name: "whodb_platform_bundle_import_plan", Description: "Plan a bundle import into the selected hosted project without executing writes.", ReadOnly: true},
		{Name: "whodb_platform_bundle_import", Description: "Import a project bundle into the selected hosted project; returns a confirmation token by default and is hidden in --read-only and --safe-mode.", ReadOnly: false},
		{Name: "whodb_platform_clone", Description: "Clone a hosted dataset, ontology, transform, or function; returns a confirmation token by default and is hidden in --read-only and --safe-mode.", ReadOnly: false},
	}
	return tools
}

func buildPlatformMCP() PlatformMCP {
	return PlatformMCP{
		EnabledByFlag:           "--platform",
		RequiresLogin:           true,
		RequiresWorkspace:       true,
		DefaultHost:             platformapi.DefaultHost,
		ToolPrefix:              "whodb_platform_",
		LocalToolsExposed:       false,
		SupportsFields:          true,
		LocalToolSelectionFlags: []string{"--tools", "--disable-tools"},
		SetupCommands: []string{
			"whodb-cli login",
			"whodb-cli use --org <org> --project <project>",
			"whodb-cli mcp serve --platform",
		},
		WriteModes: PlatformMCPModes{
			Default:    "confirmation_required",
			ReadOnly:   "write_tools_hidden",
			SafeMode:   "write_tools_hidden",
			AllowWrite: "executes_immediately",
		},
		Prompts: []MCPPrompt{
			{Name: "whodb_platform_overview", Description: "Understand hosted WhoDB platform MCP mode, workspace selection, permissions, and field projection."},
			{Name: "whodb_platform_read_workflow", Description: "Use hosted WhoDB platform read tools safely and efficiently."},
			{Name: "whodb_platform_write_safety", Description: "Handle hosted WhoDB platform write confirmations and destructive actions safely."},
			{Name: "whodb_platform_source_workflow", Description: "Manage hosted WhoDB platform sources from discovery through create, update, and delete."},
		},
		Resources: []MCPResource{
			{URI: "whodb://platform/schema", Description: "Machine-readable hosted WhoDB platform MCP contract and enabled platform tools", MIMEType: "application/json"},
			{URI: "whodb://platform/workspace", Description: "Current hosted WhoDB login and selected workspace metadata", MIMEType: "application/json"},
			{URI: "whodb://platform/tool-guide", Description: "Hosted WhoDB platform MCP tool categories, read/write behavior, and field projection guidance", MIMEType: "application/json"},
		},
	}
}

func buildSafetyModes() []SafetyMode {
	return []SafetyMode{
		{Name: "confirm-writes", Description: "Write operations require confirmation before execution."},
		{Name: "safe-mode", Description: "Read-only mode with strict validation."},
		{Name: "read-only", Description: "Blocks all write operations."},
		{Name: "allow-write", Description: "Allows write operations without confirmation."},
	}
}

func buildWorkflows() []WorkflowSummary {
	workflows := []WorkflowSummary{
		{Name: "connection-doctor", Description: "Resolve and test one connection, then inspect basic metadata."},
		{Name: "schema-audit", Description: "Inspect storage units and run data-quality checks for one schema."},
		{Name: "schema-diff", Description: "Compare schema metadata between two connections."},
	}
	slices.SortFunc(workflows, func(a, b WorkflowSummary) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return workflows
}
