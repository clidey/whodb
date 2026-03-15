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

package gorm_plugin

import (
	"database/sql"
	"encoding/hex"
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
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/dromara/carbon/v2"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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

func (p *GormPlugin) ConvertRecordValuesToMap(values []engine.Record) (map[string]any, error) {
	data := make(map[string]any, len(values))
	for _, value := range values {
		// Check if this is a NULL value
		if value.Extra != nil && value.Extra["IsNull"] == "true" {
			data[value.Key] = nil
		} else {
			isNullable := value.Extra != nil && value.Extra["IsNullable"] == "true"
			val, err := p.GormPluginFunctions.ConvertStringValue(value.Value, value.Extra["Type"], isNullable)
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

	rows, err := db.Raw(query, schema, tableName).Rows()

	if err != nil {
		// Primary keys might not exist, which is ok - return empty array
		log.Debug(fmt.Sprintf("No primary keys found for table %s.%s: %v", schema, tableName, err))
		return primaryKeys, nil
	}
	defer rows.Close()

	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			log.WithError(err).Error(fmt.Sprintf("Failed to scan primary key column name for table %s.%s", schema, tableName))
			continue
		}
		primaryKeys = append(primaryKeys, columnName)
	}

	// It's ok if there are no primary keys - return empty array
	return primaryKeys, nil
}

// GetColumnTypes uses GORM's Migrator when possible, falls back to raw SQL
func (p *GormPlugin) GetColumnTypes(db *gorm.DB, schema, tableName string) (map[string]ColumnTypeInfo, error) {
	migrator := NewMigratorHelper(db, p.GormPluginFunctions)

	fullTableName := p.FormTableName(schema, tableName)
	return migrator.GetColumnTypes(fullTableName)
}

// typeConverter defines how to handle null values and value conversion for a type category.
type typeConverter struct {
	// types is the set of base type names this converter handles
	types mapset.Set[string]
	// isNumeric indicates whether empty strings should be rejected
	isNumeric bool
	// nullValue returns the appropriate null representation for this type
	nullValue func() any
	// convert parses the string value into the appropriate Go type.
	// baseType and columnType are provided for converters that need them.
	convert func(p *GormPlugin, value, baseType, columnType string, isNullable bool) (any, error)
}

// typeConverters defines the conversion table for all supported type categories.
// Order matters: the first matching converter wins.
var typeConverters = []typeConverter{
	{
		types: intTypes, isNumeric: true,
		nullValue: func() any { return sql.NullInt64{Valid: false} },
		convert:   convertInt,
	},
	{
		types: uintTypes, isNumeric: true,
		nullValue: func() any { return nil },
		convert:   convertUint,
	},
	{
		types: bigIntTypes, isNumeric: true,
		nullValue: func() any { return nil },
		convert:   convertBigInt,
	},
	{
		types: decimalTypes, isNumeric: true,
		nullValue: func() any { return nil },
		convert:   convertDecimal,
	},
	{
		types: floatTypes, isNumeric: true,
		nullValue: func() any { return sql.NullFloat64{Valid: false} },
		convert:   convertFloat,
	},
	{
		types: boolTypes,
		nullValue: func() any { return sql.NullBool{Valid: false} },
		convert:   convertBool,
	},
	{
		types:     dateTypes,
		nullValue: func() any { return sql.NullTime{Valid: false} },
		convert:   convertDate,
	},
	{
		types:     dateTimeTypes,
		nullValue: func() any { return sql.NullTime{Valid: false} },
		convert:   convertDateTime,
	},
	{
		types:     binaryTypes,
		nullValue: func() any { return sql.NullString{Valid: false} },
		convert:   convertBinary,
	},
	{
		types:     uuidTypes,
		nullValue: func() any { return sql.NullString{Valid: false} },
		convert:   convertUUID,
	},
	{
		types:     jsonTypes,
		nullValue: func() any { return sql.NullString{Valid: false} },
		convert:   convertJSON,
	},
	{
		types:     networkTypes,
		nullValue: func() any { return sql.NullString{Valid: false} },
		convert:   convertNetwork,
	},
	{
		types:     geometryTypes,
		nullValue: func() any { return sql.NullString{Valid: false} },
		convert:   convertString,
	},
	{
		types:     xmlTypes,
		nullValue: func() any { return sql.NullString{Valid: false} },
		convert:   convertString,
	},
}

// ConvertStringValue converts a string value to the appropriate Go type based on column type.
func (p *GormPlugin) ConvertStringValue(value, columnType string, isNullable bool) (any, error) {
	columnType = strings.ToUpper(columnType)
	columnType = p.NormalizeType(columnType)

	baseType := common.ParseTypeSpec(columnType).BaseType

	// Check if plugin wants to handle this data type
	if customValue, handled, err := p.GormPluginFunctions.HandleCustomDataType(value, columnType, isNullable); handled {
		return customValue, err
	}

	// Find matching converter
	for i := range typeConverters {
		tc := &typeConverters[i]
		if !tc.types.Contains(baseType) {
			continue
		}

		// Handle NULL values
		if isNullable && (value == "" || strings.EqualFold(value, "NULL")) {
			return tc.nullValue(), nil
		}

		// Reject empty strings for numeric types
		if value == "" && tc.isNumeric {
			return nil, fmt.Errorf("cannot convert empty string to %s (column is not nullable)", columnType)
		}

		return tc.convert(p, value, baseType, columnType, isNullable)
	}

	// Default: treat as string/text
	if isNullable && (value == "" || strings.EqualFold(value, "NULL")) {
		return sql.NullString{Valid: false}, nil
	}
	if isNullable {
		return sql.NullString{String: value, Valid: true}, nil
	}
	return value, nil
}

func convertInt(_ *GormPlugin, value, _, columnType string, isNullable bool) (any, error) {
	parsedValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, err
	}
	if isNullable {
		return sql.NullInt64{Int64: parsedValue, Valid: true}, nil
	}
	return parsedValue, nil
}

func convertUint(_ *GormPlugin, value, baseType, _ string, isNullable bool) (any, error) {
	bitSize := 64
	if len(baseType) > 4 {
		if size, err := strconv.Atoi(baseType[4:]); err == nil {
			bitSize = size
		}
	}
	parsedValue, err := strconv.ParseUint(value, 10, bitSize)
	if err != nil {
		return nil, err
	}
	if isNullable {
		return &parsedValue, nil
	}
	return parsedValue, nil
}

func convertBigInt(_ *GormPlugin, value, _, _ string, isNullable bool) (any, error) {
	parsedValue := new(big.Int)
	if _, ok := parsedValue.SetString(value, 10); !ok {
		return nil, fmt.Errorf("invalid big integer value: %s", value)
	}
	return parsedValue, nil
}

func convertDecimal(_ *GormPlugin, value, _, _ string, isNullable bool) (any, error) {
	parsedValue, err := decimal.NewFromString(value)
	if err != nil {
		return nil, err
	}
	if isNullable {
		return &parsedValue, nil
	}
	return parsedValue, nil
}

func convertFloat(_ *GormPlugin, value, _, _ string, isNullable bool) (any, error) {
	parsedValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, err
	}
	if isNullable {
		return sql.NullFloat64{Float64: parsedValue, Valid: true}, nil
	}
	return parsedValue, nil
}

func convertBool(_ *GormPlugin, value, _, _ string, isNullable bool) (any, error) {
	parsedValue, err := strconv.ParseBool(value)
	if err != nil {
		return nil, err
	}
	if isNullable {
		return sql.NullBool{Bool: parsedValue, Valid: true}, nil
	}
	return parsedValue, nil
}

func convertDate(p *GormPlugin, value, _, _ string, isNullable bool) (any, error) {
	date, err := p.parseDate(value)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %v", err)
	}
	if isNullable {
		return sql.NullTime{Time: date, Valid: true}, nil
	}
	return date, nil
}

func convertDateTime(p *GormPlugin, value, baseType, _ string, isNullable bool) (any, error) {
	// TIME-only and YEAR types pass through as strings
	if baseType == "TIME" || baseType == "TIME WITH TIME ZONE" || baseType == "YEAR" {
		if isNullable {
			return sql.NullString{String: value, Valid: true}, nil
		}
		return value, nil
	}
	datetime, err := p.parseDateTime(value)
	if err != nil {
		return nil, fmt.Errorf("invalid datetime format: %v", err)
	}
	if isNullable {
		return sql.NullTime{Time: datetime, Valid: true}, nil
	}
	return datetime, nil
}

func convertBinary(_ *GormPlugin, value, _, _ string, isNullable bool) (any, error) {
	var blobData []byte
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		var err error
		blobData, err = hex.DecodeString(value[2:])
		if err != nil {
			return nil, fmt.Errorf("invalid hex binary format: %v", err)
		}
	} else {
		blobData = []byte(value)
	}
	if isNullable && len(blobData) == 0 {
		return sql.NullString{Valid: false}, nil
	}
	return blobData, nil
}

func convertUUID(_ *GormPlugin, value, _, _ string, isNullable bool) (any, error) {
	if _, err := uuid.Parse(value); err != nil {
		return nil, fmt.Errorf("invalid UUID format: %v", err)
	}
	if isNullable {
		return sql.NullString{String: value, Valid: true}, nil
	}
	return value, nil
}

func convertJSON(_ *GormPlugin, value, _, _ string, _ bool) (any, error) {
	if !json.Valid([]byte(value)) {
		return nil, fmt.Errorf("invalid JSON format")
	}
	return value, nil
}

func convertNetwork(_ *GormPlugin, value, baseType, _ string, isNullable bool) (any, error) {
	if baseType == "IPV4" || baseType == "IPV6" || baseType == "INET" || baseType == "CIDR" {
		ipStr := value
		if strings.Contains(value, "/") {
			ipStr = strings.Split(value, "/")[0]
		}
		if ip := net.ParseIP(ipStr); ip == nil {
			return nil, fmt.Errorf("invalid IP address format: %s", value)
		}
	}
	if isNullable {
		return sql.NullString{String: value, Valid: true}, nil
	}
	return value, nil
}

func convertString(_ *GormPlugin, value, _, _ string, isNullable bool) (any, error) {
	if isNullable {
		return sql.NullString{String: value, Valid: true}, nil
	}
	return value, nil
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

// GetLastInsertID returns the most recently auto-generated ID.
// The default implementation returns 0 (unsupported).
// Database plugins that support this should override it.
func (p *GormPlugin) GetLastInsertID(db *gorm.DB) (int64, error) {
	return 0, nil
}

// ConvertArrayValue parses array literal values (e.g., "[1, 2, 3]") into Go slices.
// Used by ClickHouse for ARRAY() type conversion.
func (p *GormPlugin) ConvertArrayValue(value string, columnType string) (any, error) {
	// Extract the element type from ARRAY(Type)
	upperType := strings.ToUpper(columnType)
	if strings.HasPrefix(upperType, "ARRAY(") {
		columnType = columnType[6:] // strip "ARRAY(" (6 chars)
	}
	if strings.HasSuffix(columnType, ")") {
		columnType = columnType[:len(columnType)-1]
	}
	elementType := columnType

	// Remove brackets and split by comma
	value = strings.Trim(value, "[]")
	if value == "" {
		return []any{}, nil
	}

	elements := strings.Split(value, ",")
	result := make([]any, 0, len(elements))

	for _, element := range elements {
		element = strings.TrimSpace(element)
		if element == "" {
			continue
		}

		converted, err := p.GormPluginFunctions.ConvertStringValue(element, elementType, false)
		if err != nil {
			log.WithError(err).WithField("element", element).WithField("elementType", elementType).Error("Failed to convert array element")
			return nil, fmt.Errorf("converting array element: %w", err)
		}
		result = append(result, converted)
	}

	return result, nil
}
