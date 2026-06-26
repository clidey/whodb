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
	"fmt"
	"io"

	"github.com/clidey/whodb/core/src/engine"
)

// ExportSQLData streams ClickHouse SQL Data Export statements.
func (p *ClickHousePlugin) ExportSQLData(config *engine.PluginConfig, req *engine.SQLDataExportRequest, writer io.Writer) error {
	if req != nil && req.Mode == engine.SQLDataExportModeUpdate {
		return fmt.Errorf("SQL Update Export is not supported for ClickHouse")
	}
	return p.GormPlugin.ExportSQLData(config, req, writer)
}
