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
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"github.com/clidey/whodb/cli/internal/agentmanifest"
	"github.com/clidey/whodb/cli/internal/config"
	dbmgr "github.com/clidey/whodb/cli/internal/database"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func init() { rootCmd.AddCommand(newSetupCommand()) }

func newSetupCommand() *cobra.Command {
	command := &cobra.Command{Use: "setup", Short: "Configure the single WhoDB MCP server", Long: `Configure platform and database-only modules for whodb mcp serve.
Run without a subcommand for the guided wizard. Agents should use inspect,
validate, apply and verify. These subcommands emit JSON and never prompt.
Authentication uses whodb login; never put credentials in setup JSON.`, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return fmt.Errorf("interactive setup requires a terminal; agents should start with whodb setup inspect")
		}
		return runSetupWizard(cmd)
	}}
	command.AddCommand(&cobra.Command{Use: "inspect", Short: "Show saved settings, available targets, schema and setup choices as JSON", Args: cobra.NoArgs, RunE: runSetupInspect})
	for _, action := range []string{"validate", "apply"} {
		action := action
		child := &cobra.Command{Use: action, Short: action + " a JSON settings patch without interactive prompts", Args: cobra.NoArgs, SilenceUsage: true, RunE: func(cmd *cobra.Command, args []string) error { return runSetupPatch(cmd, action) }}
		child.Flags().String("file", "", "Settings patch file, or - for stdin")
		if action == "apply" {
			child.Flags().Bool("yes", false, "Save the proposed settings after user approval")
		}
		command.AddCommand(child)
	}
	command.AddCommand(&cobra.Command{Use: "verify", Short: "Check configured host logins and database connections; emit JSON", Args: cobra.NoArgs, SilenceUsage: true, RunE: runSetupVerify})
	return command
}

type setupHost struct {
	Host    string `json:"host"`
	Email   string `json:"email,omitempty"`
	Org     string `json:"org,omitempty"`
	Project string `json:"project,omitempty"`
}
type setupConnection struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func runSetupInspect(cmd *cobra.Command, args []string) error {
	settings, err := config.LoadMCPSettings()
	if err != nil {
		return err
	}
	cfg, err := config.LoadConfigWithoutSecrets()
	if err != nil {
		return err
	}
	hosts := make([]setupHost, 0, len(cfg.Platform.Hosts))
	for _, host := range cfg.Platform.Hosts {
		hosts = append(hosts, setupHost{host.URL, host.Email, host.DefaultOrgID, host.DefaultProjectID})
	}
	manager, err := dbmgr.NewManagerWithConfig(cfg)
	if err != nil {
		return err
	}
	connections := []setupConnection{}
	for _, conn := range manager.ListConnections() {
		connections = append(connections, setupConnection{conn.Name, conn.Type})
	}
	toolChoices := []string{}
	for _, tool := range agentmanifest.Build().MCPTools {
		toolChoices = append(toolChoices, tool.Name)
	}
	schema, err := jsonschema.For[config.MCPSettings](nil)
	if err != nil {
		return err
	}
	validationError := ""
	if err := validateMCPSettings(&settings); err != nil {
		validationError = err.Error()
	}
	return writeAutomationEnvelope(cmd, "setup.inspect", map[string]any{
		"tool_choices": toolChoices,
		"settings":     settings, "validation_error": validationError, "hosts": hosts, "connections": connections, "settings_schema": schema,
		"module_choices": []string{"platform", "database"}, "write_mode_choices": []string{"confirm", "read-only", "allow"},
		"mcp_entry": map[string]any{"command": "whodb", "args": []string{"mcp", "serve"}},
		"workflow":  []string{"Ask the user which modules, targets and access settings they want.", "Validate a JSON patch with whodb setup validate --file <path>.", "Show the proposed settings and obtain user approval.", "Apply with whodb setup apply --file <path> --yes.", "Use whodb login --host <url> for browser sign-in; do not request credentials in chat.", "Run whodb setup verify, then restart the MCP connection if settings changed."},
	})
}

func readSetupPatch(cmd *cobra.Command) ([]byte, error) {
	file, _ := cmd.Flags().GetString("file")
	if file == "" {
		return nil, fmt.Errorf("--file is required; use --file - for stdin")
	}
	if file == "-" {
		return io.ReadAll(cmd.InOrStdin())
	}
	return os.ReadFile(file)
}

func runSetupPatch(cmd *cobra.Command, action string) error {
	current, err := config.LoadMCPSettings()
	if err != nil {
		return err
	}
	data, err := readSetupPatch(cmd)
	if err != nil {
		return err
	}
	proposed, err := config.MergeMCPSettings(current, data)
	if err == nil {
		err = validateMCPSettings(&proposed)
	}
	if err != nil {
		if writeErr := writeCommandJSON(cmd, map[string]any{"command": "setup." + action, "success": false, "error": err.Error()}); writeErr != nil {
			return writeErr
		}
		return err
	}
	before, _ := json.Marshal(current)
	after, _ := json.Marshal(proposed)
	changed := string(before) != string(after)
	applied := false
	if action == "apply" {
		approved, _ := cmd.Flags().GetBool("yes")
		if !approved {
			return fmt.Errorf("review with setup validate, then pass --yes after user approval")
		}
		if err := config.SaveMCPSettings(proposed); err != nil {
			return err
		}
		applied = true
	}
	return writeAutomationEnvelope(cmd, "setup."+action, map[string]any{"settings": proposed, "previous_settings": current, "valid": true, "changed": changed, "applied": applied, "restart_required": changed, "restart_instruction": "Restart the MCP connection after applying changed settings; keep mcp.json at whodb mcp serve."})
}

type setupCheck struct {
	Module      string   `json:"module"`
	Target      string   `json:"target"`
	OK          bool     `json:"ok"`
	Error       string   `json:"error,omitempty"`
	NextCommand []string `json:"next_command,omitempty"`
}

func runSetupVerify(cmd *cobra.Command, args []string) error {
	settings, err := config.LoadMCPSettings()
	if err != nil {
		return err
	}
	if err := validateMCPSettings(&settings); err != nil {
		return err
	}
	cfg, err := config.LoadConfigWithoutSecrets()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()
	checks := []setupCheck{}
	if settings.HasModule("platform") {
		hosts := []string{}
		for _, host := range cfg.Platform.Hosts {
			hosts = append(hosts, host.URL)
		}
		if p := settings.PlatformPolicy; p != nil {
			if len(p.AllowedHosts) > 0 {
				hosts = append([]string(nil), p.AllowedHosts...)
			}
			if len(p.AllowedWorkspaces) > 0 {
				hosts = nil
				for _, target := range p.AllowedWorkspaces {
					if (len(p.AllowedHosts) == 0 || slices.Contains(p.AllowedHosts, target.Host)) && !slices.Contains(hosts, target.Host) {
						hosts = append(hosts, target.Host)
					}
				}
			}
		}
		if target := settings.DefaultWorkspace; target != nil && !slices.Contains(hosts, target.Host) {
			hosts = append(hosts, target.Host)
		}
		if len(hosts) == 0 {
			checks = append(checks, setupCheck{Module: "platform", Error: "No host logins configured", NextCommand: []string{"whodb", "login"}})
		}
		for _, host := range hosts {
			check := setupCheck{Module: "platform", Target: host, NextCommand: []string{"whodb", "login", "--host", host}}
			entry, ok := cfg.GetPlatformHost(host)
			if !ok || entry.AccountID == "" {
				check.Error = "Sign-in required"
			} else {
				client, clientErr := platformapi.NewAuthenticatedClient(host, platformapi.NewOIDCTokenSource(host, entry.AccountID, cfg))
				if clientErr == nil {
					_, clientErr = client.Me(ctx)
				}
				if clientErr != nil {
					check.Error = clientErr.Error()
				} else {
					check.OK = true
					check.NextCommand = nil
					targets := []config.MCPWorkspace{}
					if p := settings.PlatformPolicy; p != nil {
						targets = append(targets, p.AllowedWorkspaces...)
						targets = append(targets, p.ReadOnlyWorkspaces...)
					}
					if settings.DefaultWorkspace != nil {
						targets = append(targets, *settings.DefaultWorkspace)
					}
					checked := map[config.MCPWorkspace]bool{}
					for _, target := range targets {
						if target.Host != host || target.Org == "" || checked[target] {
							continue
						}
						checked[target] = true
						workspaceCheck := setupCheck{Module: "platform", Target: host + "|" + target.Org + "|" + target.Project}
						if err := verifySetupWorkspace(ctx, client, target); err != nil {
							workspaceCheck.Error = err.Error()
						} else {
							workspaceCheck.OK = true
						}
						checks = append(checks, workspaceCheck)
					}
				}
			}
			checks = append(checks, check)
		}
	}
	if settings.HasModule("database") {
		manager, err := dbmgr.NewManager()
		if err != nil {
			return err
		}
		names := settings.AllowedConnections
		if len(names) == 0 {
			for _, conn := range manager.ListConnections() {
				names = append(names, conn.Name)
			}
		}
		if settings.DefaultConnection != "" && !slices.Contains(names, settings.DefaultConnection) {
			names = append(names, settings.DefaultConnection)
		}
		if len(names) == 0 {
			checks = append(checks, setupCheck{Module: "database", Error: "No database connections configured", NextCommand: []string{"whodb", "connections", "add", "--help"}})
		}
		for _, name := range names {
			check := setupCheck{Module: "database", Target: name, NextCommand: []string{"whodb", "connections", "test", name}}
			conn, _, connectErr := manager.ResolveConnection(name)
			if connectErr == nil {
				connectErr = manager.Connect(conn)
			}
			if connectErr == nil {
				check.OK = true
				check.NextCommand = nil
				if err := manager.Disconnect(); err != nil {
					check.OK = false
					check.Error = err.Error()
				}
			} else {
				check.Error = connectErr.Error()
			}
			checks = append(checks, check)
		}
	}
	ok := true
	for _, check := range checks {
		ok = ok && check.OK
	}
	if err := writeCommandJSON(cmd, map[string]any{"command": "setup.verify", "success": ok, "checks": checks}); err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("some setup checks need attention")
	}
	return nil
}

type setupWorkspaceClient interface {
	Organizations(context.Context) ([]platformapi.Organization, error)
	Projects(context.Context, string) ([]platformapi.Project, error)
}

func verifySetupWorkspace(ctx context.Context, client setupWorkspaceClient, target config.MCPWorkspace) error {
	orgs, err := client.Organizations(ctx)
	if err != nil {
		return err
	}
	found := false
	for _, org := range orgs {
		if org.ID == target.Org {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("configured organization ID is not accessible")
	}
	projects, err := client.Projects(ctx, target.Org)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if project.ID == target.Project {
			return nil
		}
	}
	return fmt.Errorf("configured project ID is not accessible")
}
