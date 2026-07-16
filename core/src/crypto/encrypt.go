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

// Package crypto provides authenticated symmetric encryption for values that
// must be stored at rest (for example, database credentials persisted in the
// local session store). It uses AES-256-GCM with a random nonce per message.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
)

// ErrInvalidKey is returned when the supplied key is not a 64-character hex
// string decoding to 32 bytes (the AES-256 key size).
var ErrInvalidKey = errors.New("crypto: key must be 64 hex chars (32 bytes)")

// ErrCiphertextTooShort is returned when the ciphertext is smaller than the
// GCM nonce and therefore cannot have been produced by Encrypt.
var ErrCiphertextTooShort = errors.New("crypto: ciphertext too short")

// Encrypt encrypts plaintext using AES-256-GCM. keyHex must be a 64-character
// hex string (32 bytes). The returned slice is the GCM nonce followed by the
// sealed ciphertext, suitable for storage and later use with Decrypt.
func Encrypt(keyHex, plaintext string) ([]byte, error) {
	gcm, err := newGCM(keyHex)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

// Decrypt decrypts ciphertext produced by Encrypt using the same key. It
// returns an error when the key is invalid, the ciphertext is malformed, or
// authentication fails (wrong key or tampered ciphertext).
func Decrypt(keyHex string, ciphertext []byte) (string, error) {
	gcm, err := newGCM(keyHex)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(ciphertext) < ns {
		return "", ErrCiphertextTooShort
	}
	plain, err := gcm.Open(nil, ciphertext[:ns], ciphertext[ns:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// newGCM builds an AES-256-GCM AEAD from a hex-encoded 32-byte key.
func newGCM(keyHex string) (cipher.AEAD, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return nil, ErrInvalidKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
