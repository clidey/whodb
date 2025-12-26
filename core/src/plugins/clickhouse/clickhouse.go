/*
 * // Copyright 2025 Clidey, Inc.
 * //
 * // Licensed under the Apache License, Version 2.0 (the "License");
 * // you may not use this file except in compliance with the License.
 * // You may obtain a copy of the License at
 * //
 * //     http://www.apache.org/licenses/LICENSE-2.0
 * //
 * // Unless required by applicable law or agreed to in writing, software
 * // distributed under the License is distributed on an "AS IS" BASIS,
 * // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * // See the License for the specific language governing permissions and
 * // limitations under the License.
 */

package clickhouse

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"
	"time"

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

func (p *ClickHousePlugin) executeRawSQL(config *engine.PluginConfig, query string, params ...interface{}) (*engine.GetRowsResult, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.GetRowsResult, error) {
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
	// Handle all ClickHouse array and special types
	return strings.HasPrefix(typeName, "Array(") ||
		strings.HasPrefix(typeName, "Tuple(") ||
		strings.HasPrefix(typeName, "Map(") ||
		strings.HasPrefix(typeName, "Nested(") ||
		strings.HasPrefix(typeName, "Int128") ||
		strings.HasPrefix(typeName, "Int256") ||
		strings.HasPrefix(typeName, "UInt128") ||
		strings.HasPrefix(typeName, "UInt256") ||
		strings.HasPrefix(typeName, "Decimal") || // Decimal32, Decimal64, Decimal128, Decimal256
		strings.HasPrefix(typeName, "FixedString") ||
		strings.HasPrefix(typeName, "Enum") || // Enum8, Enum16
		strings.Contains(typeName, "DateTime64") ||
		typeName == "IPv4" ||
		typeName == "IPv6" ||
		typeName == "UUID" ||
		typeName == "Date32" ||
		typeName == "JSON" ||
		typeName == "Point" ||
		typeName == "Ring" ||
		typeName == "Polygon" ||
		typeName == "MultiPolygon"
}

// GetColumnScanner returns appropriate scanner for ClickHouse column types
func (p *ClickHousePlugin) GetColumnScanner(typeName string) interface{} {
	// For special ClickHouse types, use interface{} to handle any type
	var value interface{}
	return &value
}

// FormatColumnValue formats the value for display
func (p *ClickHousePlugin) FormatColumnValue(typeName string, value interface{}) (string, error) {
	// Handle the interface{} pointer we created in GetColumnScanner
	if ptr, ok := value.(*interface{}); ok && ptr != nil {
		actualValue := *ptr
		if actualValue == nil {
			return "", nil
		}

		// Handle different ClickHouse types
		switch v := actualValue.(type) {
		case *big.Int:
			if v == nil {
				return "", nil
			}
			return v.String(), nil
		case []string:
			// Array of strings
			return fmt.Sprintf("[%s]", strings.Join(v, ", ")), nil
		case []interface{}:
			// Array of mixed types
			parts := make([]string, len(v))
			for i, item := range v {
				parts[i] = fmt.Sprintf("%v", item)
			}
			return fmt.Sprintf("[%s]", strings.Join(parts, ", ")), nil
		case map[string]interface{}:
			if b, err := json.Marshal(v); err == nil {
				return string(b), nil
			}
			return fmt.Sprintf("%v", v), nil
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
			if strings.Contains(typeName, "Date") && !strings.Contains(typeName, "DateTime") {
				// Date types - show only date part
				return v.Format("2006-01-02"), nil
			}
			// DateTime types - show full timestamp
			if strings.Contains(typeName, "DateTime64") {
				// High precision datetime
				return v.Format("2006-01-02 15:04:05.999999999"), nil
			}
			return v.Format("2006-01-02 15:04:05"), nil
		case decimal.Decimal:
			// Decimal32, Decimal64, Decimal128, Decimal256
			return v.String(), nil
		case []byte:
			// FixedString or binary data
			if strings.HasPrefix(typeName, "FixedString") {
				// Trim null bytes for FixedString
				trimmed := strings.TrimRight(string(v), "\x00")
				return trimmed, nil
			}
			// Other binary data
			return fmt.Sprintf("0x%x", v), nil
		default:
			// For other types, use default formatting
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

		// Enrich columns with primary key information
		for i := range columns {
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

func NewClickHousePlugin() *engine.Plugin {
	clickhousePlugin := &ClickHousePlugin{}
	clickhousePlugin.Type = engine.DatabaseType_ClickHouse
	clickhousePlugin.PluginFunctions = clickhousePlugin
	clickhousePlugin.GormPluginFunctions = clickhousePlugin
	return &clickhousePlugin.Plugin
}
