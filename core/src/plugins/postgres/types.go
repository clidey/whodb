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
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

// AliasMap maps PostgreSQL type aliases to their canonical names.
// All keys and values are UPPERCASE.
var AliasMap = map[string]string{
	"INT":                         "INTEGER",
	"INT2":                        "SMALLINT",
	"INT4":                        "INTEGER",
	"INT8":                        "BIGINT",
	"SERIAL2":                     "SMALLSERIAL",
	"SERIAL4":                     "SERIAL",
	"SERIAL8":                     "BIGSERIAL",
	"FLOAT":                       "DOUBLE PRECISION",
	"FLOAT4":                      "REAL",
	"FLOAT8":                      "DOUBLE PRECISION",
	"BOOL":                        "BOOLEAN",
	"VARCHAR":                     "CHARACTER VARYING",
	"CHAR":                        "CHARACTER",
	"BPCHAR":                      "CHARACTER",
	"TIMESTAMP WITHOUT TIME ZONE": "TIMESTAMP",
	"TIMESTAMPTZ":                 "TIMESTAMP WITH TIME ZONE",
	"TIME WITHOUT TIME ZONE":      "TIME",
	"TIMETZ":                      "TIME WITH TIME ZONE",
}

// TypeDefinitions contains the canonical PostgreSQL types with metadata for UI.
var TypeDefinitions = []engine.TypeDefinition{
	{ID: "SMALLINT", Label: "smallint", Category: engine.TypeCategoryNumeric},
	{ID: "INTEGER", Label: "integer", Category: engine.TypeCategoryNumeric},
	{ID: "BIGINT", Label: "bigint", Category: engine.TypeCategoryNumeric},
	{ID: "SERIAL", Label: "serial", Category: engine.TypeCategoryNumeric},
	{ID: "BIGSERIAL", Label: "bigserial", Category: engine.TypeCategoryNumeric},
	{ID: "SMALLSERIAL", Label: "smallserial", Category: engine.TypeCategoryNumeric},
	{ID: "DECIMAL", Label: "decimal", HasPrecision: true, DefaultPrecision: engine.IntPtr(10), Category: engine.TypeCategoryNumeric},
	{ID: "NUMERIC", Label: "numeric", HasPrecision: true, DefaultPrecision: engine.IntPtr(10), Category: engine.TypeCategoryNumeric},
	{ID: "REAL", Label: "real", Category: engine.TypeCategoryNumeric},
	{ID: "DOUBLE PRECISION", Label: "double precision", Category: engine.TypeCategoryNumeric},
	{ID: "MONEY", Label: "money", Category: engine.TypeCategoryNumeric},
	{ID: "CHARACTER VARYING", Label: "varchar", HasLength: true, DefaultLength: engine.IntPtr(255), Category: engine.TypeCategoryText},
	{ID: "CHARACTER", Label: "char", HasLength: true, DefaultLength: engine.IntPtr(1), Category: engine.TypeCategoryText},
	{ID: "TEXT", Label: "text", Category: engine.TypeCategoryText},
	{ID: "BYTEA", Label: "bytea", Category: engine.TypeCategoryBinary},
	{ID: "TIMESTAMP", Label: "timestamp", Category: engine.TypeCategoryDatetime},
	{ID: "TIMESTAMP WITH TIME ZONE", Label: "timestamptz", Category: engine.TypeCategoryDatetime},
	{ID: "DATE", Label: "date", Category: engine.TypeCategoryDatetime},
	{ID: "TIME", Label: "time", Category: engine.TypeCategoryDatetime},
	{ID: "TIME WITH TIME ZONE", Label: "timetz", Category: engine.TypeCategoryDatetime},
	// {ID: "INTERVAL", Label: "interval", Category: engine.TypeCategoryDatetime},
	{ID: "BOOLEAN", Label: "boolean", Category: engine.TypeCategoryBoolean},
	{ID: "JSON", Label: "json", Category: engine.TypeCategoryJSON},
	{ID: "JSONB", Label: "jsonb", Category: engine.TypeCategoryJSON},
	{ID: "UUID", Label: "uuid", Category: engine.TypeCategoryOther},
	{ID: "CIDR", Label: "cidr", Category: engine.TypeCategoryOther},
	{ID: "INET", Label: "inet", Category: engine.TypeCategoryOther},
	{ID: "MACADDR", Label: "macaddr", Category: engine.TypeCategoryOther},
	{ID: "POINT", Label: "point", Category: engine.TypeCategoryOther},
	{ID: "LINE", Label: "line", Category: engine.TypeCategoryOther},
	{ID: "LSEG", Label: "lseg", Category: engine.TypeCategoryOther},
	{ID: "BOX", Label: "box", Category: engine.TypeCategoryOther},
	{ID: "PATH", Label: "path", Category: engine.TypeCategoryOther},
	{ID: "CIRCLE", Label: "circle", Category: engine.TypeCategoryOther},
	{ID: "POLYGON", Label: "polygon", Category: engine.TypeCategoryOther},
	{ID: "XML", Label: "xml", Category: engine.TypeCategoryOther},
	{ID: "ARRAY", Label: "array", Category: engine.TypeCategoryOther},
	{ID: "HSTORE", Label: "hstore", Category: engine.TypeCategoryOther},
}

// NormalizeType converts a PostgreSQL type alias to its canonical form.
func NormalizeType(typeName string) string {
	return common.NormalizeTypeWithMap(typeName, AliasMap)
}

// GetDatabaseMetadata returns PostgreSQL metadata for frontend configuration.
func (p *PostgresPlugin) GetDatabaseMetadata() *engine.DatabaseMetadata {
	operators := make([]string, 0, len(supportedOperators))
	for op := range supportedOperators {
		operators = append(operators, op)
	}
	return &engine.DatabaseMetadata{
		DatabaseType:    engine.DatabaseType_Postgres,
		TypeDefinitions: TypeDefinitions,
		Operators:       operators,
		AliasMap:        AliasMap,
	}
}
