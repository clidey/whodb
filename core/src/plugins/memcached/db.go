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

package memcached

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/common/ssl"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// minVersion is the minimum memcached version required for lru_crawler metadump.
const minVersion = "1.4.31"

// DB creates a new memcached client from the given plugin config.
// It handles port parsing, TLS configuration, authentication, and version validation.
func DB(config *engine.PluginConfig) (*Client, error) {
	port := common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "11211")
	if _, err := strconv.Atoi(port); err != nil {
		log.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Failed to parse Memcached port number")
		return nil, err
	}
	addr := net.JoinHostPort(config.Credentials.Hostname, port)

	// Configure SSL/TLS
	sslConfig := ssl.ParseSSLConfig(engine.DatabaseType_Memcached, config.Credentials.Advanced, config.Credentials.Hostname, config.Credentials.IsProfile)

	var client *Client
	var err error

	if sslConfig != nil && sslConfig.IsEnabled() {
		tlsConfig, tlsErr := ssl.BuildTLSConfig(sslConfig, config.Credentials.Hostname)
		if tlsErr != nil {
			log.WithError(tlsErr).WithFields(map[string]any{
				"hostname": config.Credentials.Hostname,
				"sslMode":  sslConfig.Mode,
			}).Error("Failed to build TLS configuration for Memcached")
			return nil, tlsErr
		}
		client, err = DialTLS(addr, tlsConfig)
	} else {
		client, err = Dial(addr)
	}
	if err != nil {
		log.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Failed to connect to Memcached")
		return nil, err
	}

	// Authenticate if both username and password are provided (text protocol -Y auth)
	if config.Credentials.Username != "" && config.Credentials.Password != "" {
		if authErr := client.Authenticate(config.Credentials.Username, config.Credentials.Password); authErr != nil {
			client.Close()
			log.WithError(authErr).WithField("hostname", config.Credentials.Hostname).Error("Failed to authenticate with Memcached")
			return nil, authErr
		}
	}

	// Validate server version (≥1.4.31 required for lru_crawler metadump)
	version, err := client.Version()
	if err != nil {
		client.Close()
		log.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Failed to get Memcached version")
		return nil, err
	}
	if !isVersionSupported(version) {
		client.Close()
		return nil, fmt.Errorf("memcached version %s is not supported, minimum required: %s", version, minVersion)
	}

	return client, nil
}

// isVersionSupported checks if a version string is ≥ minVersion.
func isVersionSupported(version string) bool {
	return compareVersions(version, minVersion) >= 0
}

// compareVersions compares two semver-like version strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		var aVal, bVal int
		if i < len(aParts) {
			aVal, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bVal, _ = strconv.Atoi(bParts[i])
		}
		if aVal < bVal {
			return -1
		}
		if aVal > bVal {
			return 1
		}
	}
	return 0
}
