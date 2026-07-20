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
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

// SystemObjectAttributeKey marks a storage unit as provisioned by the database
// engine, an installed extension, or platform operator tooling rather than
// authored by the user. The namespaced key signals a transport-only marker:
// consumers promote it to a typed field at their contract boundary instead of
// displaying it as object metadata.
const SystemObjectAttributeKey = "whodb:system-object"

// systemObjectsQuery fingerprints System Objects from catalog structure, never
// from object names, so a user table can only match by genuinely being (1) an
// extension member, (2) a foreign table served by file_fdw, or (3) a regular
// table whose inheritance children are all file_fdw foreign tables. The one
// deliberate name-based rule is (4): Spilo pins its failed_authentication_[0-7]
// views, which carry no structural marker of their own.
const systemObjectsQuery = `
	SELECT c.relname
	FROM pg_catalog.pg_class c
	JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
	WHERE n.nspname = ?
		AND (
			EXISTS (
				SELECT 1
				FROM pg_catalog.pg_depend d
				WHERE d.classid = 'pg_catalog.pg_class'::pg_catalog.regclass
					AND d.objid = c.oid
					AND d.deptype = 'e'
			)
			OR (c.relkind = 'f' AND EXISTS (
				SELECT 1
				FROM pg_catalog.pg_foreign_table ft
				JOIN pg_catalog.pg_foreign_server fs ON fs.oid = ft.ftserver
				JOIN pg_catalog.pg_foreign_data_wrapper w ON w.oid = fs.srvfdw
				WHERE ft.ftrelid = c.oid AND w.fdwname = 'file_fdw'
			))
			OR (c.relkind = 'r'
				AND EXISTS (
					SELECT 1 FROM pg_catalog.pg_inherits i WHERE i.inhparent = c.oid
				)
				AND NOT EXISTS (
					SELECT 1
					FROM pg_catalog.pg_inherits i
					JOIN pg_catalog.pg_class child ON child.oid = i.inhrelid
					WHERE i.inhparent = c.oid
						AND NOT (child.relkind = 'f' AND EXISTS (
							SELECT 1
							FROM pg_catalog.pg_foreign_table ft
							JOIN pg_catalog.pg_foreign_server fs ON fs.oid = ft.ftserver
							JOIN pg_catalog.pg_foreign_data_wrapper w ON w.oid = fs.srvfdw
							WHERE ft.ftrelid = child.oid AND w.fdwname = 'file_fdw'
						))
				))
			OR (c.relkind = 'v' AND c.relname ~ '^failed_authentication_[0-7]$')
		)
`

// querySystemObjectNames runs the fingerprint query and returns the names of
// System Objects in the schema.
func (p *PostgresPlugin) querySystemObjectNames(config *engine.PluginConfig, schema string) (map[string]bool, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (map[string]bool, error) {
		var names []string
		if err := db.Raw(systemObjectsQuery, schema).Scan(&names).Error; err != nil {
			return nil, err
		}
		systemNames := make(map[string]bool, len(names))
		for _, name := range names {
			systemNames[name] = true
		}
		return systemNames, nil
	})
}

// markSystemObjects appends the System Object attribute to every classified
// unit. It fails open: any classification error leaves the listing intact and
// unmarked.
func (p *PostgresPlugin) markSystemObjects(config *engine.PluginConfig, schema string, units []engine.StorageUnit) {
	classify := p.classifySystemObjects
	if classify == nil {
		classify = p.querySystemObjectNames
	}
	systemNames, err := classify(config, schema)
	if err != nil {
		log.WithError(err).Warn("System Object classification failed; listing objects unmarked")
		return
	}
	for i := range units {
		if systemNames[units[i].Name] {
			units[i].Attributes = append(units[i].Attributes, engine.Record{Key: SystemObjectAttributeKey, Value: "true"})
		}
	}
}
