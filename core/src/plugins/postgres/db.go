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
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// validateHostname ensures the hostname doesn't contain URL-reserved characters
// that could lead to injection attacks
func validateHostname(hostname string) error {
	// Check for URL-reserved characters that could enable injection
	invalidChars := []string{"@", "?", "#", "/", "\\"}
	for _, char := range invalidChars {
		if strings.Contains(hostname, char) {
			return fmt.Errorf("invalid hostname: contains URL-reserved character '%s'", char)
		}
	}
	return nil
}

// validateDatabase ensures the database name doesn't contain URL-encoded characters
// or patterns that could lead to path traversal attacks
func validateDatabase(database string) error {
	// Check for URL-encoded forward slashes that could enable path traversal
	if strings.Contains(database, "%2f") || strings.Contains(database, "%2F") {
		return fmt.Errorf("invalid database name: contains URL-encoded forward slash")
	}
	
	// Check for literal path traversal patterns
	if strings.Contains(database, "../") || strings.Contains(database, "..\\") {
		return fmt.Errorf("invalid database name: contains path traversal pattern")
	}
	
	// Check for other URL-encoded characters that could be problematic
	problematicEncoded := []string{"%00", "%20", "%22", "%27", "%3B", "%3C", "%3E"}
	for _, encoded := range problematicEncoded {
		if strings.Contains(strings.ToLower(database), encoded) {
			return fmt.Errorf("invalid database name: contains URL-encoded character '%s'", encoded)
		}
	}
	
	return nil
}

func (p *PostgresPlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
	connectionInput, err := p.ParseConnectionConfig(config)
	if err != nil {
		return nil, err
	}

	// Validate hostname to prevent injection attacks
	if err := validateHostname(connectionInput.Hostname); err != nil {
		return nil, err
	}

	// Validate database name to prevent path traversal attacks
	if err := validateDatabase(connectionInput.Database); err != nil {
		return nil, err
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
	
	// Add extra options as query parameters
	if connectionInput.ExtraOptions != nil {
		for key, value := range connectionInput.ExtraOptions {
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

