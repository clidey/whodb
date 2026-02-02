//go:build e2e_cli || e2e_postgres

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

package e2e_test

import (
	"testing"

	"github.com/clidey/whodb/cli/e2e/testharness"
)

// ============================================================================
// Version and Help Tests (no database required)
// ============================================================================

func TestCLI_Version(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()

	stdout, stderr, exitCode := testharness.RunCLI(t, "version")

	testharness.RequireSuccess(t, stderr, exitCode)
	testharness.AssertContains(t, stdout, "whodb-cli")
}

func TestCLI_Version_Short(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()

	stdout, stderr, exitCode := testharness.RunCLI(t, "version", "--short")

	testharness.RequireSuccess(t, stderr, exitCode)
	// Short version should be shorter than full version
	if len(stdout) > 50 {
		t.Errorf("Expected short version output, got: %s", stdout)
	}
}

func TestCLI_Help(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()

	stdout, stderr, exitCode := testharness.RunCLI(t, "--help")

	testharness.RequireSuccess(t, stderr, exitCode)
	testharness.AssertContains(t, stdout, "WhoDB CLI")
	testharness.AssertContains(t, stdout, "Available Commands")
}

func TestCLI_Query_Help(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()

	stdout, stderr, exitCode := testharness.RunCLI(t, "query", "--help")

	testharness.RequireSuccess(t, stderr, exitCode)
	testharness.AssertContains(t, stdout, "Execute a SQL query")
	testharness.AssertContains(t, stdout, "--connection")
	testharness.AssertContains(t, stdout, "--format")
}

func TestCLI_Export_Help(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()

	stdout, stderr, exitCode := testharness.RunCLI(t, "export", "--help")

	testharness.RequireSuccess(t, stderr, exitCode)
	testharness.AssertContains(t, stdout, "Export table data")
	testharness.AssertContains(t, stdout, "--table")
	testharness.AssertContains(t, stdout, "--output")
}

// ============================================================================
// No Connections Available Tests
// ============================================================================

func TestCLI_ConnectionsList_Empty(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()

	stdout, stderr, exitCode := testharness.RunCLI(t,
		"connections", "list",
		"--format", "json",
		"--quiet",
	)

	testharness.RequireSuccess(t, stderr, exitCode)
	// Should return empty JSON array
	testharness.AssertContains(t, stdout, "[]")
}

func TestCLI_Schemas_NoConnections(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()

	_, stderr, exitCode := testharness.RunCLI(t,
		"schemas",
		"--quiet",
	)

	testharness.RequireFailure(t, exitCode)
	testharness.AssertContains(t, stderr, "no connections available")
}

func TestCLI_Tables_NoConnections(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()

	_, stderr, exitCode := testharness.RunCLI(t,
		"tables",
		"--quiet",
	)

	testharness.RequireFailure(t, exitCode)
	testharness.AssertContains(t, stderr, "no connections available")
}

func TestCLI_Query_NoConnections(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()

	_, stderr, exitCode := testharness.RunCLI(t,
		"query",
		"--quiet",
		"SELECT 1",
	)

	testharness.RequireFailure(t, exitCode)
	testharness.AssertContains(t, stderr, "no connections available")
}

func TestCLI_Export_NoConnections(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()

	_, stderr, exitCode := testharness.RunCLI(t,
		"export",
		"--table", "users",
		"--output", "/tmp/test.csv",
		"--quiet",
	)

	testharness.RequireFailure(t, exitCode)
	testharness.AssertContains(t, stderr, "no connections available")
}

// ============================================================================
// Invalid Arguments Tests
// ============================================================================

func TestCLI_Query_NoSQL(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()

	_, stderr, exitCode := testharness.RunCLI(t, "query")

	testharness.RequireFailure(t, exitCode)
	testharness.AssertContains(t, stderr, "missing SQL query")
}

func TestCLI_Columns_NoTable(t *testing.T) {
	cfg := testharness.DefaultPostgresConfig()
	cleanup := testharness.SetupEnv(t, cfg)
	defer cleanup()

	_, stderr, exitCode := testharness.RunCLI(t,
		"columns",
		"--connection", "test-pg",
		"--quiet",
	)

	testharness.RequireFailure(t, exitCode)
	testharness.AssertContains(t, stderr, "--table flag is required")
}

func TestCLI_Export_NoOutput(t *testing.T) {
	cfg := testharness.DefaultPostgresConfig()
	cleanup := testharness.SetupEnv(t, cfg)
	defer cleanup()

	_, stderr, exitCode := testharness.RunCLI(t,
		"export",
		"--connection", "test-pg",
		"--table", "users",
		"--quiet",
	)

	testharness.RequireFailure(t, exitCode)
	testharness.AssertContains(t, stderr, "--output is required")
}

func TestCLI_Export_NoTableOrQuery(t *testing.T) {
	cfg := testharness.DefaultPostgresConfig()
	cleanup := testharness.SetupEnv(t, cfg)
	defer cleanup()

	_, stderr, exitCode := testharness.RunCLI(t,
		"export",
		"--connection", "test-pg",
		"--output", "/tmp/test.csv",
		"--quiet",
	)

	testharness.RequireFailure(t, exitCode)
	testharness.AssertContains(t, stderr, "either --table or --query is required")
}

func TestCLI_Export_BothTableAndQuery(t *testing.T) {
	cfg := testharness.DefaultPostgresConfig()
	cleanup := testharness.SetupEnv(t, cfg)
	defer cleanup()

	_, stderr, exitCode := testharness.RunCLI(t,
		"export",
		"--connection", "test-pg",
		"--table", "users",
		"--query", "SELECT 1",
		"--output", "/tmp/test.csv",
		"--quiet",
	)

	testharness.RequireFailure(t, exitCode)
	testharness.AssertContains(t, stderr, "cannot use both --table and --query")
}

// ============================================================================
// Invalid Connection Tests
// ============================================================================

func TestCLI_Query_InvalidConnection(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()

	_, stderr, exitCode := testharness.RunCLI(t,
		"query",
		"--connection", "nonexistent-db",
		"--quiet",
		"SELECT 1",
	)

	testharness.RequireFailure(t, exitCode)
	testharness.AssertContains(t, stderr, "connection")
}

func TestCLI_Schemas_InvalidConnection(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()

	_, stderr, exitCode := testharness.RunCLI(t,
		"schemas",
		"--connection", "nonexistent-db",
		"--quiet",
	)

	testharness.RequireFailure(t, exitCode)
	testharness.AssertContains(t, stderr, "connection")
}

// ============================================================================
// History Tests (no database required)
// ============================================================================

func TestCLI_History_List_Empty(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()

	stdout, stderr, exitCode := testharness.RunCLI(t,
		"history", "list",
		"--format", "json",
	)

	testharness.RequireSuccess(t, stderr, exitCode)
	testharness.AssertContains(t, stdout, "[]")
}

func TestCLI_History_Clear(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()

	_, stderr, exitCode := testharness.RunCLI(t,
		"history", "clear",
	)

	testharness.RequireSuccess(t, stderr, exitCode)
}

func TestCLI_History_Search_Empty(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()

	stdout, stderr, exitCode := testharness.RunCLI(t,
		"history", "search", "SELECT",
		"--format", "json",
	)

	testharness.RequireSuccess(t, stderr, exitCode)
	testharness.AssertContains(t, stdout, "[]")
}

// ============================================================================
// Output Format Tests
// ============================================================================

func TestCLI_InvalidFormat(t *testing.T) {
	cleanup := testharness.SetupCleanEnv(t)
	defer cleanup()

	_, stderr, exitCode := testharness.RunCLI(t,
		"connections", "list",
		"--format", "invalid_format",
	)

	testharness.RequireFailure(t, exitCode)
	testharness.AssertContains(t, stderr, "format")
}
