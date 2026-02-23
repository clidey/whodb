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

package mysql

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"github.com/clidey/whodb/core/src/plugins/ssl"
	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func (p *MySQLPlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
	return p.openDB(config, false)
}

func (p *MySQLPlugin) openDB(config *engine.PluginConfig, multiStatements bool) (*gorm.DB, error) {
	connectionInput, err := p.ParseConnectionConfig(config)
	if err != nil {
		return nil, err
	}

	mysqlConfig := mysqldriver.NewConfig()
	mysqlConfig.User = connectionInput.Username
	mysqlConfig.Passwd = connectionInput.Password
	mysqlConfig.Net = "tcp"
	mysqlConfig.Addr = net.JoinHostPort(connectionInput.Hostname, strconv.Itoa(connectionInput.Port))
	mysqlConfig.DBName = connectionInput.Database
	mysqlConfig.AllowCleartextPasswords = connectionInput.AllowClearTextPasswords
	mysqlConfig.ParseTime = connectionInput.ParseTime
	mysqlConfig.Loc = connectionInput.Loc
	mysqlConfig.Params = connectionInput.ExtraOptions
	mysqlConfig.MultiStatements = multiStatements

	// Configure SSL/TLS
	sslMode := "disabled"
	if connectionInput.SSLConfig != nil && connectionInput.SSLConfig.IsEnabled() {
		sslMode = string(connectionInput.SSLConfig.Mode)

		// MySQL driver requires registering TLS configs by name
		// Handle "preferred" mode specially - it's a DSN parameter, not a registered config
		if connectionInput.SSLConfig.Mode == ssl.SSLModePreferred {
			mysqlConfig.TLSConfig = "preferred"
		} else {
			// Build and register TLS config for other modes
			tlsConfig, err := ssl.BuildTLSConfig(connectionInput.SSLConfig, connectionInput.Hostname)
			if err != nil {
				log.WithError(err).WithFields(map[string]any{
					"hostname": connectionInput.Hostname,
					"sslMode":  connectionInput.SSLConfig.Mode,
				}).Error("Failed to build TLS configuration for MySQL")
				return nil, err
			}

			// Register TLS config with unique name
			configName := fmt.Sprintf("whodb_%s_%d", connectionInput.Database, time.Now().UnixNano())
			if err := mysqldriver.RegisterTLSConfig(configName, tlsConfig); err != nil {
				log.WithError(err).WithField("configName", configName).Error("Failed to register TLS config for MySQL")
				return nil, err
			}
			mysqlConfig.TLSConfig = configName
		}
	}

	l := log.WithFields(map[string]any{
		"hostname": connectionInput.Hostname,
		"port":     connectionInput.Port,
		"database": connectionInput.Database,
		"username": connectionInput.Username,
		"sslMode":  sslMode,
	})

	db, err := gorm.Open(mysql.Open(mysqlConfig.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(plugins.GetGormLogConfig())})
	if err != nil {
		l.WithError(err).Error("Failed to connect to MySQL database")
		return nil, err
	}

	// Configure connection pool for better reconnection behavior
	if err := plugins.ConfigureConnectionPool(db); err != nil {
		l.WithError(err).Warn("Failed to configure connection pool")
	}

	return db, nil
}
