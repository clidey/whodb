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

package gorm_plugin

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
)

// StreamRawExecute streams a raw SQL result row by row without materializing
// the entire result set in memory first.
func (p *GormPlugin) StreamRawExecute(config *engine.PluginConfig, query string, writer engine.QueryStreamWriter, params ...any) error {
	if config != nil && config.MultiStatement {
		return engine.ErrMultiStatementUnsupported
	}

	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		// codeql[go/sql-injection]: StreamRawExecute intentionally runs user-authored SQL from the query editor/import flow.
		rows, err := db.Raw(query, params...).Rows()
		if err != nil {
			return false, err
		}
		defer func() { _ = rows.Close() }()

		columns, typeMap, resultColumns, err := p.describeRawColumns(rows)
		if err != nil {
			return false, err
		}
		if err := writer.WriteColumns(resultColumns); err != nil {
			return false, fmt.Errorf("failed to write streamed columns: %w", err)
		}

		for rows.Next() {
			row, err := p.scanRawRow(rows, columns, typeMap)
			if err != nil {
				return false, err
			}
			if err := writer.WriteRow(row); err != nil {
				return false, fmt.Errorf("failed to write streamed row: %w", err)
			}
		}
		if err := rows.Err(); err != nil {
			return false, err
		}

		return true, nil
	})
	return err
}
