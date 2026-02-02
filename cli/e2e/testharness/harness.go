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
	"strings"
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
// It passes the current process's environment to the subprocess.
func RunCLI(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	binaryPath := CLIBinaryPath(t)

	cmd := exec.Command(binaryPath, args...)

	// Explicitly pass the current environment to the subprocess
	// This ensures any modifications made via os.Setenv are inherited
	cmd.Env = os.Environ()

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

// SetupCleanEnv sets up a clean environment with no database connections.
// Returns a cleanup function that restores the original environment.
func SetupCleanEnv(t *testing.T) func() {
	t.Helper()

	// Database prefixes to clear (including numbered variants like WHODB_POSTGRES_1)
	dbPrefixes := []string{
		"WHODB_POSTGRES", "WHODB_MYSQL", "WHODB_MARIADB", "WHODB_MONGODB",
		"WHODB_REDIS", "WHODB_CLICKHOUSE", "WHODB_ELASTICSEARCH", "WHODB_SQLITE",
	}

	// Collect all matching env vars (including numbered ones)
	origValues := make(map[string]string)

	// Store HOME/USERPROFILE
	origValues["HOME"] = os.Getenv("HOME")
	origValues["USERPROFILE"] = os.Getenv("USERPROFILE")

	// Find and store all database env vars
	for _, env := range os.Environ() {
		for _, prefix := range dbPrefixes {
			if strings.HasPrefix(env, prefix) {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					origValues[parts[0]] = parts[1]
				}
			}
		}
	}

	// Create a temp home directory to avoid config file conflicts
	tempHome, err := os.MkdirTemp("", "whodb-cli-e2e-clean-")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}

	// Clear all database env vars (including numbered ones)
	for _, env := range os.Environ() {
		for _, prefix := range dbPrefixes {
			if strings.HasPrefix(env, prefix) {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) >= 1 {
					os.Unsetenv(parts[0])
				}
			}
		}
	}

	// Preserve BAML library path so it can be found with the new HOME
	// BAML caches its library under different paths per OS:
	// - Linux/Mac: ~/.cache/baml/libs/<version>/
	// - Windows: %LOCALAPPDATA%/baml/libs/<version>/
	origHome := origValues["HOME"]
	if origHome != "" {
		// Try common BAML cache locations
		possibleCacheDirs := []string{
			filepath.Join(origHome, ".cache", "baml", "libs"),                    // Linux/Mac
			filepath.Join(os.Getenv("LOCALAPPDATA"), "baml", "libs"),             // Windows
			filepath.Join(origValues["USERPROFILE"], ".cache", "baml", "libs"),   // Windows fallback
		}

		for _, cacheDir := range possibleCacheDirs {
			if cacheDir == "" {
				continue
			}
			// Find any .so or .dll file in the cache
			entries, err := os.ReadDir(cacheDir)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if entry.IsDir() {
					versionDir := filepath.Join(cacheDir, entry.Name())
					files, err := os.ReadDir(versionDir)
					if err != nil {
						continue
					}
					for _, f := range files {
						name := f.Name()
						if strings.HasPrefix(name, "libbaml") && (strings.HasSuffix(name, ".so") || strings.HasSuffix(name, ".dll") || strings.HasSuffix(name, ".dylib")) {
							os.Setenv("BAML_LIBRARY_PATH", filepath.Join(versionDir, name))
							break
						}
					}
				}
			}
		}
	}

	// Set temp home
	os.Setenv("HOME", tempHome)
	os.Setenv("USERPROFILE", tempHome)
	os.Setenv("WHODB_CLI_ANALYTICS_DISABLED", "true")

	return func() {
		// Clear any env vars that were set during test
		for _, env := range os.Environ() {
			for _, prefix := range dbPrefixes {
				if strings.HasPrefix(env, prefix) {
					parts := strings.SplitN(env, "=", 2)
					if len(parts) >= 1 {
						os.Unsetenv(parts[0])
					}
				}
			}
		}

		// Restore original values
		for env, val := range origValues {
			if val != "" {
				os.Setenv(env, val)
			} else {
				os.Unsetenv(env)
			}
		}
		os.Unsetenv("WHODB_CLI_ANALYTICS_DISABLED")
		os.Unsetenv("BAML_LIBRARY_PATH")
		os.RemoveAll(tempHome)
	}
}

// SetupMultipleConnections sets up multiple database connections via environment variables.
// Returns a cleanup function.
func SetupMultipleConnections(t *testing.T, connections map[string]PostgresConfig) func() {
	t.Helper()

	// Store original values
	origPostgres := os.Getenv("WHODB_POSTGRES")
	origHome := os.Getenv("HOME")

	// Create temp home
	tempHome, err := os.MkdirTemp("", "whodb-cli-e2e-multi-")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}

	// Build JSON array for WHODB_POSTGRES
	var connArray []string
	for alias, cfg := range connections {
		connJSON := fmt.Sprintf(`{"alias":"%s","host":"%s","user":"%s","password":"%s","database":"%s","port":"%s"}`,
			alias, cfg.Host, cfg.User, cfg.Password, cfg.Database, cfg.Port)
		connArray = append(connArray, connJSON)
	}
	fullJSON := "[" + strings.Join(connArray, ",") + "]"

	os.Setenv("WHODB_POSTGRES", fullJSON)
	os.Setenv("HOME", tempHome)
	os.Setenv("USERPROFILE", tempHome)
	os.Setenv("WHODB_CLI_ANALYTICS_DISABLED", "true")

	return func() {
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
		os.RemoveAll(tempHome)
	}
}

// GetTempHome returns the temp home directory path from the current environment.
// Useful for tests that need to check config file locations.
func GetTempHome(t *testing.T) string {
	t.Helper()
	return os.Getenv("HOME")
}
