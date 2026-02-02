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

package testharness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// PostgresConfig holds the test database connection details.
type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	Schema   string
}

// DefaultPostgresConfig returns the config matching docker-compose.yml.
func DefaultPostgresConfig() PostgresConfig {
	return PostgresConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "user",
		Password: "jio53$*(@nfe)",
		Database: "test_db",
		Schema:   "test_schema",
	}
}

// SetupEnv sets environment variables for the CLI to use and returns a cleanup function.
func SetupEnv(t *testing.T, cfg PostgresConfig) func() {
	t.Helper()

	// Build the connection JSON for the environment variable
	connJSON := fmt.Sprintf(`[{"alias":"test-pg","host":"%s","user":"%s","password":"%s","database":"%s","port":"%s"}]`,
		cfg.Host, cfg.User, cfg.Password, cfg.Database, cfg.Port)

	// Store original values
	origPostgres := os.Getenv("WHODB_POSTGRES")
	origHome := os.Getenv("HOME")

	// Create a temp home directory to avoid config file conflicts
	tempHome, err := os.MkdirTemp("", "whodb-cli-e2e-")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}

	// Set environment variables
	os.Setenv("WHODB_POSTGRES", connJSON)
	os.Setenv("HOME", tempHome)
	// Also set for Windows compatibility
	os.Setenv("USERPROFILE", tempHome)
	// Disable analytics during tests
	os.Setenv("WHODB_CLI_ANALYTICS_DISABLED", "true")

	return func() {
		// Restore original values
		if origPostgres != "" {
			os.Setenv("WHODB_POSTGRES", origPostgres)
		} else {
			os.Unsetenv("WHODB_POSTGRES")
		}
		if origHome != "" {
			os.Setenv("HOME", origHome)
			os.Setenv("USERPROFILE", origHome)
		}
		os.Unsetenv("WHODB_CLI_ANALYTICS_DISABLED")
		// Clean up temp directory
		os.RemoveAll(tempHome)
	}
}

// CLIBinaryPath returns the path to the CLI binary.
func CLIBinaryPath(t *testing.T) string {
	t.Helper()

	// Get the path to the cli directory
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Failed to get caller information")
	}

	// testharness/harness.go -> e2e -> cli
	cliDir := filepath.Dir(filepath.Dir(filepath.Dir(filename)))
	binaryName := "whodb-cli"
	if runtime.GOOS == "windows" {
		binaryName = "whodb-cli.exe"
	}

	binaryPath := filepath.Join(cliDir, binaryName)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("CLI binary not found at %s. Run 'go build -o whodb-cli .' in the cli directory first.", binaryPath)
	}

	return binaryPath
}

// RunCLI executes the CLI with given args and returns stdout, stderr, and exit code.
func RunCLI(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	binaryPath := CLIBinaryPath(t)

	cmd := exec.Command(binaryPath, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()

	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("Failed to run CLI: %v", err)
		}
	}

	return stdout, stderr, exitCode
}

// RequireSuccess fails the test if exitCode != 0.
func RequireSuccess(t *testing.T, stderr string, exitCode int) {
	t.Helper()

	if exitCode != 0 {
		t.Fatalf("CLI command failed with exit code %d\nStderr: %s", exitCode, stderr)
	}
}

// RequireFailure fails the test if exitCode == 0.
func RequireFailure(t *testing.T, exitCode int) {
	t.Helper()

	if exitCode == 0 {
		t.Fatal("Expected CLI command to fail, but it succeeded")
	}
}

// ParseJSONArray parses stdout as a JSON array of maps.
func ParseJSONArray(t *testing.T, stdout string) []map[string]any {
	t.Helper()

	var result []map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Failed to parse JSON array: %v\nOutput: %s", err, stdout)
	}
	return result
}
