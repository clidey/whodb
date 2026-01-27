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

package ssl

import (
	"testing"

	"github.com/clidey/whodb/core/src/engine"
)

// Test helper functions - moved from ssl.go as they're only used in tests

// requiresCertificates returns true if the SSL mode requires CA certificate.
// This is a test helper, not exported from the package.
func (c *SSLConfig) requiresCertificates() bool {
	if c == nil {
		return false
	}
	switch c.Mode {
	case SSLModeVerifyCA, SSLModeVerifyIdentity:
		return true
	default:
		return false
	}
}

// requiresHostnameVerification returns true if the SSL mode requires server hostname verification.
// This is a test helper, not exported from the package.
func (c *SSLConfig) requiresHostnameVerification() bool {
	if c == nil {
		return false
	}
	return c.Mode == SSLModeVerifyIdentity
}

func TestGetSSLModes(t *testing.T) {
	tests := []struct {
		name     string
		dbType   engine.DatabaseType
		wantNil  bool
		minModes int
	}{
		{
			name:     "Postgres has SSL modes",
			dbType:   engine.DatabaseType_Postgres,
			wantNil:  false,
			minModes: 4,
		},
		{
			name:     "MySQL has SSL modes",
			dbType:   engine.DatabaseType_MySQL,
			wantNil:  false,
			minModes: 5,
		},
		{
			name:     "MariaDB has SSL modes",
			dbType:   engine.DatabaseType_MariaDB,
			wantNil:  false,
			minModes: 5,
		},
		{
			name:     "ClickHouse has SSL modes",
			dbType:   engine.DatabaseType_ClickHouse,
			wantNil:  false,
			minModes: 3,
		},
		{
			name:     "MongoDB has SSL modes",
			dbType:   engine.DatabaseType_MongoDB,
			wantNil:  false,
			minModes: 3,
		},
		{
			name:     "Redis has SSL modes",
			dbType:   engine.DatabaseType_Redis,
			wantNil:  false,
			minModes: 3,
		},
		{
			name:     "Elasticsearch has SSL modes",
			dbType:   engine.DatabaseType_ElasticSearch,
			wantNil:  false,
			minModes: 3,
		},
		{
			name:    "Sqlite3 has no SSL modes",
			dbType:  engine.DatabaseType_Sqlite3,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modes := GetSSLModes(tt.dbType)
			if tt.wantNil {
				if modes != nil {
					t.Errorf("GetSSLModes(%s) = %v, want nil", tt.dbType, modes)
				}
			} else {
				if modes == nil {
					t.Errorf("GetSSLModes(%s) = nil, want non-nil", tt.dbType)
				} else if len(modes) < tt.minModes {
					t.Errorf("GetSSLModes(%s) returned %d modes, want at least %d", tt.dbType, len(modes), tt.minModes)
				}
			}
		})
	}
}

func TestGetSSLModesContainsDisabled(t *testing.T) {
	// All databases with SSL support should have a disabled option
	dbTypes := []engine.DatabaseType{
		engine.DatabaseType_Postgres,
		engine.DatabaseType_MySQL,
		engine.DatabaseType_MariaDB,
		engine.DatabaseType_ClickHouse,
		engine.DatabaseType_MongoDB,
		engine.DatabaseType_Redis,
		engine.DatabaseType_ElasticSearch,
	}

	for _, dbType := range dbTypes {
		t.Run(string(dbType), func(t *testing.T) {
			modes := GetSSLModes(dbType)
			hasDisabled := false
			for _, m := range modes {
				if m.Value == SSLModeDisabled {
					hasDisabled = true
					break
				}
			}
			if !hasDisabled {
				t.Errorf("GetSSLModes(%s) does not contain disabled mode", dbType)
			}
		})
	}
}

func TestValidateSSLMode(t *testing.T) {
	tests := []struct {
		name   string
		dbType engine.DatabaseType
		mode   SSLMode
		want   bool
	}{
		// Postgres modes
		{"Postgres disabled valid", engine.DatabaseType_Postgres, SSLModeDisabled, true},
		{"Postgres required valid", engine.DatabaseType_Postgres, SSLModeRequired, true},
		{"Postgres verify-ca valid", engine.DatabaseType_Postgres, SSLModeVerifyCA, true},
		{"Postgres verify-identity valid", engine.DatabaseType_Postgres, SSLModeVerifyIdentity, true},
		{"Postgres preferred invalid", engine.DatabaseType_Postgres, SSLModePreferred, false},

		// MySQL modes
		{"MySQL disabled valid", engine.DatabaseType_MySQL, SSLModeDisabled, true},
		{"MySQL preferred valid", engine.DatabaseType_MySQL, SSLModePreferred, true},
		{"MySQL required valid", engine.DatabaseType_MySQL, SSLModeRequired, true},
		{"MySQL insecure invalid", engine.DatabaseType_MySQL, SSLModeInsecure, false},

		// ClickHouse modes
		{"ClickHouse disabled valid", engine.DatabaseType_ClickHouse, SSLModeDisabled, true},
		{"ClickHouse enabled valid", engine.DatabaseType_ClickHouse, SSLModeEnabled, true},
		{"ClickHouse insecure valid", engine.DatabaseType_ClickHouse, SSLModeInsecure, true},
		{"ClickHouse verify-ca invalid", engine.DatabaseType_ClickHouse, SSLModeVerifyCA, false},

		// SQLite (no SSL support)
		{"Sqlite3 disabled invalid", engine.DatabaseType_Sqlite3, SSLModeDisabled, false},
		{"Sqlite3 enabled invalid", engine.DatabaseType_Sqlite3, SSLModeEnabled, false},

		// Invalid mode
		{"Unknown mode", engine.DatabaseType_Postgres, SSLMode("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateSSLMode(tt.dbType, tt.mode)
			if got != tt.want {
				t.Errorf("ValidateSSLMode(%s, %s) = %v, want %v", tt.dbType, tt.mode, got, tt.want)
			}
		})
	}
}

func TestHasSSLSupport(t *testing.T) {
	tests := []struct {
		name   string
		dbType engine.DatabaseType
		want   bool
	}{
		{"Postgres supports SSL", engine.DatabaseType_Postgres, true},
		{"MySQL supports SSL", engine.DatabaseType_MySQL, true},
		{"MariaDB supports SSL", engine.DatabaseType_MariaDB, true},
		{"ClickHouse supports SSL", engine.DatabaseType_ClickHouse, true},
		{"MongoDB supports SSL", engine.DatabaseType_MongoDB, true},
		{"Redis supports SSL", engine.DatabaseType_Redis, true},
		{"Elasticsearch supports SSL", engine.DatabaseType_ElasticSearch, true},
		{"Sqlite3 no SSL support", engine.DatabaseType_Sqlite3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasSSLSupport(tt.dbType)
			if got != tt.want {
				t.Errorf("HasSSLSupport(%s) = %v, want %v", tt.dbType, got, tt.want)
			}
		})
	}
}

func TestSSLConfigIsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		config *SSLConfig
		want   bool
	}{
		{"nil config", nil, false},
		{"empty mode", &SSLConfig{Mode: ""}, false},
		{"disabled mode", &SSLConfig{Mode: SSLModeDisabled}, false},
		{"required mode", &SSLConfig{Mode: SSLModeRequired}, true},
		{"verify-ca mode", &SSLConfig{Mode: SSLModeVerifyCA}, true},
		{"verify-identity mode", &SSLConfig{Mode: SSLModeVerifyIdentity}, true},
		{"enabled mode", &SSLConfig{Mode: SSLModeEnabled}, true},
		{"insecure mode", &SSLConfig{Mode: SSLModeInsecure}, true},
		{"preferred mode", &SSLConfig{Mode: SSLModePreferred}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.IsEnabled()
			if got != tt.want {
				t.Errorf("SSLConfig.IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSSLConfigRequiresCertificates(t *testing.T) {
	tests := []struct {
		name   string
		config *SSLConfig
		want   bool
	}{
		{"nil config", nil, false},
		{"disabled mode", &SSLConfig{Mode: SSLModeDisabled}, false},
		{"required mode", &SSLConfig{Mode: SSLModeRequired}, false},
		{"enabled mode", &SSLConfig{Mode: SSLModeEnabled}, false},
		{"insecure mode", &SSLConfig{Mode: SSLModeInsecure}, false},
		{"verify-ca mode", &SSLConfig{Mode: SSLModeVerifyCA}, true},
		{"verify-identity mode", &SSLConfig{Mode: SSLModeVerifyIdentity}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.requiresCertificates()
			if got != tt.want {
				t.Errorf("SSLConfig.requiresCertificates() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSSLConfigRequiresHostnameVerification(t *testing.T) {
	tests := []struct {
		name   string
		config *SSLConfig
		want   bool
	}{
		{"nil config", nil, false},
		{"disabled mode", &SSLConfig{Mode: SSLModeDisabled}, false},
		{"required mode", &SSLConfig{Mode: SSLModeRequired}, false},
		{"verify-ca mode", &SSLConfig{Mode: SSLModeVerifyCA}, false},
		{"verify-identity mode", &SSLConfig{Mode: SSLModeVerifyIdentity}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.requiresHostnameVerification()
			if got != tt.want {
				t.Errorf("SSLConfig.requiresHostnameVerification() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegisterDatabaseSSLModes(t *testing.T) {
	// Use a fake database type for testing
	fakeDBType := engine.DatabaseType("TestDB")

	// Verify it doesn't exist initially
	if modes := GetSSLModes(fakeDBType); modes != nil {
		t.Fatalf("GetSSLModes(%s) should return nil before registration", fakeDBType)
	}

	// Register modes
	testModes := []SSLModeInfo{
		{SSLModeDisabled, "Off", "No encryption"},
		{SSLModeEnabled, "On", "Full encryption"},
	}
	RegisterDatabaseSSLModes(fakeDBType, testModes)

	// Verify registration worked
	modes := GetSSLModes(fakeDBType)
	if modes == nil {
		t.Fatalf("GetSSLModes(%s) returned nil after registration", fakeDBType)
	}
	if len(modes) != 2 {
		t.Errorf("GetSSLModes(%s) returned %d modes, want 2", fakeDBType, len(modes))
	}

	// Verify mode values
	if modes[0].Value != SSLModeDisabled {
		t.Errorf("First mode value = %s, want %s", modes[0].Value, SSLModeDisabled)
	}
	if modes[1].Value != SSLModeEnabled {
		t.Errorf("Second mode value = %s, want %s", modes[1].Value, SSLModeEnabled)
	}

	// Cleanup
	delete(additionalSSLModes, fakeDBType)
}

func TestNormalizeSSLMode(t *testing.T) {
	tests := []struct {
		name   string
		dbType engine.DatabaseType
		input  string
		want   SSLMode
	}{
		// PostgreSQL aliases
		{"Postgres disable -> disabled", engine.DatabaseType_Postgres, "disable", SSLModeDisabled},
		{"Postgres require -> required", engine.DatabaseType_Postgres, "require", SSLModeRequired},
		{"Postgres verify-full -> verify-identity", engine.DatabaseType_Postgres, "verify-full", SSLModeVerifyIdentity},
		{"Postgres verify-ca unchanged", engine.DatabaseType_Postgres, "verify-ca", SSLModeVerifyCA},
		{"Postgres disabled unchanged", engine.DatabaseType_Postgres, "disabled", SSLModeDisabled},

		// MySQL aliases
		{"MySQL DISABLED -> disabled", engine.DatabaseType_MySQL, "DISABLED", SSLModeDisabled},
		{"MySQL REQUIRED -> required", engine.DatabaseType_MySQL, "REQUIRED", SSLModeRequired},
		{"MySQL PREFERRED -> preferred", engine.DatabaseType_MySQL, "PREFERRED", SSLModePreferred},
		{"MySQL VERIFY_CA -> verify-ca", engine.DatabaseType_MySQL, "VERIFY_CA", SSLModeVerifyCA},
		{"MySQL VERIFY_IDENTITY -> verify-identity", engine.DatabaseType_MySQL, "VERIFY_IDENTITY", SSLModeVerifyIdentity},

		// MariaDB (same as MySQL)
		{"MariaDB REQUIRED -> required", engine.DatabaseType_MariaDB, "REQUIRED", SSLModeRequired},

		// No aliases for simple-mode databases
		{"ClickHouse enabled unchanged", engine.DatabaseType_ClickHouse, "enabled", SSLModeEnabled},
		{"MongoDB insecure unchanged", engine.DatabaseType_MongoDB, "insecure", SSLModeInsecure},

		// Unknown mode passed through
		{"Unknown mode unchanged", engine.DatabaseType_Postgres, "unknown-mode", SSLMode("unknown-mode")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeSSLMode(tt.dbType, tt.input)
			if got != tt.want {
				t.Errorf("NormalizeSSLMode(%s, %q) = %q, want %q", tt.dbType, tt.input, got, tt.want)
			}
		})
	}
}

func TestGetSSLModeAliases(t *testing.T) {
	tests := []struct {
		name      string
		dbType    engine.DatabaseType
		mode      SSLMode
		wantCount int
		wantAlias string // one expected alias to check
	}{
		{"Postgres required has alias", engine.DatabaseType_Postgres, SSLModeRequired, 1, "require"},
		{"Postgres verify-identity has alias", engine.DatabaseType_Postgres, SSLModeVerifyIdentity, 1, "verify-full"},
		{"Postgres disabled has alias", engine.DatabaseType_Postgres, SSLModeDisabled, 1, "disable"},
		{"Postgres verify-ca no alias", engine.DatabaseType_Postgres, SSLModeVerifyCA, 0, ""},
		{"MySQL required has alias", engine.DatabaseType_MySQL, SSLModeRequired, 1, "REQUIRED"},
		{"ClickHouse no aliases", engine.DatabaseType_ClickHouse, SSLModeEnabled, 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aliases := GetSSLModeAliases(tt.dbType, tt.mode)
			if len(aliases) != tt.wantCount {
				t.Errorf("GetSSLModeAliases(%s, %s) returned %d aliases, want %d", tt.dbType, tt.mode, len(aliases), tt.wantCount)
			}
			if tt.wantAlias != "" {
				found := false
				for _, a := range aliases {
					if a == tt.wantAlias {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GetSSLModeAliases(%s, %s) = %v, want to contain %q", tt.dbType, tt.mode, aliases, tt.wantAlias)
				}
			}
		})
	}
}

func TestSSLModeInfoFields(t *testing.T) {
	// Verify that all registered modes have non-empty fields
	dbTypes := []engine.DatabaseType{
		engine.DatabaseType_Postgres,
		engine.DatabaseType_MySQL,
		engine.DatabaseType_MariaDB,
		engine.DatabaseType_ClickHouse,
		engine.DatabaseType_MongoDB,
		engine.DatabaseType_Redis,
		engine.DatabaseType_ElasticSearch,
	}

	for _, dbType := range dbTypes {
		t.Run(string(dbType), func(t *testing.T) {
			modes := GetSSLModes(dbType)
			for _, mode := range modes {
				if mode.Value == "" {
					t.Errorf("Mode in %s has empty Value", dbType)
				}
				if mode.Label == "" {
					t.Errorf("Mode %s in %s has empty Label", mode.Value, dbType)
				}
				if mode.Description == "" {
					t.Errorf("Mode %s in %s has empty Description", mode.Value, dbType)
				}
			}
		})
	}
}
