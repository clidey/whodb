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

package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/clidey/whodb/core/src/env"
)

func TestResolveSessionEncryptionKeyFromEnv(t *testing.T) {
	valid := "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	env.EncryptionKey = valid
	t.Cleanup(func() { env.EncryptionKey = "" })

	got, err := ResolveSessionEncryptionKey(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveSessionEncryptionKey: %v", err)
	}
	if got != valid {
		t.Fatalf("got %q want %q", got, valid)
	}
}

func TestResolveSessionEncryptionKeyEnvMalformed(t *testing.T) {
	env.EncryptionKey = "not-hex"
	t.Cleanup(func() { env.EncryptionKey = "" })

	if _, err := ResolveSessionEncryptionKey(t.TempDir()); err == nil {
		t.Fatal("expected error for malformed WHODB_ENCRYPTION_KEY")
	}
}

func TestResolveSessionEncryptionKeyFileGenerateThenReuse(t *testing.T) {
	env.EncryptionKey = ""
	dir := t.TempDir()

	first, err := ResolveSessionEncryptionKey(dir)
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	if err := validateKeyHex(first); err != nil {
		t.Fatalf("generated key invalid: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, sessionKeyFileName)); err != nil {
		t.Fatalf("key file not written: %v", err)
	}

	second, err := ResolveSessionEncryptionKey(dir)
	if err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if first != second {
		t.Fatal("expected the persisted key to be reused")
	}
}

func TestResolveSessionEncryptionKeyDesktopKeyring(t *testing.T) {
	keyring.MockInit()
	env.EncryptionKey = ""
	t.Setenv("WHODB_DESKTOP", "true")

	first, err := ResolveSessionEncryptionKey(t.TempDir())
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	if err := validateKeyHex(first); err != nil {
		t.Fatalf("generated key invalid: %v", err)
	}

	second, err := ResolveSessionEncryptionKey(t.TempDir())
	if err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if first != second {
		t.Fatal("expected the keyring key to be reused")
	}
}
