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

package gorm_plugin

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/types"
	"gorm.io/gorm"
)

var (
	intTypes      = common.IntTypes
	uintTypes     = common.UintTypes
	floatTypes    = common.FloatTypes
	boolTypes     = common.BoolTypes
	dateTypes     = common.DateTypes
	dateTimeTypes = common.DateTimeTypes
	uuidTypes     = common.UuidTypes
	binaryTypes   = common.BinaryTypes
	// geometryTypes = common.GeometryTypes // not defined yet
)

// Identifier quoting and escaping is handled by GORM Dialector via SQLBuilder.

func (p *GormPlugin) ConvertRecordValuesToMap(values []engine.Record) (map[string]interface{}, error) {
	data := make(map[string]interface{}, len(values))
	for _, value := range values {
		// Check if this is a NULL value
		if value.Extra != nil && value.Extra["IsNull"] == "true" {
			data[value.Key] = nil
		} else {
			val, err := p.GormPluginFunctions.ConvertStringValueDuringMap(value.Value, value.Extra["Type"])
			if err != nil {
				return nil, err
			}
			data[value.Key] = val
		}
	}
	return data, nil
}

// GetPrimaryKeyColumns returns the primary key columns for a table using raw SQL
func (p *GormPlugin) GetPrimaryKeyColumns(db *gorm.DB, schema string, tableName string) ([]string, error) {
	var primaryKeys []string
	query := p.GetPrimaryKeyColQuery()

	// If no query is provided by the plugin, return empty (no primary keys)
	if query == "" {
		return primaryKeys, nil
	}

	var rows *sql.Rows
	var err error

	// SQLite doesn't use schema in queries
	if p.Type == engine.DatabaseType_Sqlite3 {
		rows, err = db.Raw(query, tableName).Rows()
	} else {
		rows, err = db.Raw(query, schema, tableName).Rows()
	}

	if err != nil {
		// Primary keys might not exist, which is ok - return empty array
		log.Logger.Debug(fmt.Sprintf("No primary keys found for table %s.%s: %v", schema, tableName, err))
		return primaryKeys, nil
	}
	defer rows.Close()

	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			log.Logger.WithError(err).Error(fmt.Sprintf("Failed to scan primary key column name for table %s.%s", schema, tableName))
			continue
		}
		primaryKeys = append(primaryKeys, columnName)
	}

	// It's ok if there are no primary keys - return empty array
	return primaryKeys, nil
}

// GetColumnTypes uses GORM's Migrator when possible, falls back to raw SQL
// GetOrderedColumnsWithTypes returns columns in definition order with their types
func (p *GormPlugin) GetOrderedColumnsWithTypes(db *gorm.DB, schema, tableName string) ([]engine.Column, map[string]string, error) {
	migrator := NewMigratorHelper(db, p)

	// Build full table name for Migrator
	var fullTableName string
	if schema != "" && p.Type != engine.DatabaseType_Sqlite3 {
		fullTableName = schema + "." + tableName
	} else {
		fullTableName = tableName
	}

	// Get ordered columns
	columns, err := migrator.GetOrderedColumns(fullTableName)
	if err != nil {
		return nil, nil, err
	}

	// Build column types map for backward compatibility
	columnTypes := make(map[string]string, len(columns))
	for _, col := range columns {
		columnTypes[col.Name] = col.Type
	}

	return columns, columnTypes, nil
}

func (p *GormPlugin) GetColumnTypes(db *gorm.DB, schema, tableName string) (map[string]string, error) {
	// Try using Migrator first
	migrator := NewMigratorHelper(db, p)

	// Build full table name for Migrator
	var fullTableName string
	if schema != "" && p.Type != engine.DatabaseType_Sqlite3 {
		fullTableName = schema + "." + tableName
	} else {
		fullTableName = tableName
	}

	// Attempt to get column types using Migrator
	columnTypes, err := migrator.GetColumnTypes(fullTableName)
	if err == nil && len(columnTypes) > 0 {
		return columnTypes, nil
	}

	// Fall back to raw SQL if Migrator didn't work
	log.Logger.Debug("Migrator failed to get column types, falling back to raw SQL")

	columnTypes = make(map[string]string)
	query := p.GetColTypeQuery()

	var rows *sql.Rows

	if p.Type == engine.DatabaseType_Sqlite3 {
		rows, err = db.Raw(query, tableName).Rows()
	} else {
		rows, err = db.Raw(query, schema, tableName).Rows()
	}

	if err != nil {
		log.Logger.WithError(err).WithField("schema", schema).WithField("tableName", tableName).Error("Failed to execute column types query")
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var columnName, dataType string
		if err := rows.Scan(&columnName, &dataType); err != nil {
			log.Logger.WithError(err).WithField("schema", schema).WithField("tableName", tableName).Error("Failed to scan column type data")
			return nil, err
		}
		columnTypes[columnName] = dataType
	}

	if err := rows.Err(); err != nil {
		log.Logger.WithError(err).WithField("schema", schema).WithField("tableName", tableName).Error("Row iteration error while getting column types")
		return nil, err
	}

	return columnTypes, nil
}

// ConvertStringValue uses the new type converter system with vetted libraries
func (p *GormPlugin) ConvertStringValue(value, columnType string) (interface{}, error) {
	// Initialize converter if needed
	if p.typeConverter == nil {
		p.InitPlugin()
	}

	// First check if plugin wants to handle this data type with custom logic
	isNullable := p.typeConverter.IsNullable(columnType)
	baseType := p.typeConverter.GetBaseType(columnType)

	if customValue, handled, err := p.GormPluginFunctions.HandleCustomDataType(value, baseType, isNullable); handled {
		return customValue, err
	}

	// Handle Array type (ClickHouse specific)
	if strings.HasPrefix(strings.ToUpper(columnType), "ARRAY(") {
		return p.convertArrayValue(value, columnType)
	}

	// Use the new type converter for all standard conversions
	return p.typeConverter.ConvertFromString(value, columnType)
}

func (p *GormPlugin) convertArrayValue(value string, columnType string) (interface{}, error) {
	// For ClickHouse, use the improved parser if available
	if p.Type == engine.DatabaseType_ClickHouse && p.typeConverter != nil {
		parser := &types.ClickHouseArrayParser{}
		elementType := strings.TrimPrefix(columnType, "Array(")
		elementType = strings.TrimSuffix(elementType, ")")
		return parser.ParseArray(value, elementType)
	}

	// Fallback to simple parsing for other databases
	// Extract the element type from Array(Type)
	elementType := strings.TrimPrefix(columnType, "Array(")
	elementType = strings.TrimSuffix(elementType, ")")

	// Remove brackets and split by comma
	value = strings.Trim(value, "[]")
	if value == "" {
		return []interface{}{}, nil
	}

	elements := strings.Split(value, ",")
	result := make([]interface{}, 0, len(elements))

	for _, element := range elements {
		element = strings.TrimSpace(element)
		if element == "" {
			continue
		}

		converted, err := p.GormPluginFunctions.ConvertStringValue(element, elementType)
		if err != nil {
			log.Logger.WithError(err).WithField("element", element).WithField("elementType", elementType).Error("Failed to convert array element")
			return nil, fmt.Errorf("converting array element: %w", err)
		}
		result = append(result, converted)
	}

	return result, nil
}
