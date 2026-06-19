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

package platform

import "strings"

const redactedSourceValue = "********"

// RedactedSourceConfig is a source connection configuration with secrets masked.
type RedactedSourceConfig struct {
	Hostname string            `json:"hostname"`
	Port     string            `json:"port"`
	Username string            `json:"username"`
	Password string            `json:"password"`
	Database string            `json:"database"`
	Advanced map[string]string `json:"advanced"`
}

// RedactedValue returns the placeholder used for masked source secrets.
func RedactedValue() string {
	return redactedSourceValue
}

// MergeSourceConfig applies explicit field updates to an existing source config.
func MergeSourceConfig(existing *SourceConfig, values map[string]string, advanced map[string]string) SourceConfig {
	merged := SourceConfig{}
	if existing != nil {
		merged = *existing
		merged.Advanced = map[string]string{}
		for key, value := range existing.Advanced {
			merged.Advanced[key] = value
		}
	}
	if merged.Advanced == nil {
		merged.Advanced = map[string]string{}
	}
	for key, value := range values {
		AssignSourceConfigField(&merged, key, value)
	}
	for key, value := range advanced {
		merged.Advanced[key] = value
	}
	return merged
}

// AssignSourceConfigField maps a public source field name into a source config.
func AssignSourceConfigField(config *SourceConfig, key, value string) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "hostname":
		config.Hostname = value
	case "port":
		config.Port = value
	case "username":
		config.Username = value
	case "password":
		config.Password = value
	case "database":
		config.Database = value
	default:
		if config.Advanced == nil {
			config.Advanced = map[string]string{}
		}
		config.Advanced[key] = value
	}
}

// RedactSourceConfig masks secrets in source connection configuration.
func RedactSourceConfig(config *SourceConfig, sourceType *SourceType) RedactedSourceConfig {
	if config == nil {
		return RedactedSourceConfig{Advanced: map[string]string{}}
	}
	safe := RedactedSourceConfig{
		Hostname: config.Hostname,
		Port:     config.Port,
		Username: config.Username,
		Password: RedactSourceValue("Password", config.Password, sourceType),
		Database: config.Database,
		Advanced: map[string]string{},
	}
	for key, value := range config.Advanced {
		safe.Advanced[key] = RedactSourceValue(key, value, sourceType)
	}
	return safe
}

// RedactSourceValue masks a value when its source field is secret.
func RedactSourceValue(key, value string, sourceType *SourceType) string {
	if value == "" {
		return ""
	}
	if SourceConfigFieldSecret(key, sourceType) {
		return redactedSourceValue
	}
	return value
}

// SourceConfigFieldSecret reports whether a source config field contains secret material.
func SourceConfigFieldSecret(key string, sourceType *SourceType) bool {
	if strings.EqualFold(key, "Password") {
		return true
	}
	if sourceType != nil {
		for _, field := range sourceType.ConnectionFields {
			if strings.EqualFold(field.Key, key) {
				return SourceFieldSecret(field)
			}
		}
	}
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "_", " "))
	for _, part := range []string{"password", "secret", "token", "private key"} {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	return false
}

// SourceFieldSecret reports whether a source type field should be treated as secret.
func SourceFieldSecret(field SourceConnectionField) bool {
	if strings.EqualFold(field.Kind, "Password") {
		return true
	}
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(field.Key), "_", " "))
	for _, part := range []string{"password", "secret", "token", "private key"} {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	return false
}
