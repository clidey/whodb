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

package postgres

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/common/ssl"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	sourcecatalogspecs "github.com/clidey/whodb/core/src/sourcecatalog/specs"
)

// CockroachDBPlugin extends PostgresPlugin with CockroachDB-specific overrides.
// CockroachDB is PostgreSQL wire-compatible but lacks some pg_catalog functions.
type CockroachDBPlugin struct {
	PostgresPlugin
}

type cockroachDBColumnInfo struct {
	columnName             string
	dataType               string
	columnDefault          sql.NullString
	isNullable             string
	isGenerated            string
	characterMaximumLength sql.NullInt64
	numericPrecision       sql.NullInt64
	numericScale           sql.NullInt64
	isPrimary              bool
	referencedTable        sql.NullString
	referencedColumn       sql.NullString
}

const cockroachDBColumnInfoQuery = `
	SELECT
		c.column_name,
		c.data_type,
		c.column_default,
		c.is_nullable,
		c.is_generated,
		c.character_maximum_length,
		c.numeric_precision,
		c.numeric_scale,
		pk.column_name IS NOT NULL AS is_primary,
		fk.referenced_table,
		fk.referenced_column
	FROM information_schema.columns c
	LEFT JOIN (
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
			AND tc.table_name = kcu.table_name
		WHERE tc.constraint_type = 'PRIMARY KEY'
			AND tc.table_schema = ?
			AND tc.table_name = ?
	) pk ON pk.column_name = c.column_name
	LEFT JOIN (
		SELECT
			kcu.column_name,
			ccu.table_name AS referenced_table,
			ccu.column_name AS referenced_column
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
			AND tc.table_name = kcu.table_name
		JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = ?
			AND tc.table_name = ?
	) fk ON fk.column_name = c.column_name
	WHERE c.table_schema = ?
		AND c.table_name = ?
	ORDER BY c.ordinal_position;
`

// GetTableInfoQuery returns a CockroachDB-compatible table info query.
// CockroachDB does not support pg_size_pretty() or pg_total_relation_size(),
// so we query only the table name and type from information_schema.
func (p *CockroachDBPlugin) GetTableInfoQuery() string {
	return `
		SELECT
			t.table_name,
			t.table_type
		FROM
			information_schema.tables t
		WHERE
			t.table_schema = ?;
	`
}

// GetTableNameAndAttributes parses CockroachDB table info rows.
func (p *CockroachDBPlugin) GetTableNameAndAttributes(rows *sql.Rows) (string, []engine.Record) {
	var tableName, tableType string
	if err := rows.Scan(&tableName, &tableType); err != nil {
		log.WithError(err).Error("Failed to scan CockroachDB table info row data")
		return "", nil
	}

	attributes := []engine.Record{
		{Key: "Type", Value: tableType},
	}

	return tableName, attributes
}

// GetStorageUnitExistsQuery returns a CockroachDB-compatible table existence check.
// CockroachDB does not support to_regclass().
func (p *CockroachDBPlugin) GetStorageUnitExistsQuery() string {
	return `SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2)`
}

// GetAllSchemasQuery returns user-facing CockroachDB schemas only.
func (p *CockroachDBPlugin) GetAllSchemasQuery() string {
	return `
			SELECT schema_name AS schemaname
			FROM information_schema.schemata
		WHERE has_schema_privilege(schema_name, 'USAGE')
			AND schema_name NOT IN ('information_schema', 'pg_catalog', 'crdb_internal', 'pg_extension');
	`
}

// GetColumnsForTable returns CockroachDB table columns without using GORM's PostgreSQL migrator.
func (p *CockroachDBPlugin) GetColumnsForTable(config *engine.PluginConfig, schema string, storageUnit string) ([]engine.Column, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]engine.Column, error) {
		return p.getCockroachDBColumns(db, schema, storageUnit)
	})
}

// GetColumnTypes returns CockroachDB column type metadata from information_schema.
func (p *CockroachDBPlugin) GetColumnTypes(db *gorm.DB, schema, tableName string) (map[string]gorm_plugin.ColumnTypeInfo, error) {
	columns, err := p.getCockroachDBColumns(db, schema, tableName)
	if err != nil {
		return nil, err
	}

	columnTypes := make(map[string]gorm_plugin.ColumnTypeInfo, len(columns))
	for _, column := range columns {
		columnTypes[column.Name] = gorm_plugin.ColumnTypeInfo{
			Type:       column.Type,
			IsNullable: column.IsNullable,
		}
	}

	return columnTypes, nil
}

func (p *CockroachDBPlugin) getCockroachDBColumns(db *gorm.DB, schema string, storageUnit string) ([]engine.Column, error) {
	rows, err := db.Raw(cockroachDBColumnInfoQuery, schema, storageUnit, schema, storageUnit, schema, storageUnit).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := []engine.Column{}
	for rows.Next() {
		var info cockroachDBColumnInfo
		if err := rows.Scan(
			&info.columnName,
			&info.dataType,
			&info.columnDefault,
			&info.isNullable,
			&info.isGenerated,
			&info.characterMaximumLength,
			&info.numericPrecision,
			&info.numericScale,
			&info.isPrimary,
			&info.referencedTable,
			&info.referencedColumn,
		); err != nil {
			return nil, err
		}
		columns = append(columns, p.buildCockroachDBColumn(info))
	}

	return columns, rows.Err()
}

func (p *CockroachDBPlugin) buildCockroachDBColumn(info cockroachDBColumnInfo) engine.Column {
	fullType := cockroachDBFullType(info)
	baseType := strings.ToUpper(info.dataType)
	column := engine.Column{
		Name:            info.columnName,
		Type:            p.NormalizeType(fullType),
		IsNullable:      strings.EqualFold(info.isNullable, "YES"),
		IsPrimary:       info.isPrimary,
		IsAutoIncrement: cockroachDBColumnIsAutoIncrement(info.columnDefault),
		IsComputed:      strings.EqualFold(info.isGenerated, "ALWAYS"),
	}

	if cockroachDBTypeHasLength(baseType) && info.characterMaximumLength.Valid {
		length := int(info.characterMaximumLength.Int64)
		column.Length = &length
	}

	if cockroachDBTypeHasPrecision(baseType) && info.numericPrecision.Valid {
		precision := int(info.numericPrecision.Int64)
		scale := 0
		if info.numericScale.Valid {
			scale = int(info.numericScale.Int64)
		}
		column.Precision = &precision
		column.Scale = &scale
	}

	if info.referencedTable.Valid && info.referencedColumn.Valid {
		column.IsForeignKey = true
		column.ReferencedTable = &info.referencedTable.String
		column.ReferencedColumn = &info.referencedColumn.String
	}

	return column
}

func cockroachDBFullType(info cockroachDBColumnInfo) string {
	baseType := strings.ToUpper(info.dataType)
	if cockroachDBTypeHasLength(baseType) && info.characterMaximumLength.Valid && info.characterMaximumLength.Int64 > 0 {
		return fmt.Sprintf("%s(%d)", baseType, info.characterMaximumLength.Int64)
	}

	if cockroachDBTypeHasPrecision(baseType) && info.numericPrecision.Valid && info.numericPrecision.Int64 > 0 {
		scale := int64(0)
		if info.numericScale.Valid {
			scale = info.numericScale.Int64
		}
		return fmt.Sprintf("%s(%d,%d)", baseType, info.numericPrecision.Int64, scale)
	}

	return baseType
}

func cockroachDBTypeHasLength(baseType string) bool {
	return baseType == "CHARACTER VARYING" || baseType == "CHARACTER"
}

func cockroachDBTypeHasPrecision(baseType string) bool {
	return baseType == "DECIMAL" || baseType == "NUMERIC"
}

func cockroachDBColumnIsAutoIncrement(columnDefault sql.NullString) bool {
	return columnDefault.Valid && strings.Contains(strings.ToLower(columnDefault.String), "nextval(")
}

// MarkGeneratedColumns is a no-op because CockroachDB column discovery already marks computed columns.
func (p *CockroachDBPlugin) MarkGeneratedColumns(config *engine.PluginConfig, schema string, storageUnit string, columns []engine.Column) error {
	return nil
}

// GetBulkInsertBatchSize returns a smaller row batch for CockroachDB bulk inserts.
func (p *CockroachDBPlugin) GetBulkInsertBatchSize() int {
	return 1
}

// HandleCustomDataType converts CockroachDB-specific writable values.
func (p *CockroachDBPlugin) HandleCustomDataType(value string, columnType string, isNullable bool) (any, bool, error) {
	baseType := common.ParseTypeSpec(p.NormalizeType(strings.ToUpper(columnType))).BaseType
	if baseType != "BYTEA" {
		return nil, false, nil
	}

	if isNullable && (value == "" || strings.EqualFold(value, "NULL")) {
		return nil, true, nil
	}

	blobData, hasHexPrefix, err := gorm_plugin.DecodeHexLiteral(value)
	if err != nil {
		return nil, true, fmt.Errorf("invalid hex binary format: %w", err)
	}
	if !hasHexPrefix {
		blobData = []byte(value)
	}

	return gorm.Expr("decode(?, 'hex')", hex.EncodeToString(blobData)), true, nil
}

// IsGeometryType returns false for CockroachDB since it has limited geometry support
// compared to PostGIS and does not use the same binary encoding.
func (p *CockroachDBPlugin) IsGeometryType(columnType string) bool {
	return false
}

// GetSSLStatus determines SSL status for CockroachDB connections.
// CockroachDB does not have pg_stat_ssl, so we query the session's ssl variable
// via SHOW ssl (returns "on"/"off" as a string).
func (p *CockroachDBPlugin) GetSSLStatus(config *engine.PluginConfig) (*engine.SSLStatus, error) {
	if cached := plugins.GetCachedSSLStatus(config); cached != nil {
		return cached, nil
	}

	status, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.SSLStatus, error) {
		var result struct {
			SSL string `gorm:"column:ssl"`
		}

		query := db.Raw("SHOW ssl").Scan(&result)
		if query.Error != nil {
			return nil, query.Error
		}

		if result.SSL != "on" {
			return &engine.SSLStatus{
				IsEnabled: false,
				Mode:      string(ssl.SSLModeDisabled),
			}, nil
		}

		sslConfig := ssl.ParseSSLConfig(engine.DatabaseType(p.Type), config.Credentials.Advanced, config.Credentials.Hostname, config.Credentials.IsProfile)
		mode := "enabled"
		if sslConfig != nil {
			mode = string(sslConfig.Mode)
		}

		return &engine.SSLStatus{
			IsEnabled: true,
			Mode:      mode,
		}, nil
	})

	if err == nil && status != nil {
		plugins.SetCachedSSLStatus(config, status)
	}
	return status, err
}

// CockroachDB-supported type definitions (excludes MONEY, XML, HSTORE, geometric types,
// CIDR, MACADDR, TIMETZ which CockroachDB does not support).
var cockroachDBTypeDefinitions = sourcecatalogspecs.CockroachDBTypeDefinitions

// NewCockroachDBPlugin creates a CockroachDB plugin with PostgreSQL compatibility
// and CockroachDB-specific overrides for unsupported catalog functions.
func NewCockroachDBPlugin() *engine.Plugin {
	crdbPlugin := &CockroachDBPlugin{}
	crdbPlugin.Type = engine.DatabaseType_CockroachDB
	crdbPlugin.PluginFunctions = crdbPlugin
	crdbPlugin.GormPluginFunctions = crdbPlugin
	return &crdbPlugin.Plugin
}

func init() {
	engine.RegisterPlugin(NewCockroachDBPlugin())
}
