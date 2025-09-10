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
	"strconv"
	"strings"
	"time"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/google/uuid"
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

// todo: test this thouroughly for each DB to ensure that the casting is correct and there's no data loss
// todo: how do we handle if user doesn't pass in a value, or it's null
func (p *GormPlugin) ConvertStringValue(value, columnType string) (interface{}, error) {
	// handle nullable type. clickhouse specific
	isNullable := false
	if strings.HasPrefix(columnType, "Nullable(") {
		isNullable = true
		columnType = strings.TrimPrefix(columnType, "Nullable(")
		columnType = strings.TrimSuffix(columnType, ")")
	}

	columnType = strings.ToUpper(columnType)

	// Check if plugin wants to handle this data type
	if customValue, handled, err := p.GormPluginFunctions.HandleCustomDataType(value, columnType, isNullable); handled {
		return customValue, err
	}

	// Handle NULL values
	if isNullable && (value == "" || strings.EqualFold(value, "NULL")) {
		switch {
		case intTypes.Contains(columnType):
			return sql.NullInt64{Valid: false}, nil
		case uintTypes.Contains(columnType):
			return nil, nil // Go's sql package does not have sql.NullUint64
		case floatTypes.Contains(columnType):
			return sql.NullFloat64{Valid: false}, nil
		case boolTypes.Contains(columnType):
			return sql.NullBool{Valid: false}, nil
		case dateTypes.Contains(columnType), dateTimeTypes.Contains(columnType):
			return sql.NullTime{Valid: false}, nil
		case binaryTypes.Contains(columnType):
			fallthrough // treat null binary as null string
		default: // Assume text
			return sql.NullString{Valid: false}, nil
		}
	}

	// Handle Array type. clickhouse specific
	if strings.HasPrefix(columnType, "Array(") {
		return p.convertArrayValue(value, columnType)
	}

	// Remove any LowCardinality() wrapper. clickhouse specific
	if strings.HasPrefix(columnType, "LowCardinality(") {
		columnType = strings.TrimPrefix(columnType, "LowCardinality(")
		columnType = strings.TrimSuffix(columnType, ")")
	}

	switch {
	case intTypes.Contains(columnType):
		parsedValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			log.Logger.WithError(err).WithField("value", value).WithField("columnType", columnType).Error("Failed to parse integer value")
			return nil, err
		}
		if isNullable {
			return sql.NullInt64{Int64: parsedValue, Valid: true}, nil
		}
		return parsedValue, nil
	case uintTypes.Contains(columnType): //todo: this unsigned stuff is meant to be for clickhouse, double check if it's needed
		bitSize := 64
		if len(columnType) > 4 {
			if size, err := strconv.Atoi(columnType[4:]); err == nil {
				bitSize = size
			}
		}
		parsedValue, err := strconv.ParseUint(value, 10, bitSize)
		if err != nil {
			log.Logger.WithError(err).WithField("value", value).WithField("columnType", columnType).Error("Failed to parse unsigned integer value")
			return nil, err
		}
		if isNullable {
			return &parsedValue, nil
		}
		return parsedValue, nil
	case floatTypes.Contains(columnType):
		parsedValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Logger.WithError(err).WithField("value", value).WithField("columnType", columnType).Error("Failed to parse float value")
			return nil, err
		}
		if isNullable {
			return sql.NullFloat64{Float64: parsedValue, Valid: true}, nil
		}
		return parsedValue, nil
	case boolTypes.Contains(columnType):
		parsedValue, err := strconv.ParseBool(value)
		if err != nil {
			log.Logger.WithError(err).WithField("value", value).WithField("columnType", columnType).Error("Failed to parse boolean value")
			return nil, err
		}
		if isNullable {
			return sql.NullBool{Bool: parsedValue, Valid: true}, nil
		}
		return parsedValue, nil
	case dateTypes.Contains(columnType):
		date, err := p.parseDate(value)
		if err != nil {
			log.Logger.WithError(err).WithField("value", value).WithField("columnType", columnType).Error("Failed to parse date value")
			return nil, fmt.Errorf("invalid date format: %v", err)
		}
		if isNullable {
			return sql.NullTime{Time: date, Valid: true}, nil
		}
		return date, nil
	case dateTimeTypes.Contains(columnType):
		datetime, err := p.parseDateTime(value)
		if err != nil {
			log.Logger.WithError(err).WithField("value", value).WithField("columnType", columnType).Error("Failed to parse datetime value")
			return nil, fmt.Errorf("invalid datetime format: %v", err)
		}
		if isNullable {
			return sql.NullTime{Time: datetime, Valid: true}, nil
		}
		return datetime, nil
	case binaryTypes.Contains(columnType):
		blobData := []byte(value)
		if isNullable && len(blobData) == 0 {
			return sql.NullString{Valid: false}, nil
		}
		return blobData, nil
	// todo: geometry types need to be sorted out more thoughtfully
	// case geometryTypes.Contains(columnType):
	// 	geom, err := wkt.Unmarshal(value)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("invalid geometry format: %v", err)
	// 	}
	// 	return geom, nil
	case uuidTypes.Contains(columnType):
		_, err := uuid.Parse(value)
		if err != nil {
			log.Logger.WithError(err).WithField("value", value).WithField("columnType", columnType).Error("Failed to parse UUID value")
			return nil, fmt.Errorf("invalid UUID format: %v", err)
		}
		fallthrough // let it be handled as a string for now
	default: // should be always string/text/etc
		if isNullable {
			return sql.NullString{String: value, Valid: true}, nil
		}
		return value, nil
	}
}

func (p *GormPlugin) parseDateTime(value string) (time.Time, error) {
	// List of formats to try
	formats := []string{
		time.RFC3339,           // "2006-01-02T15:04:05Z07:00"
		"2006-01-02T15:04:05Z", // UTC timezone
		"2006-01-02 15:04:05",  // No timezone
		"2006-01-02T15:04:05",  // No timezone with T
	}

	var lastErr error
	for _, format := range formats {
		t, err := time.Parse(format, value)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}

	return time.Time{}, fmt.Errorf("could not parse datetime '%s': %v", value, lastErr)
}

// parseDate converts a string to a time.Time object for ClickHouse Date
func (p *GormPlugin) parseDate(value string) (time.Time, error) {
	formats := []string{
		"2006-01-02", // Standard date format
		time.RFC3339, // Try full datetime format and truncate to date
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	var lastErr error
	for _, format := range formats {
		t, err := time.Parse(format, value)
		if err == nil {
			// Truncate to date only (no time component)
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()), nil
		}
		lastErr = err
	}

	return time.Time{}, fmt.Errorf("could not parse date '%s': %v", value, lastErr)
}

func (p *GormPlugin) convertArrayValue(value string, columnType string) (interface{}, error) {
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
