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
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
	"github.com/zalando/go-keyring"
)

// CLI tests use an in-memory keyring and never access the developer's credential store.
func TestMain(m *testing.M) { keyring.MockInit(); os.Exit(m.Run()) }

func executeSetupTest(t *testing.T, input string, args ...string) (map[string]any, error) {
	t.Helper()
	cmd := newSetupCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader(input))
	cmd.SetArgs(args)
	err := cmd.Execute()
	var result map[string]any
	if out.Len() > 0 {
		if decodeErr := json.Unmarshal(out.Bytes(), &result); decodeErr != nil {
			t.Fatalf("invalid JSON %s: %v", out.String(), decodeErr)
		}
	}
	return result, err
}

func TestSetupPatchApprovalAndPreservation(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	cfg, err := config.LoadConfigWithoutSecrets()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Connections = []config.Connection{{Name: "saved", Type: "Postgres", Password: "do-not-print", SSHPassword: "ssh-secret"}}
	cfg.UpsertPlatformHost(config.PlatformHost{URL: "https://example.test", AccountID: "user", DefaultOrgID: "org-1"})
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	path, _ := config.GetConfigPath()
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	patch := `{"modules":["platform","database"],"platform_policy":{"read_only_hosts":["https://example.test/"]}}`
	validated, err := executeSetupTest(t, patch, "validate", "--file", "-")
	if err != nil {
		t.Fatal(err)
	}
	if validated["data"].(map[string]any)["applied"] != false {
		t.Fatal(validated)
	}
	unchanged, _ := os.ReadFile(path)
	if !bytes.Equal(before, unchanged) {
		t.Fatal("validation modified config")
	}
	if _, err := executeSetupTest(t, patch, "apply", "--file", "-"); err == nil {
		t.Fatal("applied without approval")
	}
	applied, err := executeSetupTest(t, patch, "apply", "--file", "-", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if applied["data"].(map[string]any)["restart_required"] != true {
		t.Fatal(applied)
	}
	settings, err := config.LoadMCPSettings()
	if err != nil {
		t.Fatal(err)
	}
	if !settings.HasModule("database") || settings.PlatformPolicy.ReadOnlyHosts[0] != "https://example.test" {
		t.Fatal(settings)
	}
	after, _ := os.ReadFile(path)
	var oldSections, newSections map[string]json.RawMessage
	if err := json.Unmarshal(before, &oldSections); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(after, &newSections); err != nil {
		t.Fatal(err)
	}
	var oldCLI, newCLI any
	_ = json.Unmarshal(oldSections["cli"], &oldCLI)
	_ = json.Unmarshal(newSections["cli"], &newCLI)
	oldJSON, _ := json.Marshal(oldCLI)
	newJSON, _ := json.Marshal(newCLI)
	if !bytes.Equal(oldJSON, newJSON) {
		t.Fatal("MCP setup rewrote CLI credentials or hosts")
	}
	_, err = executeSetupTest(t, `{"write_mode":"read-only"}`, "apply", "--file", "-", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	settings, _ = config.LoadMCPSettings()
	if settings.WriteMode != "read-only" || !settings.HasModule("database") || settings.PlatformPolicy == nil {
		t.Fatal("patch erased unspecified fields")
	}
	inspected, err := executeSetupTest(t, "", "inspect")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(inspected)
	if bytes.Contains(raw, []byte("do-not-print")) || bytes.Contains(raw, []byte("ssh-secret")) {
		t.Fatal("inspect exposed credentials")
	}
}

func TestSetupRejectsInvalidSettings(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	for _, patch := range []string{`{"password":"secret"}`, `{"default_workspace":{"host":"https://prod.example.test"},"platform_policy":{"allowed_hosts":["https://uat.example.test"]}}`, `{"modules":[]}`, `{"modules":["typo"]}`, `{"write_mode":"sometimes"}`, `{"enabled_tools":["typo"]}`, `{"platform_policy":{"read_only_host":[]}}`, `{"allowed_connections":["uat"],"default_connection":"prod"}`, `null`} {
		if _, err := executeSetupTest(t, patch, "validate", "--file", "-"); err == nil {
			t.Errorf("accepted %s", patch)
		}
	}
	settings, _ := config.LoadMCPSettings()
	if settings.WriteMode != "confirm" || len(settings.Modules) != 1 {
		t.Fatal(settings)
	}
}

func TestMCPReadsSavedModulesAndOverrides(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	resetMCPServeFlagsForTest(t)
	if err := config.SaveMCPSettings(config.MCPSettings{Modules: []string{"platform", "database"}, WriteMode: "read-only", AllowedConnections: []string{"uat"}}); err != nil {
		t.Fatal(err)
	}
	settings, err := configureMCPMode(mcpServeCmd)
	if err != nil {
		t.Fatal(err)
	}
	if !settings.HasModule("platform") || !settings.HasModule("database") || settings.WriteMode != "read-only" || len(settings.AllowedConnections) != 1 {
		t.Fatal(settings)
	}
	if err := mcpServeCmd.Flags().Set("database", "true"); err != nil {
		t.Fatal(err)
	}
	settings, err = configureMCPMode(mcpServeCmd)
	if err != nil {
		t.Fatal(err)
	}
	if settings.HasModule("platform") || !settings.HasModule("database") {
		t.Fatal(settings)
	}
	saved, _ := config.LoadMCPSettings()
	if !saved.HasModule("platform") {
		t.Fatal("flags changed saved settings")
	}
}

func TestSetupWizardCancellation(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	cmd := newSetupCommand()
	cmd.SetIn(strings.NewReader("database\nread-only\nno\n*\n-\nno\nno\n"))
	cmd.SetErr(&bytes.Buffer{})
	if err := runSetupWizard(cmd); err != nil {
		t.Fatal(err)
	}
	saved, _ := config.LoadMCPSettings()
	if !saved.HasModule("platform") || saved.WriteMode != "confirm" {
		t.Fatal("cancelled wizard saved settings")
	}
}

func TestSetupWizardSavesAndReportsRestart(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	cmd := newSetupCommand()
	var output bytes.Buffer
	cmd.SetErr(&output)
	cmd.SetIn(strings.NewReader("database\nread-only\nno\n*\n-\nno\nyes\nno\n"))
	if err := runSetupWizard(cmd); err != nil {
		t.Fatal(err)
	}
	saved, err := config.LoadMCPSettings()
	if err != nil {
		t.Fatal(err)
	}
	if !saved.HasModule("database") || saved.HasModule("platform") || saved.WriteMode != "read-only" {
		t.Fatal(saved)
	}
	if !strings.Contains(output.String(), "Restart") {
		t.Fatal("wizard did not explain restart")
	}
}

func TestSetupVerifyReportsMissingLogin(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	if err := config.SaveMCPSettings(config.MCPSettings{Modules: []string{"platform"}, WriteMode: "confirm", PlatformPolicy: &config.MCPPlatformPolicy{AllowedHosts: []string{"https://not-configured.example.test"}}}); err != nil {
		t.Fatal(err)
	}
	report, err := executeSetupTest(t, "", "verify")
	if err == nil {
		t.Fatal("missing login passed verification")
	}
	if report["success"] != false {
		t.Fatal(report)
	}
	checks := report["checks"].([]any)
	if len(checks) != 1 || checks[0].(map[string]any)["error"] != "Sign-in required" {
		t.Fatal(checks)
	}
}

func TestSavedWorkspaceYieldsToProcessTarget(t *testing.T) {
	resetMCPServeFlagsForTest(t)
	for _, key := range []string{platformapi.SessionHostEnv, platformapi.SessionOrgEnv, platformapi.SessionProjectEnv} {
		t.Setenv(key, "")
	}
	target := &config.MCPWorkspace{Host: "https://uat.example.test", Org: "org-uat", Project: "project-uat"}
	if err := configureMCPPlatformScope(target); err != nil {
		t.Fatal(err)
	}
	if os.Getenv(platformapi.SessionProjectEnv) != "project-uat" {
		t.Fatal("saved default not applied")
	}
	t.Setenv(platformapi.SessionHostEnv, "https://prod.example.test")
	t.Setenv(platformapi.SessionOrgEnv, "")
	t.Setenv(platformapi.SessionProjectEnv, "")
	if err := configureMCPPlatformScope(target); err != nil {
		t.Fatal(err)
	}
	if os.Getenv(platformapi.SessionHostEnv) != "https://prod.example.test" || os.Getenv(platformapi.SessionProjectEnv) != "" {
		t.Fatal("saved workspace leaked into explicit host target")
	}
}

type setupWorkspaceFixture struct{}

func (setupWorkspaceFixture) Organizations(context.Context) ([]platformapi.Organization, error) {
	return []platformapi.Organization{{ID: "org-1"}}, nil
}
func (setupWorkspaceFixture) Projects(context.Context, string) ([]platformapi.Project, error) {
	return []platformapi.Project{{ID: "project-1"}}, nil
}
func TestVerifyConfiguredWorkspace(t *testing.T) {
	target := config.MCPWorkspace{Host: "https://example.test", Org: "org-1", Project: "project-1"}
	if err := verifySetupWorkspace(context.Background(), setupWorkspaceFixture{}, target); err != nil {
		t.Fatal(err)
	}
	target.Project = "missing"
	if err := verifySetupWorkspace(context.Background(), setupWorkspaceFixture{}, target); err == nil {
		t.Fatal("inaccessible project passed")
	}
}
