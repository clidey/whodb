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

package mockdata

import (
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
)

// generateTableRows generates rows for a single table.
// If isTargetTable is true, uses bulk insert for better performance.
func (g *Generator) generateTableRows(
	plugin engine.PluginFunctions,
	config *engine.PluginConfig,
	schema, table string,
	rowCount int,
	overwrite bool,
	isTargetTable bool,
) (int, error) {
	insertMode := "single"
	if isTargetTable {
		insertMode = "bulk"
	}
	log.WithFields(map[string]any{
		"table":      table,
		"rowCount":   rowCount,
		"overwrite":  overwrite,
		"insertMode": insertMode,
	}).Info("Generating mock data for table")

	// Get columns and constraints
	columns, err := plugin.GetColumnsForTable(config, schema, table)
	if err != nil {
		return 0, fmt.Errorf("failed to get columns: %w", err)
	}

	// Enrich columns with auto-increment and computed column flags
	if err := plugin.MarkGeneratedColumns(config, schema, table, columns); err != nil {
		log.WithError(err).Warn("Failed to mark generated columns for " + table)
	}

	// Log column details for debugging PK/auto-increment/computed issues
	columnDetails := make([]map[string]any, 0, len(columns))
	for _, col := range columns {
		if col.IsPrimary || col.IsAutoIncrement || col.IsComputed {
			columnDetails = append(columnDetails, map[string]any{
				"name":       col.Name,
				"type":       col.Type,
				"isPrimary":  col.IsPrimary,
				"isAutoIncr": col.IsAutoIncrement,
				"isComputed": col.IsComputed,
			})
		}
	}
	log.WithFields(map[string]any{
		"table":          table,
		"columnCount":    len(columns),
		"specialColumns": columnDetails,
	}).Debug("Retrieved column definitions")

	constraints, err := plugin.GetColumnConstraints(config, schema, table)
	if err != nil {
		log.WithError(err).WithField("table", table).Warn("Failed to get constraints, using defaults")
		constraints = make(map[string]map[string]any)
	} else {
		// Log constraint details for debugging
		constraintDetails := make(map[string][]string)
		for col, c := range constraints {
			var details []string
			if checkVals, ok := c["check_values"]; ok {
				details = append(details, fmt.Sprintf("check_values=%v", checkVals))
			}
			if checkMin, ok := c["check_min"]; ok {
				details = append(details, fmt.Sprintf("check_min=%v", checkMin))
			}
			if checkMax, ok := c["check_max"]; ok {
				details = append(details, fmt.Sprintf("check_max=%v", checkMax))
			}
			if len(details) > 0 {
				constraintDetails[col] = details
			}
		}
		log.WithFields(map[string]any{
			"table":            table,
			"constraintCount":  len(constraints),
			"checkConstraints": constraintDetails,
		}).Debug("Retrieved column constraints")
	}

	// Get FK relationships
	fks, err := plugin.GetForeignKeyRelationships(config, schema, table)
	if err != nil {
		log.WithError(err).WithField("table", table).Warn("Failed to get FK relationships")
		fks = make(map[string]*engine.ForeignKeyRelationship)
	} else if len(fks) > 0 {
		log.WithFields(map[string]any{
			"table":   table,
			"fkCount": len(fks),
		}).Debug("Retrieved FK relationships")
	}

	// Load existing PKs to prevent uniqueness violations (only when not overwriting)
	// When overwriting, tables are already cleared at the top level in Generate()
	if !overwrite {
		if err := g.loadExistingPKsForUniqueness(plugin, config, schema, table, columns); err != nil {
			log.WithError(err).WithField("table", table).Warn("Failed to load existing PKs for uniqueness check")
			// Continue anyway - worst case we get unique constraint errors
		}
	}

	// Target table: use bulk insert (no PK tracking needed)
	// Parent tables: use single inserts (need to track PKs for FK references)
	if isTargetTable {
		return g.generateRowsBulk(plugin, config, schema, table, columns, constraints, fks, rowCount)
	}
	return g.generateRowsSingle(plugin, config, schema, table, columns, constraints, fks, rowCount)
}

// generateRowsBulk generates all rows first, then bulk inserts them.
// Used for target table where PK tracking isn't needed.
func (g *Generator) generateRowsBulk(
	plugin engine.PluginFunctions,
	config *engine.PluginConfig,
	schema, table string,
	columns []engine.Column,
	constraints map[string]map[string]any,
	fks map[string]*engine.ForeignKeyRelationship,
	rowCount int,
) (int, error) {
	log.WithFields(map[string]any{
		"table":    table,
		"rowCount": rowCount,
	}).Debug("Starting bulk row generation")

	// Generate all rows first
	rows := make([][]engine.Record, 0, rowCount)
	failedCount := 0
	pkCollisions := 0
	for i := 0; i < rowCount; i++ {
		var row []engine.Record
		var err error

		// Try to generate a row with unique PK (retry on collision)
		for attempt := 0; attempt < MaxPKRetries; attempt++ {
			row, err = g.generateRow(columns, constraints, fks, table)
			if err != nil {
				break // Generation error, not a collision
			}

			// Check if PK is unique
			if !g.isPKValueUsed(table, columns, row) {
				// Mark PK as used and proceed
				g.markPKValueUsed(table, columns, row)
				break
			}

			// PK collision - retry
			if attempt == MaxPKRetries-1 {
				err = fmt.Errorf("failed to generate unique PK after %d attempts", MaxPKRetries)
				pkCollisions++
			}
		}

		if err != nil {
			log.WithError(err).WithField("table", table).WithField("row", i).Error("Failed to generate row")
			failedCount++
			continue
		}
		rows = append(rows, row)
	}

	if failedCount > 0 {
		log.WithFields(map[string]any{
			"table":        table,
			"failedCount":  failedCount,
			"pkCollisions": pkCollisions,
		}).Warn("Some rows failed to generate")
	}

	if len(rows) == 0 {
		log.WithField("table", table).Warn("No rows generated")
		return 0, nil
	}

	log.WithFields(map[string]any{
		"table":    table,
		"rowCount": len(rows),
	}).Debug("Generated rows, starting bulk insert")

	// Bulk insert all rows
	if _, err := plugin.BulkAddRows(config, schema, table, rows); err != nil {
		log.WithError(err).WithField("table", table).WithField("rowCount", len(rows)).Error("Failed to bulk insert rows")
		return 0, fmt.Errorf("failed to bulk insert rows: %w", err)
	}

	log.WithFields(map[string]any{
		"table":     table,
		"generated": len(rows),
	}).Info("Completed bulk mock data generation")
	return len(rows), nil
}

// generateRowsSingle generates and inserts rows one at a time.
// Used for parent tables where PK tracking is needed for FK references.
func (g *Generator) generateRowsSingle(
	plugin engine.PluginFunctions,
	config *engine.PluginConfig,
	schema, table string,
	columns []engine.Column,
	constraints map[string]map[string]any,
	fks map[string]*engine.ForeignKeyRelationship,
	rowCount int,
) (int, error) {
	log.WithFields(map[string]any{
		"table":    table,
		"rowCount": rowCount,
	}).Debug("Starting single-row generation (PK tracking enabled)")

	// Find auto-increment PK column (if any) for tracking database-generated IDs
	var autoIncrementPKCol *engine.Column
	for i := range columns {
		if columns[i].IsPrimary && columns[i].IsAutoIncrement {
			autoIncrementPKCol = &columns[i]
			log.WithFields(map[string]any{
				"table":  table,
				"column": columns[i].Name,
			}).Debug("Found auto-increment PK column for tracking")
			break
		}
	}

	generated := 0
	failedGenerate := 0
	failedInsert := 0
	pkCollisions := 0

	for i := 0; i < rowCount; i++ {
		var row []engine.Record
		var err error

		// Try to generate a row with unique PK (retry on collision)
		for attempt := 0; attempt < MaxPKRetries; attempt++ {
			row, err = g.generateRow(columns, constraints, fks, table)
			if err != nil {
				break // Generation error, not a collision
			}

			// Check if PK is unique (skip for auto-increment as DB handles it)
			if autoIncrementPKCol == nil && g.isPKValueUsed(table, columns, row) {
				// PK collision - retry
				if attempt == MaxPKRetries-1 {
					err = fmt.Errorf("failed to generate unique PK after %d attempts", MaxPKRetries)
					pkCollisions++
				}
				continue
			}

			// Mark PK as used (for non-auto-increment PKs)
			if autoIncrementPKCol == nil {
				g.markPKValueUsed(table, columns, row)
			}
			break
		}

		if err != nil {
			log.WithError(err).WithField("table", table).WithField("row", i).Error("Failed to generate row")
			failedGenerate++
			continue
		}

		// Insert row and get auto-generated ID (if any)
		generatedID, err := plugin.AddRowReturningID(config, schema, table, row)
		if err != nil {
			log.WithError(err).WithField("table", table).WithField("row", i).Error("Failed to insert row")
			// Return error immediately to trigger transaction rollback
			// This ensures atomicity - either all tables are populated or none are
			return 0, fmt.Errorf("failed to insert row %d into %s: %w", i, table, err)
		}

		// Track generated PK for FK references
		if autoIncrementPKCol != nil {
			// Auto-increment PK: track the database-generated ID
			if generatedID > 0 {
				g.trackAutoGeneratedPK(table, autoIncrementPKCol.Name, generatedID)
			} else {
				// Failed to get the auto-generated ID - this is a problem
				// because FK references won't work
				log.WithFields(map[string]any{
					"table":  table,
					"column": autoIncrementPKCol.Name,
					"row":    i,
				}).Warn("Failed to get auto-generated PK - FK references may fail")
			}
		} else {
			// Non-auto-increment PK: track from the generated row values
			g.trackGeneratedPK(table, columns, row)
		}
		generated++

		// Log progress every 100 rows for larger batches
		if rowCount >= 100 && generated%100 == 0 {
			log.WithFields(map[string]any{
				"table":    table,
				"progress": fmt.Sprintf("%d/%d", generated, rowCount),
			}).Debug("Row generation progress")
		}
	}

	if failedGenerate > 0 || failedInsert > 0 {
		log.WithFields(map[string]any{
			"table":          table,
			"failedGenerate": failedGenerate,
			"failedInsert":   failedInsert,
			"pkCollisions":   pkCollisions,
		}).Warn("Some rows failed during generation")
	}

	log.WithFields(map[string]any{
		"table":      table,
		"generated":  generated,
		"trackedPKs": len(g.generatedPKs[table]),
	}).Info("Completed single-row mock data generation")
	return generated, nil
}

// generateRow creates a single row with proper FK handling.
// Uses a per-row cache to ensure composite FKs use values from the same parent row.
func (g *Generator) generateRow(
	columns []engine.Column,
	constraints map[string]map[string]any,
	fks map[string]*engine.ForeignKeyRelationship,
	table string,
) ([]engine.Record, error) {
	records := make([]engine.Record, 0, len(columns))

	// Cache for selected parent rows - ensures composite FKs are consistent
	// Maps parent table name -> selected row (or nil for NULL)
	selectedParentRows := make(map[string]map[string]any)

	// Track skip reasons for debugging
	skippedAutoIncr := 0
	skippedComputed := 0
	skippedNil := 0

	for _, col := range columns {
		// Skip auto-increment columns - database will generate these values
		if col.IsAutoIncrement {
			log.WithFields(map[string]any{
				"table":  table,
				"column": col.Name,
			}).Debug("Skipping auto-increment column")
			skippedAutoIncr++
			continue
		}

		// Skip computed/database-managed columns
		if col.IsComputed {
			log.WithFields(map[string]any{
				"table":  table,
				"column": col.Name,
				"type":   col.Type,
			}).Debug("Skipping computed column")
			skippedComputed++
			continue
		}

		// Use case-insensitive lookup for constraints (needed for SQLite)
		colConstraints := getConstraintsForColumn(constraints, col.Name)
		var value any
		var err error

		// Check if this is a FK column
		if fk, isFk := fks[col.Name]; isFk {
			value, err = g.generateFKValue(col, colConstraints, fk, table, selectedParentRows)
			if err != nil {
				return nil, fmt.Errorf("failed to generate FK value for column %s: %w", col.Name, err)
			}
		} else {
			value = g.generateColumnValue(col, colConstraints)
		}

		// Skip nil values - let the database use default values instead of explicit NULLs.
		if value == nil {
			skippedNil++
			continue
		}

		// Get constraint type if available (important for MongoDB schema validation)
		constraintType := ""
		if colConstraints != nil {
			if t, ok := colConstraints["type"].(string); ok {
				constraintType = t
			}
		}

		// Convert value to record
		record := valueToRecord(col, value, constraintType)
		records = append(records, record)
	}

	// Log summary if no records were generated (indicates a bug)
	if len(records) == 0 {
		log.WithFields(map[string]any{
			"table":           table,
			"totalColumns":    len(columns),
			"skippedAutoIncr": skippedAutoIncr,
			"skippedComputed": skippedComputed,
			"skippedNil":      skippedNil,
		}).Warn("No records generated - all columns were skipped")
	}

	return records, nil
}

// generateFKValue generates a value for a FK column.
// selectedParentRows caches which parent row was selected for each table,
// ensuring composite FKs use consistent values from the same parent row.
// Returns an error if no parent PKs are available for a non-nullable FK.
func (g *Generator) generateFKValue(
	col engine.Column,
	constraints map[string]any,
	fk *engine.ForeignKeyRelationship,
	currentTable string,
	selectedParentRows map[string]map[string]any,
) (any, error) {
	isNullable := gorm_plugin.IsNullable(constraints)

	log.WithFields(map[string]any{
		"table":      currentTable,
		"column":     col.Name,
		"isNullable": isNullable,
		"fkTarget":   fk.ReferencedTable,
	}).Debug("Generating FK value")

	// Self-reference: return NULL if nullable, otherwise error
	if fk.ReferencedTable == currentTable {
		if isNullable {
			log.WithFields(map[string]any{
				"table":  currentTable,
				"column": col.Name,
			}).Debug("Self-referencing FK set to NULL")
			return nil, nil
		}
		// Non-nullable self-reference - cannot generate valid data
		return nil, fmt.Errorf("cannot generate data for non-nullable self-referencing FK column %s", col.Name)
	}

	// Check if we've already selected a row for this parent table
	// (to ensure composite FK columns are consistently NULL or not)
	if selectedRow, exists := selectedParentRows[fk.ReferencedTable]; exists {
		// Use the already-selected row for composite FK consistency
		if selectedRow == nil {
			// Previously decided to use NULL
			return nil, nil
		}
		value := selectedRow[fk.ReferencedColumn]
		log.WithFields(map[string]any{
			"table":            currentTable,
			"column":           col.Name,
			"parentTable":      fk.ReferencedTable,
			"referencedColumn": fk.ReferencedColumn,
			"value":            value,
		}).Debug("Using cached parent row for composite FK")
		return value, nil
	}

	// First FK column for this parent table - decide NULL or select a row
	if isNullable && g.faker.Float64() < NullableFKProbability {
		log.WithFields(map[string]any{
			"table":  currentTable,
			"column": col.Name,
		}).Debug("Nullable FK set to NULL (random)")
		selectedParentRows[fk.ReferencedTable] = nil // Cache the NULL decision
		return nil, nil
	}

	// Get parent PK rows
	parentRows := g.getPKRowsForTable(fk.ReferencedTable)
	log.WithFields(map[string]any{
		"column":          col.Name,
		"referencedTable": fk.ReferencedTable,
		"parentRowCount":  len(parentRows),
	}).Debug("Looking up parent PKs for FK")

	if len(parentRows) == 0 {
		// No parent PKs available - this is an error for non-nullable FK columns
		if isNullable {
			log.WithFields(map[string]any{
				"column":          col.Name,
				"referencedTable": fk.ReferencedTable,
			}).Debug("No parent PKs available, using NULL for nullable FK")
			selectedParentRows[fk.ReferencedTable] = nil
			return nil, nil
		}
		return nil, fmt.Errorf("no parent rows available for non-nullable FK column %s (references %s)", col.Name, fk.ReferencedTable)
	}

	// Pick a random parent row and cache it
	idx := g.faker.Number(0, len(parentRows)-1)
	selectedRow := parentRows[idx]
	selectedParentRows[fk.ReferencedTable] = selectedRow

	// Get the specific column value
	value := selectedRow[fk.ReferencedColumn]

	log.WithFields(map[string]any{
		"table":            currentTable,
		"column":           col.Name,
		"parentTable":      fk.ReferencedTable,
		"referencedColumn": fk.ReferencedColumn,
		"value":            value,
		"availableRows":    len(parentRows),
	}).Debug("Selected FK value from parent table")

	return value, nil
}

// generateColumnValue generates a value for a regular column.
func (g *Generator) generateColumnValue(col engine.Column, constraints map[string]any) any {
	// Check for nullable with random NULL
	if gorm_plugin.IsNullable(constraints) {
		if g.faker.Float64() < RegularNullProbability {
			return nil
		}
	}

	// Check for is_json constraint (MariaDB stores JSON as LONGTEXT with JSON_VALID check)
	if constraints != nil {
		if isJSON, ok := constraints["is_json"].(bool); ok && isJSON {
			return GenerateByType("json", g.databaseType, nil, g.faker)
		}
	}

	// Merge column metadata (length, precision, scale) into constraints if available
	effectiveConstraints := constraints
	needsCopy := false
	if col.Length != nil && *col.Length > 0 && (constraints == nil || constraints["length"] == nil) {
		needsCopy = true
	}
	if col.Precision != nil && *col.Precision > 0 && (constraints == nil || constraints["precision"] == nil) {
		needsCopy = true
	}
	if needsCopy {
		effectiveConstraints = make(map[string]any)
		for k, v := range constraints { // no-op if constraints is nil
			effectiveConstraints[k] = v
		}
		if col.Length != nil && *col.Length > 0 {
			effectiveConstraints["length"] = *col.Length
		}
		if col.Precision != nil && *col.Precision > 0 {
			effectiveConstraints["precision"] = *col.Precision
		}
		if col.Scale != nil && *col.Scale >= 0 {
			effectiveConstraints["scale"] = *col.Scale
		}
	}

	// Try column name pattern matching first (for text types)
	dbType := detectDatabaseType(col.Type)
	if dbType == "text" {
		// Skip pattern matching if we have check_values (ENUM)
		if effectiveConstraints != nil {
			if _, hasCheckValues := effectiveConstraints["check_values"]; hasCheckValues {
				return GenerateByType(col.Type, g.databaseType, effectiveConstraints, g.faker)
			}
		}

		maxLen := 0
		if col.Length != nil {
			maxLen = *col.Length
		}
		if val, matched := MatchColumnName(col.Name, maxLen, g.faker); matched {
			return val
		}
	}

	// Fall back to type-based generation
	return GenerateByType(col.Type, g.databaseType, effectiveConstraints, g.faker)
}
