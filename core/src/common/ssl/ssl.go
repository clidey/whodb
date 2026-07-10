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

// Package ssl provides SSL/TLS configuration types and source-backed
// normalization helpers for database connections.
package ssl

import (
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/sourcecatalog"
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

	// KeySSLCACertPath is used by profile-based, server-side connections.
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
	ModeInfoDisabled       = SSLModeInfo{Value: SSLModeDisabled, Label: "Disabled", Description: "No SSL/TLS encryption"}
	ModeInfoPreferred      = SSLModeInfo{Value: SSLModePreferred, Label: "Preferred", Description: "Use TLS if server supports it"}
	ModeInfoRequired       = SSLModeInfo{Value: SSLModeRequired, Label: "Required", Description: "Require TLS, skip certificate verification"}
	ModeInfoVerifyCA       = SSLModeInfo{Value: SSLModeVerifyCA, Label: "Verify CA", Description: "Verify server certificate against CA"}
	ModeInfoVerifyIdentity = SSLModeInfo{Value: SSLModeVerifyIdentity, Label: "Verify Identity", Description: "Verify CA and server hostname"}
	ModeInfoEnabled        = SSLModeInfo{Value: SSLModeEnabled, Label: "Enabled", Description: "Enable TLS with certificate verification"}
	ModeInfoInsecure       = SSLModeInfo{Value: SSLModeInsecure, Label: "Insecure", Description: "Enable TLS, skip certificate verification"}
)

// GetSSLModes returns the source-declared SSL modes for a database type.
// Returns nil for database types that do not expose SSL configuration.
func GetSSLModes(dbType engine.DatabaseType) []SSLModeInfo {
	spec, ok := sourcecatalog.Find(string(dbType))
	if !ok || len(spec.SSLModes) == 0 {
		return nil
	}

	modes := make([]SSLModeInfo, 0, len(spec.SSLModes))
	for _, mode := range spec.SSLModes {
		modes = append(modes, SSLModeInfo{
			Value:       SSLMode(mode.Value),
			Label:       mode.Label,
			Description: mode.Description,
		})
	}
	return modes
}

// ValidateSSLMode checks if the given mode is valid for the database type.
func ValidateSSLMode(dbType engine.DatabaseType, mode SSLMode) bool {
	for _, item := range GetSSLModes(dbType) {
		if item.Value == mode {
			return true
		}
	}
	return false
}

// NormalizeSSLMode converts database-native SSL mode names to our unified names.
// For example, PostgreSQL's "require" becomes "required", "verify-full" becomes "verify-identity".
// If no alias exists, returns the original mode unchanged.
func NormalizeSSLMode(dbType engine.DatabaseType, mode string) SSLMode {
	trimmed := strings.TrimSpace(mode)
	spec, ok := sourcecatalog.Find(string(dbType))
	if !ok {
		return SSLMode(trimmed)
	}

	for _, item := range spec.SSLModes {
		if strings.EqualFold(trimmed, item.Value) {
			return SSLMode(item.Value)
		}
		for _, alias := range item.Aliases {
			if strings.EqualFold(trimmed, alias) {
				return SSLMode(item.Value)
			}
		}
	}

	return SSLMode(trimmed)
}

// GetSSLModeAliases returns all accepted alias names for a specific SSL mode.
// For example, for PostgreSQL's "required" mode, this returns ["require"].
// This is used by the frontend to know which alternative names are accepted.
func GetSSLModeAliases(dbType engine.DatabaseType, mode SSLMode) []string {
	spec, ok := sourcecatalog.Find(string(dbType))
	if !ok {
		return nil
	}

	for _, item := range spec.SSLModes {
		if item.Value == string(mode) {
			return append([]string(nil), item.Aliases...)
		}
	}
	return nil
}

// HasSSLSupport returns true if the database type supports SSL configuration.
func HasSSLSupport(dbType engine.DatabaseType) bool {
	return len(GetSSLModes(dbType)) > 0
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
	log.Debugf("[SSL] ParseSSLConfig: received %d advanced records", len(advanced))
	for _, rec := range advanced {
		// Log only the key and value length. Advanced records include certificate
		// and private-key PEM content (e.g. "SSL Client Key Content"), which must
		// never be written to logs.
		log.Debugf("[SSL] ParseSSLConfig: key=%q valueLen=%d", rec.Key, len(rec.Value))
	}
	modeStr := common.GetRecordValueOrDefault(advanced, KeySSLMode, string(SSLModeDisabled))
	log.Debugf("[SSL] ParseSSLConfig: modeStr=%q (looking for key=%q)", modeStr, KeySSLMode)

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
