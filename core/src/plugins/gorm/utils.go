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
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/dromara/carbon/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/twpayne/go-geom/encoding/wkt"
	"gorm.io/gorm"
)

var (
	intTypes      = common.IntTypes
	uintTypes     = common.UintTypes
	bigIntTypes   = common.BigIntTypes
	decimalTypes  = common.DecimalTypes
	floatTypes    = common.FloatTypes
	boolTypes     = common.BoolTypes
	dateTypes     = common.DateTypes
	dateTimeTypes = common.DateTimeTypes
	uuidTypes     = common.UuidTypes
	binaryTypes   = common.BinaryTypes
	jsonTypes     = common.JsonTypes
	networkTypes  = common.NetworkTypes
	geometryTypes = common.GeometryTypes
	xmlTypes      = common.XMLTypes
)

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
	migrator := NewMigratorHelper(db, p.GormPluginFunctions)

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
	migrator := NewMigratorHelper(db, p.GormPluginFunctions)

	// Build full table name for Migrator
	var fullTableName string
	if schema != "" && p.Type != engine.DatabaseType_Sqlite3 {
		fullTableName = schema + "." + tableName
	} else {
		fullTableName = tableName
	}

	return migrator.GetColumnTypes(fullTableName)
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
	columnType = p.NormalizeType(columnType)

	// Extract base type for type set comparisons (e.g., "DECIMAL(10,2)" -> "DECIMAL")
	// This is needed because type sets contain only base types, not parameterized types
	baseType := common.ParseTypeSpec(columnType).BaseType

	// Check if plugin wants to handle this data type
	if customValue, handled, err := p.GormPluginFunctions.HandleCustomDataType(value, columnType, isNullable); handled {
		return customValue, err
	}

	// Handle NULL values
	if isNullable && (value == "" || strings.EqualFold(value, "NULL")) {
		switch {
		case intTypes.Contains(baseType):
			return sql.NullInt64{Valid: false}, nil
		case uintTypes.Contains(baseType), // Go's sql package does not have sql.NullUint64
			bigIntTypes.Contains(baseType),  // big.Int doesn't have a nullable variant, use nil
			decimalTypes.Contains(baseType): // decimal.Decimal doesn't have a nullable variant, use nil
			return nil, nil
		case floatTypes.Contains(baseType):
			return sql.NullFloat64{Valid: false}, nil
		case boolTypes.Contains(baseType):
			return sql.NullBool{Valid: false}, nil
		case dateTypes.Contains(baseType), dateTimeTypes.Contains(baseType):
			return sql.NullTime{Valid: false}, nil
		case binaryTypes.Contains(baseType):
			fallthrough //	todo: treating binary as string but might have to treat as binary
		default: // Assume text
			return sql.NullString{Valid: false}, nil
		}
	}

	// Handle Array type. clickhouse specific
	if strings.HasPrefix(baseType, "ARRAY") {
		return p.convertArrayValue(value, columnType)
	}

	// Remove any LowCardinality() wrapper. clickhouse specific
	if strings.HasPrefix(baseType, "LOWCARDINALITY") {
		columnType = strings.TrimPrefix(columnType, "LowCardinality(")
		columnType = strings.TrimSuffix(columnType, ")")
		baseType = common.ParseTypeSpec(columnType).BaseType
	}

	switch {
	case intTypes.Contains(baseType):
		parsedValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			log.Logger.WithError(err).WithField("value", value).WithField("columnType", columnType).Error("Failed to parse integer value")
			return nil, err
		}
		if isNullable {
			return sql.NullInt64{Int64: parsedValue, Valid: true}, nil
		}
		return parsedValue, nil
	case uintTypes.Contains(baseType):
		// Extract bit size from base type (e.g., UINT32 -> 32, UINT64 -> 64)
		bitSize := 64
		if len(baseType) > 4 {
			if size, err := strconv.Atoi(baseType[4:]); err == nil {
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
	case bigIntTypes.Contains(baseType):
		// Use math/big.Int for INT128, INT256, UINT128, UINT256 which exceed int64/uint64 range
		parsedValue := new(big.Int)
		_, ok := parsedValue.SetString(value, 10)
		if !ok {
			log.Logger.WithField("value", value).WithField("columnType", columnType).Error("Failed to parse big integer value")
			return nil, fmt.Errorf("invalid big integer value: %s", value)
		}
		if isNullable {
			return parsedValue, nil
		}
		return parsedValue, nil
	case decimalTypes.Contains(baseType):
		// Use shopspring/decimal for high-precision numeric types (DECIMAL, NUMERIC, NUMBER, MONEY)
		// This preserves full precision unlike float64 which only has ~15-17 significant digits
		parsedValue, err := decimal.NewFromString(value)
		if err != nil {
			log.Logger.WithError(err).WithField("value", value).WithField("columnType", columnType).Error("Failed to parse decimal value")
			return nil, err
		}
		if isNullable {
			return &parsedValue, nil
		}
		return parsedValue, nil
	case floatTypes.Contains(baseType):
		parsedValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Logger.WithError(err).WithField("value", value).WithField("columnType", columnType).Error("Failed to parse float value")
			return nil, err
		}
		if isNullable {
			return sql.NullFloat64{Float64: parsedValue, Valid: true}, nil
		}
		return parsedValue, nil
	case boolTypes.Contains(baseType):
		parsedValue, err := strconv.ParseBool(value)
		if err != nil {
			log.Logger.WithError(err).WithField("value", value).WithField("columnType", columnType).Error("Failed to parse boolean value")
			return nil, err
		}
		if isNullable {
			return sql.NullBool{Bool: parsedValue, Valid: true}, nil
		}
		return parsedValue, nil
	case dateTypes.Contains(baseType):
		date, err := p.parseDate(value)
		if err != nil {
			log.Logger.WithError(err).WithField("value", value).WithField("columnType", columnType).Error("Failed to parse date value")
			return nil, fmt.Errorf("invalid date format: %v", err)
		}
		if isNullable {
			return sql.NullTime{Time: date, Valid: true}, nil
		}
		return date, nil
	case dateTimeTypes.Contains(baseType):
		datetime, err := p.parseDateTime(value)
		if err != nil {
			log.Logger.WithError(err).WithField("value", value).WithField("columnType", columnType).Error("Failed to parse datetime value")
			return nil, fmt.Errorf("invalid datetime format: %v", err)
		}
		if isNullable {
			return sql.NullTime{Time: datetime, Valid: true}, nil
		}
		return datetime, nil
	case binaryTypes.Contains(baseType):
		blobData := []byte(value)
		if isNullable && len(blobData) == 0 {
			return sql.NullString{Valid: false}, nil
		}
		return blobData, nil
	case uuidTypes.Contains(baseType):
		_, err := uuid.Parse(value)
		if err != nil {
			log.Logger.WithError(err).WithField("value", value).WithField("columnType", columnType).Error("Failed to parse UUID value")
			return nil, fmt.Errorf("invalid UUID format: %v", err)
		}
		if isNullable {
			return sql.NullString{String: value, Valid: true}, nil
		}
		return value, nil
	case jsonTypes.Contains(baseType):
		if !json.Valid([]byte(value)) {
			log.Logger.WithField("value", value).WithField("columnType", columnType).Error("Invalid JSON value")
			return nil, fmt.Errorf("invalid JSON format")
		}
		// todo: not sure about nullable json, can json even be nullable
		return json.RawMessage(value), nil
	case networkTypes.Contains(baseType):
		// MAC addresses (MACADDR, MACADDR8) stay as strings
		if baseType == "IPV4" || baseType == "IPV6" || baseType == "INET" || baseType == "CIDR" {
			// For CIDR, extract just the IP part
			ipStr := value
			if strings.Contains(value, "/") {
				ipStr = strings.Split(value, "/")[0]
			}
			ip := net.ParseIP(ipStr)
			if ip == nil {
				log.Logger.WithField("value", value).WithField("columnType", columnType).Error("Invalid IP address")
				return nil, fmt.Errorf("invalid IP address format: %s", value)
			}
			if isNullable {
				return sql.NullString{String: value, Valid: true}, nil
			}
			return value, nil
		}
		if isNullable {
			return sql.NullString{String: value, Valid: true}, nil
		}
		return value, nil
	case geometryTypes.Contains(baseType):
		// Validate WKT (Well-Known Text) format using go-geom
		_, err := wkt.Unmarshal(value)
		if err != nil {
			log.Logger.WithError(err).WithField("value", value).WithField("columnType", columnType).Error("Invalid geometry WKT format")
			return nil, fmt.Errorf("invalid geometry WKT format: %v", err)
		}
		if isNullable {
			return sql.NullString{String: value, Valid: true}, nil
		}
		return value, nil
	case xmlTypes.Contains(baseType):
		fallthrough
	default: // should be always string/text/etc
		if isNullable {
			return sql.NullString{String: value, Valid: true}, nil
		}
		return value, nil
	}
}

func (p *GormPlugin) parseDateTime(value string) (time.Time, error) {
	c := carbon.Parse(value)
	if c.Error != nil {
		return time.Time{}, fmt.Errorf("could not parse datetime '%s': %v", value, c.Error)
	}
	if c.IsInvalid() {
		return time.Time{}, fmt.Errorf("could not parse datetime '%s': invalid date", value)
	}
	return c.StdTime(), nil
}

// parseDate converts a string to a time.Time object, truncated to date only
func (p *GormPlugin) parseDate(value string) (time.Time, error) {
	c := carbon.Parse(value)
	if c.Error != nil {
		return time.Time{}, fmt.Errorf("could not parse date '%s': %v", value, c.Error)
	}
	if c.IsInvalid() {
		return time.Time{}, fmt.Errorf("could not parse date '%s': invalid date", value)
	}
	// Truncate to date only (no time component)
	return c.StartOfDay().StdTime(), nil
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
