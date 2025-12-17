// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clickhouse

import (
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

// AliasMap maps ClickHouse type aliases to their canonical names.
// Note: ClickHouse uses mixed-case type names (Int8, String, etc.)
var AliasMap = map[string]string{
	"TINYINT":   "Int8",
	"SMALLINT":  "Int16",
	"INT":       "Int32",
	"INTEGER":   "Int32",
	"BIGINT":    "Int64",
	"FLOAT":     "Float32",
	"DOUBLE":    "Float64",
	"BOOLEAN":   "Bool",
	"TEXT":      "String",
	"VARCHAR":   "String",
	"CHAR":      "FixedString",
	"TIMESTAMP": "DateTime",
}

// TypeDefinitions contains the canonical ClickHouse types with metadata for UI.
var TypeDefinitions = []engine.TypeDefinition{
	{ID: "Int8", Label: "Int8", Category: engine.TypeCategoryNumeric},
	{ID: "Int16", Label: "Int16", Category: engine.TypeCategoryNumeric},
	{ID: "Int32", Label: "Int32", Category: engine.TypeCategoryNumeric},
	{ID: "Int64", Label: "Int64", Category: engine.TypeCategoryNumeric},
	{ID: "Int128", Label: "Int128", Category: engine.TypeCategoryNumeric},
	{ID: "Int256", Label: "Int256", Category: engine.TypeCategoryNumeric},
	{ID: "UInt8", Label: "UInt8", Category: engine.TypeCategoryNumeric},
	{ID: "UInt16", Label: "UInt16", Category: engine.TypeCategoryNumeric},
	{ID: "UInt32", Label: "UInt32", Category: engine.TypeCategoryNumeric},
	{ID: "UInt64", Label: "UInt64", Category: engine.TypeCategoryNumeric},
	{ID: "UInt128", Label: "UInt128", Category: engine.TypeCategoryNumeric},
	{ID: "UInt256", Label: "UInt256", Category: engine.TypeCategoryNumeric},
	{ID: "Float32", Label: "Float32", Category: engine.TypeCategoryNumeric},
	{ID: "Float64", Label: "Float64", Category: engine.TypeCategoryNumeric},
	{ID: "Decimal", Label: "Decimal", HasPrecision: true, DefaultPrecision: engine.IntPtr(10), Category: engine.TypeCategoryNumeric},
	{ID: "Decimal32", Label: "Decimal32", HasPrecision: true, DefaultPrecision: engine.IntPtr(9), Category: engine.TypeCategoryNumeric},
	{ID: "Decimal64", Label: "Decimal64", HasPrecision: true, DefaultPrecision: engine.IntPtr(18), Category: engine.TypeCategoryNumeric},
	{ID: "Decimal128", Label: "Decimal128", HasPrecision: true, DefaultPrecision: engine.IntPtr(38), Category: engine.TypeCategoryNumeric},
	{ID: "String", Label: "String", Category: engine.TypeCategoryText},
	{ID: "FixedString", Label: "FixedString", HasLength: true, DefaultLength: engine.IntPtr(16), Category: engine.TypeCategoryText},
	{ID: "Date", Label: "Date", Category: engine.TypeCategoryDatetime},
	{ID: "Date32", Label: "Date32", Category: engine.TypeCategoryDatetime},
	{ID: "DateTime", Label: "DateTime", Category: engine.TypeCategoryDatetime},
	{ID: "DateTime64", Label: "DateTime64", Category: engine.TypeCategoryDatetime},
	{ID: "Bool", Label: "Bool", Category: engine.TypeCategoryBoolean},
	{ID: "UUID", Label: "UUID", Category: engine.TypeCategoryOther},
	{ID: "JSON", Label: "JSON", Category: engine.TypeCategoryJSON},
	{ID: "IPv4", Label: "IPv4", Category: engine.TypeCategoryOther},
	{ID: "IPv6", Label: "IPv6", Category: engine.TypeCategoryOther},
	{ID: "Enum8", Label: "Enum8", Category: engine.TypeCategoryOther},
	{ID: "Enum16", Label: "Enum16", Category: engine.TypeCategoryOther},
}

// NormalizeType converts a ClickHouse type alias to its canonical form.
func NormalizeType(typeName string) string {
	return common.NormalizeTypeWithMap(typeName, AliasMap)
}

// GetDatabaseMetadata returns ClickHouse metadata for frontend configuration.
func (p *ClickHousePlugin) GetDatabaseMetadata() *engine.DatabaseMetadata {
	operators := make([]string, 0, len(supportedOperators))
	for op := range supportedOperators {
		operators = append(operators, op)
	}
	return &engine.DatabaseMetadata{
		DatabaseType:    engine.DatabaseType_ClickHouse,
		TypeDefinitions: TypeDefinitions,
		Operators:       operators,
		AliasMap:        AliasMap,
	}
}
