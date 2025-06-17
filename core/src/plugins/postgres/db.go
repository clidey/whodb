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
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func (p *PostgresPlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
	connectionInput, err := p.ParseConnectionConfig(config)
	if err != nil {
		return nil, err
	}

	// Use key-value DSN format to prevent SQL injection
	// This format properly separates parameters and prevents injection
	dsnParts := []string{
		fmt.Sprintf("host=%s", connectionInput.Hostname),
		fmt.Sprintf("user=%s", connectionInput.Username),
		fmt.Sprintf("password=%s", connectionInput.Password),
		fmt.Sprintf("dbname=%s", connectionInput.Database),
		fmt.Sprintf("port=%d", connectionInput.Port),
		"sslmode=prefer",
	}

	// Add extra options safely with validation
	if connectionInput.ExtraOptions != nil {
		for key, value := range connectionInput.ExtraOptions {
			if isValidPostgresParam(key) {
				dsnParts = append(dsnParts, fmt.Sprintf("%s=%s", strings.ToLower(key), value))
			}
		}
	}

	dsn := strings.Join(dsnParts, " ")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// isValidPostgresParam validates PostgreSQL connection parameter names
// to prevent injection of arbitrary DSN parameters
func isValidPostgresParam(key string) bool {
	validParams := map[string]bool{
		"sslmode":           true,
		"sslcert":           true,
		"sslkey":            true,
		"sslrootcert":       true,
		"sslcrl":            true,
		"application_name":  true,
		"connect_timeout":   true,
		"search_path":       true,
		"timezone":          true,
		"statement_timeout": true,
		"lock_timeout":      true,
		"idle_in_transaction_session_timeout": true,
	}
	return validParams[strings.ToLower(key)]
}
