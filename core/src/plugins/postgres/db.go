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
	"github.com/clidey/whodb/core/src/engine"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func (p *PostgresPlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
	connectionInput, err := p.ParseConnectionConfig(config)
	if err != nil {
		return nil, err
	}

	pgxConfig, err := pgx.ParseConfig("")
	if err != nil {
		return nil, err
	}

	pgxConfig.Host = connectionInput.Hostname
	pgxConfig.Port = uint16(connectionInput.Port)
	pgxConfig.User = connectionInput.Username
	pgxConfig.Password = connectionInput.Password
	pgxConfig.Database = connectionInput.Database

	if connectionInput.ExtraOptions != nil {
		if pgxConfig.RuntimeParams == nil {
			pgxConfig.RuntimeParams = make(map[string]string)
		}
		for key, value := range connectionInput.ExtraOptions {
			pgxConfig.RuntimeParams[key] = value
		}
	}

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: stdlib.OpenDB(*pgxConfig)}), &gorm.Config{})

	if err != nil {
		return nil, err
	}
	return db, nil
}