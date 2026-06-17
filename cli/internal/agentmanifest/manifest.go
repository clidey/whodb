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
		{Name: "whodb_platform_sources", Description: "List hosted WhoDB sources when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_source_objects", Description: "Browse hosted source objects when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_source_columns", Description: "Inspect hosted source object columns when MCP starts with --platform.", ReadOnly: true},
		{Name: "whodb_platform_source_rows", Description: "Preview hosted source object rows when MCP starts with --platform.", ReadOnly: true},
	}
	return tools
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
