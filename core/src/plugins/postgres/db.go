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

package postgres

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"

	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// validateInput uses allowlist validation to ensure only safe characters are used
// This prevents all forms of injection and path traversal attacks
func validateInput(input, inputType string) error {
	if len(input) == 0 {
		return fmt.Errorf("%s cannot be empty", inputType)
	}
	
	// Define allowlist patterns for different input types
	var allowedPattern *regexp.Regexp
	var maxLength int
	
	switch inputType {
	case "database":
		// Database names: only alphanumeric, underscore, hyphen (no dots to prevent traversal)
		allowedPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
		maxLength = 63 // PostgreSQL database name limit
	case "hostname":
		// Hostnames: alphanumeric, dots, hyphens (standard hostname characters)
		allowedPattern = regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
		maxLength = 253 // RFC hostname limit
	case "username":
		// Usernames: alphanumeric and underscore only
		allowedPattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
		maxLength = 63 // PostgreSQL username limit
	default:
		return fmt.Errorf("unknown input type: %s", inputType)
	}
	
	// Check length
	if len(input) > maxLength {
		return fmt.Errorf("%s too long: maximum %d characters", inputType, maxLength)
	}
	
	// Check against allowlist pattern
	if !allowedPattern.MatchString(input) {
		return fmt.Errorf("invalid %s: contains disallowed characters (only alphanumeric, underscore, hyphen allowed)", inputType)
	}
	
	return nil
}

func (p *PostgresPlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
	connectionInput, err := p.ParseConnectionConfig(config)
	if err != nil {
		return nil, err
	}

	// Validate all connection parameters using allowlist validation
	if err := validateInput(connectionInput.Hostname, "hostname"); err != nil {
		return nil, fmt.Errorf("hostname validation failed: %w", err)
	}

	if err := validateInput(connectionInput.Database, "database"); err != nil {
		return nil, fmt.Errorf("database validation failed: %w", err)
	}

	if err := validateInput(connectionInput.Username, "username"); err != nil {
		return nil, fmt.Errorf("username validation failed: %w", err)
	}

	// Construct PostgreSQL URL securely using url.URL struct
	u := &url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(connectionInput.Username, connectionInput.Password),
		Host:   net.JoinHostPort(connectionInput.Hostname, strconv.Itoa(connectionInput.Port)),
		Path:   "/" + connectionInput.Database,
	}

	// Add query parameters securely
	q := u.Query()
	q.Set("sslmode", "prefer")
	
	// Validate and add extra options as query parameters (allowlist approach)
	if connectionInput.ExtraOptions != nil {
		allowedOptions := map[string]bool{
			"sslmode":     true,
			"sslcert":     true,
			"sslkey":      true,
			"sslrootcert": true,
			"connect_timeout": true,
			"application_name": true,
		}
		
		for key, value := range connectionInput.ExtraOptions {
			// Only allow predefined safe options
			if !allowedOptions[key] {
				return nil, fmt.Errorf("extra option '%s' is not allowed for security reasons", key)
			}
			
			// Validate option values using basic allowlist (no special characters)
			if !regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`).MatchString(value) {
				return nil, fmt.Errorf("extra option value for '%s' contains invalid characters", key)
			}
			
			q.Set(key, value)
		}
	}
	
	u.RawQuery = q.Encode()
	dsn := u.String()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

