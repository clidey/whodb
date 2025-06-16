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

func escapeConnectionParam(x string) string {
	// PostgreSQL libpq connection string escaping rules:
	// 1. Single quotes must be doubled: ' -> ''
	// 2. Backslashes must be doubled: \ -> \\
	x = strings.ReplaceAll(x, "\\", "\\\\")
	x = strings.ReplaceAll(x, "'", "''")
	return x
}

func validateConnectionParam(param, paramName string) error {
	// Check for null bytes which can terminate the connection string
	if strings.Contains(param, "\x00") {
		return fmt.Errorf("invalid %s: contains null byte", paramName)
	}
	
	// Check for potentially dangerous control characters
	for _, char := range param {
		if char < 32 && char != '\t' && char != '\n' && char != '\r' {
			return fmt.Errorf("invalid %s: contains control character", paramName)
		}
	}
	
	return nil
}

func isValidConnectionParamKey(key string) bool {
	// Connection parameter keys should only contain alphanumeric characters and underscores
	for _, char := range key {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || 
			 (char >= '0' && char <= '9') || char == '_') {
			return false
		}
	}
	return len(key) > 0
}

func (p *PostgresPlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
	connectionInput, err := p.ParseConnectionConfig(config)
	if err != nil {
		return nil, err
	}

	// Validate connection parameters for security
	if err := validateConnectionParam(connectionInput.Hostname, "hostname"); err != nil {
		return nil, err
	}
	if err := validateConnectionParam(connectionInput.Username, "username"); err != nil {
		return nil, err
	}
	if err := validateConnectionParam(connectionInput.Password, "password"); err != nil {
		return nil, err
	}
	if err := validateConnectionParam(connectionInput.Database, "database"); err != nil {
		return nil, err
	}

	host := escapeConnectionParam(connectionInput.Hostname)
	username := escapeConnectionParam(connectionInput.Username)
	password := escapeConnectionParam(connectionInput.Password)
	database := escapeConnectionParam(connectionInput.Database)

	params := strings.Builder{}
	if connectionInput.ExtraOptions != nil {
		for key, value := range connectionInput.ExtraOptions {
			// Validate extra option values
			if err := validateConnectionParam(value, fmt.Sprintf("extra option '%s'", key)); err != nil {
				return nil, err
			}
			// Validate key names (should only contain alphanumeric characters and underscores)
			if !isValidConnectionParamKey(key) {
				return nil, fmt.Errorf("invalid extra option key '%s': only alphanumeric characters and underscores allowed", key)
			}
			params.WriteString(fmt.Sprintf("%v='%v' ", strings.ToLower(key), escapeConnectionParam(value)))
		}
	}

	dsn := fmt.Sprintf("host='%v' user='%v' password='%v' dbname='%v' port='%v'",
		host, username, password, database, connectionInput.Port)

	if params.Len() > 0 {
		dsn = fmt.Sprintf("%v %v", dsn, params.String())
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
