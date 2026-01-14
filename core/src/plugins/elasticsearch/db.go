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

package elasticsearch

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins/ssl"
	"github.com/elastic/go-elasticsearch/v8"
)

func DB(config *engine.PluginConfig) (*elasticsearch.Client, error) {
	var addresses []string
	port, err := strconv.Atoi(common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "9200"))
	if err != nil {
		log.Logger.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Invalid port number for ElasticSearch connection")
		return nil, err
	}

	hostName := url.QueryEscape(config.Credentials.Hostname)

	// Configure SSL/TLS
	sslMode := "disabled"
	sslConfig := ssl.ParseSSLConfig(engine.DatabaseType_ElasticSearch, config.Credentials.Advanced, config.Credentials.Hostname, config.Credentials.IsProfile)

	// Determine scheme based on SSL mode
	scheme := "http"
	if sslConfig != nil && sslConfig.IsEnabled() {
		scheme = "https"
		sslMode = string(sslConfig.Mode)
	}

	addressUrl := url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(hostName, strconv.Itoa(port)),
	}

	addresses = []string{
		addressUrl.String(),
	}

	cfg := elasticsearch.Config{
		Addresses: addresses,
		Username:  config.Credentials.Username,
		Password:  config.Credentials.Password,
	}

	// Configure TLS if enabled
	if sslConfig != nil && sslConfig.IsEnabled() {
		tlsConfig, err := ssl.BuildTLSConfig(sslConfig, config.Credentials.Hostname)
		if err != nil {
			log.Logger.WithError(err).WithFields(map[string]interface{}{
				"hostname": config.Credentials.Hostname,
				"sslMode":  sslConfig.Mode,
			}).Error("Failed to build TLS configuration for Elasticsearch")
			return nil, err
		}

		if tlsConfig != nil {
			// we need to use a custom Transport
			cfg.Transport = &http.Transport{
				TLSClientConfig: tlsConfig,
			}
		}
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname": config.Credentials.Hostname,
			"port":     port,
			"sslMode":  sslMode,
		}).Error("Failed to create ElasticSearch client")
		return nil, err
	}

	res, err := client.Info()
	if err != nil || res.IsError() {
		log.Logger.WithError(err).WithFields(map[string]interface{}{
			"hostname": config.Credentials.Hostname,
			"port":     port,
			"sslMode":  sslMode,
		}).Error("Failed to ping ElasticSearch server")
		return nil, fmt.Errorf("error pinging Elasticsearch: %v", err)
	}

	return client, nil
}
