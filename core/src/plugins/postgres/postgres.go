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

package postgres

import (
	"database/sql"
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	"github.com/clidey/whodb/core/src/types"
	mapset "github.com/deckarep/golang-set/v2"
	"gorm.io/gorm"
)

var (
	supportedColumnDataTypes = mapset.NewSet(
		"SMALLINT", "INTEGER", "BIGINT", "DECIMAL", "NUMERIC", "REAL", "DOUBLE PRECISION", "SMALLSERIAL",
		"SERIAL", "BIGSERIAL", "MONEY",
		"CHAR", "VARCHAR", "TEXT", "BYTEA",
		"TIMESTAMP", "TIMESTAMPTZ", "DATE", "TIME", "TIMETZ",
		"BOOLEAN", "POINT", "LINE", "LSEG", "BOX", "PATH", "POLYGON", "CIRCLE",
		"CIDR", "INET", "MACADDR", "UUID", "XML", "JSON", "JSONB", "ARRAY", "HSTORE",
	)

	supportedOperators = map[string]string{
		"=": "=", ">=": ">=", ">": ">", "<=": "<=", "<": "<", "<>": "<>",
		"!=": "!=", "!>": "!>", "!<": "!<", "BETWEEN": "BETWEEN", "NOT BETWEEN": "NOT BETWEEN",
		"LIKE": "LIKE", "NOT LIKE": "NOT LIKE", "IN": "IN", "NOT IN": "NOT IN",
		"IS NULL": "IS NULL", "IS NOT NULL": "IS NOT NULL", "AND": "AND", "OR": "OR", "NOT": "NOT",
	}
)

type PostgresPlugin struct {
	gorm_plugin.GormPlugin
}

func (p *PostgresPlugin) GetSupportedColumnDataTypes() mapset.Set[string] {
	return supportedColumnDataTypes
}

func (p *PostgresPlugin) GetSupportedOperators() map[string]string {
	return supportedOperators
}

func (p *PostgresPlugin) FormTableName(schema string, storageUnit string) string {
	// Keep raw concatenation; actual SQL builders will quote via GORM Dialector
	if schema == "" {
		return storageUnit
	}
	return schema + "." + storageUnit
}

func (p *PostgresPlugin) GetAllSchemasQuery() string {
	return "SELECT schema_name AS schemaname FROM information_schema.schemata"
}

func (p *PostgresPlugin) GetSchemaTableQuery() string {
	return `
		SELECT 
			table_name AS "TABLE_NAME", 
			column_name AS "COLUMN_NAME", 
			data_type AS "DATA_TYPE"
		FROM information_schema.columns
		WHERE table_schema = ?
		ORDER BY table_name, ordinal_position
	`
}

func (p *PostgresPlugin) GetTableInfoQuery() string {
	return `
		SELECT
			t.table_name,
			t.table_type,
			pg_size_pretty(pg_total_relation_size(quote_ident(t.table_schema) || '.' || quote_ident(t.table_name))) AS total_size,
			pg_size_pretty(pg_relation_size(quote_ident(t.table_schema) || '.' || quote_ident(t.table_name))) AS data_size,
			COALESCE(s.n_live_tup, 0) AS row_count
		FROM
			information_schema.tables t
		LEFT JOIN
			pg_stat_user_tables s ON t.table_name = s.relname
		WHERE
			t.table_schema = ?;
	`

	// AND t.table_type = 'BASE TABLE' this removes the view tables
}

func (p *PostgresPlugin) GetPlaceholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

func (p *PostgresPlugin) GetTableNameAndAttributes(rows *sql.Rows, db *gorm.DB) (string, []engine.Record) {
	var tableName, tableType, totalSize, dataSize string
	var rowCount int64
	if err := rows.Scan(&tableName, &tableType, &totalSize, &dataSize, &rowCount); err != nil {
		log.Logger.WithError(err).Error("Failed to scan table info row data")
		return "", nil
	}

	rowCountRecordValue := "unknown"
	if rowCount >= 0 {
		rowCountRecordValue = fmt.Sprintf("%d", rowCount)
	}

	attributes := []engine.Record{
		{Key: "Type", Value: tableType},
		{Key: "Total Size", Value: totalSize},
		{Key: "Data Size", Value: dataSize},
		{Key: "Count", Value: rowCountRecordValue},
	}

	return tableName, attributes
}

func (p *PostgresPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]string, error) {
		var databases []struct {
			Datname string `gorm:"column:datname"`
		}
		if err := db.Table("pg_database").
			Select("datname").
			Where("datistemplate = ?", false).
			Scan(&databases).Error; err != nil {
			return nil, err
		}
		var databaseNames []string
		for _, database := range databases {
			databaseNames = append(databaseNames, database.Datname)
		}
		return databaseNames, nil
	})
}

func (p *PostgresPlugin) executeRawSQL(config *engine.PluginConfig, query string, params ...interface{}) (*engine.GetRowsResult, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.GetRowsResult, error) {
		rows, err := db.Raw(query, params...).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return p.ConvertRawToRows(rows)
	})
}

func (p *PostgresPlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return p.executeRawSQL(config, query)
}

// RegisterTypes registers PostgreSQL-specific types
func (p *PostgresPlugin) RegisterTypes(registry *types.TypeRegistry) error {
	// Create parser for proper conversion using pgx
	parser := types.NewPostgreSQLArrayParserSimple()

	// Register PostgreSQL array types with pgx-based parsing
	err := registry.RegisterType(&types.TypeDefinition{
		Name:     "PostgresIntArray",
		Category: types.TypeCategoryArray,
		SQLTypes: []string{"_INT2", "_INT4", "_INT8"},
		FromString: func(s string) (any, error) {
			// Use pgx parser for proper integer array handling
			return parser.ParseArraySimple(s, "_INT4")
		},
		ToString: func(v any) (string, error) {
			return parser.FormatArraySimple(v), nil
		},
	})
	if err != nil {
		return err
	}

	// Register float array types
	err = registry.RegisterType(&types.TypeDefinition{
		Name:     "PostgresFloatArray",
		Category: types.TypeCategoryArray,
		SQLTypes: []string{"_FLOAT4", "_FLOAT8", "_NUMERIC", "_DECIMAL"},
		FromString: func(s string) (any, error) {
			return parser.ParseArraySimple(s, "_FLOAT8")
		},
		ToString: func(v any) (string, error) {
			return parser.FormatArraySimple(v), nil
		},
	})
	if err != nil {
		return err
	}

	err = registry.RegisterType(&types.TypeDefinition{
		Name:     "PostgresTextArray",
		Category: types.TypeCategoryArray,
		SQLTypes: []string{"_TEXT", "_VARCHAR", "_CHAR"},
		FromString: func(s string) (any, error) {
			return parser.ParseArraySimple(s, "_TEXT")
		},
		ToString: func(v any) (string, error) {
			return parser.FormatArraySimple(v), nil
		},
	})
	if err != nil {
		return err
	}

	// Register boolean array type
	err = registry.RegisterType(&types.TypeDefinition{
		Name:     "PostgresBoolArray",
		Category: types.TypeCategoryArray,
		SQLTypes: []string{"_BOOL"},
		FromString: func(s string) (any, error) {
			return parser.ParseArraySimple(s, "_BOOL")
		},
		ToString: func(v any) (string, error) {
			return parser.FormatArraySimple(v), nil
		},
	})
	if err != nil {
		return err
	}

	err = registry.RegisterType(&types.TypeDefinition{
		Name:     "PostgresUUIDArray",
		Category: types.TypeCategoryArray,
		SQLTypes: []string{"_UUID"},
		FromString: func(s string) (any, error) {
			return parser.ParseArraySimple(s, "_UUID")
		},
		ToString: func(v any) (string, error) {
			return parser.FormatArraySimple(v), nil
		},
	})
	if err != nil {
		return err
	}

	// Register date/time array types
	err = registry.RegisterType(&types.TypeDefinition{
		Name:     "PostgresDateTimeArray",
		Category: types.TypeCategoryArray,
		SQLTypes: []string{"_DATE", "_TIMESTAMP", "_TIMESTAMPTZ", "_TIME", "_TIMETZ"},
		FromString: func(s string) (any, error) {
			return parser.ParseArraySimple(s, "_TIMESTAMP")
		},
		ToString: func(v any) (string, error) {
			return parser.FormatArraySimple(v), nil
		},
	})
	if err != nil {
		return err
	}

	// Register JSON array types
	err = registry.RegisterType(&types.TypeDefinition{
		Name:     "PostgresJSONArray",
		Category: types.TypeCategoryArray,
		SQLTypes: []string{"_JSON", "_JSONB"},
		FromString: func(s string) (any, error) {
			return parser.ParseArraySimple(s, "_JSON")
		},
		ToString: func(v any) (string, error) {
			return parser.FormatArraySimple(v), nil
		},
	})
	if err != nil {
		return err
	}

	// Register database-specific handler
	converter := p.GetTypeConverter()
	if converter != nil {
		handler := types.NewPostgreSQLHandler()
		converter.RegisterDatabaseHandler("postgresql", handler)
	}

	return nil
}

func NewPostgresPlugin() *engine.Plugin {
	plugin := &PostgresPlugin{}
	plugin.Type = engine.DatabaseType_Postgres
	plugin.PluginFunctions = plugin
	plugin.GormPluginFunctions = plugin

	// Initialize type converter with PostgreSQL-specific types
	registry := types.NewTypeRegistry()
	types.InitializeDefaultTypes(registry)
	err := plugin.RegisterTypes(registry)
	if err != nil {
		return nil
	}
	plugin.SetTypeConverter(types.NewUniversalTypeConverter("postgresql", registry))

	return &plugin.Plugin
}
