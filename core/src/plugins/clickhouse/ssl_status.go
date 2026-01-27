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
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	"github.com/clidey/whodb/core/src/plugins/ssl"
)

// GetSSLStatus returns the SSL status based on the configured SSL mode for ClickHouse.
// ClickHouse native protocol doesn't expose connection-level SSL state via queries,
// so we use config-based detection like Redis and Elasticsearch.
func (p *ClickHousePlugin) GetSSLStatus(config *engine.PluginConfig) (*engine.SSLStatus, error) {
	if cached := plugins.GetCachedSSLStatus(config); cached != nil {
		return cached, nil
	}

	sslConfig := ssl.ParseSSLConfig(engine.DatabaseType_ClickHouse, config.Credentials.Advanced, config.Credentials.Hostname, config.Credentials.IsProfile)

	var status *engine.SSLStatus
	if sslConfig == nil || !sslConfig.IsEnabled() {
		status = &engine.SSLStatus{
			IsEnabled: false,
			Mode:      string(ssl.SSLModeDisabled),
		}
	} else {
		status = &engine.SSLStatus{
			IsEnabled: true,
			Mode:      string(sslConfig.Mode),
		}
	}

	plugins.SetCachedSSLStatus(config, status)
	return status, nil
}
