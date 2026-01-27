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

// Package ssl provides SSL/TLS configuration types and utilities for database connections.
// It defines an extensible registry of SSL modes per database type, allowing EE to register
// additional modes without modifying CE code.
package ssl

import (
	"slices"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

type SSLMode string

const (
	SSLModeDisabled       SSLMode = "disabled"        // No SSL/TLS encryption
	SSLModePreferred      SSLMode = "preferred"       // Use TLS if server supports it (MySQL)
	SSLModeRequired       SSLMode = "required"        // Require TLS, skip certificate verification
	SSLModeVerifyCA       SSLMode = "verify-ca"       // Verify server certificate against CA
	SSLModeVerifyIdentity SSLMode = "verify-identity" // Verify CA + hostname (PostgreSQL: verify-full)
	SSLModeEnabled        SSLMode = "enabled"         // Simple TLS toggle with verification (CH/Mongo/Redis/ES)
	SSLModeInsecure       SSLMode = "insecure"        // TLS enabled but skip all verification
)

// Advanced option keys for SSL configuration.
// Content keys: Used by frontend (reads files client-side, sends content directly)
// Path keys: Used by server-side profiles only
const (
	KeySSLMode              = "SSL Mode"
	KeySSLCACertContent     = "SSL CA Content"
	KeySSLClientCertContent = "SSL Client Cert Content"
	KeySSLClientKeyContent  = "SSL Client Key Content"
	KeySSLServerName        = "SSL Server Name"

	// Path keys - only for profile-based connections (server-side, admin-controlled)
	KeySSLCACertPath     = "SSL CA Path"
	KeySSLClientCertPath = "SSL Client Cert Path"
	KeySSLClientKeyPath  = "SSL Client Key Path"
)

// CertificateInput holds PEM certificate content or a file path to the cert.
// Content: Used by frontend (reads files client-side, sends content via API)
// Path: Used by profile-based connections only (server reads file directly)
type CertificateInput struct {
	Content string // PEM content directly (from frontend or inline in profile)
	Path    string // File path (profile-based connections only, server reads file)
}

// SSLConfig holds all SSL-related configuration for a database connection
type SSLConfig struct {
	Mode       SSLMode          // SSL mode (required)
	CACert     CertificateInput // CA certificate for server verification
	ClientCert CertificateInput // Client certificate for mutual TLS
	ClientKey  CertificateInput // Client private key for mutual TLS
	ServerName string           // Server hostname for verification (overrides connection hostname)
}

// SSLModeInfo provides metadata for a single SSL mode, used for frontend display
type SSLModeInfo struct {
	Value       SSLMode // The mode value used in configuration
	Label       string  // Human-readable label for UI (localization key: ssl.modes.{value}.label)
	Description string  // Detailed description for tooltips (localization key: ssl.modes.{value}.description)
}

// Pre-defined mode info entries (frontend can override with localized strings using Value as key)
var (
	ModeInfoDisabled       = SSLModeInfo{SSLModeDisabled, "Disabled", "No SSL/TLS encryption"}
	ModeInfoPreferred      = SSLModeInfo{SSLModePreferred, "Preferred", "Use TLS if server supports it"}
	ModeInfoRequired       = SSLModeInfo{SSLModeRequired, "Required", "Require TLS, skip certificate verification"}
	ModeInfoVerifyCA       = SSLModeInfo{SSLModeVerifyCA, "Verify CA", "Verify server certificate against CA"}
	ModeInfoVerifyIdentity = SSLModeInfo{SSLModeVerifyIdentity, "Verify Identity", "Verify CA and server hostname"}
	ModeInfoEnabled        = SSLModeInfo{SSLModeEnabled, "Enabled", "Enable TLS with certificate verification"}
	ModeInfoInsecure       = SSLModeInfo{SSLModeInsecure, "Insecure", "Enable TLS, skip certificate verification"}
)

// Alias mappings for database-native SSL mode names
var (
	postgresAliases = map[string]SSLMode{
		"disable":     SSLModeDisabled,
		"require":     SSLModeRequired,
		"verify-full": SSLModeVerifyIdentity,
	}
	mysqlAliases = map[string]SSLMode{
		"DISABLED":        SSLModeDisabled,
		"PREFERRED":       SSLModePreferred,
		"REQUIRED":        SSLModeRequired,
		"VERIFY_CA":       SSLModeVerifyCA,
		"VERIFY_IDENTITY": SSLModeVerifyIdentity,
	}
)

// sslModeAliases maps database-native SSL mode names to our unified mode names.
var sslModeAliases = map[engine.DatabaseType]map[string]SSLMode{
	engine.DatabaseType_Postgres: postgresAliases,
	engine.DatabaseType_MySQL:    mysqlAliases,
	engine.DatabaseType_MariaDB:  mysqlAliases,
}

// Common mode sets for databases with similar SSL support
var (
	modesStandard      = []SSLModeInfo{ModeInfoDisabled, ModeInfoRequired, ModeInfoVerifyCA, ModeInfoVerifyIdentity}
	modesWithPreferred = []SSLModeInfo{ModeInfoDisabled, ModeInfoPreferred, ModeInfoRequired, ModeInfoVerifyCA, ModeInfoVerifyIdentity}
	modesSimple        = []SSLModeInfo{ModeInfoDisabled, ModeInfoEnabled, ModeInfoInsecure}
)

var (
	// databaseSSLModes holds CE database SSL modes
	databaseSSLModes = map[engine.DatabaseType][]SSLModeInfo{
		engine.DatabaseType_Postgres:      modesStandard,
		engine.DatabaseType_MySQL:         modesWithPreferred,
		engine.DatabaseType_MariaDB:       modesWithPreferred,
		engine.DatabaseType_ClickHouse:    modesSimple,
		engine.DatabaseType_MongoDB:       modesSimple,
		engine.DatabaseType_Redis:         modesSimple,
		engine.DatabaseType_ElasticSearch: modesSimple,
	}

	// additionalSSLModes holds EE-registered modes
	additionalSSLModes = make(map[engine.DatabaseType][]SSLModeInfo)
)

// RegisterDatabaseSSLModes allows EE to register SSL modes for EE databases.
func RegisterDatabaseSSLModes(dbType engine.DatabaseType, modes []SSLModeInfo) {
	additionalSSLModes[dbType] = modes
}

// RegisterSSLModeAliases allows EE to register SSL mode aliases for EE databases.
func RegisterSSLModeAliases(dbType engine.DatabaseType, aliases map[string]SSLMode) {
	sslModeAliases[dbType] = aliases
}

// GetSSLModes returns available SSL modes for a database type.
// Returns nil for database types that don't support SSL (e.g., Sqlite3).
func GetSSLModes(dbType engine.DatabaseType) []SSLModeInfo {
	// Check CE modes first
	if modes, ok := databaseSSLModes[dbType]; ok {
		return modes
	}

	// Check EE-registered modes
	if modes, ok := additionalSSLModes[dbType]; ok {
		return modes
	}

	return nil
}

// ValidateSSLMode checks if the given mode is valid for the database type.
func ValidateSSLMode(dbType engine.DatabaseType, mode SSLMode) bool {
	return slices.ContainsFunc(GetSSLModes(dbType), func(m SSLModeInfo) bool {
		return m.Value == mode
	})
}

// NormalizeSSLMode converts database-native SSL mode names to our unified names.
// For example, PostgreSQL's "require" becomes "required", "verify-full" becomes "verify-identity".
// If no alias exists, returns the original mode unchanged.
func NormalizeSSLMode(dbType engine.DatabaseType, mode string) SSLMode {
	if aliases, ok := sslModeAliases[dbType]; ok {
		if normalized, found := aliases[mode]; found {
			return normalized
		}
	}
	// No alias found, return as-is
	return SSLMode(mode)
}

// GetSSLModeAliases returns all accepted alias names for a specific SSL mode.
// For example, for PostgreSQL's "required" mode, this returns ["require"].
// This is used by the frontend to know which alternative names are accepted.
func GetSSLModeAliases(dbType engine.DatabaseType, mode SSLMode) []string {
	aliases, ok := sslModeAliases[dbType]
	if !ok {
		return nil
	}

	var result []string
	for alias, normalizedMode := range aliases {
		if normalizedMode == mode {
			result = append(result, alias)
		}
	}
	return result
}

// HasSSLSupport returns true if the database type supports SSL configuration.
func HasSSLSupport(dbType engine.DatabaseType) bool {
	return GetSSLModes(dbType) != nil
}

// IsEnabled returns true if the SSL config has a non-disabled mode.
func (c *SSLConfig) IsEnabled() bool {
	return c != nil && c.Mode != SSLModeDisabled && c.Mode != ""
}

// ParseSSLConfig extracts SSL configuration from advanced options.
// This is a shared implementation for databases using simple SSL modes (enabled/insecure/disabled).
// Parameters:
//   - dbType: database type for mode validation
//   - advanced: key-value records containing SSL settings
//   - hostname: default server name for certificate verification
//   - isProfile: if true, allows path-based certificate loading (admin-controlled)
func ParseSSLConfig(dbType engine.DatabaseType, advanced []engine.Record, hostname string, isProfile bool) *SSLConfig {
	log.Logger.Debugf("[SSL] ParseSSLConfig: received %d advanced records", len(advanced))
	for _, rec := range advanced {
		log.Logger.Debugf("[SSL] ParseSSLConfig: key=%q value=%q", rec.Key, rec.Value)
	}
	modeStr := common.GetRecordValueOrDefault(advanced, KeySSLMode, string(SSLModeDisabled))
	log.Logger.Debugf("[SSL] ParseSSLConfig: modeStr=%q (looking for key=%q)", modeStr, KeySSLMode)

	// Normalize database-native mode names
	mode := NormalizeSSLMode(dbType, modeStr)

	// Validate the normalized mode for this database
	if !ValidateSSLMode(dbType, mode) {
		return nil
	}

	if mode == SSLModeDisabled {
		return nil
	}

	config := &SSLConfig{
		Mode: mode,
		CACert: CertificateInput{
			Content: common.GetRecordValueOrDefault(advanced, KeySSLCACertContent, ""),
		},
		ClientCert: CertificateInput{
			Content: common.GetRecordValueOrDefault(advanced, KeySSLClientCertContent, ""),
		},
		ClientKey: CertificateInput{
			Content: common.GetRecordValueOrDefault(advanced, KeySSLClientKeyContent, ""),
		},
		ServerName: common.GetRecordValueOrDefault(advanced, KeySSLServerName, hostname),
	}

	// Path-based loading only for profile connections
	if isProfile {
		config.CACert.Path = common.GetRecordValueOrDefault(advanced, KeySSLCACertPath, "")
		config.ClientCert.Path = common.GetRecordValueOrDefault(advanced, KeySSLClientCertPath, "")
		config.ClientKey.Path = common.GetRecordValueOrDefault(advanced, KeySSLClientKeyPath, "")
	}

	return config
}
