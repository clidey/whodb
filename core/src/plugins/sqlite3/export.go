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
	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

// ExportData exports a table through the raw-query stream, which keeps date
// and time columns as stored text the same way the SQLite grid does.
func (p *Sqlite3Plugin) ExportData(config *engine.PluginConfig, schema string, storageUnit string, writer engine.ExportWriter, selection *engine.ExportSelection) error {
	if selection.HasRows() {
		return p.GormPlugin.ExportData(config, schema, storageUnit, writer, selection)
	}

	tableName, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (string, error) {
		return p.CreateSQLBuilder(db).QualifiedTableName(schema, storageUnit)
	})
	if err != nil {
		return err
	}

	metadata, _ := sourcecatalog.ResolveSessionMetadata(string(p.Type))
	return p.StreamRawExecute(config, "SELECT * FROM "+tableName, &gorm_plugin.ExportStreamWriter{Writer: writer, Metadata: metadata})
}
