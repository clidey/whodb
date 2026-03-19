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
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
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
	databaseType   string                      // Database type for type-specific generation (e.g., "PostgreSQL")
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
		log.WithField("table", table).Debug("Table already visited, skipping")
		return nil
	}
	visited[table] = true

	log.WithFields(map[string]any{
		"table":    table,
		"rowCount": rowCount,
	}).Debug("Collecting dependencies for table")

	// Get FK relationships for this table
	fks, err := plugin.GetForeignKeyRelationships(config, schema, table)
	if err != nil {
		log.WithError(err).WithField("table", table).Warn("Failed to get FK relationships")
		fks = make(map[string]*engine.ForeignKeyRelationship)
	}

	if len(fks) > 0 {
		fkColumns := make([]string, 0, len(fks))
		for col := range fks {
			fkColumns = append(fkColumns, col)
		}
		log.WithFields(map[string]any{
			"table":     table,
			"fkColumns": fkColumns,
			"fkCount":   len(fks),
		}).Debug("Found foreign key relationships")
	}

	// Check if mock data generation is allowed using env function
	isBlocked := !IsMockDataGenerationAllowed(table)
	if isBlocked {
		log.WithField("table", table).Debug("Mock data generation blocked for table")
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
			log.WithFields(map[string]any{
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

		log.WithFields(map[string]any{
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

	log.WithFields(map[string]any{
		"inDegree": inDegree,
	}).Debug("Topological sort in-degree calculated")

	// Queue nodes with no dependencies (parents/root tables)
	var queue []string
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	log.WithField("initialQueue", queue).Debug("Topological sort starting with root tables")

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

	log.WithField("order", result).Debug("Topological sort complete")
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
	log.WithFields(map[string]any{
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
	log.Debug("Analyzing table dependencies")
	analysis, err := g.AnalyzeDependencies(plugin, config, schema, table, rowCount)
	if err != nil {
		return nil, err
	}

	if analysis.Error != "" {
		log.WithField("error", analysis.Error).Error("Dependency analysis failed")
		return nil, fmt.Errorf("%s", analysis.Error)
	}

	log.WithFields(map[string]any{
		"generationOrder": analysis.GenerationOrder,
		"totalTables":     len(analysis.Tables),
		"totalRows":       analysis.TotalRows,
	}).Info("Dependency analysis complete")

	result.Warnings = analysis.Warnings

	// If overwrite mode, clear target table and all child tables first (in FK-safe order)
	if overwrite {
		log.Debug("Overwrite mode: clearing target table and child tables")
		graph, err := plugin.GetGraph(config, schema)
		if err != nil {
			log.WithError(err).Warn("Failed to get graph for FK-safe clearing, attempting direct clear")
			// Fall back to direct clear (may fail with FK constraints)
			if _, err := plugin.ClearTableData(config, schema, table); err != nil {
				return nil, fmt.Errorf("failed to clear table %s: %w", table, err)
			}
		} else {
			if err := plugins.ClearTableWithDependencies(plugin, config, schema, table, graph); err != nil {
				return nil, fmt.Errorf("failed to clear tables for overwrite: %w", err)
			}
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
			log.WithFields(map[string]any{
				"step":      fmt.Sprintf("%d/%d", i+1, len(analysis.Tables)),
				"table":     tblDep.Table,
				"rowCount":  tblDep.RowCount,
				"dependsOn": tblDep.DependsOn,
			}).Info("Processing table")

			// Load existing PKs for blocked tables
			if tblDep.UsesExistingData {
				log.WithField("table", tblDep.Table).Debug("Table uses existing data, loading PKs")
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
		log.WithError(err).Error("Mock data generation failed, transaction rolled back")
		return nil, err
	}

	log.WithFields(map[string]any{
		"totalGenerated": result.TotalGenerated,
		"tablesCount":    len(result.Details),
	}).Info("Mock data generation complete")

	return result, nil
}
