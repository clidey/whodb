/*
 * Copyright 2025 Clidey, Inc.
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

package duckdb

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

func (p *DuckDBPlugin) GetGraph(config *engine.PluginConfig, schema string) (*model.Graph, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*model.Graph, error) {
		tables, err := p.getTablesWithRelationships(db, schema)
		if err != nil {
			return nil, err
		}

		nodes := make([]*model.Node, 0, len(tables))
		edges := make([]*model.Edge, 0)

		for _, table := range tables {
			node := &model.Node{
				ID:    table.Name,
				Label: table.Name,
				Type:  "table",
				Metadata: map[string]interface{}{
					"schema": schema,
					"type":   "table",
				},
			}
			nodes = append(nodes, node)
		}

		// DuckDB doesn't have built-in foreign key constraints like traditional databases
		// but we can analyze column names and types to suggest relationships
		edges = p.inferRelationships(tables)

		return &model.Graph{
			Nodes: nodes,
			Edges: edges,
		}, nil
	})
}

type TableInfo struct {
	Name    string
	Columns []ColumnInfo
}

type ColumnInfo struct {
	Name     string
	DataType string
}

func (p *DuckDBPlugin) getTablesWithRelationships(db *gorm.DB, schema string) ([]TableInfo, error) {
	var tables []TableInfo

	// Get all tables
	tableRows, err := db.Raw(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'main' 
		AND table_type = 'BASE TABLE'
	`).Rows()
	if err != nil {
		return nil, err
	}
	defer tableRows.Close()

	var tableNames []string
	for tableRows.Next() {
		var tableName string
		if err := tableRows.Scan(&tableName); err != nil {
			log.Printf("Error scanning table name: %v", err)
			continue
		}
		tableNames = append(tableNames, tableName)
	}

	// Get columns for each table
	for _, tableName := range tableNames {
		columnRows, err := db.Raw(`
			SELECT column_name, data_type 
			FROM information_schema.columns 
			WHERE table_name = ? AND table_schema = 'main'
			ORDER BY ordinal_position
		`, tableName).Rows()
		if err != nil {
			log.Printf("Error getting columns for table %s: %v", tableName, err)
			continue
		}

		var columns []ColumnInfo
		for columnRows.Next() {
			var column ColumnInfo
			if err := columnRows.Scan(&column.Name, &column.DataType); err != nil {
				log.Printf("Error scanning column info: %v", err)
				continue
			}
			columns = append(columns, column)
		}
		columnRows.Close()

		tables = append(tables, TableInfo{
			Name:    tableName,
			Columns: columns,
		})
	}

	return tables, nil
}

// inferRelationships analyzes column names to suggest possible relationships
func (p *DuckDBPlugin) inferRelationships(tables []TableInfo) []*model.Edge {
	var edges []*model.Edge
	
	// Create a map of table names to their primary key columns (if identifiable)
	primaryKeys := make(map[string][]string)
	for _, table := range tables {
		for _, column := range table.Columns {
			// Common patterns for primary keys
			if strings.EqualFold(column.Name, "id") || 
			   strings.EqualFold(column.Name, table.Name+"_id") ||
			   strings.HasSuffix(strings.ToLower(column.Name), "_id") {
				primaryKeys[table.Name] = append(primaryKeys[table.Name], column.Name)
			}
		}
	}

	// Look for foreign key relationships based on naming conventions
	for _, table := range tables {
		for _, column := range table.Columns {
			columnLower := strings.ToLower(column.Name)
			
			// Skip if this looks like a primary key for this table
			if strings.EqualFold(column.Name, "id") || 
			   strings.EqualFold(column.Name, table.Name+"_id") {
				continue
			}
			
			// Look for foreign key patterns (ends with _id)
			if strings.HasSuffix(columnLower, "_id") {
				// Extract the potential referenced table name
				referencedTable := strings.TrimSuffix(columnLower, "_id")
				
				// Check if there's a table with this name or similar
				for _, otherTable := range tables {
					if strings.EqualFold(otherTable.Name, referencedTable) {
						// Found a potential relationship
						edge := &model.Edge{
							ID:     fmt.Sprintf("%s_%s_to_%s", table.Name, column.Name, otherTable.Name),
							Source: table.Name,
							Target: otherTable.Name,
							Type:   "foreign_key",
							Label:  fmt.Sprintf("%s.%s", table.Name, column.Name),
							Metadata: map[string]interface{}{
								"source_column": column.Name,
								"type":          "inferred_foreign_key",
							},
						}
						edges = append(edges, edge)
						break
					}
				}
			}
		}
	}

	return edges
}