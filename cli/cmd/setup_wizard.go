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
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/connectionopts"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type setupPrompter struct {
	reader *bufio.Reader
	out    io.Writer
}

func (p setupPrompter) ask(label, current string) (string, error) {
	fmt.Fprintf(p.out, "%s [%s]: ", label, current)
	line, err := p.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("setup cancelled before saving: %w", err)
	}
	value := strings.TrimSpace(line)
	if value == "" {
		value = current
	}
	return value, nil
}
func (p setupPrompter) yes(label string) (bool, error) {
	value, err := p.ask(label, "no")
	if err != nil {
		return false, err
	}
	switch strings.ToLower(value) {
	case "yes", "y":
		return true, nil
	case "no", "n":
		return false, nil
	default:
		return false, fmt.Errorf("answer yes or no")
	}
}
func setupList(values []string) string {
	if len(values) == 0 {
		return "*"
	}
	return strings.Join(values, ",")
}
func parseSetupList(value string) []string {
	if value == "*" {
		return nil
	}
	items := strings.Split(value, ",")
	for i := range items {
		items[i] = strings.TrimSpace(items[i])
	}
	return items
}
func (p setupPrompter) list(label string, current []string) ([]string, error) {
	value, err := p.ask(label+" (comma separated; * for unrestricted)", setupList(current))
	if err != nil {
		return nil, err
	}
	return parseSetupList(value), nil
}

func runSetupWizard(cmd *cobra.Command) error {
	current, err := config.LoadMCPSettings()
	if err != nil {
		return err
	}
	raw, _ := json.Marshal(current)
	proposed, err := config.MergeMCPSettings(current, raw)
	if err != nil {
		return err
	}
	cfg, err := config.LoadConfigWithoutSecrets()
	if err != nil {
		return err
	}
	p := setupPrompter{bufio.NewReader(cmd.InOrStdin()), cmd.ErrOrStderr()}
	fmt.Fprintln(p.out, "WhoDB setup — one MCP entry: whodb mcp serve\nPress Enter to keep a setting. Credentials stay out of setup JSON.")
	modules, err := p.ask("Modules: platform, database, or both", strings.Join(current.Modules, ","))
	if err != nil {
		return err
	}
	if modules == "both" {
		modules = "platform,database"
	}
	proposed.Modules = parseSetupList(modules)
	proposed.WriteMode, err = p.ask("Write access: confirm, read-only, or allow", current.WriteMode)
	if err != nil {
		return err
	}
	loginHosts := []string{}
	if proposed.HasModule("platform") {
		fmt.Fprintln(p.out, "Saved platform hosts:")
		for _, host := range cfg.Platform.Hosts {
			fmt.Fprintf(p.out, "  %s (%s)\n", host.URL, host.Email)
		}
		if proposed.PlatformPolicy == nil {
			proposed.PlatformPolicy = &config.MCPPlatformPolicy{}
		}
		policy := proposed.PlatformPolicy
		policy.AllowedHosts, err = p.list("Allowed platform hosts", policy.AllowedHosts)
		if err != nil {
			return err
		}
		value, askErr := p.ask("Read-only hosts (comma separated; - for none)", listOrDash(policy.ReadOnlyHosts))
		if askErr != nil {
			return askErr
		}
		policy.ReadOnlyHosts = parseListOrDash(value)
		edit, askErr := p.yes("Edit restrictions for individual workspaces?")
		if askErr != nil {
			return askErr
		}
		if edit {
			fmt.Fprintln(p.out, "Workspace rules use canonical IDs from whodb orgs and whodb projects.")
			policy.AllowedWorkspaces, err = promptSetupWorkspaces(p, "Allowed workspaces", policy.AllowedWorkspaces)
			if err != nil {
				return err
			}
			policy.ReadOnlyWorkspaces, err = promptSetupWorkspaces(p, "Read-only workspaces", policy.ReadOnlyWorkspaces)
			if err != nil {
				return err
			}
		}
		defaultTargets := []config.MCPWorkspace{}
		if proposed.DefaultWorkspace != nil {
			defaultTargets = append(defaultTargets, *proposed.DefaultWorkspace)
		}
		defaults, defaultErr := promptSetupWorkspaces(p, "Default workspace (optional)", defaultTargets)
		if defaultErr != nil {
			return defaultErr
		}
		if len(defaults) > 1 {
			return fmt.Errorf("choose one default workspace")
		}
		proposed.DefaultWorkspace = nil
		if len(defaults) == 1 {
			proposed.DefaultWorkspace = &defaults[0]
		}
		value, askErr = p.ask("Hosts to sign into after saving (comma separated; - to skip)", "-")
		if askErr != nil {
			return askErr
		}
		loginHosts = parseListOrDash(value)
	}
	var newConnection *config.Connection
	if proposed.HasModule("database") {
		fmt.Fprintln(p.out, "Saved database connections:")
		for _, conn := range cfg.Connections {
			fmt.Fprintf(p.out, "  %s (%s)\n", conn.Name, conn.Type)
		}
		add, askErr := p.yes("Add a database connection?")
		if askErr != nil {
			return askErr
		}
		if add {
			newConnection, err = promptSetupConnection(p, cfg)
			if err != nil {
				return err
			}
		}
		proposed.AllowedConnections, err = p.list("Allowed database connection names", proposed.AllowedConnections)
		if err != nil {
			return err
		}
		value, askErr := p.ask("Default database connection (- for none)", valueOrDash(proposed.DefaultConnection))
		if askErr != nil {
			return askErr
		}
		proposed.DefaultConnection = value
		if value == "-" {
			proposed.DefaultConnection = ""
		}
	}
	edit, err := p.yes("Change tool selection?")
	if err != nil {
		return err
	}
	if edit {
		fmt.Fprintln(p.out, "Use full whodb_platform_* names for platform tools; database tools accept names such as query or whodb_query.")
		proposed.EnabledTools, err = p.list("Enabled tools", proposed.EnabledTools)
		if err != nil {
			return err
		}
		value, askErr := p.ask("Disabled tools (comma separated; - for none)", listOrDash(proposed.DisabledTools))
		if askErr != nil {
			return askErr
		}
		proposed.DisabledTools = parseListOrDash(value)
	}
	if err := validateMCPSettings(&proposed); err != nil {
		return err
	}
	preview, _ := json.MarshalIndent(proposed, "", "  ")
	fmt.Fprintf(p.out, "\nProposed MCP settings:\n%s\n", preview)
	if newConnection != nil {
		fmt.Fprintf(p.out, "New connection: %s (%s); password is not shown.\n", newConnection.Name, newConnection.Type)
	}
	if len(loginHosts) > 0 {
		fmt.Fprintf(p.out, "Browser sign-in after saving: %s\n", strings.Join(loginHosts, ", "))
	}
	approved, err := p.yes("Save these settings?")
	if err != nil {
		return err
	}
	if !approved {
		fmt.Fprintln(p.out, "Setup cancelled; settings were not saved.")
		return nil
	}
	if newConnection != nil {
		latest, err := config.LoadConfig()
		if err != nil {
			return err
		}
		latest.AddConnection(*newConnection)
		if err := latest.Save(); err != nil {
			return err
		}
	}
	if err := config.SaveMCPSettings(proposed); err != nil {
		return err
	}
	fmt.Fprintln(p.out, "Saved. Keep the MCP entry at whodb mcp serve. Restart an existing MCP connection to apply changes.")
	for _, host := range loginHosts {
		if err := runSetupLogin(cmd, host); err != nil {
			fmt.Fprintf(p.out, "Settings saved, but login needs attention: %v\n", err)
		}
	}
	verify, err := p.yes("Verify configured connections now?")
	if err != nil {
		return err
	}
	if verify {
		return runSetupVerify(cmd, nil)
	}
	fmt.Fprintln(p.out, "Run whodb setup verify when ready.")
	return nil
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
func listOrDash(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ",")
}
func parseListOrDash(value string) []string {
	if value == "-" {
		return nil
	}
	return parseSetupList(value)
}
func promptSetupWorkspaces(p setupPrompter, label string, targets []config.MCPWorkspace) ([]config.MCPWorkspace, error) {
	values := []string{}
	for _, target := range targets {
		values = append(values, target.Host+"|"+target.Org+"|"+target.Project)
	}
	value, err := p.ask(label+" (host|org-id|project-id, comma separated; - for none)", listOrDash(values))
	if err != nil {
		return nil, err
	}
	result := []config.MCPWorkspace{}
	for _, item := range parseListOrDash(value) {
		parts := strings.Split(item, "|")
		if len(parts) != 3 {
			return nil, fmt.Errorf("workspace must be host|org-id|project-id")
		}
		result = append(result, config.MCPWorkspace{Host: strings.TrimSpace(parts[0]), Org: strings.TrimSpace(parts[1]), Project: strings.TrimSpace(parts[2])})
	}
	return result, nil
}

func runSetupLogin(cmd *cobra.Command, host string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	child := exec.CommandContext(cmd.Context(), executable, "login", "--host", host)
	child.Stdin = cmd.InOrStdin()
	child.Stdout = cmd.ErrOrStderr()
	child.Stderr = cmd.ErrOrStderr()
	return child.Run()
}

func promptSetupConnection(p setupPrompter, cfg *config.Config) (*config.Connection, error) {
	name, err := p.ask("Connection name", "")
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("connection name is required")
	}
	if _, err := cfg.GetConnection(name); err == nil {
		return nil, fmt.Errorf("connection %q already exists; edit it with whodb connections", name)
	}
	kind, err := p.ask("Database type", "Postgres")
	if err != nil {
		return nil, err
	}
	spec, ok := lookupDatabaseType(kind)
	if !ok {
		return nil, fmt.Errorf("unsupported database type %q", kind)
	}
	conn := config.Connection{Name: name, Type: spec.ID}
	conn.Host, err = p.ask("Host or file path", "localhost")
	if err != nil {
		return nil, err
	}
	port, err := p.ask("Port (0 for connector default)", "0")
	if err != nil {
		return nil, err
	}
	conn.Port, err = strconv.Atoi(port)
	if err != nil {
		return nil, err
	}
	conn.Username, err = p.ask("Username", "")
	if err != nil {
		return nil, err
	}
	conn.Database, err = p.ask("Database", "")
	if err != nil {
		return nil, err
	}
	ssl, err := p.ask("SSL mode (- for connector default)", "-")
	if err != nil {
		return nil, err
	}
	if ssl != "-" {
		conn.Advanced, err = connectionopts.ApplySSLSettings(spec.ID, nil, connectionopts.SSLSettings{Mode: ssl})
		if err != nil {
			return nil, err
		}
	}
	fmt.Fprint(p.out, "Password (hidden; Enter for none): ")
	secret, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(p.out)
	if err != nil {
		return nil, err
	}
	conn.Password = string(secret)
	conn, err = normalizeDirectConnection(conn, spec, true)
	if err != nil {
		return nil, err
	}
	return &conn, nil
}
