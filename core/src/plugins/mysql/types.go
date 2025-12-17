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

package mysql

import (
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

// AliasMap maps MySQL type aliases to their canonical names.
// All keys and values are UPPERCASE.
var AliasMap = map[string]string{
	"INTEGER":           "INT",
	"BOOL":              "BOOLEAN",
	"DEC":               "DECIMAL",
	"FIXED":             "DECIMAL",
	"NUMERIC":           "DECIMAL",
	"DOUBLE PRECISION":  "DOUBLE",
	"REAL":              "DOUBLE",
	"CHARACTER":         "CHAR",
	"CHARACTER VARYING": "VARCHAR",
}

// TypeDefinitions contains the canonical MySQL types with metadata for UI.
var TypeDefinitions = []engine.TypeDefinition{
	{ID: "TINYINT", Label: "TINYINT", Category: engine.TypeCategoryNumeric},
	{ID: "SMALLINT", Label: "SMALLINT", Category: engine.TypeCategoryNumeric},
	{ID: "MEDIUMINT", Label: "MEDIUMINT", Category: engine.TypeCategoryNumeric},
	{ID: "INT", Label: "INT", Category: engine.TypeCategoryNumeric},
	{ID: "BIGINT", Label: "BIGINT", Category: engine.TypeCategoryNumeric},
	{ID: "DECIMAL", Label: "DECIMAL", HasPrecision: true, DefaultPrecision: engine.IntPtr(10), Category: engine.TypeCategoryNumeric},
	{ID: "FLOAT", Label: "FLOAT", Category: engine.TypeCategoryNumeric},
	{ID: "DOUBLE", Label: "DOUBLE", Category: engine.TypeCategoryNumeric},
	{ID: "VARCHAR", Label: "VARCHAR", HasLength: true, DefaultLength: engine.IntPtr(255), Category: engine.TypeCategoryText},
	{ID: "CHAR", Label: "CHAR", HasLength: true, DefaultLength: engine.IntPtr(1), Category: engine.TypeCategoryText},
	{ID: "TINYTEXT", Label: "TINYTEXT", Category: engine.TypeCategoryText},
	{ID: "TEXT", Label: "TEXT", Category: engine.TypeCategoryText},
	{ID: "MEDIUMTEXT", Label: "MEDIUMTEXT", Category: engine.TypeCategoryText},
	{ID: "LONGTEXT", Label: "LONGTEXT", Category: engine.TypeCategoryText},
	{ID: "BINARY", Label: "BINARY", HasLength: true, DefaultLength: engine.IntPtr(1), Category: engine.TypeCategoryBinary},
	{ID: "VARBINARY", Label: "VARBINARY", HasLength: true, DefaultLength: engine.IntPtr(255), Category: engine.TypeCategoryBinary},
	{ID: "TINYBLOB", Label: "TINYBLOB", Category: engine.TypeCategoryBinary},
	{ID: "BLOB", Label: "BLOB", Category: engine.TypeCategoryBinary},
	{ID: "MEDIUMBLOB", Label: "MEDIUMBLOB", Category: engine.TypeCategoryBinary},
	{ID: "LONGBLOB", Label: "LONGBLOB", Category: engine.TypeCategoryBinary},
	{ID: "DATE", Label: "DATE", Category: engine.TypeCategoryDatetime},
	{ID: "TIME", Label: "TIME", Category: engine.TypeCategoryDatetime},
	{ID: "DATETIME", Label: "DATETIME", Category: engine.TypeCategoryDatetime},
	{ID: "TIMESTAMP", Label: "TIMESTAMP", Category: engine.TypeCategoryDatetime},
	{ID: "YEAR", Label: "YEAR", Category: engine.TypeCategoryDatetime},
	{ID: "BOOLEAN", Label: "BOOL", Category: engine.TypeCategoryBoolean},
	{ID: "JSON", Label: "JSON", Category: engine.TypeCategoryJSON},
	{ID: "ENUM", Label: "ENUM", Category: engine.TypeCategoryOther},
	{ID: "SET", Label: "SET", Category: engine.TypeCategoryOther},
}

// NormalizeType converts a MySQL type alias to its canonical form.
func NormalizeType(typeName string) string {
	return common.NormalizeTypeWithMap(typeName, AliasMap)
}

// GetDatabaseMetadata returns MySQL/MariaDB metadata for frontend configuration.
func (p *MySQLPlugin) GetDatabaseMetadata() *engine.DatabaseMetadata {
	operators := make([]string, 0, len(supportedOperators))
	for op := range supportedOperators {
		operators = append(operators, op)
	}
	return &engine.DatabaseMetadata{
		DatabaseType:    p.Type, // Uses the plugin's actual type (MySQL or MariaDB)
		TypeDefinitions: TypeDefinitions,
		Operators:       operators,
		AliasMap:        AliasMap,
	}
}
