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

package sqlite3

import (
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

// AliasMap maps SQLite type aliases to their canonical storage class or affinity.
// SQLite's type affinity rules mean many types map to the same storage class.
// All keys and values are UPPERCASE.
var AliasMap = map[string]string{
	"INT":               "INTEGER",
	"TINYINT":           "INTEGER",
	"SMALLINT":          "INTEGER",
	"MEDIUMINT":         "INTEGER",
	"BIGINT":            "INTEGER",
	"INT2":              "INTEGER",
	"INT8":              "INTEGER",
	"DOUBLE":            "REAL",
	"DOUBLE PRECISION":  "REAL",
	"FLOAT":             "REAL",
	"CHARACTER":         "TEXT",
	"VARCHAR":           "TEXT",
	"VARYING CHARACTER": "TEXT",
	"NCHAR":             "TEXT",
	"NATIVE CHARACTER":  "TEXT",
	"NVARCHAR":          "TEXT",
	"CLOB":              "TEXT",
	"CHAR":              "TEXT",
	"DECIMAL":           "NUMERIC",
	"BOOL":              "BOOLEAN",
	"TIMESTAMP":         "DATETIME",
}

// TypeDefinitions contains the canonical SQLite types with metadata for UI.
var TypeDefinitions = []engine.TypeDefinition{
	{ID: "NULL", Label: "NULL", Category: engine.TypeCategoryOther},
	{ID: "INTEGER", Label: "INTEGER", Category: engine.TypeCategoryNumeric},
	{ID: "REAL", Label: "REAL", Category: engine.TypeCategoryNumeric},
	{ID: "TEXT", Label: "TEXT", Category: engine.TypeCategoryText},
	{ID: "BLOB", Label: "BLOB", Category: engine.TypeCategoryBinary},
	{ID: "NUMERIC", Label: "NUMERIC", Category: engine.TypeCategoryNumeric},
	{ID: "BOOLEAN", Label: "BOOLEAN", Category: engine.TypeCategoryBoolean},
	{ID: "DATE", Label: "DATE", Category: engine.TypeCategoryDatetime},
	{ID: "DATETIME", Label: "DATETIME", Category: engine.TypeCategoryDatetime},
}

// NormalizeType converts a SQLite type alias to its canonical form.
func NormalizeType(typeName string) string {
	return common.NormalizeTypeWithMap(typeName, AliasMap)
}

// GetDatabaseMetadata returns SQLite metadata for frontend configuration.
func (p *Sqlite3Plugin) GetDatabaseMetadata() *engine.DatabaseMetadata {
	operators := make([]string, 0, len(supportedOperators))
	for op := range supportedOperators {
		operators = append(operators, op)
	}
	return &engine.DatabaseMetadata{
		DatabaseType:    engine.DatabaseType_Sqlite3,
		TypeDefinitions: TypeDefinitions,
		Operators:       operators,
		AliasMap:        AliasMap,
	}
}
