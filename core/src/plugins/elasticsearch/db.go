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

package elasticsearch

import (
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/elastic/go-elasticsearch/v8"
)

func DB(config *engine.PluginConfig) (*elasticsearch.Client, error) {
	var addresses []string
	port, err := strconv.Atoi(common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "9200"))
	if err != nil {
		log.Logger.WithError(err).WithField("hostname", config.Credentials.Hostname).Error("Invalid port number for ElasticSearch connection")
		return nil, err
	}
	sslMode := common.GetRecordValueOrDefault(config.Credentials.Advanced, "SSL Mode", "disable")

	hostName := url.QueryEscape(config.Credentials.Hostname)

	scheme := "https"
	if sslMode == "disable" {
		scheme = "http"
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
		Username:  url.QueryEscape(config.Credentials.Username),
		Password:  url.QueryEscape(config.Credentials.Password),
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Logger.WithError(err).WithField("hostname", config.Credentials.Hostname).WithField("port", port).Error("Failed to create ElasticSearch client")
		return nil, err
	}

	res, err := client.Info()
	if err != nil || res.IsError() {
		log.Logger.WithError(err).WithField("hostname", config.Credentials.Hostname).WithField("port", port).Error("Failed to ping ElasticSearch server")
		return nil, fmt.Errorf("error pinging Elasticsearch: %v", err)
	}

	return client, nil
}
