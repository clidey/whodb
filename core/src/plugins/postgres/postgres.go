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
	return fmt.Sprintf("%s.%s", schema, storageUnit)
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
		if err := db.Raw("SELECT datname AS datname FROM pg_database WHERE datistemplate = false").Scan(&databases).Error; err != nil {
			return nil, err
		}
		databaseNames := []string{}
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

// GetFunctions returns all functions in the specified schema
func (p *PostgresPlugin) GetFunctions(config *engine.PluginConfig, schema string) ([]engine.DatabaseFunction, error) {
	var functions []engine.DatabaseFunction
	
	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		query := `
			SELECT 
				p.proname AS function_name,
				pg_catalog.pg_get_function_result(p.oid) AS return_type,
				pg_catalog.pg_get_function_arguments(p.oid) AS arguments,
				COALESCE(pg_catalog.pg_get_functiondef(p.oid), '') AS definition,
				l.lanname AS language,
				p.prokind = 'a' AS is_aggregate
			FROM pg_catalog.pg_proc p
			LEFT JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
			LEFT JOIN pg_catalog.pg_language l ON l.oid = p.prolang
			WHERE n.nspname = ?
			  AND p.prokind IN ('f', 'a', 'w')  -- function, aggregate, or window function
			ORDER BY p.proname`
		
		rows, err := db.Raw(query, schema).Rows()
		if err != nil {
			return false, err
		}
		defer rows.Close()
		
		for rows.Next() {
			var name, returnType, arguments, definition, language string
			var isAggregate bool
			
			err := rows.Scan(&name, &returnType, &arguments, &definition, &language, &isAggregate)
			if err != nil {
				return false, err
			}
			
			// Parse arguments into parameters
			var parameters []engine.Record
			if arguments != "" {
				// Simple parsing - could be enhanced
				argPairs := strings.Split(arguments, ", ")
				for _, pair := range argPairs {
					parts := strings.SplitN(pair, " ", 2)
					if len(parts) == 2 {
						parameters = append(parameters, engine.Record{
							Key:   parts[0], // parameter name
							Value: parts[1], // parameter type
						})
					}
				}
			}
			
			functions = append(functions, engine.DatabaseFunction{
				Name:        name,
				ReturnType:  returnType,
				Parameters:  parameters,
				Definition:  definition,
				Language:    language,
				IsAggregate: isAggregate,
			})
		}
		
		return true, nil
	})
	
	return functions, err
}

// GetProcedures returns all procedures in the specified schema
func (p *PostgresPlugin) GetProcedures(config *engine.PluginConfig, schema string) ([]engine.DatabaseProcedure, error) {
	var procedures []engine.DatabaseProcedure
	
	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		query := `
			SELECT 
				p.proname AS procedure_name,
				pg_catalog.pg_get_function_arguments(p.oid) AS arguments,
				COALESCE(pg_catalog.pg_get_functiondef(p.oid), '') AS definition,
				l.lanname AS language
			FROM pg_catalog.pg_proc p
			LEFT JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
			LEFT JOIN pg_catalog.pg_language l ON l.oid = p.prolang
			WHERE n.nspname = ?
			  AND p.prokind = 'p'  -- procedures only
			ORDER BY p.proname`
		
		rows, err := db.Raw(query, schema).Rows()
		if err != nil {
			return false, err
		}
		defer rows.Close()
		
		for rows.Next() {
			var name, arguments, definition, language string
			
			err := rows.Scan(&name, &arguments, &definition, &language)
			if err != nil {
				return false, err
			}
			
			// Parse arguments into parameters
			var parameters []engine.Record
			if arguments != "" {
				argPairs := strings.Split(arguments, ", ")
				for _, pair := range argPairs {
					parts := strings.SplitN(pair, " ", 2)
					if len(parts) == 2 {
						parameters = append(parameters, engine.Record{
							Key:   parts[0],
							Value: parts[1],
						})
					}
				}
			}
			
			procedures = append(procedures, engine.DatabaseProcedure{
				Name:       name,
				Parameters: parameters,
				Definition: definition,
				Language:   language,
			})
		}
		
		return true, nil
	})
	
	return procedures, err
}

// GetTriggers returns all triggers in the specified schema
func (p *PostgresPlugin) GetTriggers(config *engine.PluginConfig, schema string) ([]engine.DatabaseTrigger, error) {
	var triggers []engine.DatabaseTrigger
	
	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		query := `
			SELECT 
				t.tgname AS trigger_name,
				c.relname AS table_name,
				CASE
					WHEN t.tgtype & 2 = 2 THEN 'BEFORE'
					ELSE 'AFTER'
				END AS timing,
				CASE
					WHEN t.tgtype & 4 = 4 THEN 'INSERT'
					WHEN t.tgtype & 8 = 8 THEN 'DELETE'
					WHEN t.tgtype & 16 = 16 THEN 'UPDATE'
					ELSE 'UNKNOWN'
				END AS event,
				pg_catalog.pg_get_triggerdef(t.oid) AS definition
			FROM pg_catalog.pg_trigger t
			JOIN pg_catalog.pg_class c ON c.oid = t.tgrelid
			JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = ?
			  AND NOT t.tgisinternal
			ORDER BY c.relname, t.tgname`
		
		rows, err := db.Raw(query, schema).Rows()
		if err != nil {
			return false, err
		}
		defer rows.Close()
		
		for rows.Next() {
			var name, tableName, timing, event, definition string
			
			err := rows.Scan(&name, &tableName, &timing, &event, &definition)
			if err != nil {
				return false, err
			}
			
			triggers = append(triggers, engine.DatabaseTrigger{
				Name:       name,
				TableName:  tableName,
				Event:      event,
				Timing:     timing,
				Definition: definition,
			})
		}
		
		return true, nil
	})
	
	return triggers, err
}

// GetIndexes returns all indexes in the specified schema
func (p *PostgresPlugin) GetIndexes(config *engine.PluginConfig, schema string) ([]engine.DatabaseIndex, error) {
	var indexes []engine.DatabaseIndex
	
	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		query := `
			SELECT 
				i.indexname AS index_name,
				i.tablename AS table_name,
				string_agg(a.attname, ', ' ORDER BY array_position(ix.indkey, a.attnum)) AS columns,
				am.amname AS index_type,
				ix.indisunique AS is_unique,
				ix.indisprimary AS is_primary,
				pg_size_pretty(pg_relation_size(ix.indexrelid)) AS size
			FROM pg_indexes i
			JOIN pg_class c ON c.relname = i.indexname AND c.relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = i.schemaname)
			JOIN pg_index ix ON ix.indexrelid = c.oid
			JOIN pg_class tc ON tc.oid = ix.indrelid
			JOIN pg_am am ON am.oid = c.relam
			LEFT JOIN pg_attribute a ON a.attrelid = tc.oid AND a.attnum = ANY(ix.indkey)
			WHERE i.schemaname = ?
			GROUP BY i.indexname, i.tablename, am.amname, ix.indisunique, ix.indisprimary, ix.indexrelid
			ORDER BY i.tablename, i.indexname`
		
		rows, err := db.Raw(query, schema).Rows()
		if err != nil {
			return false, err
		}
		defer rows.Close()
		
		for rows.Next() {
			var name, tableName, columns, indexType, size string
			var isUnique, isPrimary bool
			
			err := rows.Scan(&name, &tableName, &columns, &indexType, &isUnique, &isPrimary, &size)
			if err != nil {
				return false, err
			}
			
			columnList := strings.Split(columns, ", ")
			
			indexes = append(indexes, engine.DatabaseIndex{
				Name:      name,
				TableName: tableName,
				Columns:   columnList,
				Type:      indexType,
				IsUnique:  isUnique,
				IsPrimary: isPrimary,
				Size:      size,
			})
		}
		
		return true, nil
	})
	
	return indexes, err
}

// GetSequences returns all sequences in the specified schema
func (p *PostgresPlugin) GetSequences(config *engine.PluginConfig, schema string) ([]engine.DatabaseSequence, error) {
	var sequences []engine.DatabaseSequence
	
	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		query := `
			SELECT 
				s.relname AS sequence_name,
				format_type(sq.seqtypid, NULL) AS data_type,
				sq.seqstart AS start_value,
				sq.seqincrement AS increment,
				sq.seqmin AS min_value,
				sq.seqmax AS max_value,
				sq.seqcache AS cache_size,
				sq.seqcycle AS is_cycle
			FROM pg_class s
			JOIN pg_namespace n ON n.oid = s.relnamespace
			JOIN pg_sequence sq ON sq.seqrelid = s.oid
			WHERE n.nspname = ?
			  AND s.relkind = 'S'
			ORDER BY s.relname`
		
		rows, err := db.Raw(query, schema).Rows()
		if err != nil {
			return false, err
		}
		defer rows.Close()
		
		for rows.Next() {
			var name, dataType string
			var startValue, increment, minValue, maxValue, cacheSize int64
			var isCycle bool
			
			err := rows.Scan(&name, &dataType, &startValue, &increment, &minValue, &maxValue, &cacheSize, &isCycle)
			if err != nil {
				return false, err
			}
			
			sequences = append(sequences, engine.DatabaseSequence{
				Name:       name,
				DataType:   dataType,
				StartValue: startValue,
				Increment:  increment,
				MinValue:   minValue,
				MaxValue:   maxValue,
				CacheSize:  cacheSize,
				IsCycle:    isCycle,
			})
		}
		
		return true, nil
	})
	
	return sequences, err
}

// GetTypes returns all user-defined types in the specified schema
func (p *PostgresPlugin) GetTypes(config *engine.PluginConfig, schema string) ([]engine.DatabaseType, error) {
	var types []engine.DatabaseType
	
	_, err := plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		query := `
			SELECT 
				t.typname AS type_name,
				n.nspname AS schema_name,
				CASE t.typtype
					WHEN 'c' THEN 'COMPOSITE'
					WHEN 'd' THEN 'DOMAIN'
					WHEN 'e' THEN 'ENUM'
					WHEN 'r' THEN 'RANGE'
					ELSE 'OTHER'
				END AS type_type,
				COALESCE(
					CASE t.typtype
						WHEN 'e' THEN (
							SELECT string_agg(e.enumlabel, ', ' ORDER BY e.enumsortorder)
							FROM pg_enum e
							WHERE e.enumtypid = t.oid
						)
						ELSE pg_catalog.format_type(t.oid, NULL)
					END,
					''
				) AS definition
			FROM pg_type t
			JOIN pg_namespace n ON n.oid = t.typnamespace
			WHERE n.nspname = ?
			  AND t.typtype IN ('c', 'd', 'e', 'r')
			  AND NOT EXISTS (
				SELECT 1 FROM pg_class c WHERE c.oid = t.typrelid AND c.relkind IN ('r', 'v', 'm')
			  )
			ORDER BY t.typname`
		
		rows, err := db.Raw(query, schema).Rows()
		if err != nil {
			return false, err
		}
		defer rows.Close()
		
		for rows.Next() {
			var name, schemaName, typeType, definition string
			
			err := rows.Scan(&name, &schemaName, &typeType, &definition)
			if err != nil {
				return false, err
			}
			
			types = append(types, engine.DatabaseType{
				Name:       name,
				Schema:     schemaName,
				Type:       typeType,
				Definition: definition,
			})
		}
		
		return true, nil
	})
	
	return types, err
}

func NewPostgresPlugin() *engine.Plugin {
	plugin := &PostgresPlugin{}
	plugin.Type = engine.DatabaseType_Postgres
	plugin.PluginFunctions = plugin
	plugin.GormPluginFunctions = plugin
	return &plugin.Plugin
}
