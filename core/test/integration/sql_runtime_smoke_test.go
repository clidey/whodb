//go:build integration

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

package integration

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/query"
)

type sqlSmokeCase struct {
	name               string
	schema             string
	sampleTable        string
	expectedDatabase   string
	supportsSchemas    bool
	expectedFKColumn   string
	expectedConstraint string
	rawQuery           string
}

func findTarget(t *testing.T, name string) target {
	t.Helper()

	for _, candidate := range targets {
		if candidate.name == name {
			return candidate
		}
	}

	t.Fatalf("target %q not configured", name)
	return target{}
}

func storageUnitMap(units []engine.StorageUnit) map[string]engine.StorageUnit {
	result := make(map[string]engine.StorageUnit, len(units))
	for _, unit := range units {
		result[unit.Name] = unit
	}
	return result
}

func graphUnitByName(units []engine.GraphUnit, name string) *engine.GraphUnit {
	for i := range units {
		if units[i].Unit.Name == name {
			return &units[i]
		}
	}
	return nil
}

func columnByName(columns []engine.Column, name string) *engine.Column {
	for i := range columns {
		if columns[i].Name == name {
			return &columns[i]
		}
	}
	return nil
}

func TestSeededSQLRuntimeSmoke(t *testing.T) {
	t.Parallel()

	tests := []sqlSmokeCase{
		{
			name:               "postgres",
			schema:             "test_schema",
			sampleTable:        "orders",
			expectedDatabase:   "test_db",
			supportsSchemas:    true,
			expectedFKColumn:   "user_id",
			expectedConstraint: "status",
			rawQuery:           "SELECT status FROM test_schema.orders ORDER BY id LIMIT 1",
		},
		{
			name:               "mysql",
			schema:             "test_db",
			sampleTable:        "orders",
			expectedDatabase:   "test_db",
			supportsSchemas:    true,
			expectedFKColumn:   "user_id",
			expectedConstraint: "status",
			rawQuery:           "SELECT status FROM orders ORDER BY id LIMIT 1",
		},
		{
			name:               "clickhouse",
			schema:             "test_db",
			sampleTable:        "orders",
			expectedDatabase:   "test_db",
			supportsSchemas:    false,
			expectedFKColumn:   "",
			expectedConstraint: "status",
			rawQuery:           "SELECT status FROM test_db.orders ORDER BY id LIMIT 1",
		},
	}

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {
			target := findTarget(t, tc.name)

			databases, err := target.plugin.GetDatabases(target.config)
			if err != nil {
				t.Fatalf("GetDatabases failed: %v", err)
			}
			if !slices.Contains(databases, tc.expectedDatabase) {
				t.Fatalf("expected databases %#v to contain %q", databases, tc.expectedDatabase)
			}

			schemas, err := target.plugin.GetAllSchemas(target.config)
			if tc.supportsSchemas {
				if err != nil {
					t.Fatalf("GetAllSchemas failed: %v", err)
				}
				if !slices.Contains(schemas, tc.schema) {
					t.Fatalf("expected schemas %#v to contain %q", schemas, tc.schema)
				}
			} else if err == nil {
				t.Fatalf("expected %s GetAllSchemas to be unsupported", tc.name)
			}

			units, err := target.plugin.GetStorageUnits(target.config, tc.schema)
			if err != nil {
				t.Fatalf("GetStorageUnits failed: %v", err)
			}
			unitMap := storageUnitMap(units)
			if _, ok := unitMap[tc.sampleTable]; !ok {
				t.Fatalf("expected storage units to contain %q, got %#v", tc.sampleTable, units)
			}

			exists, err := target.plugin.StorageUnitExists(target.config, tc.schema, tc.sampleTable)
			if err != nil || !exists {
				t.Fatalf("expected storage unit %q to exist, exists=%t err=%v", tc.sampleTable, exists, err)
			}

			missingTable := "missing_" + tc.sampleTable
			exists, err = target.plugin.StorageUnitExists(target.config, tc.schema, missingTable)
			if err != nil {
				t.Fatalf("StorageUnitExists failed for missing table: %v", err)
			}
			if exists {
				t.Fatalf("expected storage unit %q not to exist", missingTable)
			}

			columns, err := target.plugin.GetColumnsForTable(target.config, tc.schema, tc.sampleTable)
			if err != nil {
				t.Fatalf("GetColumnsForTable failed: %v", err)
			}
			if len(columns) == 0 {
				t.Fatalf("expected columns for %s.%s", tc.schema, tc.sampleTable)
			}
			if columnByName(columns, "id") == nil && columnByName(columns, "ID") == nil {
				t.Fatalf("expected id column in %#v", columns)
			}

			rows, err := target.plugin.GetRows(target.config, &engine.GetRowsRequest{
				Schema:      tc.schema,
				StorageUnit: tc.sampleTable,
				Sort:        []*query.SortCondition{},
				PageSize:    5,
			})
			if err != nil {
				t.Fatalf("GetRows failed: %v", err)
			}
			if len(rows.Rows) == 0 {
				t.Fatalf("expected seeded rows for %s.%s", tc.schema, tc.sampleTable)
			}

			count, err := target.plugin.GetRowCount(target.config, tc.schema, tc.sampleTable, nil)
			if err != nil {
				t.Fatalf("GetRowCount failed: %v", err)
			}
			if count < int64(len(rows.Rows)) {
				t.Fatalf("expected row count >= visible rows, got count=%d rows=%d", count, len(rows.Rows))
			}

			constraints, err := target.plugin.GetColumnConstraints(target.config, tc.schema, tc.sampleTable)
			if err != nil {
				t.Fatalf("GetColumnConstraints failed: %v", err)
			}
			if tc.expectedConstraint != "" {
				colConstraints, ok := constraints[tc.expectedConstraint]
				if !ok {
					t.Fatalf("expected constraints for column %q in %#v", tc.expectedConstraint, constraints)
				}
				if values, ok := colConstraints["check_values"].([]string); ok && len(values) == 0 {
					t.Fatalf("expected non-empty check_values for %q", tc.expectedConstraint)
				}
			}

			fkRelationships, err := target.plugin.GetForeignKeyRelationships(target.config, tc.schema, tc.sampleTable)
			if err != nil {
				t.Fatalf("GetForeignKeyRelationships failed: %v", err)
			}
			if tc.expectedFKColumn != "" {
				fk, ok := fkRelationships[tc.expectedFKColumn]
				if !ok {
					t.Fatalf("expected foreign key for column %q in %#v", tc.expectedFKColumn, fkRelationships)
				}
				if !strings.EqualFold(fk.ReferencedTable, "users") || !strings.EqualFold(fk.ReferencedColumn, "id") {
					t.Fatalf("unexpected foreign key relationship: %#v", fk)
				}
			}

			graphUnits, err := target.plugin.GetGraph(target.config, tc.schema)
			if err != nil {
				t.Fatalf("GetGraph failed: %v", err)
			}
			if len(graphUnits) == 0 {
				t.Fatalf("expected graph units for schema %q", tc.schema)
			}
			if graphUnitByName(graphUnits, tc.sampleTable) == nil {
				t.Fatalf("expected graph to contain storage unit %q", tc.sampleTable)
			}

			rawResult, err := target.plugin.RawExecute(target.config, tc.rawQuery)
			if err != nil {
				t.Fatalf("RawExecute failed: %v", err)
			}
			if len(rawResult.Rows) == 0 {
				t.Fatalf("expected RawExecute rows for query %q", tc.rawQuery)
			}

			var exported [][]string
			if err := target.plugin.ExportData(target.config, tc.schema, tc.sampleTable, func(row []string) error {
				copyRow := append([]string(nil), row...)
				exported = append(exported, copyRow)
				return nil
			}, nil); err != nil {
				t.Fatalf("ExportData failed: %v", err)
			}
			if len(exported) < 2 {
				t.Fatalf("expected export header plus data rows, got %#v", exported)
			}
		})
	}
}

func TestClickHouseMutationRuntime(t *testing.T) {
	target := findTarget(t, "clickhouse")
	table := fmt.Sprintf("intg_runtime_ch_%d", time.Now().UnixNano())

	_, _ = target.plugin.RawExecute(target.config, fmt.Sprintf("DROP TABLE IF EXISTS %s.%s", target.schema, table))
	defer target.plugin.RawExecute(target.config, fmt.Sprintf("DROP TABLE IF EXISTS %s.%s", target.schema, table))

	created, err := target.plugin.AddStorageUnit(target.config, target.schema, table, []engine.Record{
		{Key: "id", Value: "UInt32", Extra: map[string]string{"Primary": "true", "Nullable": "false"}},
		{Key: "status", Value: "String", Extra: map[string]string{"Nullable": "false"}},
	})
	if err != nil || !created {
		t.Fatalf("AddStorageUnit failed: created=%t err=%v", created, err)
	}

	added, err := target.plugin.AddRow(target.config, target.schema, table, []engine.Record{
		{Key: "id", Value: "1", Extra: map[string]string{"Type": "UInt32"}},
		{Key: "status", Value: "pending", Extra: map[string]string{"Type": "String"}},
	})
	if err != nil || !added {
		t.Fatalf("AddRow failed: added=%t err=%v", added, err)
	}

	bulkAdded, err := target.plugin.BulkAddRows(target.config, target.schema, table, [][]engine.Record{
		{
			{Key: "id", Value: "2", Extra: map[string]string{"Type": "UInt32"}},
			{Key: "status", Value: "queued", Extra: map[string]string{"Type": "String"}},
		},
		{
			{Key: "id", Value: "3", Extra: map[string]string{"Type": "UInt32"}},
			{Key: "status", Value: "queued", Extra: map[string]string{"Type": "String"}},
		},
	})
	if err != nil || !bulkAdded {
		t.Fatalf("BulkAddRows failed: added=%t err=%v", bulkAdded, err)
	}

	updated, err := target.plugin.UpdateStorageUnit(target.config, target.schema, table, map[string]string{
		"id":     "1",
		"status": "completed",
	}, []string{"status"})
	if err != nil || !updated {
		t.Fatalf("UpdateStorageUnit failed: updated=%t err=%v", updated, err)
	}

	var currentRows *engine.GetRowsResult
	for range 5 {
		time.Sleep(200 * time.Millisecond)
		currentRows, err = target.plugin.GetRows(target.config, &engine.GetRowsRequest{
			Schema:      target.schema,
			StorageUnit: table,
			Sort:        []*query.SortCondition{{Column: "id", Direction: query.SortDirectionAsc}},
			PageSize:    10,
		})
		if err != nil {
			t.Fatalf("GetRows failed: %v", err)
		}
		if len(currentRows.Rows) == 3 && strings.Contains(strings.Join(currentRows.Rows[0], ","), "completed") {
			break
		}
	}
	if currentRows == nil || len(currentRows.Rows) != 3 {
		t.Fatalf("expected three ClickHouse rows after inserts, got %#v", currentRows)
	}

	var exported [][]string
	if err := target.plugin.ExportData(target.config, target.schema, table, func(row []string) error {
		exported = append(exported, append([]string(nil), row...))
		return nil
	}, nil); err != nil {
		t.Fatalf("ExportData failed: %v", err)
	}
	if len(exported) < 2 {
		t.Fatalf("expected export rows, got %#v", exported)
	}

	cleared, err := target.plugin.ClearTableData(target.config, target.schema, table)
	if err != nil || !cleared {
		t.Fatalf("ClearTableData failed: cleared=%t err=%v", cleared, err)
	}

	for range 5 {
		time.Sleep(200 * time.Millisecond)
		count, err := target.plugin.GetRowCount(target.config, target.schema, table, nil)
		if err != nil {
			t.Fatalf("GetRowCount after clear failed: %v", err)
		}
		if count == 0 {
			return
		}
	}

	t.Fatalf("expected ClickHouse table %q to be empty after ClearTableData", table)
}
