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

package types

import "encoding/json"

// DatabaseCredentials holds database connection details parsed from environment
// variables or configuration files. The Advanced field accepts both "advanced"
// and the legacy "config" JSON key for backwards compatibility.
type DatabaseCredentials struct {
	Alias    string            `json:"alias"`
	Hostname string            `json:"host"`
	Username string            `json:"user"`
	Password string            `json:"password"`
	Database string            `json:"database"`
	Port     string            `json:"port"`
	Advanced map[string]string `json:"advanced"`
	Extra    map[string]any

	IsProfile bool
	Type      string
	CustomId  string
	Source    string
}

// UnmarshalJSON supports both "advanced" and the legacy "config" JSON key.
func (d *DatabaseCredentials) UnmarshalJSON(data []byte) error {
	type Alias DatabaseCredentials
	aux := &struct {
		*Alias
		Config map[string]string `json:"config"`
	}{Alias: (*Alias)(d)}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if d.Advanced == nil && aux.Config != nil {
		d.Advanced = aux.Config
	}
	return nil
}
