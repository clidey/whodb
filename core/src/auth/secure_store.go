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

package auth

import (
    "encoding/json"

    "github.com/clidey/whodb/core/src/engine"
    "github.com/clidey/whodb/core/src/env"
    "github.com/zalando/go-keyring"
)

var keyringService = getKeyringService()

func getKeyringService() string {
    base := "WhoDB"
    if env.IsEnterpriseEdition {
        base = "WhoDB-EE"
    }
    if env.IsDevelopment {
        return base + "-Dev"
    }
    return base
}

// GetKeyringServiceName exposes the resolved keyring service label for logging.
func GetKeyringServiceName() string { return keyringService }

func keyForProfile(id string) string {
	return "profile:" + id
}

func SaveCredentials(id string, creds *engine.Credentials) error {
	if id == "" || creds == nil {
		return nil
	}
	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	return keyring.Set(keyringService, keyForProfile(id), string(data))
}

func LoadCredentials(id string) (*engine.Credentials, error) {
	if id == "" {
		return nil, keyring.ErrNotFound
	}
	val, err := keyring.Get(keyringService, keyForProfile(id))
	if err != nil {
		return nil, err
	}
	var creds engine.Credentials
	if err := json.Unmarshal([]byte(val), &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

func DeleteCredentials(id string) error {
	if id == "" {
		return nil
	}
	return keyring.Delete(keyringService, keyForProfile(id))
}
