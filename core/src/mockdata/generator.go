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
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
)

const (
	// DefaultForeignKeyDensityRatio determines parent row count: requestedRows / ratio = parent rows
	// E.g., 20 child rows with ratio 20 → 1 parent row
	// This can be overridden via the FKDensityRatio parameter in Generate()
	DefaultForeignKeyDensityRatio = 20

	// NullableFKProbability is the chance a nullable FK column gets NULL
	NullableFKProbability = 0.2

	// RegularNullProbability is the chance a regular nullable column gets NULL
	RegularNullProbability = 0.1

	// MaxPKRetries is the maximum number of attempts to generate a unique PK value
	MaxPKRetries = 100

	// MaxExistingPKsToLoad limits how many existing rows to load for PK uniqueness checks.
	// Beyond this limit, uniqueness violations may occur for tables with many rows.
	MaxExistingPKsToLoad = 10000
)

// Use shared database type sets
var (
	intTypes      = common.IntTypes
	uintTypes     = common.UintTypes
	floatTypes    = common.FloatTypes
	boolTypes     = common.BoolTypes
	dateTypes     = common.DateTypes
	dateTimeTypes = common.DateTimeTypes
	uuidTypes     = common.UuidTypes
	textTypes     = common.TextTypes
	jsonTypes     = common.JsonTypes
)

// TableDependency represents a table in the dependency chain with generation info.
type TableDependency struct {
	Table            string
	DependsOn        []string
	RowCount         int
	IsBlocked        bool
	UsesExistingData bool
}

// GenerationResult contains the result of mock data generation.
type GenerationResult struct {
	TotalGenerated int
	Details        []TableDetail
	Warnings       []string
}

// TableDetail contains per-table generation info.
type TableDetail struct {
	Table            string
	RowsGenerated    int
	UsedExistingData bool
}

// DependencyAnalysis contains the result of analyzing table dependencies.
type DependencyAnalysis struct {
	GenerationOrder []string
	Tables          []TableDependency
	TotalRows       int
	Warnings        []string
	Error           string
}

// Generator handles mock data generation with FK support.
type Generator struct {
	faker          *gofakeit.Faker
	fkDensityRatio int                         // Parent rows = child rows / ratio (default: 20)
	generatedPKs   map[string][]map[string]any // Cache of generated PK rows per table (supports composite PKs)
	existingPKs    map[string][]map[string]any // Cache of existing PK rows for blocked tables
	usedPKValues   map[string]map[string]bool  // Track used PK values: table -> pkValueString -> true
	databaseType   string                      // Database type for type-specific generation (e.g., "MSSQL", "PostgreSQL")
}

// NewGenerator creates a new mock data generator with the specified FK density ratio.
// The ratio determines how many parent rows are created per N child rows.
// E.g., ratio=20 means 1 parent row per 20 child rows.
// Pass 0 or negative to use the default ratio.
func NewGenerator(fkDensityRatio int) *Generator {
	if fkDensityRatio <= 0 {
		fkDensityRatio = DefaultForeignKeyDensityRatio
	}
	return &Generator{
		faker:          gofakeit.New(uint64(time.Now().UnixNano())),
		fkDensityRatio: fkDensityRatio,
		generatedPKs:   make(map[string][]map[string]any),
		existingPKs:    make(map[string][]map[string]any),
		usedPKValues:   make(map[string]map[string]bool),
	}
}

// findChildTables returns all tables that have FK references to the given table.
// The graph structure stores parent -> children relationships, so we look at the
// target table's Relations to find its children (tables that reference it).
func findChildTables(graph []engine.GraphUnit, targetTable string) []string {
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

// clearTableWithDependencies clears a table and all tables that reference it (children first).
// This ensures FK constraints are respected by deleting in the correct order.
func (g *Generator) clearTableWithDependencies(
	plugin engine.PluginFunctions,
	config *engine.PluginConfig,
	schema, table string,
	graph []engine.GraphUnit,
	cleared map[string]bool,
) error {
	// Avoid clearing the same table twice (handles cycles)
	if cleared[table] {
		return nil
	}

	// Find tables that reference this table (children)
	children := findChildTables(graph, table)

	// Clear children first (recursive)
	for _, child := range children {
		if err := g.clearTableWithDependencies(plugin, config, schema, child, graph, cleared); err != nil {
			return err
		}
	}

	// Now safe to clear this table
	log.Logger.WithFields(map[string]any{
		"table":    table,
		"children": children,
	}).Debug("Clearing table data (children already cleared)")

	if _, err := plugin.ClearTableData(config, schema, table); err != nil {
		return fmt.Errorf("failed to clear table %s: %w", table, err)
	}

	cleared[table] = true
	g.usedPKValues[table] = make(map[string]bool)
	return nil
}

// AnalyzeDependencies collects all tables in dependency order with cycle detection.
// Returns an analysis result with ordered tables, row counts, and any errors.
func (g *Generator) AnalyzeDependencies(
	plugin engine.PluginFunctions,
	config *engine.PluginConfig,
	schema, table string,
	rowCount int,
) (*DependencyAnalysis, error) {
	analysis := &DependencyAnalysis{
		Warnings: []string{},
	}

	// Collect all FK relationships recursively
	visited := make(map[string]bool)
	tableInfo := make(map[string]*TableDependency)
	adjacency := make(map[string][]string)

	if err := g.collectDependencies(plugin, config, schema, table, rowCount, visited, tableInfo, adjacency); err != nil {
		analysis.Error = err.Error()
		return analysis, nil
	}

	// Topological sort with cycle detection
	order, err := topoSort(adjacency)
	if err != nil {
		analysis.Error = err.Error()
		return analysis, nil
	}

	// Build result in generation order
	analysis.GenerationOrder = order
	for _, tbl := range order {
		if info, ok := tableInfo[tbl]; ok {
			analysis.Tables = append(analysis.Tables, *info)
			analysis.TotalRows += info.RowCount
			if info.UsesExistingData {
				analysis.Warnings = append(analysis.Warnings,
					fmt.Sprintf("Table '%s' has mock data disabled; will use existing data", tbl))
			}
		}
	}

	return analysis, nil
}

// collectDependencies recursively collects all table dependencies.
func (g *Generator) collectDependencies(
	plugin engine.PluginFunctions,
	config *engine.PluginConfig,
	schema, table string,
	rowCount int,
	visited map[string]bool,
	tableInfo map[string]*TableDependency,
	adjacency map[string][]string,
) error {
	if visited[table] {
		log.Logger.WithField("table", table).Debug("Table already visited, skipping")
		return nil
	}
	visited[table] = true

	log.Logger.WithFields(map[string]any{
		"table":    table,
		"rowCount": rowCount,
	}).Debug("Collecting dependencies for table")

	// Get FK relationships for this table
	fks, err := plugin.GetForeignKeyRelationships(config, schema, table)
	if err != nil {
		log.Logger.WithError(err).WithField("table", table).Warn("Failed to get FK relationships")
		fks = make(map[string]*engine.ForeignKeyRelationship)
	}

	if len(fks) > 0 {
		fkColumns := make([]string, 0, len(fks))
		for col := range fks {
			fkColumns = append(fkColumns, col)
		}
		log.Logger.WithFields(map[string]any{
			"table":     table,
			"fkColumns": fkColumns,
			"fkCount":   len(fks),
		}).Debug("Found foreign key relationships")
	}

	// Check if mock data generation is allowed using env function
	isBlocked := !env.IsMockDataGenerationAllowed(table)
	if isBlocked {
		log.Logger.WithField("table", table).Debug("Mock data generation blocked for table")
	}

	// Create table info
	info := &TableDependency{
		Table:            table,
		RowCount:         rowCount,
		IsBlocked:        isBlocked,
		UsesExistingData: isBlocked,
		DependsOn:        []string{},
	}

	// Ensure table is in adjacency map even if it has no FKs
	// This is needed for topoSort to include tables with no dependencies
	if _, exists := adjacency[table]; !exists {
		adjacency[table] = []string{}
	}

	// Process each FK
	for colName, fk := range fks {
		// Skip self-references
		if fk.ReferencedTable == table {
			log.Logger.WithFields(map[string]any{
				"table":  table,
				"column": colName,
			}).Debug("Skipping self-referencing FK")
			continue
		}

		info.DependsOn = append(info.DependsOn, fk.ReferencedTable)

		// Track adjacency for topo sort (child depends on parent)
		adjacency[table] = append(adjacency[table], fk.ReferencedTable)

		// Calculate parent row count based on density ratio
		// Ensure at least 1 parent row exists
		parentRowCount := max(1, rowCount/g.fkDensityRatio)

		log.Logger.WithFields(map[string]any{
			"childTable":     table,
			"parentTable":    fk.ReferencedTable,
			"column":         colName,
			"parentRowCount": parentRowCount,
		}).Debug("Following FK to parent table")

		// Recursively collect parent dependencies
		if err := g.collectDependencies(plugin, config, schema, fk.ReferencedTable, parentRowCount, visited, tableInfo, adjacency); err != nil {
			return err
		}
	}

	tableInfo[table] = info
	return nil
}

// topoSort performs topological sort using Kahn's algorithm with cycle detection.
// Returns tables in order they should be populated (parents before children).
// adjacency maps: child -> [parents it depends on]
func topoSort(adjacency map[string][]string) ([]string, error) {
	// inDegree[node] = number of dependencies (parents) that node has
	inDegree := make(map[string]int)
	// reverse maps: parent -> [children that depend on it]
	reverse := make(map[string][]string)

	// Build in-degree and reverse adjacency
	for node, deps := range adjacency {
		// node's in-degree is the count of its dependencies
		inDegree[node] = len(deps)

		for _, dep := range deps {
			// Ensure parent is in inDegree map (may have 0 deps itself)
			if _, ok := inDegree[dep]; !ok {
				inDegree[dep] = 0
			}
			// Track that node depends on dep (for when dep is processed)
			reverse[dep] = append(reverse[dep], node)
		}
	}

	log.Logger.WithFields(map[string]any{
		"inDegree": inDegree,
	}).Debug("Topological sort in-degree calculated")

	// Queue nodes with no dependencies (parents/root tables)
	var queue []string
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	log.Logger.WithField("initialQueue", queue).Debug("Topological sort starting with root tables")

	var result []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		// When a parent is processed, reduce in-degree of its children
		for _, child := range reverse[node] {
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	// Check for cycle - if not all nodes processed, there's a cycle
	if len(result) != len(inDegree) {
		var cycleNodes []string
		for node, degree := range inDegree {
			if degree > 0 {
				cycleNodes = append(cycleNodes, node)
			}
		}
		return nil, fmt.Errorf("circular dependency detected involving tables: %s", strings.Join(cycleNodes, " → "))
	}

	log.Logger.WithField("order", result).Debug("Topological sort complete")
	return result, nil
}

// Generate creates mock data for the target table and its FK dependencies.
// All inserts are wrapped in a transaction for atomicity - either all tables
// are populated successfully, or all changes are rolled back on any error.
func (g *Generator) Generate(
	plugin engine.PluginFunctions,
	config *engine.PluginConfig,
	schema, table string,
	rowCount int,
	overwrite bool,
) (*GenerationResult, error) {
	log.Logger.WithFields(map[string]any{
		"schema":    schema,
		"table":     table,
		"rowCount":  rowCount,
		"overwrite": overwrite,
	}).Info("Starting mock data generation")

	// Reset caches for fresh generation
	g.generatedPKs = make(map[string][]map[string]any)
	g.existingPKs = make(map[string][]map[string]any)
	g.usedPKValues = make(map[string]map[string]bool)

	// Set database type for type-specific generation
	if metadata := plugin.GetDatabaseMetadata(); metadata != nil {
		g.databaseType = string(metadata.DatabaseType)
	}

	result := &GenerationResult{
		Details:  []TableDetail{},
		Warnings: []string{},
	}

	// Analyze dependencies (read-only, outside transaction)
	log.Logger.Debug("Analyzing table dependencies")
	analysis, err := g.AnalyzeDependencies(plugin, config, schema, table, rowCount)
	if err != nil {
		return nil, err
	}

	if analysis.Error != "" {
		log.Logger.WithField("error", analysis.Error).Error("Dependency analysis failed")
		return nil, fmt.Errorf("%s", analysis.Error)
	}

	log.Logger.WithFields(map[string]any{
		"generationOrder": analysis.GenerationOrder,
		"totalTables":     len(analysis.Tables),
		"totalRows":       analysis.TotalRows,
	}).Info("Dependency analysis complete")

	result.Warnings = analysis.Warnings

	// If overwrite mode, clear target table and all child tables first (in FK-safe order)
	if overwrite {
		log.Logger.Debug("Overwrite mode: clearing target table and child tables")
		graph, err := plugin.GetGraph(config, schema)
		if err != nil {
			log.Logger.WithError(err).Warn("Failed to get graph for FK-safe clearing, attempting direct clear")
			// Fall back to direct clear (may fail with FK constraints)
			if _, err := plugin.ClearTableData(config, schema, table); err != nil {
				return nil, fmt.Errorf("failed to clear table %s: %w", table, err)
			}
		} else {
			cleared := make(map[string]bool)
			if err := g.clearTableWithDependencies(plugin, config, schema, table, graph, cleared); err != nil {
				return nil, fmt.Errorf("failed to clear tables for overwrite: %w", err)
			}
			log.Logger.WithField("clearedTables", len(cleared)).Info("Cleared tables for overwrite")
		}
	}

	// Wrap all inserts in a transaction for atomicity
	// If any insert fails, all changes are rolled back
	err = plugin.WithTransaction(config, func(tx any) error {
		// Create a transactional config that passes the transaction to all operations
		txConfig := &engine.PluginConfig{
			Credentials:   config.Credentials,
			ExternalModel: config.ExternalModel,
			Transaction:   tx,
		}

		// Generate data for each table in order
		for i, tblDep := range analysis.Tables {
			log.Logger.WithFields(map[string]any{
				"step":      fmt.Sprintf("%d/%d", i+1, len(analysis.Tables)),
				"table":     tblDep.Table,
				"rowCount":  tblDep.RowCount,
				"dependsOn": tblDep.DependsOn,
			}).Info("Processing table")

			// Load existing PKs for blocked tables
			if tblDep.UsesExistingData {
				log.Logger.WithField("table", tblDep.Table).Debug("Table uses existing data, loading PKs")
				if err := g.loadExistingPKs(plugin, txConfig, schema, tblDep.Table); err != nil {
					return fmt.Errorf("failed to load existing PKs for %s: %w", tblDep.Table, err)
				}
				result.Details = append(result.Details, TableDetail{
					Table:            tblDep.Table,
					RowsGenerated:    0,
					UsedExistingData: true,
				})
				continue
			}

			// Generate rows for this table
			isTargetTable := tblDep.Table == table
			generated, err := g.generateTableRows(plugin, txConfig, schema, tblDep.Table, tblDep.RowCount, overwrite && isTargetTable, isTargetTable)
			if err != nil {
				return fmt.Errorf("failed to generate rows for %s: %w", tblDep.Table, err)
			}

			result.TotalGenerated += generated
			result.Details = append(result.Details, TableDetail{
				Table:            tblDep.Table,
				RowsGenerated:    generated,
				UsedExistingData: false,
			})
		}

		return nil
	})

	if err != nil {
		log.Logger.WithError(err).Error("Mock data generation failed, transaction rolled back")
		return nil, err
	}

	log.Logger.WithFields(map[string]any{
		"totalGenerated": result.TotalGenerated,
		"tablesCount":    len(result.Details),
	}).Info("Mock data generation complete")

	return result, nil
}

// loadExistingPKs loads primary key values from an existing table.
// Supports composite PKs by storing maps of column->value.
func (g *Generator) loadExistingPKs(
	plugin engine.PluginFunctions,
	config *engine.PluginConfig,
	schema, table string,
) error {
	log.Logger.WithField("table", table).Debug("Loading existing PKs from table")

	// Get columns to find all PK columns
	columns, err := plugin.GetColumnsForTable(config, schema, table)
	if err != nil {
		return err
	}

	// Collect all PK column names
	var pkColNames []string
	for _, col := range columns {
		if col.IsPrimary {
			pkColNames = append(pkColNames, col.Name)
		}
	}

	if len(pkColNames) == 0 {
		// No PK found, try to use first column
		if len(columns) > 0 {
			pkColNames = []string{columns[0].Name}
			log.Logger.WithFields(map[string]any{
				"table":  table,
				"column": columns[0].Name,
			}).Debug("No PK column found, using first column")
		} else {
			return fmt.Errorf("table %s has no columns", table)
		}
	} else {
		log.Logger.WithFields(map[string]any{
			"table":     table,
			"pkColumns": pkColNames,
		}).Debug("Found PK columns")
	}

	// Get existing rows
	rows, err := plugin.GetRows(config, schema, table, nil, nil, 50, 0)
	if err != nil {
		return err
	}

	// Build column name -> index map
	colIdxMap := make(map[string]int)
	for i, col := range rows.Columns {
		colIdxMap[col.Name] = i
	}

	// Extract PK values for each row
	pks := make([]map[string]any, 0, len(rows.Rows))
	for _, row := range rows.Rows {
		pkRow := make(map[string]any)
		for _, pkColName := range pkColNames {
			if idx, exists := colIdxMap[pkColName]; exists && idx < len(row) {
				pkRow[pkColName] = row[idx]
			}
		}
		if len(pkRow) > 0 {
			pks = append(pks, pkRow)
		}
	}

	g.existingPKs[table] = pks
	log.Logger.WithFields(map[string]any{
		"table":     table,
		"pkCount":   len(pks),
		"pkColumns": pkColNames,
	}).Debug("Loaded existing PKs")

	return nil
}

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
	log.Logger.WithFields(map[string]any{
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
	log.Logger.WithFields(map[string]any{
		"table":          table,
		"columnCount":    len(columns),
		"specialColumns": columnDetails,
	}).Debug("Retrieved column definitions")

	constraints, err := plugin.GetColumnConstraints(config, schema, table)
	if err != nil {
		log.Logger.WithError(err).WithField("table", table).Warn("Failed to get constraints, using defaults")
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
		log.Logger.WithFields(map[string]any{
			"table":            table,
			"constraintCount":  len(constraints),
			"checkConstraints": constraintDetails,
		}).Debug("Retrieved column constraints")
	}

	// Get FK relationships
	fks, err := plugin.GetForeignKeyRelationships(config, schema, table)
	if err != nil {
		log.Logger.WithError(err).WithField("table", table).Warn("Failed to get FK relationships")
		fks = make(map[string]*engine.ForeignKeyRelationship)
	} else if len(fks) > 0 {
		log.Logger.WithFields(map[string]any{
			"table":   table,
			"fkCount": len(fks),
		}).Debug("Retrieved FK relationships")
	}

	// Load existing PKs to prevent uniqueness violations (only when not overwriting)
	// When overwriting, tables are already cleared at the top level in Generate()
	if !overwrite {
		if err := g.loadExistingPKsForUniqueness(plugin, config, schema, table, columns); err != nil {
			log.Logger.WithError(err).WithField("table", table).Warn("Failed to load existing PKs for uniqueness check")
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
	log.Logger.WithFields(map[string]any{
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
			log.Logger.WithError(err).WithField("table", table).WithField("row", i).Error("Failed to generate row")
			failedCount++
			continue
		}
		rows = append(rows, row)
	}

	if failedCount > 0 {
		log.Logger.WithFields(map[string]any{
			"table":        table,
			"failedCount":  failedCount,
			"pkCollisions": pkCollisions,
		}).Warn("Some rows failed to generate")
	}

	if len(rows) == 0 {
		log.Logger.WithField("table", table).Warn("No rows generated")
		return 0, nil
	}

	log.Logger.WithFields(map[string]any{
		"table":    table,
		"rowCount": len(rows),
	}).Debug("Generated rows, starting bulk insert")

	// Bulk insert all rows
	if _, err := plugin.BulkAddRows(config, schema, table, rows); err != nil {
		log.Logger.WithError(err).WithField("table", table).WithField("rowCount", len(rows)).Error("Failed to bulk insert rows")
		return 0, fmt.Errorf("failed to bulk insert rows: %w", err)
	}

	log.Logger.WithFields(map[string]any{
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
	log.Logger.WithFields(map[string]any{
		"table":    table,
		"rowCount": rowCount,
	}).Debug("Starting single-row generation (PK tracking enabled)")

	// Find auto-increment PK column (if any) for tracking database-generated IDs
	var autoIncrementPKCol *engine.Column
	for i := range columns {
		if columns[i].IsPrimary && columns[i].IsAutoIncrement {
			autoIncrementPKCol = &columns[i]
			log.Logger.WithFields(map[string]any{
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
			log.Logger.WithError(err).WithField("table", table).WithField("row", i).Error("Failed to generate row")
			failedGenerate++
			continue
		}

		// Insert row and get auto-generated ID (if any)
		generatedID, err := plugin.AddRowReturningID(config, schema, table, row)
		if err != nil {
			log.Logger.WithError(err).WithField("table", table).WithField("row", i).Error("Failed to insert row")
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
				log.Logger.WithFields(map[string]any{
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
			log.Logger.WithFields(map[string]any{
				"table":    table,
				"progress": fmt.Sprintf("%d/%d", generated, rowCount),
			}).Debug("Row generation progress")
		}
	}

	if failedGenerate > 0 || failedInsert > 0 {
		log.Logger.WithFields(map[string]any{
			"table":          table,
			"failedGenerate": failedGenerate,
			"failedInsert":   failedInsert,
			"pkCollisions":   pkCollisions,
		}).Warn("Some rows failed during generation")
	}

	log.Logger.WithFields(map[string]any{
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
			log.Logger.WithFields(map[string]any{
				"table":  table,
				"column": col.Name,
			}).Debug("Skipping auto-increment column")
			skippedAutoIncr++
			continue
		}

		// Skip computed/database-managed columns
		if col.IsComputed {
			log.Logger.WithFields(map[string]any{
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
		log.Logger.WithFields(map[string]any{
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
	isNullable := false
	if constraints != nil {
		if n, ok := constraints["nullable"].(bool); ok {
			isNullable = n
		}
	}

	log.Logger.WithFields(map[string]any{
		"table":      currentTable,
		"column":     col.Name,
		"isNullable": isNullable,
		"fkTarget":   fk.ReferencedTable,
	}).Debug("Generating FK value")

	// Self-reference: return NULL if nullable, otherwise error
	if fk.ReferencedTable == currentTable {
		if isNullable {
			log.Logger.WithFields(map[string]any{
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
		log.Logger.WithFields(map[string]any{
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
		log.Logger.WithFields(map[string]any{
			"table":  currentTable,
			"column": col.Name,
		}).Debug("Nullable FK set to NULL (random)")
		selectedParentRows[fk.ReferencedTable] = nil // Cache the NULL decision
		return nil, nil
	}

	// Get parent PK rows
	parentRows := g.getPKRowsForTable(fk.ReferencedTable)
	log.Logger.WithFields(map[string]any{
		"column":          col.Name,
		"referencedTable": fk.ReferencedTable,
		"parentRowCount":  len(parentRows),
	}).Debug("Looking up parent PKs for FK")

	if len(parentRows) == 0 {
		// No parent PKs available - this is an error for non-nullable FK columns
		if isNullable {
			log.Logger.WithFields(map[string]any{
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

	log.Logger.WithFields(map[string]any{
		"table":            currentTable,
		"column":           col.Name,
		"parentTable":      fk.ReferencedTable,
		"referencedColumn": fk.ReferencedColumn,
		"value":            value,
		"availableRows":    len(parentRows),
	}).Debug("Selected FK value from parent table")

	return value, nil
}

// getPKRowsForTable returns available PK rows for a table.
// Each row is a map of column name -> value to support composite PKs.
func (g *Generator) getPKRowsForTable(table string) []map[string]any {
	// Check generated PKs first
	if pks, ok := g.generatedPKs[table]; ok && len(pks) > 0 {
		return pks
	}
	// Fall back to existing PKs
	if pks, ok := g.existingPKs[table]; ok && len(pks) > 0 {
		return pks
	}
	return nil
}

// trackAutoGeneratedPK stores an auto-generated PK value from the database.
// Used when the PK is auto-increment and the database generates the value.
func (g *Generator) trackAutoGeneratedPK(table string, pkColumnName string, generatedID int64) {
	pkRow := map[string]any{
		pkColumnName: generatedID,
	}
	g.generatedPKs[table] = append(g.generatedPKs[table], pkRow)
	log.Logger.WithFields(map[string]any{
		"table":     table,
		"pkColumn":  pkColumnName,
		"pkValue":   generatedID,
		"totalRows": len(g.generatedPKs[table]),
	}).Debug("Tracked auto-generated PK")
}

// trackGeneratedPK stores all PK column values from a generated row.
// Supports composite PKs by storing a map of column->value.
func (g *Generator) trackGeneratedPK(table string, columns []engine.Column, row []engine.Record) {
	// Build a map of record key -> value for quick lookup
	rowValues := make(map[string]string)
	for _, rec := range row {
		rowValues[rec.Key] = rec.Value
	}

	// Collect all PK column values
	pkRow := make(map[string]any)
	for _, col := range columns {
		if col.IsPrimary {
			if val, exists := rowValues[col.Name]; exists {
				pkRow[col.Name] = val
			}
		}
	}

	if len(pkRow) > 0 {
		g.generatedPKs[table] = append(g.generatedPKs[table], pkRow)
		log.Logger.WithFields(map[string]any{
			"table":     table,
			"pkColumns": pkRow,
			"totalRows": len(g.generatedPKs[table]),
		}).Debug("Tracked generated PK row")
	}
}

// generateColumnValue generates a value for a regular column.
func (g *Generator) generateColumnValue(col engine.Column, constraints map[string]any) any {
	// Check for nullable with random NULL
	if constraints != nil {
		if n, ok := constraints["nullable"].(bool); ok && n {
			if g.faker.Float64() < RegularNullProbability {
				return nil
			}
		}
	}

	// Check for is_json constraint (MariaDB stores JSON as LONGTEXT with JSON_VALID check)
	if constraints != nil {
		if isJSON, ok := constraints["is_json"].(bool); ok && isJSON {
			return GenerateByType("json", g.databaseType, nil, g.faker)
		}
	}

	// Merge column length into constraints if available
	effectiveConstraints := constraints
	if col.Length != nil && *col.Length > 0 {
		if constraints == nil || constraints["length"] == nil {
			effectiveConstraints = make(map[string]any)
			for k, v := range constraints { // no-op if constraints is nil
				effectiveConstraints[k] = v
			}
			effectiveConstraints["length"] = *col.Length
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

// valueToRecord converts a value to an engine.Record.
// If constraintType is provided (non-empty), it overrides the column type for type hints.
// This is important for MongoDB where schema validation types are more authoritative
// than types inferred from documents.
func valueToRecord(col engine.Column, value any, constraintType string) engine.Record {
	typeHint := col.Type
	if constraintType != "" {
		typeHint = constraintType
	}

	extra := map[string]string{
		"Type": typeHint,
	}

	var valueStr string
	if value == nil {
		valueStr = ""
		extra["IsNull"] = "true"
	} else {
		valueStr = fmt.Sprintf("%v", value)
	}

	return engine.Record{
		Key:   col.Name,
		Value: valueStr,
		Extra: extra,
	}
}

// detectDatabaseType returns the simplified database type for a column.
func detectDatabaseType(columnType string) string {
	upperType := strings.ToUpper(columnType)

	// Handle PostgreSQL arrays first
	if strings.Contains(upperType, "[]") {
		return "array"
	}

	// Handle timestamp/time types with timezone suffixes before other checks
	// PostgreSQL uses "TIMESTAMP WITH TIME ZONE", "TIME WITH TIME ZONE", etc.
	if strings.HasPrefix(upperType, "TIMESTAMP") {
		return "datetime"
	}
	if strings.HasPrefix(upperType, "TIME") && !strings.HasPrefix(upperType, "TINYINT") {
		return "datetime"
	}

	// Remove size specifiers like VARCHAR(255) -> VARCHAR
	if idx := strings.Index(upperType, "("); idx > 0 {
		upperType = upperType[:idx]
	}
	upperType = strings.TrimSpace(upperType)

	switch {
	case intTypes.Contains(upperType):
		return "int"
	case uintTypes.Contains(upperType):
		return "uint"
	case floatTypes.Contains(upperType):
		return "float"
	case boolTypes.Contains(upperType):
		return "bool"
	case dateTypes.Contains(upperType):
		return "date"
	case dateTimeTypes.Contains(upperType):
		return "datetime"
	case uuidTypes.Contains(upperType):
		return "uuid"
	case jsonTypes.Contains(upperType):
		return "json"
	case textTypes.Contains(upperType):
		return "text"
	default:
		return "text"
	}
}

// getConstraintsForColumn performs case-insensitive lookup for column constraints.
// This is necessary because SQLite (and some other databases) treat column names
// as case-insensitive, but the constraint map keys might not match the column case.
func getConstraintsForColumn(constraints map[string]map[string]any, columnName string) map[string]any {
	// Try exact match first
	if c, ok := constraints[columnName]; ok {
		return c
	}
	// Try case-insensitive match
	lowerName := strings.ToLower(columnName)
	for key, value := range constraints {
		if strings.ToLower(key) == lowerName {
			return value
		}
	}
	return nil
}

// pkValueKey creates a string key for a PK value to use in the usedPKValues set.
// For composite PKs, concatenates all column values with a separator.
func pkValueKey(pkColumns []engine.Column, row []engine.Record) string {
	// Build a map of record key -> value for quick lookup
	rowValues := make(map[string]string)
	for _, rec := range row {
		rowValues[rec.Key] = rec.Value
	}

	// Concatenate PK column values
	var parts []string
	for _, col := range pkColumns {
		if col.IsPrimary && !col.IsAutoIncrement {
			if val, exists := rowValues[col.Name]; exists {
				parts = append(parts, fmt.Sprintf("%s=%s", col.Name, val))
			}
		}
	}
	return strings.Join(parts, "|")
}

// loadExistingPKsForUniqueness loads existing PK values from a table into the usedPKValues set.
// This prevents generating PK values that already exist in the database.
func (g *Generator) loadExistingPKsForUniqueness(
	plugin engine.PluginFunctions,
	config *engine.PluginConfig,
	schema, table string,
	columns []engine.Column,
) error {
	// Find non-auto-increment PK columns
	var pkColumns []engine.Column
	for _, col := range columns {
		if col.IsPrimary && !col.IsAutoIncrement {
			pkColumns = append(pkColumns, col)
		}
	}

	// No non-auto-increment PKs to check
	if len(pkColumns) == 0 {
		return nil
	}

	log.Logger.WithFields(map[string]any{
		"table":     table,
		"pkColumns": len(pkColumns),
	}).Debug("Loading existing PK values for uniqueness check")

	// Initialize the used PK set for this table
	if g.usedPKValues[table] == nil {
		g.usedPKValues[table] = make(map[string]bool)
	}

	// Get existing rows (limit to avoid memory issues)
	rows, err := plugin.GetRows(config, schema, table, nil, nil, MaxExistingPKsToLoad, 0)
	if err != nil {
		return fmt.Errorf("failed to load existing rows for PK uniqueness check: %w", err)
	}

	// Build column name -> index map
	colIdxMap := make(map[string]int)
	for i, col := range rows.Columns {
		colIdxMap[col.Name] = i
	}

	// Extract and store PK values
	for _, row := range rows.Rows {
		// Build a pseudo-record for pkValueKey
		records := make([]engine.Record, 0, len(pkColumns))
		for _, pkCol := range pkColumns {
			if idx, exists := colIdxMap[pkCol.Name]; exists && idx < len(row) {
				records = append(records, engine.Record{Key: pkCol.Name, Value: row[idx]})
			}
		}
		if len(records) > 0 {
			key := pkValueKey(pkColumns, records)
			g.usedPKValues[table][key] = true
		}
	}

	log.Logger.WithFields(map[string]any{
		"table":          table,
		"existingPKs":    len(g.usedPKValues[table]),
		"totalRowsFound": len(rows.Rows),
	}).Debug("Loaded existing PK values")

	return nil
}

// isPKValueUsed checks if a PK value combination is already used.
func (g *Generator) isPKValueUsed(table string, columns []engine.Column, row []engine.Record) bool {
	if g.usedPKValues[table] == nil {
		return false
	}
	key := pkValueKey(columns, row)
	if key == "" {
		return false // No non-auto-increment PKs
	}
	return g.usedPKValues[table][key]
}

// markPKValueUsed marks a PK value combination as used.
func (g *Generator) markPKValueUsed(table string, columns []engine.Column, row []engine.Record) {
	if g.usedPKValues[table] == nil {
		g.usedPKValues[table] = make(map[string]bool)
	}
	key := pkValueKey(columns, row)
	if key != "" {
		g.usedPKValues[table][key] = true
	}
}
