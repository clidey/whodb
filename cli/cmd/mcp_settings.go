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
	"fmt"
	"slices"
	"strings"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	whodbmcp "github.com/clidey/whodb/cli/pkg/mcp"
	"github.com/spf13/cobra"
)

var databaseMCPTools = []string{"query", "schemas", "tables", "columns", "connections", "confirm", "pending", "explain", "diff", "erd", "audit", "suggestions"}

func validateMCPSettings(settings *config.MCPSettings) error {
	if len(settings.Modules) == 0 {
		return fmt.Errorf("enable at least one module: platform or database")
	}
	seen := map[string]bool{}
	for _, module := range settings.Modules {
		if (module != "platform" && module != "database") || seen[module] {
			return fmt.Errorf("invalid or duplicate MCP module %q", module)
		}
		seen[module] = true
	}
	if !slices.Contains([]string{"confirm", "read-only", "allow"}, settings.WriteMode) {
		return fmt.Errorf("write_mode must be confirm, read-only or allow")
	}
	for _, list := range [][]string{settings.EnabledTools, settings.DisabledTools} {
		for _, tool := range list {
			if strings.HasPrefix(tool, "whodb_platform_") {
				if err := whodbmcp.ValidatePlatformTools([]string{tool}, nil); err != nil {
					return err
				}
			} else if !slices.Contains(databaseMCPTools, strings.TrimPrefix(tool, "whodb_")) {
				return fmt.Errorf("unknown MCP tool %q; platform tools use full whodb_platform_* names", tool)
			}
		}
	}
	if settings.DefaultConnection != "" && len(settings.AllowedConnections) > 0 && !slices.Contains(settings.AllowedConnections, settings.DefaultConnection) {
		return fmt.Errorf("default_connection must be in allowed_connections")
	}
	if target := settings.DefaultWorkspace; target != nil {
		if target.Host == "" {
			return fmt.Errorf("default_workspace requires a host")
		}
		host, err := platformapi.NormalizeHost(target.Host)
		if err != nil {
			return err
		}
		target.Host = host
		if err := (platformapi.SessionScope{Host: target.Host, Org: target.Org, Project: target.Project}).Validate(); err != nil {
			return err
		}
	}
	if err := whodbmcp.ValidatePlatformPolicy((*whodbmcp.PlatformPolicy)(settings.PlatformPolicy)); err != nil {
		return err
	}
	if target, p := settings.DefaultWorkspace, settings.PlatformPolicy; target != nil && p != nil {
		if len(p.AllowedHosts) > 0 && !slices.Contains(p.AllowedHosts, target.Host) {
			return fmt.Errorf("default_workspace host is not allowed by platform_policy")
		}
		if len(p.AllowedWorkspaces) > 0 && !slices.Contains(p.AllowedWorkspaces, *target) {
			return fmt.Errorf("default_workspace must match an allowed workspace using canonical IDs")
		}
	}
	return nil
}

func effectiveMCPSettings(cmd *cobra.Command) (config.MCPSettings, error) {
	settings, err := config.LoadMCPSettings()
	if err != nil {
		return settings, err
	}
	if cmd.Flags().Changed("platform") && mcpDatabase {
		return settings, fmt.Errorf("--platform and --database cannot be used together")
	}
	if cmd.Flags().Changed("platform") && !mcpPlatform {
		return settings, fmt.Errorf("use --database instead of --platform=false")
	}
	if mcpDatabase {
		settings.Modules = []string{"database"}
	} else if cmd.Flags().Changed("platform") {
		settings.Modules = []string{"platform"}
	}
	if cmd.Flags().Changed("tools") {
		settings.EnabledTools = mcpEnabledTools
	}
	if cmd.Flags().Changed("disable-tools") {
		settings.DisabledTools = mcpDisabledTools
	}
	if cmd.Flags().Changed("default-connection") {
		settings.DefaultConnection = mcpConnection
	}
	if cmd.Flags().Changed("allowed-connections") {
		settings.AllowedConnections = mcpAllowedConnections
	}
	if mcpPlatformPolicy != "" {
		if !settings.HasModule("platform") {
			return settings, fmt.Errorf("--platform-policy requires platform mode")
		}
		policy, err := whodbmcp.LoadPlatformPolicy(mcpPlatformPolicy)
		if err != nil {
			return settings, err
		}
		settings.PlatformPolicy = (*config.MCPPlatformPolicy)(policy)
	}
	if mcpSafeMode || (cmd.Flags().Changed("read-only") && mcpReadOnly) {
		settings.WriteMode = "read-only"
	} else if cmd.Flags().Changed("allow-write") && mcpAllowWrite {
		settings.WriteMode = "allow"
	} else if cmd.Flags().Changed("confirm-writes") && mcpConfirmWrites {
		settings.WriteMode = "confirm"
	}
	if !settings.HasModule("database") {
		for _, flag := range []string{"default-connection", "allowed-connections", "allow-drop", "allow-multi-statement"} {
			if cmd.Flags().Changed(flag) {
				return settings, fmt.Errorf("--%s requires --database or the saved database module", flag)
			}
		}
	}
	for _, flag := range []string{"tools", "disable-tools"} {
		if !cmd.Flags().Changed(flag) {
			continue
		}
		values, _ := cmd.Flags().GetStringSlice(flag)
		for _, tool := range values {
			platform := strings.HasPrefix(tool, "whodb_platform_")
			if (platform && !settings.HasModule("platform")) || (!platform && !settings.HasModule("database")) {
				return settings, fmt.Errorf("tool %q belongs to a disabled module; configure modules with whodb setup (database module requires --database or saved setup)", tool)
			}
		}
	}
	return settings, validateMCPSettings(&settings)
}
