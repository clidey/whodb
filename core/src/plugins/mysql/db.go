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
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/clidey/whodb/core/src/common/ssl"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"github.com/clidey/whodb/core/src/sourcedial"
)

// sourceNetwork is the MySQL driver network name that dials through the
// registered source dialer.
const sourceNetwork = "whodb-source"

var registerSourceNetwork sync.Once

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
	// The config becomes a DSN, which cannot carry a dial function, so a
	// registered dialer is reached through a named network instead.
	if sourcedial.Dialer() != nil {
		registerSourceNetwork.Do(func() {
			mysqldriver.RegisterDialContext(sourceNetwork, func(ctx context.Context, addr string) (net.Conn, error) {
				return sourcedial.Dialer()(ctx, "tcp", addr)
			})
		})
		mysqlConfig.Net = sourceNetwork
	}
	mysqlConfig.Addr = net.JoinHostPort(connectionInput.Hostname, strconv.Itoa(connectionInput.Port))
	mysqlConfig.DBName = connectionInput.Database
	mysqlConfig.AllowCleartextPasswords = connectionInput.AllowClearTextPasswords
	mysqlConfig.ParseTime = connectionInput.ParseTime
	mysqlConfig.Loc = connectionInput.Loc
	mysqlConfig.Timeout = time.Duration(connectionInput.ConnectionTimeout) * time.Second
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

	dsn := mysqlConfig.FormatDSN()
	// The DSN is parsed again on open, so a hostname carrying DSN syntax (for
	// example "x@tcp(10.0.0.5:3306") could switch the network back to plain
	// tcp and skip the registered dialer. Require the parsed DSN to keep both.
	if mysqlConfig.Net == sourceNetwork {
		parsedConfig, err := mysqldriver.ParseDSN(dsn)
		if err != nil {
			return nil, err
		}
		if parsedConfig.Net != sourceNetwork || parsedConfig.Addr != mysqlConfig.Addr {
			return nil, errors.New("invalid MySQL hostname")
		}
	}
	if !env.GetIsLocalMode() {
		// Check the effective DSN because option keys can contain parameter delimiters.
		parsedConfig, err := mysqldriver.ParseDSN(dsn)
		if err != nil {
			return nil, err
		}
		if parsedConfig.AllowAllFiles {
			return nil, errors.New("allowAllFiles is disabled in server mode")
		}
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(plugins.GetGormLogConfig())})
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
