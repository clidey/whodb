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

package ssh

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTunnel_NoAuthMethods(t *testing.T) {
	// Unset SSH_AUTH_SOCK so agent auth is not available
	t.Setenv("SSH_AUTH_SOCK", "")

	_, err := NewTunnel("localhost", 22, "user", "", "", "dbhost", 5432)
	if err == nil {
		t.Fatal("expected error when no auth methods are provided")
	}
}

func TestNewTunnel_InvalidKeyFile(t *testing.T) {
	_, err := NewTunnel("localhost", 22, "user", "/nonexistent/key/file", "", "dbhost", 5432)
	if err == nil {
		t.Fatal("expected error for nonexistent key file")
	}
}

func TestNewTunnel_BadKeyFileContent(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "bad_key")
	if err := os.WriteFile(keyPath, []byte("not a valid ssh key"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err := NewTunnel("localhost", 22, "user", keyPath, "", "dbhost", 5432)
	if err == nil {
		t.Fatal("expected error for invalid key file content")
	}
}

func TestBuildAuthMethods_PasswordOnly(t *testing.T) {
	methods, err := buildAuthMethods("", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) == 0 {
		t.Fatal("expected at least one auth method for password")
	}
}

func TestBuildAuthMethods_NoCredentials(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")

	methods, err := buildAuthMethods("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 0 {
		t.Fatalf("expected zero auth methods, got %d", len(methods))
	}
}

func TestTunnel_LocalPort_BeforeStart(t *testing.T) {
	tunnel := &Tunnel{}
	if port := tunnel.LocalPort(); port != 0 {
		t.Fatalf("expected port 0 before start, got %d", port)
	}
}

func TestTunnel_StopWithoutStart(t *testing.T) {
	tunnel := &Tunnel{
		done: make(chan struct{}),
	}
	// Stop should not panic even if Start was never called
	tunnel.Stop()
	tunnel.Stop() // double stop should be safe
}
