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
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	mapset "github.com/deckarep/golang-set/v2"
	"gorm.io/gorm"
)

type GormPlugin struct {
	engine.Plugin
	GormPluginFunctions
}

type GormPluginFunctions interface {
	// these below are meant to be generic-ish implementations by the base gorm plugin
	ParseConnectionConfig(config *engine.PluginConfig) (*ConnectionInput, error)

	ConvertStringValue(value, columnType string) (interface{}, error)
	ConvertRawToRows(raw *sql.Rows) (*engine.GetRowsResult, error)
	ConvertRecordValuesToMap(values []engine.Record) (map[string]interface{}, error)

	EscapeIdentifier(identifier string) string

	// these below are meant to be implemented by the specific database plugins
	DB(config *engine.PluginConfig) (*gorm.DB, error)
	GetSupportedColumnDataTypes() mapset.Set[string]

	GetTableInfoQuery() string
	GetSchemaTableQuery() string
	GetPrimaryKeyColQuery() string
	GetColTypeQuery() string
	GetAllSchemasQuery() string
	GetCreateTableQuery(schema string, storageUnit string, columns []engine.Record) string

	FormTableName(schema string, storageUnit string) string
	EscapeSpecificIdentifier(identifier string) string
	ConvertStringValueDuringMap(value, columnType string) (interface{}, error)

	GetGraphQueryDB(db *gorm.DB, schema string) *gorm.DB
	GetTableNameAndAttributes(rows *sql.Rows, db *gorm.DB) (string, []engine.Record)
}

func (p *GormPlugin) GetStorageUnits(config *engine.PluginConfig, schema string) ([]engine.StorageUnit, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]engine.StorageUnit, error) {
		storageUnits := []engine.StorageUnit{}
		rows, err := db.Raw(p.GetTableInfoQuery(), schema).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		allTablesWithColumns, err := p.GetTableSchema(db, schema)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			tableName, attributes := p.GetTableNameAndAttributes(rows, db)
			if attributes == nil && tableName == "" {
				// skip if error getting attributes
				continue
			}

			attributes = append(attributes, allTablesWithColumns[tableName]...)

			storageUnits = append(storageUnits, engine.StorageUnit{
				Name:       tableName,
				Attributes: attributes,
			})
		}
		return storageUnits, nil
	})
}

func (p *GormPlugin) GetTableSchema(db *gorm.DB, schema string) (map[string][]engine.Record, error) {
	var result []struct {
		TableName  string `gorm:"column:TABLE_NAME"`
		ColumnName string `gorm:"column:COLUMN_NAME"`
		DataType   string `gorm:"column:DATA_TYPE"`
	}

	query := p.GetSchemaTableQuery()

	if err := db.Raw(query, schema).Scan(&result).Error; err != nil {
		return nil, err
	}

	tableColumnsMap := make(map[string][]engine.Record)
	for _, row := range result {
		tableColumnsMap[row.TableName] = append(tableColumnsMap[row.TableName], engine.Record{Key: row.ColumnName, Value: row.DataType})
	}

	return tableColumnsMap, nil
}

func (p *GormPlugin) GetAllSchemas(config *engine.PluginConfig) ([]string, error) {
	return plugins.WithConnection[[]string](config, p.DB, func(db *gorm.DB) ([]string, error) {
		var schemas []interface{}
		query := p.GetAllSchemasQuery()
		if err := db.Raw(query).Scan(&schemas).Error; err != nil {
			return nil, err
		}
		schemaNames := []string{}
		for _, schema := range schemas {
			schemaNames = append(schemaNames, fmt.Sprintf("%s", schema))
		}
		return schemaNames, nil
	})
}

func (p *GormPlugin) GetRows(config *engine.PluginConfig, schema string, storageUnit string, where *model.WhereCondition, pageSize int, pageOffset int) (*engine.GetRowsResult, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (*engine.GetRowsResult, error) {
		schema = p.EscapeIdentifier(schema)
		storageUnit = p.EscapeIdentifier(storageUnit)
		fullTableName := p.FormTableName(schema, storageUnit)

		query := db.Table(fullTableName)

		query, err := p.applyWhereConditions(query, where)
		if err != nil {
			return nil, err
		}

		query = query.Limit(pageSize).Offset(pageOffset)

		rows, err := query.Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		result, err := p.ConvertRawToRows(rows)
		if err != nil {
			return nil, err
		}

		// postprocess to add any missing data types
		for i, col := range result.Columns {
			if _, err := strconv.Atoi(col.Type); err == nil {
				result.Columns[i].Type = p.FindMissingDataType(db, col.Type)
			}
		}
		return result, nil
	})
}

func (p *GormPlugin) applyWhereConditions(query *gorm.DB, condition *model.WhereCondition) (*gorm.DB, error) {
	if condition == nil {
		return query, nil
	}

	switch condition.Type {
	case model.WhereConditionTypeAtomic:
		if condition.Atomic != nil {
			value, err := p.ConvertStringValue(condition.Atomic.Value, condition.Atomic.ColumnType)
			if err != nil {
				return nil, err
			}
			query = query.Where(fmt.Sprintf("%s = ?", condition.Atomic.Key), value)
		}

	case model.WhereConditionTypeAnd:
		if condition.And != nil {
			for _, child := range condition.And.Children {
				var err error
				query, err = p.applyWhereConditions(query, child)
				if err != nil {
					return nil, err
				}
			}
		}

	case model.WhereConditionTypeOr:
		if condition.Or != nil {
			orQueries := query
			for _, child := range condition.Or.Children {
				childQuery, err := p.applyWhereConditions(query, child)
				if err != nil {
					return nil, err
				}
				orQueries = orQueries.Or(childQuery)
			}
			query = orQueries
		}
	}

	return query, nil
}

func (p *GormPlugin) ConvertRawToRows(rows *sql.Rows) (*engine.GetRowsResult, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	// Create a map for faster column type lookup
	typeMap := make(map[string]*sql.ColumnType, len(columnTypes))
	for _, colType := range columnTypes {
		typeMap[colType.Name()] = colType
	}

	result := &engine.GetRowsResult{
		Columns: make([]engine.Column, 0, len(columns)),
		Rows:    make([][]string, 0, 100),
	}

	// todo: might have to extract some of this stuff into db specific functions
	// Build columns with type information
	for _, col := range columns {
		if colType, exists := typeMap[col]; exists {
			colTypeName := colType.DatabaseTypeName()
			if p.Type == engine.DatabaseType_Postgres && strings.HasPrefix(colTypeName, "_") {
				colTypeName = strings.Replace(colTypeName, "_", "[]", 1)
			}
			result.Columns = append(result.Columns, engine.Column{Name: col, Type: colTypeName})
		}
	}

	for rows.Next() {
		columnPointers := make([]interface{}, len(columns))
		row := make([]string, len(columns))

		for i, col := range columns {
			colType := typeMap[col]
			typeName := colType.DatabaseTypeName()

			switch typeName {
			case "VARBINARY", "BINARY", "IMAGE", "BYTEA", "BLOB", "HIERARCHYID":
				columnPointers[i] = new(sql.RawBytes)
			case "GEOMETRY", "POINT", "LINESTRING", "POLYGON", "GEOGRAPHY",
				"MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON":
				columnPointers[i] = new(sql.RawBytes)
			default:
				columnPointers[i] = new(sql.NullString)
			}
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		for i, colPtr := range columnPointers {
			colType := typeMap[columns[i]]
			typeName := colType.DatabaseTypeName()

			switch typeName {
			case "VARBINARY", "BINARY", "IMAGE", "BYTEA", "BLOB":
				rawBytes := colPtr.(*sql.RawBytes)
				if rawBytes == nil || len(*rawBytes) == 0 {
					row[i] = ""
				} else {
					row[i] = "0x" + hex.EncodeToString(*rawBytes)
				}
			default:
				val := colPtr.(*sql.NullString)
				if val.Valid {
					row[i] = val.String
				} else {
					row[i] = ""
				}
			}
		}

		result.Rows = append(result.Rows, row)
	}

	return result, nil
}

// todo: extract this into a gormplugin default if needed for other plugins
func (p *GormPlugin) FindMissingDataType(db *gorm.DB, columnType string) string {
	if p.Type == engine.DatabaseType_Postgres {
		var typname string
		if err := db.Raw("SELECT typname FROM pg_type WHERE oid = ?", columnType).Scan(&typname).Error; err != nil {
			typname = columnType
		}
		return strings.ToUpper(typname)
	}
	return columnType
}
