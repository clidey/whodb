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

package duckdb

import (
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

// AliasMap maps DuckDB type aliases to their canonical names.
var AliasMap = map[string]string{
	// Integer aliases
	"INT":     "INTEGER",
	"INT1":    "TINYINT",
	"INT2":    "SMALLINT",
	"INT4":    "INTEGER",
	"INT8":    "BIGINT",
	"SIGNED":  "BIGINT",
	"LONG":    "BIGINT",
	"SHORT":   "SMALLINT",
	// Float aliases
	"FLOAT4":  "FLOAT",
	"FLOAT8":  "DOUBLE",
	"REAL":    "FLOAT",
	"NUMERIC": "DECIMAL",
	// Boolean aliases
	"BOOL":    "BOOLEAN",
	"LOGICAL": "BOOLEAN",
	// String aliases
	"STRING":  "VARCHAR",
	"TEXT":    "VARCHAR",
	"CHAR":    "VARCHAR",
	"BPCHAR":  "VARCHAR",
	// Binary aliases
	"BYTEA":    "BLOB",
	"BINARY":   "BLOB",
	"VARBINARY": "BLOB",
	// Datetime aliases
	"DATETIME":    "TIMESTAMP",
	"TIMESTAMPTZ": "TIMESTAMP WITH TIME ZONE",
}

// TypeDefinitions contains the canonical DuckDB types with metadata for UI.
var TypeDefinitions = []engine.TypeDefinition{
	// Boolean
	{ID: "BOOLEAN", Label: "boolean", Category: engine.TypeCategoryBoolean},
	// Signed integers
	{ID: "TINYINT", Label: "tinyint", Category: engine.TypeCategoryNumeric},
	{ID: "SMALLINT", Label: "smallint", Category: engine.TypeCategoryNumeric},
	{ID: "INTEGER", Label: "integer", Category: engine.TypeCategoryNumeric},
	{ID: "BIGINT", Label: "bigint", Category: engine.TypeCategoryNumeric},
	{ID: "HUGEINT", Label: "hugeint", Category: engine.TypeCategoryNumeric},
	// Unsigned integers
	{ID: "UTINYINT", Label: "utinyint", Category: engine.TypeCategoryNumeric},
	{ID: "USMALLINT", Label: "usmallint", Category: engine.TypeCategoryNumeric},
	{ID: "UINTEGER", Label: "uinteger", Category: engine.TypeCategoryNumeric},
	{ID: "UBIGINT", Label: "ubigint", Category: engine.TypeCategoryNumeric},
	// Floating point
	{ID: "FLOAT", Label: "float", Category: engine.TypeCategoryNumeric},
	{ID: "DOUBLE", Label: "double", Category: engine.TypeCategoryNumeric},
	{ID: "DECIMAL", Label: "decimal", HasPrecision: true, DefaultPrecision: intPtr(18), Category: engine.TypeCategoryNumeric},
	// Text
	{ID: "VARCHAR", Label: "varchar", HasLength: true, DefaultLength: intPtr(255), Category: engine.TypeCategoryText},
	// Binary
	{ID: "BLOB", Label: "blob", Category: engine.TypeCategoryBinary},
	// Datetime
	{ID: "DATE", Label: "date", Category: engine.TypeCategoryDatetime},
	{ID: "TIME", Label: "time", Category: engine.TypeCategoryDatetime},
	{ID: "TIMESTAMP", Label: "timestamp", Category: engine.TypeCategoryDatetime},
	{ID: "TIMESTAMP WITH TIME ZONE", Label: "timestamptz", Category: engine.TypeCategoryDatetime},
	{ID: "INTERVAL", Label: "interval", Category: engine.TypeCategoryDatetime},
	// JSON
	{ID: "JSON", Label: "json", Category: engine.TypeCategoryJSON},
	// Nested/Composite
	{ID: "LIST", Label: "list", Category: engine.TypeCategoryOther},
	{ID: "ARRAY", Label: "array", Category: engine.TypeCategoryOther},
	{ID: "STRUCT", Label: "struct", Category: engine.TypeCategoryOther},
	{ID: "MAP", Label: "map", Category: engine.TypeCategoryOther},
	{ID: "UNION", Label: "union", Category: engine.TypeCategoryOther},
	// Other
	{ID: "UUID", Label: "uuid", Category: engine.TypeCategoryOther},
}

func intPtr(i int) *int {
	return &i
}

// NormalizeType converts a DuckDB type alias to its canonical form.
func NormalizeType(typeName string) string {
	return common.NormalizeTypeWithMap(typeName, AliasMap)
}

// GetDatabaseMetadata returns DuckDB metadata for frontend configuration.
func (p *DuckDBPlugin) GetDatabaseMetadata() *engine.DatabaseMetadata {
	operators := make([]string, 0, len(supportedOperators))
	for op := range supportedOperators {
		operators = append(operators, op)
	}
	return &engine.DatabaseMetadata{
		DatabaseType:    engine.DatabaseType_DuckDB,
		TypeDefinitions: TypeDefinitions,
		Operators:       operators,
		AliasMap:        AliasMap,
		Capabilities: engine.Capabilities{
			SupportsScratchpad: true,
			SupportsChat:       true,
			SupportsGraph:      true,
			SupportsSchema:     true,
			SupportsModifiers:  true,
		},
	}
}
