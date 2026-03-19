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

package plugins

import (
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// FindChildTables returns all tables that have FK references to the given table.
// The graph structure stores parent -> children relationships, so we look at the
// target table's Relations to find its children (tables that reference it).
func FindChildTables(graph []engine.GraphUnit, targetTable string) []string {
	for _, unit := range graph {
		if unit.Unit.Name == targetTable {
			var children []string
			for _, rel := range unit.Relations {
				children = append(children, rel.Name)
			}
			return children
		}
	}
	return nil
}

// NullifyFKColumn sets the FK column in childTable that references parentTable to NULL.
// Used to break circular FK constraints before deleting rows.
func NullifyFKColumn(
	plugin engine.PluginFunctions,
	config *engine.PluginConfig,
	schema, childTable, parentTable string,
	graph []engine.GraphUnit,
) error {
	for _, unit := range graph {
		if unit.Unit.Name == parentTable {
			for _, rel := range unit.Relations {
				if rel.Name == childTable && rel.SourceColumn != nil {
					return plugin.NullifyFKColumn(config, schema, childTable, *rel.SourceColumn)
				}
			}
			return nil
		}
	}
	return nil
}

// ClearTableWithDependencies clears a table and all tables that reference it (children first).
// This ensures FK constraints are respected by deleting in the correct order.
// Circular FK references (e.g., employees <-> departments) are handled by NULLing out
// the FK column that forms the cycle before any rows are deleted.
func ClearTableWithDependencies(
	plugin engine.PluginFunctions,
	config *engine.PluginConfig,
	schema, table string,
	graph []engine.GraphUnit,
) error {
	cleared := make(map[string]bool)
	return clearTableRecursive(plugin, config, schema, table, graph, cleared)
}

func clearTableRecursive(
	plugin engine.PluginFunctions,
	config *engine.PluginConfig,
	schema, table string,
	graph []engine.GraphUnit,
	cleared map[string]bool,
) error {
	if cleared[table] {
		return nil
	}
	cleared[table] = true

	children := FindChildTables(graph, table)

	for _, child := range children {
		if cleared[child] {
			// Cycle detected: NULL out the FK column in child that points to this
			// table so the subsequent DELETE won't violate the constraint.
			if err := NullifyFKColumn(plugin, config, schema, child, table, graph); err != nil {
				log.WithFields(map[string]any{
					"child":  child,
					"parent": table,
				}).WithError(err).Error("Failed to nullify FK column for cycle breaking")
			}
			continue
		}
		if err := clearTableRecursive(plugin, config, schema, child, graph, cleared); err != nil {
			return err
		}
	}

	log.WithFields(map[string]any{
		"table":    table,
		"children": children,
	}).Debug("Clearing table data (children already cleared)")

	if _, err := plugin.ClearTableData(config, schema, table); err != nil {
		return fmt.Errorf("failed to clear table %s: %w", table, err)
	}

	return nil
}
