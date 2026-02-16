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

package postgres

import (
	"maps"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"github.com/clidey/whodb/core/src/plugins/ssl"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func (p *PostgresPlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
	return p.openDB(config, false)
}

func (p *PostgresPlugin) openDB(config *engine.PluginConfig, multiStatement bool) (*gorm.DB, error) {
	connectionInput, err := p.ParseConnectionConfig(config)
	if err != nil {
		return nil, err
	}

	pgxConfig, err := pgx.ParseConfig("")
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]any{
			"hostname": connectionInput.Hostname,
			"port":     connectionInput.Port,
			"database": connectionInput.Database,
			"username": connectionInput.Username,
		}).Error("Failed to parse PostgreSQL configuration")
		return nil, err
	}

	pgxConfig.Host = connectionInput.Hostname
	pgxConfig.Port = uint16(connectionInput.Port)
	pgxConfig.User = connectionInput.Username
	pgxConfig.Password = connectionInput.Password
	pgxConfig.Database = connectionInput.Database

	// Use simpler protocol for multi-statement SQL scripts
	if multiStatement {
		pgxConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	}

	// Configure SSL/TLS
	sslMode := "disabled"
	if connectionInput.SSLConfig != nil && connectionInput.SSLConfig.IsEnabled() {
		sslMode = string(connectionInput.SSLConfig.Mode)
		tlsConfig, err := ssl.BuildTLSConfig(connectionInput.SSLConfig, connectionInput.Hostname)
		if err != nil {
			log.Logger.WithError(err).WithFields(map[string]any{
				"hostname": connectionInput.Hostname,
				"sslMode":  connectionInput.SSLConfig.Mode,
			}).Error("Failed to build TLS configuration for PostgreSQL")
			return nil, err
		}
		pgxConfig.TLSConfig = tlsConfig
	}

	if connectionInput.ExtraOptions != nil {
		if pgxConfig.RuntimeParams == nil {
			pgxConfig.RuntimeParams = make(map[string]string)
		}
		maps.Copy(pgxConfig.RuntimeParams, connectionInput.ExtraOptions)
	}

	l := log.Logger.WithFields(map[string]any{
		"hostname": connectionInput.Hostname,
		"port":     connectionInput.Port,
		"database": connectionInput.Database,
		"username": connectionInput.Username,
		"sslMode":  sslMode,
	})

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: stdlib.OpenDB(*pgxConfig)}), &gorm.Config{Logger: logger.Default.LogMode(plugins.GetGormLogConfig())})
	if err != nil {
		l.WithError(err).Error("Failed to connect to PostgreSQL database")
		return nil, err
	}

	// Configure connection pool for better reconnection behavior
	if err := plugins.ConfigureConnectionPool(db); err != nil {
		l.WithError(err).Warn("Failed to configure connection pool")
	}

	return db, nil
}
