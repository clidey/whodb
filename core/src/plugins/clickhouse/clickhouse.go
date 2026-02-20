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

package clickhouse

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	supportedOperators = map[string]string{
		"=": "=", ">=": ">=", ">": ">", "<=": "<=", "<": "<", "!=": "!=", "<>": "<>", "==": "==",
		"LIKE": "LIKE", "NOT LIKE": "NOT LIKE", "ILIKE": "ILIKE",
		"IN": "IN", "NOT IN": "NOT IN", "GLOBAL IN": "GLOBAL IN", "GLOBAL NOT IN": "GLOBAL NOT IN",
		"BETWEEN": "BETWEEN", "NOT BETWEEN": "NOT BETWEEN",
		"IS NULL": "IS NULL", "IS NOT NULL": "IS NOT NULL",
		"AND": "AND", "OR": "OR", "NOT": "NOT",
	}
)

type ClickHousePlugin struct {
	gorm_plugin.GormPlugin
}

func (p *ClickHousePlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	return plugins.WithConnection[[]string](config, p.DB, func(db *gorm.DB) ([]string, error) {
		var databases []struct {
			Datname string `gorm:"column:datname"`
		}
		if err := db.Table("system.databases").
			Select("name AS datname").
			Scan(&databases).Error; err != nil {
			return nil, err
		}
		databaseNames := []string{}
		for _, database := range databases {
			databaseNames = append(databaseNames, database.Datname)
		}
		return databaseNames, nil
	})
}

func (p *ClickHousePlugin) FormTableName(schema string, storageUnit string) string {
	if schema == "" {
		return storageUnit
	}
	return schema + "." + storageUnit
}

func (p *ClickHousePlugin) GetSupportedOperators() map[string]string {
	return supportedOperators
}

// HandleCustomDataType handles ClickHouse compound types (Array, Map, Tuple).
func (p *ClickHousePlugin) HandleCustomDataType(value string, columnType string, isNullable bool) (any, bool, error) {
	upper := strings.ToUpper(columnType)

	var convert func(string, string) (any, error)
	switch {
	case strings.HasPrefix(upper, "ARRAY("):
		convert = convertArrayLiteral
	case strings.HasPrefix(upper, "MAP("):
		convert = convertMapLiteral
	case strings.HasPrefix(upper, "TUPLE("):
		convert = convertTupleLiteral
	default:
		return nil, false, nil
	}

	if isNullable && (value == "" || strings.EqualFold(value, "NULL")) {
		return nil, true, nil
	}
	result, err := convert(value, columnType)
	return result, true, err
}

func (p *ClickHousePlugin) ConvertStringValue(value, columnType string) (any, error) {
	normalized := strings.ToUpper(p.NormalizeType(columnType))
	if strings.Contains(normalized, "JSON") {
		if !json.Valid([]byte(value)) {
			return nil, fmt.Errorf("invalid JSON format")
		}
		return value, nil
	}
	return p.GormPlugin.ConvertStringValue(value, columnType)
}

func (p *ClickHousePlugin) GetAllSchemas(config *engine.PluginConfig) ([]string, error) {
	// technically a table is considered a schema in clickhouse
	return nil, errors.ErrUnsupported
}

func (p *ClickHousePlugin) GetTableInfoQuery() string {
	return `
		SELECT
			name,
			engine,
			total_rows,
			formatReadableSize(total_bytes) as total_size
		FROM system.tables
		WHERE database = ?
		AND name NOT LIKE '.inner%'
	`
}

func (p *ClickHousePlugin) GetStorageUnitExistsQuery() string {
	return `SELECT EXISTS(SELECT 1 FROM system.tables WHERE database = ? AND name = ?)`
}

func (p *ClickHousePlugin) GetTableNameAndAttributes(rows *sql.Rows) (string, []engine.Record) {
	var tableName, tableType string
	var totalRows *uint64
	var totalSize *string

	if err := rows.Scan(&tableName, &tableType, &totalRows, &totalSize); err != nil {
		log.Logger.WithError(err).Error("Failed to scan table name and attributes from ClickHouse system.tables query")
		return "", nil
	}

	rowCount := "0"
	if totalRows != nil {
		rowCount = strconv.FormatUint(*totalRows, 10)
	}

	size := "unknown"
	if totalSize != nil {
		size = *totalSize
	}

	attributes := []engine.Record{
		{Key: "Type", Value: tableType},
		{Key: "Total Size", Value: size},
		{Key: "Count", Value: rowCount},
	}

	return tableName, attributes
}

func (p *ClickHousePlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return p.executeRawSQL(config, query)
}

func (p *ClickHousePlugin) RawExecuteWithParams(config *engine.PluginConfig, query string, params []any) (*engine.GetRowsResult, error) {
	return p.executeRawSQL(config, query, params...)
}

// ClearTableData handles ClickHouse-specific DELETE semantics.
// ClickHouse DELETE is an async mutation (ALTER TABLE DELETE) that doesn't return results
// in the same way as traditional SQL DELETE. GORM's Delete method doesn't handle this properly.
func (p *ClickHousePlugin) ClearTableData(config *engine.PluginConfig, schema string, storageUnit string) (bool, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		tableName := p.FormTableName(schema, storageUnit)

		query := fmt.Sprintf("ALTER TABLE %s DELETE WHERE 1=1", tableName)

		// Execute the DELETE mutation
		err := db.Exec(query).Error
		if err != nil {
			// ClickHouse mutations may return "driver: bad connection" when trying to read
			// the non-existent result set. Verify the connection is still healthy.
			if err.Error() == "driver: bad connection" {
				var result int
				if db.Raw("SELECT 1").Scan(&result).Error == nil {
					// Connection is healthy, mutation was accepted
					log.Logger.WithField("table", tableName).Debug("ClickHouse DELETE mutation accepted")
					return true, nil
				}
			}
			return false, err
		}

		log.Logger.WithField("table", tableName).Debug("ClickHouse DELETE executed")
		return true, nil
	})
}

// UpdateStorageUnit handles two ClickHouse-specific issues with ALTER TABLE UPDATE:
// the driver hangs when binding typed slices (Array columns) as parameters, and
// mutations may return "driver: bad connection" on success.
func (p *ClickHousePlugin) UpdateStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string, updatedColumns []string) (bool, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		columnTypes, err := p.GetColumnTypes(db, schema, storageUnit)
		if err != nil {
			return false, err
		}

		arrayColumns := map[string]bool{}
		for _, col := range updatedColumns {
			if ct, ok := columnTypes[col]; ok && strings.HasPrefix(strings.ToUpper(ct), "ARRAY(") {
				arrayColumns[col] = true
			}
		}

		if len(arrayColumns) == 0 {
			result, err := p.GormPlugin.UpdateStorageUnit(config, schema, storageUnit, values, updatedColumns)
			if err != nil && err.Error() == "driver: bad connection" {
				var check int
				if db.Raw("SELECT 1").Scan(&check).Error == nil {
					return true, nil
				}
			}
			return result, err
		}

		pkColumns, err := p.GetPrimaryKeyColumns(db, schema, storageUnit)
		if err != nil {
			pkColumns = []string{}
		}

		conditions := make(map[string]any)
		convertedValues := make(map[string]any)

		for column, strValue := range values {
			isPK := common.ContainsString(pkColumns, column)
			isUpdated := common.ContainsString(updatedColumns, column)
			if !isPK && !isUpdated && len(pkColumns) > 0 {
				continue
			}

			columnType := columnTypes[column]
			converted, convErr := p.ConvertStringValue(strValue, columnType)
			if convErr != nil {
				converted = strValue
			}

			if isUpdated && arrayColumns[column] && reflect.ValueOf(converted).Kind() == reflect.Slice {
				converted = arrayToExpr(converted)
			}

			if isPK {
				conditions[column] = converted
			} else if isUpdated {
				convertedValues[column] = converted
			} else if len(pkColumns) == 0 {
				conditions[column] = converted
			}
		}

		if len(convertedValues) == 0 {
			return true, nil
		}

		result := p.CreateSQLBuilder(db).UpdateQuery(schema, storageUnit, convertedValues, conditions)
		if result.Error != nil {
			if result.Error.Error() == "driver: bad connection" {
				var check int
				if db.Raw("SELECT 1").Scan(&check).Error == nil {
					return true, nil
				}
			}
			return false, result.Error
		}

		return true, nil
	})
}

func (p *ClickHousePlugin) executeRawSQL(config *engine.PluginConfig, query string, params ...any) (*engine.GetRowsResult, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.GetRowsResult, error) {
		// ClickHouse's native TCP protocol only supports one statement per request.
		// Multi-statement scripts silently execute only the first statement.
		if config != nil && config.MultiStatement && isMultiStatement(query) {
			return nil, engine.ErrMultiStatementUnsupported
		}

		rows, err := db.Raw(query, params...).Rows()
		if err != nil {
			// ClickHouse mutations (ALTER TABLE UPDATE/DELETE, INSERT, etc.) execute successfully
			// but the driver returns "driver: bad connection" when trying to read the non-existent result set.
			// Verify the connection is still healthy with a simple query - if so, the mutation succeeded.
			if err.Error() == "driver: bad connection" {
				var result int
				if db.Raw("SELECT 1").Scan(&result).Error == nil {
					return &engine.GetRowsResult{
						Columns: []engine.Column{},
						Rows:    [][]string{},
					}, nil
				}
			}
			return nil, err
		}
		defer rows.Close()

		return p.ConvertRawToRows(rows)
	})
}

// ShouldHandleColumnType returns true for ClickHouse special types
func (p *ClickHousePlugin) ShouldHandleColumnType(typeName string) bool {
	upper := strings.ToUpper(typeName)
	// Handle all ClickHouse array and special types
	return strings.HasPrefix(upper, "NULLABLE(") ||
		strings.HasPrefix(upper, "LOWCARDINALITY(") ||
		strings.HasPrefix(upper, "ARRAY(") ||
		strings.HasPrefix(upper, "TUPLE(") ||
		strings.HasPrefix(upper, "MAP(") ||
		strings.HasPrefix(upper, "NESTED(") ||
		strings.HasPrefix(upper, "INT128") ||
		strings.HasPrefix(upper, "INT256") ||
		strings.HasPrefix(upper, "UINT128") ||
		strings.HasPrefix(upper, "UINT256") ||
		strings.HasPrefix(upper, "DECIMAL") || // Decimal32, Decimal64, Decimal128, Decimal256
		strings.HasPrefix(upper, "FIXEDSTRING") ||
		strings.HasPrefix(upper, "ENUM") || // Enum8, Enum16
		strings.Contains(upper, "DATETIME64") ||
		upper == "IPV4" ||
		upper == "IPV6" ||
		upper == "UUID" ||
		upper == "DATE32" ||
		upper == "JSON" ||
		upper == "POINT" ||
		upper == "RING" ||
		upper == "POLYGON" ||
		upper == "MULTIPOLYGON"
}

// GetColumnScanner returns appropriate scanner for ClickHouse column types
func (p *ClickHousePlugin) GetColumnScanner(typeName string) any {
	upper := strings.ToUpper(typeName)
	if strings.HasPrefix(upper, "INT128") || strings.HasPrefix(upper, "INT256") || strings.HasPrefix(upper, "UINT128") || strings.HasPrefix(upper, "UINT256") {
		return new(big.Int)
	}
	// For special ClickHouse types, use any to handle any type
	var value any
	return &value
}

// FormatColumnValue formats the value for display
func (p *ClickHousePlugin) FormatColumnValue(typeName string, value any) (string, error) {
	// Handle the any pointer we created in GetColumnScanner
	if ptr, ok := value.(*any); ok && ptr != nil {
		actualValue := *ptr
		if actualValue == nil {
			return "", nil
		}

		upperType := strings.ToUpper(typeName)

		// Handle different ClickHouse types
		switch v := actualValue.(type) {
		case *big.Int:
			if v == nil {
				return "", nil
			}
			return v.String(), nil
		case []string:
			// Array of strings
			quoted := make([]string, len(v))
			for i, s := range v {
				quoted[i] = "'" + s + "'"
			}
			return "[" + strings.Join(quoted, ", ") + "]", nil
		case []any:
			// Tuple or mixed array
			if strings.HasPrefix(upperType, "TUPLE") {
				return formatTuple(v), nil
			}
			return formatSlice(v), nil
		case map[string]any:
			return formatMap(v), nil
		case net.IP:
			// IPv4 or IPv6 address
			return v.String(), nil
		case *net.IP:
			// Pointer to IP address
			if v != nil {
				return v.String(), nil
			}
			return "", nil
		case uuid.UUID:
			// UUID type
			return v.String(), nil
		case time.Time:
			// DateTime, DateTime64, Date, Date32
			if strings.Contains(upperType, "DATE") && !strings.Contains(upperType, "DATETIME") {
				// Date types - show only date part
				return v.Format("2006-01-02"), nil
			}
			// DateTime types - show full timestamp
			if strings.Contains(upperType, "DATETIME64") {
				// High precision datetime
				return v.Format("2006-01-02 15:04:05.999999999"), nil
			}
			return v.Format("2006-01-02 15:04:05"), nil
		case decimal.Decimal:
			// Decimal32, Decimal64, Decimal128, Decimal256
			return v.String(), nil
		case []byte:
			// FixedString or binary data
			if strings.HasPrefix(upperType, "FIXEDSTRING") {
				// Trim null bytes for FixedString
				trimmed := strings.TrimRight(string(v), "\x00")
				return trimmed, nil
			}
			// Other binary data
			return fmt.Sprintf("0x%x", v), nil
		case string:
			if strings.HasPrefix(upperType, "FIXEDSTRING") {
				return strings.TrimRight(v, "\x00"), nil
			}
			return v, nil
		default:
			// Handle typed maps/slices from the driver (e.g., map[string]int32, []int32)
			// using reflection so the display matches ClickHouse literal syntax
			rv := reflect.ValueOf(actualValue)
			if rv.Kind() == reflect.Map {
				return formatReflectMap(rv, upperType), nil
			}
			if rv.Kind() == reflect.Slice {
				if strings.HasPrefix(upperType, "TUPLE") {
					return formatReflectTuple(rv), nil
				}
				return formatReflectSlice(rv, upperType), nil
			}
			if stringer, ok := actualValue.(fmt.Stringer); ok {
				return stringer.String(), nil
			}
			if marshaled, err := json.Marshal(actualValue); err == nil && json.Valid(marshaled) {
				return string(marshaled), nil
			}
			return fmt.Sprintf("%v", actualValue), nil
		}
	}

	// Fallback to string representation
	return fmt.Sprintf("%v", value), nil
}

// NormalizeType converts ClickHouse type aliases to their canonical form.
func (p *ClickHousePlugin) NormalizeType(typeName string) string {
	return NormalizeType(typeName)
}

// GetColumnTypes overrides the base implementation because GORM ClickHouse's
// migrator.ColumnTypes() doesn't support "database.table" format - it uses
// m.CurrentDatabase() internally and expects just the table name.
func (p *ClickHousePlugin) GetColumnTypes(db *gorm.DB, schema, tableName string) (map[string]string, error) {
	migrator := gorm_plugin.NewMigratorHelper(db, p)
	// Pass just table name - ClickHouse GORM driver handles database context
	return migrator.GetColumnTypes(tableName)
}

// GetColumnsForTable overrides the base implementation for the same reason as GetColumnTypes.
func (p *ClickHousePlugin) GetColumnsForTable(config *engine.PluginConfig, schema string, storageUnit string) ([]engine.Column, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]engine.Column, error) {
		migrator := gorm_plugin.NewMigratorHelper(db, p)

		// Pass just table name - ClickHouse GORM driver handles database context
		columns, err := migrator.GetOrderedColumns(storageUnit)
		if err != nil {
			log.Logger.WithError(err).Error(fmt.Sprintf("Failed to get columns for table %s.%s", schema, storageUnit))
			return nil, err
		}

		// Get primary keys
		primaryKeys, err := p.GetPrimaryKeyColumns(db, schema, storageUnit)
		if err != nil {
			log.Logger.WithError(err).Warn(fmt.Sprintf("Failed to get primary keys for table %s.%s", schema, storageUnit))
			primaryKeys = []string{}
		}

		// Enrich columns with primary key information + edge case fix regarding auto-increment fields
		for i := range columns {
			// ClickHouse doesn't have traditional auto-increment columns. ClickHouse 25.01+
			// introduced generateSerialID() which provides auto-increment behavior via DEFAULT
			// expressions (e.g., DEFAULT generateSerialID('counter')), but this is not a column
			// attribute - it's a default value. GORM may incorrectly detect columns as auto-increment,
			// so we explicitly disable it for all columns.
			columns[i].IsAutoIncrement = false

			// ClickHouse GORM driver embeds length in type name (e.g., "FixedString(10)")
			// but doesn't expose it via Length(). Parse it from the type string.
			if columns[i].Length == nil {
				typeSpec := common.ParseTypeSpec(columns[i].Type)
				if typeSpec.Length > 0 {
					columns[i].Length = &typeSpec.Length
				}
			}

			// Set primary key flag
			for _, pk := range primaryKeys {
				if columns[i].Name == pk {
					columns[i].IsPrimary = true
					break
				}
			}
		}

		return columns, nil
	})
}

// WithTransaction executes the operation directly since ClickHouse doesn't support traditional ACID transactions.
// The ClickHouse GORM driver's Begin() produces a connection where metadata queries like ColumnTypes() return
// empty results, which breaks mock data generation and other operations that need column information.
func (p *ClickHousePlugin) WithTransaction(config *engine.PluginConfig, operation func(tx any) error) error {
	return operation(nil)
}

func NewClickHousePlugin() *engine.Plugin {
	clickhousePlugin := &ClickHousePlugin{}
	clickhousePlugin.Type = engine.DatabaseType_ClickHouse
	clickhousePlugin.PluginFunctions = clickhousePlugin
	clickhousePlugin.GormPluginFunctions = clickhousePlugin
	return &clickhousePlugin.Plugin
}
