/*
 * Copyright 2025 Clidey, Inc.
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
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
)

// ExportCSV exports Redis data to CSV format
// Redis doesn't have tables, so we export key patterns as "tables"
func (p *RedisPlugin) ExportCSV(config *engine.PluginConfig, schema string, storageUnit string, writer func([]string) error, progressCallback func(int)) error {
	// Redis doesn't support traditional table export
	// Could implement key pattern export, but that would be different from other databases
	return fmt.Errorf("CSV export is not supported for Redis databases")
}

// ImportCSV imports CSV data into Redis
func (p *RedisPlugin) ImportCSV(config *engine.PluginConfig, schema string, storageUnit string, reader func() ([]string, error), mode engine.ImportMode, progressCallback func(engine.ImportProgress)) error {
	// Redis doesn't support traditional table import
	return fmt.Errorf("CSV import is not supported for Redis databases")
}