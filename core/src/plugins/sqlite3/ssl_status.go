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

package sqlite3

import (
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// GetSSLStatus returns nil for SQLite as it's a local file-based database
// that doesn't use network connections or SSL/TLS.
func (p *Sqlite3Plugin) GetSSLStatus(config *engine.PluginConfig) (*engine.SSLStatus, error) {
	log.Debug("[SSL] Sqlite3Plugin.GetSSLStatus: SQLite does not support SSL (local file-based)")
	return nil, nil
}
