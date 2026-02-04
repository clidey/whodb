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

package postgres

import (
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	"github.com/clidey/whodb/core/src/plugins/ssl"
	"gorm.io/gorm"
)

// GetSSLStatus queries PostgreSQL to get the actual SSL status of the connection.
// Uses pg_stat_ssl system view. Results are cached.
func (p *PostgresPlugin) GetSSLStatus(config *engine.PluginConfig) (*engine.SSLStatus, error) {
	if cached := plugins.GetCachedSSLStatus(config); cached != nil {
		return cached, nil
	}

	status, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.SSLStatus, error) {
		var result struct {
			SSL bool `gorm:"column:ssl"`
		}

		query := db.Raw("SELECT ssl FROM pg_stat_ssl WHERE pid = pg_backend_pid()").Scan(&result)
		if query.Error != nil {
			return nil, query.Error
		}

		if query.RowsAffected == 0 {
			// No row returned - might be permissions issue, fall back to config
			sslConfig := ssl.ParseSSLConfig(engine.DatabaseType_Postgres, config.Credentials.Advanced, config.Credentials.Hostname, config.Credentials.IsProfile)
			if sslConfig != nil && sslConfig.IsEnabled() {
				return &engine.SSLStatus{
					IsEnabled: true,
					Mode:      string(sslConfig.Mode),
				}, nil
			}
			return &engine.SSLStatus{
				IsEnabled: false,
				Mode:      string(ssl.SSLModeDisabled),
			}, nil
		}

		if !result.SSL {
			return &engine.SSLStatus{
				IsEnabled: false,
				Mode:      string(ssl.SSLModeDisabled),
			}, nil
		}

		sslConfig := ssl.ParseSSLConfig(engine.DatabaseType_Postgres, config.Credentials.Advanced, config.Credentials.Hostname, config.Credentials.IsProfile)
		mode := "enabled"
		if sslConfig != nil {
			mode = string(sslConfig.Mode)
		}

		return &engine.SSLStatus{
			IsEnabled: true,
			Mode:      mode,
		}, nil
	})

	if err == nil && status != nil {
		plugins.SetCachedSSLStatus(config, status)
	}
	return status, err
}
