// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package memcached

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

func DB(config *engine.PluginConfig) (*memcache.Client, error) {
	port, err := strconv.Atoi(common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "11211"))
	if err != nil {
		log.Logger.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Failed to parse Memcached port number")
		return nil, err
	}
	
	addr := net.JoinHostPort(config.Credentials.Hostname, strconv.Itoa(port))
	
	// Support multiple servers if provided in advanced config
	servers := []string{addr}
	if additionalServers := common.GetRecordValueOrDefault(config.Credentials.Advanced, "AdditionalServers", ""); additionalServers != "" {
		// Parse additional servers (comma-separated)
		for _, server := range strings.Split(additionalServers, ",") {
			server = strings.TrimSpace(server)
			if server != "" {
				servers = append(servers, server)
			}
		}
	}
	
	client := memcache.New(servers...)
	
	// Set timeout if provided
	if timeoutStr := common.GetRecordValueOrDefault(config.Credentials.Advanced, "Timeout", ""); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil && timeout > 0 {
			client.Timeout = time.Duration(timeout) * time.Second
		}
	}
	
	// Test connection by trying to get stats
	_, err = client.Stats()
	if err != nil {
		log.Logger.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Failed to connect to Memcached server")
		return nil, fmt.Errorf("failed to connect to Memcached: %w", err)
	}
	
	return client, nil
}