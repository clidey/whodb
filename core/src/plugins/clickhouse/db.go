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

package clickhouse

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"github.com/clidey/whodb/core/src/plugins/ssl"
	gorm_clickhouse "gorm.io/driver/clickhouse"
)

func (p *ClickHousePlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
	connectionInput, err := p.ParseConnectionConfig(config)
	if err != nil {
		return nil, err
	}

	auth := clickhouse.Auth{
		Database: connectionInput.Database,
		Username: connectionInput.Username,
		Password: connectionInput.Password,
	}

	address := []string{net.JoinHostPort(connectionInput.Hostname, strconv.Itoa(connectionInput.Port))}
	options := &clickhouse.Options{
		Addr:             address,
		Auth:             auth,
		DialTimeout:      time.Second * 30,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	}

	if connectionInput.HTTPProtocol != "disable" {
		options.Protocol = clickhouse.HTTP
		options.Compression = &clickhouse.Compression{
			Method: clickhouse.CompressionGZIP,
		}
	}

	if connectionInput.Debug != "disable" {
		options.Debug = true
	}

	switch connectionInput.ReadOnly {
	case "disable":
		options.Settings = clickhouse.Settings{
			"readonly": 0,
		}
	case "enable":
		options.Settings = clickhouse.Settings{
			"readonly": 1,
		}
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
			}).Error("Failed to build TLS configuration for ClickHouse")
			return nil, err
		}
		options.TLS = tlsConfig
	}

	conn := clickhouse.OpenDB(options)

	l := log.Logger.WithFields(map[string]any{
		"hostname": connectionInput.Hostname,
		"port":     connectionInput.Port,
		"database": connectionInput.Database,
		"username": connectionInput.Username,
		"sslMode":  sslMode,
		"protocol": func() string {
			if connectionInput.HTTPProtocol != "disable" {
				return "HTTP"
			}
			return "Native"
		}(),
	})

	err = conn.PingContext(context.Background())
	if err != nil {
		l.WithError(err).Error("Failed to ping ClickHouse server")
		return nil, err
	}

	db, err := gorm.Open(gorm_clickhouse.New(gorm_clickhouse.Config{
		Conn: conn,
	}), &gorm.Config{Logger: logger.Default.LogMode(plugins.GetGormLogConfig())})
	if err != nil {
		l.WithError(err).Error("Failed to open ClickHouse GORM connection")
		return nil, err
	}

	// Configure connection pool for better reconnection behavior
	if err := plugins.ConfigureConnectionPool(db); err != nil {
		l.WithError(err).Warn("Failed to configure connection pool")
	}

	return db, nil
}
