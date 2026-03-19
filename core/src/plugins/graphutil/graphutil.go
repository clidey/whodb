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

package graphutil

import (
	"strings"

	"github.com/clidey/whodb/core/src/engine"
)

// Relation represents an inferred relationship between two storage units.
type Relation struct {
	Table1       string
	Table2       string
	Relation     string
	SourceColumn string
	TargetColumn string
}

// InferForeignKeys examines a set of field names from a given storage unit and
// returns a map of referenced storage unit name to the field name that references it.
// It uses singular/plural heuristics to match field names like "user_id" to a "users" collection.
func InferForeignKeys(currentUnit string, fieldNames []string, allUnits []string) map[string]string {
	foreignKeys := make(map[string]string)

	for _, fieldName := range fieldNames {
		if fieldName == "_id" {
			continue
		}

		lowerField := strings.ToLower(fieldName)

		// Check for explicit relation hints (e.g., "order.id")
		if strings.HasSuffix(lowerField, ".id") || strings.HasSuffix(lowerField, "id") || strings.HasSuffix(lowerField, "_id") {
			for _, otherUnit := range allUnits {
				if otherUnit == currentUnit {
					continue
				}
				if strings.Contains(lowerField, strings.ToLower(otherUnit)) {
					foreignKeys[otherUnit] = fieldName
					break
				}
			}
		}

		// Check for naming convention matches (e.g., "user_id" → "users")
		for _, otherUnit := range allUnits {
			if otherUnit == currentUnit {
				continue
			}

			singularName := strings.TrimSuffix(otherUnit, "s")
			pluralName := otherUnit
			if !strings.HasSuffix(otherUnit, "s") {
				pluralName = otherUnit + "s"
			}

			if lowerField == strings.ToLower(singularName)+"_id" ||
				lowerField == strings.ToLower(singularName)+"id" ||
				lowerField == strings.ToLower(otherUnit)+"_id" ||
				lowerField == strings.ToLower(otherUnit)+"id" ||
				lowerField == strings.ToLower(pluralName)+"_id" ||
				lowerField == strings.ToLower(pluralName)+"id" {
				foreignKeys[otherUnit] = fieldName
				break
			}
		}
	}

	return foreignKeys
}

// BuildGraphUnits assembles GraphUnit slices from a relation list and the full set of storage units.
func BuildGraphUnits(relations []Relation, storageUnits []engine.StorageUnit) []engine.GraphUnit {
	tableMap := make(map[string][]engine.GraphUnitRelationship)
	for _, tr := range relations {
		sourceCol := tr.SourceColumn
		targetCol := tr.TargetColumn
		tableMap[tr.Table1] = append(tableMap[tr.Table1], engine.GraphUnitRelationship{
			Name:             tr.Table2,
			RelationshipType: engine.GraphUnitRelationshipType(tr.Relation),
			SourceColumn:     &sourceCol,
			TargetColumn:     &targetCol,
		})
	}

	tables := make([]engine.GraphUnit, 0, len(storageUnits))
	for _, storageUnit := range storageUnits {
		foundTable, ok := tableMap[storageUnit.Name]
		var rels []engine.GraphUnitRelationship
		if ok {
			rels = foundTable
		}
		tables = append(tables, engine.GraphUnit{Unit: storageUnit, Relations: rels})
	}

	return tables
}
