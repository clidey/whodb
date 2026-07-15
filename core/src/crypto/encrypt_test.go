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

package crypto

import (
	"errors"
	"strings"
	"testing"
)

const (
	testKey      = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	otherTestKey = "1f1e1d1c1b1a191817161514131211100f0e0d0c0b0a09080706050403020100"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	plaintext := `{"Password":"s3cr3t","Hostname":"db.internal"}`
	ct, err := Encrypt(testKey, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if strings.Contains(string(ct), "s3cr3t") {
		t.Fatal("ciphertext leaked plaintext")
	}
	got, err := Decrypt(testKey, ct)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != plaintext {
		t.Fatalf("round-trip mismatch: got %q want %q", got, plaintext)
	}
}

func TestEncryptNonceIsRandom(t *testing.T) {
	a, err := Encrypt(testKey, "same")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Encrypt(testKey, "same")
	if err != nil {
		t.Fatal(err)
	}
	if string(a) == string(b) {
		t.Fatal("expected distinct ciphertexts for repeated plaintext (nonce reuse)")
	}
}

func TestDecryptWrongKeyFails(t *testing.T) {
	ct, err := Encrypt(testKey, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decrypt(otherTestKey, ct); err == nil {
		t.Fatal("expected error decrypting with wrong key")
	}
}

func TestInvalidKeyLength(t *testing.T) {
	if _, err := Encrypt("abcd", "hello"); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("Encrypt short key: got %v want ErrInvalidKey", err)
	}
	if _, err := Decrypt("abcd", []byte("whatever")); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("Decrypt short key: got %v want ErrInvalidKey", err)
	}
}

func TestDecryptCiphertextTooShort(t *testing.T) {
	if _, err := Decrypt(testKey, []byte("x")); !errors.Is(err, ErrCiphertextTooShort) {
		t.Fatalf("got %v want ErrCiphertextTooShort", err)
	}
}

func TestDecryptTamperedCiphertextFails(t *testing.T) {
	ct, err := Encrypt(testKey, "hello world")
	if err != nil {
		t.Fatal(err)
	}
	ct[len(ct)-1] ^= 0xff // flip a bit in the sealed data
	if _, err := Decrypt(testKey, ct); err == nil {
		t.Fatal("expected GCM authentication failure on tampered ciphertext")
	}
}
