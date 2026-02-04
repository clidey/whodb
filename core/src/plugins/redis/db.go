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

package redis

import (
	"context"
	"net"
	"strconv"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins/ssl"
	"github.com/go-redis/redis/v8"
)

func DB(config *engine.PluginConfig) (*redis.Client, error) {
	ctx := context.Background()
	port, err := strconv.Atoi(common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "6379"))
	if err != nil {
		log.Logger.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Failed to parse Redis port number")
		return nil, err
	}
	database := 0
	if config.Credentials.Database != "" {
		var err error
		database, err = strconv.Atoi(config.Credentials.Database)
		if err != nil {
			log.Logger.WithError(err).WithField("database", config.Credentials.Database).WithField("hostname", config.Credentials.Hostname).Error("Failed to parse Redis database number")
			return nil, err
		}
	}
	addr := net.JoinHostPort(config.Credentials.Hostname, strconv.Itoa(port))

	opts := &redis.Options{
		Addr:     addr,
		Username: config.Credentials.Username,
		Password: config.Credentials.Password,
		DB:       database,
	}

	// Configure SSL/TLS
	sslMode := "disabled"
	sslConfig := ssl.ParseSSLConfig(engine.DatabaseType_Redis, config.Credentials.Advanced, config.Credentials.Hostname, config.Credentials.IsProfile)
	if sslConfig != nil && sslConfig.IsEnabled() {
		sslMode = string(sslConfig.Mode)
		tlsConfig, err := ssl.BuildTLSConfig(sslConfig, config.Credentials.Hostname)
		if err != nil {
			log.Logger.WithError(err).WithFields(map[string]interface{}{
				"hostname": config.Credentials.Hostname,
				"sslMode":  sslConfig.Mode,
			}).Error("Failed to build TLS configuration for Redis")
			return nil, err
		}
		opts.TLSConfig = tlsConfig
	}

	client := redis.NewClient(opts)
	if _, err := client.Ping(ctx).Result(); err != nil {
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname": config.Credentials.Hostname,
			"database": database,
			"sslMode":  sslMode,
		}).Error("Failed to ping Redis server")
		return nil, err
	}
	return client, nil
}
