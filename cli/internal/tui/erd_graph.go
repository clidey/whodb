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

package tui

import "github.com/clidey/whodb/core/src/engine"

type erdForeignKeyTarget struct {
	table  string
	column string
}

func buildERDTablesFromUnits(graphUnits []engine.GraphUnit) []tableWithColumns {
	tables := make([]tableWithColumns, len(graphUnits))
	for i, graphUnit := range graphUnits {
		tables[i] = tableWithColumns{StorageUnit: graphUnit.Unit}
	}
	return tables
}

func buildERDTablesFromGraph(graphUnits []engine.GraphUnit, columnLoader func(storageUnit string) ([]engine.Column, error)) []tableWithColumns {
	fkTargets := buildERDForeignKeyTargets(graphUnits)
	tables := make([]tableWithColumns, 0, len(graphUnits))

	for _, graphUnit := range graphUnits {
		columns, err := columnLoader(graphUnit.Unit.Name)
		if err != nil {
			continue
		}

		tables = append(tables, tableWithColumns{
			StorageUnit: graphUnit.Unit,
			Columns:     applyERDForeignKeyTargets(graphUnit.Unit.Name, columns, fkTargets),
		})
	}

	return tables
}

func buildERDForeignKeyTargets(graphUnits []engine.GraphUnit) map[string]map[string]erdForeignKeyTarget {
	fkTargets := make(map[string]map[string]erdForeignKeyTarget)

	for _, graphUnit := range graphUnits {
		for _, relation := range graphUnit.Relations {
			if relation.SourceColumn == nil || relation.TargetColumn == nil {
				continue
			}

			sourceTable := graphUnit.Unit.Name
			targetTable := relation.Name
			if relation.RelationshipType == "OneToMany" {
				sourceTable = relation.Name
				targetTable = graphUnit.Unit.Name
			}

			if _, ok := fkTargets[sourceTable]; !ok {
				fkTargets[sourceTable] = make(map[string]erdForeignKeyTarget)
			}

			fkTargets[sourceTable][*relation.SourceColumn] = erdForeignKeyTarget{
				table:  targetTable,
				column: *relation.TargetColumn,
			}
		}
	}

	return fkTargets
}

func applyERDForeignKeyTargets(storageUnit string, columns []engine.Column, fkTargets map[string]map[string]erdForeignKeyTarget) []engine.Column {
	targets := fkTargets[storageUnit]
	if len(targets) == 0 {
		return columns
	}

	enriched := make([]engine.Column, len(columns))
	copy(enriched, columns)

	for i := range enriched {
		target, ok := targets[enriched[i].Name]
		if !ok {
			continue
		}

		enriched[i].IsForeignKey = true

		if enriched[i].ReferencedTable == nil {
			tableName := target.table
			enriched[i].ReferencedTable = new(tableName)
		}

		if enriched[i].ReferencedColumn == nil {
			columnName := target.column
			enriched[i].ReferencedColumn = new(columnName)
		}
	}

	return enriched
}

func countGraphRelationships(graphUnits []engine.GraphUnit) int {
	count := 0
	for _, graphUnit := range graphUnits {
		count += len(graphUnit.Relations)
	}
	return count
}
