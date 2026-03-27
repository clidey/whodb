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
	"github.com/clidey/whodb/core/src/common/ssl"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// GetSSLStatus returns the SSL status based on the configured TLS settings for Memcached.
func (p *MemcachedPlugin) GetSSLStatus(config *engine.PluginConfig) (*engine.SSLStatus, error) {
	log.Debug("[SSL] MemcachedPlugin.GetSSLStatus: checking configured TLS mode")
	sslConfig := ssl.ParseSSLConfig(engine.DatabaseType_Memcached, config.Credentials.Advanced, config.Credentials.Hostname, config.Credentials.IsProfile)

	if sslConfig == nil || !sslConfig.IsEnabled() {
		log.Debug("[SSL] MemcachedPlugin.GetSSLStatus: TLS not configured or disabled")
		return &engine.SSLStatus{
			IsEnabled: false,
			Mode:      string(ssl.SSLModeDisabled),
		}, nil
	}

	log.Debugf("[SSL] MemcachedPlugin.GetSSLStatus: TLS enabled, mode=%s", sslConfig.Mode)
	return &engine.SSLStatus{
		IsEnabled: true,
		Mode:      string(sslConfig.Mode),
	}, nil
}
