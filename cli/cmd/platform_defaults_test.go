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
	"strings"
	"testing"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

func TestMCPPlatformDefaultAndDatabaseMode(t *testing.T) {
	for _, tc := range []struct {
		name      string
		flags     map[string]string
		platform  bool
		wantError bool
		notice    bool
	}{
		{name: "default", platform: true},
		{name: "database", flags: map[string]string{"database": "true"}},
		{name: "legacy platform", flags: map[string]string{"platform": "true"}, platform: true, notice: true},
		{name: "conflicting modes", flags: map[string]string{"platform": "true", "database": "true"}, wantError: true},
		{name: "old false mode", flags: map[string]string{"platform": "false"}, wantError: true},
		{name: "database flags require mode", flags: map[string]string{"tools": "schemas"}, wantError: true},
		{name: "database tool selection", flags: map[string]string{"database": "true", "tools": "schemas"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetMCPServeFlagsForTest(t)
			var stdout, stderr bytes.Buffer
			mcpServeCmd.SetOut(&stdout)
			mcpServeCmd.SetErr(&stderr)
			t.Cleanup(func() { mcpServeCmd.SetOut(nil); mcpServeCmd.SetErr(nil) })
			for name, value := range tc.flags {
				if err := mcpServeCmd.Flags().Set(name, value); err != nil {
					t.Fatal(err)
				}
			}
			err := configureMCPMode(mcpServeCmd)
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v", err)
			}
			if err == nil && mcpPlatform != tc.platform {
				t.Fatalf("platform = %v", mcpPlatform)
			}
			if tc.notice && !strings.Contains(stderr.String(), "no longer necessary") {
				t.Fatal("missing compatibility notice")
			}
			if stdout.Len() != 0 {
				t.Fatal("mode selection wrote to protocol stdout")
			}
		})
	}
}

func TestPlatformLandingWithoutLogin(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()
	t.Setenv(platformapi.SessionHostEnv, "")
	t.Setenv(platformapi.SessionOrgEnv, "")
	t.Setenv(platformapi.SessionProjectEnv, "")
	cfg, err := config.LoadConfigWithoutSecrets()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Platform = config.PlatformConfig{}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	rootCmd.SetOut(&output)
	t.Cleanup(func() { rootCmd.SetOut(nil) })
	if err := showPlatformLanding(rootCmd); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"No saved login", "whodb login", "whodb --tui", "whodb mcp serve --database"} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("missing %q in %s", expected, output.String())
		}
	}
	if rootCmd.Flags().Lookup("tui") == nil {
		t.Fatal("missing --tui")
	}
}
