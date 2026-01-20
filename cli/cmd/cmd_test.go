/*
 * Copyright 2025 Clidey, Inc.
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

package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootCmd(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	if rootCmd.Use != "whodb-cli" {
		t.Errorf("Expected Use to be 'whodb-cli', got '%s'", rootCmd.Use)
	}

	if rootCmd.Short == "" {
		t.Error("Expected Short description to be non-empty")
	}
}

func TestRootCmd_HasSubcommands(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	commands := rootCmd.Commands()
	if len(commands) == 0 {
		t.Error("Expected at least one subcommand")
	}

	hasConnect := false

	for _, cmd := range commands {
		cmdName := cmd.Use
		if cmdName == "connect" {
			hasConnect = true
		}
	}

	if !hasConnect {
		t.Error("Expected 'connect' subcommand")
	}
}

func TestRootCmd_Flags(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	flag := rootCmd.PersistentFlags().Lookup("config")
	if flag == nil {
		t.Error("Expected 'config' flag to exist")
	}

	debugFlag := rootCmd.PersistentFlags().Lookup("debug")
	if debugFlag == nil {
		t.Error("Expected 'debug' flag to exist")
	}
}

func TestInitConfig(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	initConfig()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to resolve home directory: %v", err)
	}
	configDir := filepath.Join(home, ".whodb-cli")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Errorf("Config directory was not created: %s", configDir)
	}
}
