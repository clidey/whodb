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
	"crypto/tls"
	"crypto/x509"
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
	log.Debug("[ES DB] Creating Elasticsearch client")
	var addresses []string
	port, err := strconv.Atoi(common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "9200"))
	if err != nil {
		log.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Invalid port number for ElasticSearch connection")
		return nil, err
	}
	log.Debugf("[ES DB] Port: %d", port)

	hostName := url.QueryEscape(config.Credentials.Hostname)

	// Configure SSL/TLS
	sslMode := "disabled"
	log.Debugf("[ES DB] Parsing SSL config for %s", hostName)
	sslConfig := ssl.ParseSSLConfig(engine.DatabaseType_ElasticSearch, config.Credentials.Advanced, config.Credentials.Hostname, config.Credentials.IsProfile)

	// Determine scheme based on SSL mode
	scheme := "http"
	if sslConfig != nil && sslConfig.IsEnabled() {
		scheme = "https"
		sslMode = string(sslConfig.Mode)
		log.Debugf("[ES DB] SSL enabled, mode=%s, scheme=%s", sslMode, scheme)
	} else {
		log.Debug("[ES DB] SSL disabled or not configured")
	}

	addressUrl := url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(hostName, strconv.Itoa(port)),
	}

	addresses = []string{
		addressUrl.String(),
	}
	log.Debugf("[ES DB] Connecting to: %s", addresses[0])

	cfg := elasticsearch.Config{
		Addresses: addresses,
		Username:  config.Credentials.Username,
		Password:  config.Credentials.Password,
	}

	// Configure TLS if enabled
	if sslConfig != nil && sslConfig.IsEnabled() {
		// For insecure mode, skip certificate verification
		if sslConfig.Mode == ssl.SSLModeInsecure {
			log.Debug("[ES DB] Insecure mode: skipping certificate verification")
			cfg.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}
		} else {
			// For enabled mode, use the native CACert option if CA is provided
			caCertPEM, err := sslConfig.CACert.Load()
			if err != nil {
				log.WithError(err).Error("[ES DB] Failed to load CA certificate")
				return nil, err
			}

			if caCertPEM != nil {
				log.Debugf("[ES DB] Using CA certificate (%d bytes)", len(caCertPEM))
				cfg.CACert = caCertPEM
			}

			// Handle client certificates for mutual TLS
			if !sslConfig.ClientCert.IsEmpty() && !sslConfig.ClientKey.IsEmpty() {
				log.Debug("[ES DB] Loading client certificate for mutual TLS")
				tlsConfig, err := ssl.BuildTLSConfig(sslConfig, config.Credentials.Hostname)
				if err != nil {
					log.WithError(err).Error("[ES DB] Failed to build TLS config for client certs")
					return nil, err
				}
				if tlsConfig != nil && len(tlsConfig.Certificates) > 0 {
					// If we have client certs, we need custom transport
					// But also include CA verification
					if caCertPEM != nil {
						rootCAs := x509.NewCertPool()
						rootCAs.AppendCertsFromPEM(caCertPEM)
						tlsConfig.RootCAs = rootCAs
					}
					cfg.Transport = &http.Transport{
						TLSClientConfig: tlsConfig,
					}
					cfg.CACert = nil // Clear this since we're using transport
				}
			}
		}
	}

	log.Debug("[ES DB] Creating Elasticsearch client instance")
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"hostname": config.Credentials.Hostname,
			"port":     port,
			"sslMode":  sslMode,
		}).Error("Failed to create ElasticSearch client")
		return nil, err
	}

	log.Debug("[ES DB] Pinging Elasticsearch server")
	res, err := client.Info()
	if err != nil || res.IsError() {
		errMsg := "no error"
		if err != nil {
			errMsg = err.Error()
		}
		resStatus := "N/A"
		if res != nil {
			resStatus = res.Status()
		}
		log.WithError(err).WithFields(map[string]any{
			"hostname":  config.Credentials.Hostname,
			"port":      port,
			"sslMode":   sslMode,
			"error":     errMsg,
			"resStatus": resStatus,
		}).Error("Failed to ping ElasticSearch server")
		return nil, fmt.Errorf("error pinging Elasticsearch: %v", err)
	}

	log.Debugf("[ES DB] Successfully connected to Elasticsearch at %s", addresses[0])
	return client, nil
}
