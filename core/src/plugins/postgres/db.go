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
	"net"
	"net/url"
	"strconv"

	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func (p *PostgresPlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
	connectionInput, err := p.ParseConnectionConfig(config)
	if err != nil {
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

