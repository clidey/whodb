/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	commonconfig "github.com/clidey/whodb/core/src/common/config"
)

// MCPWorkspace identifies a platform target; policy entries use canonical IDs.
type MCPWorkspace struct {
	Host    string `json:"host,omitempty" jsonschema:"Saved platform host URL. Does not change the default host."`
	Org     string `json:"org,omitempty" jsonschema:"Organization ID, slug, or unambiguous name."`
	Project string `json:"project,omitempty" jsonschema:"Project ID, slug, or unambiguous name. Requires org."`
}

// MCPPlatformPolicy stores optional restrictions for platform tools.
type MCPPlatformPolicy struct {
	AllowedHosts       []string       `json:"allowed_hosts,omitempty"`
	AllowedWorkspaces  []MCPWorkspace `json:"allowed_workspaces,omitempty"`
	ReadOnlyHosts      []string       `json:"read_only_hosts,omitempty"`
	ReadOnlyWorkspaces []MCPWorkspace `json:"read_only_workspaces,omitempty"`
}

// MCPSettings stores credential-free settings for the single MCP server entry.
type MCPSettings struct {
	Modules            []string           `json:"modules"`
	WriteMode          string             `json:"write_mode"`
	EnabledTools       []string           `json:"enabled_tools,omitempty"`
	DisabledTools      []string           `json:"disabled_tools,omitempty"`
	DefaultWorkspace   *MCPWorkspace      `json:"default_workspace,omitempty" jsonschema:"Optional default host and canonical organization/project IDs. Process overrides and explicit tool targets take precedence."`
	PlatformPolicy     *MCPPlatformPolicy `json:"platform_policy,omitempty"`
	DefaultConnection  string             `json:"default_connection,omitempty"`
	AllowedConnections []string           `json:"allowed_connections,omitempty"`
}

// DefaultMCPSettings preserves platform-first startup and write confirmations.
func DefaultMCPSettings() MCPSettings {
	return MCPSettings{Modules: []string{"platform"}, WriteMode: "confirm"}
}

// HasModule reports whether a tool module is enabled.
func (s MCPSettings) HasModule(name string) bool {
	for _, module := range s.Modules {
		if module == name {
			return true
		}
	}
	return false
}

// LoadMCPSettings reads the separate MCP section without loading credentials.
func LoadMCPSettings() (MCPSettings, error) {
	settings := DefaultMCPSettings()
	if _, err := GetConfigDir(); err != nil {
		return settings, err
	}
	err := commonconfig.ReadSection("mcp", &settings, getConfigOptions())
	return settings, err
}

// SaveMCPSettings saves validated MCP settings without rewriting CLI credentials.
func SaveMCPSettings(settings MCPSettings) error {
	return commonconfig.WriteSection("mcp", settings, getConfigOptions())
}

// MergeMCPSettings overlays supplied JSON fields, preserving unspecified settings.
// Arrays replace existing arrays; null clears platform_policy.
func MergeMCPSettings(current MCPSettings, patch []byte) (MCPSettings, error) {
	data, err := json.Marshal(current)
	if err != nil {
		return current, err
	}
	var result MCPSettings
	if err := json.Unmarshal(data, &result); err != nil {
		return current, err
	}
	if trimmed := bytes.TrimSpace(patch); len(trimmed) == 0 || trimmed[0] != '{' {
		return current, fmt.Errorf("setup settings must be a JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(patch))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return current, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return current, fmt.Errorf("setup settings must contain one JSON object")
	}
	return result, nil
}
