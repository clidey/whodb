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
)

// AliasMap maps SQLite type aliases to their canonical storage class or affinity.
// SQLite's type affinity rules mean many types map to the same storage class.
// All keys and values are UPPERCASE.
var AliasMap = map[string]string{
	// INTEGER affinity aliases
	"INT":       "INTEGER",
	"TINYINT":   "INTEGER",
	"SMALLINT":  "INTEGER",
	"MEDIUMINT": "INTEGER",
	"BIGINT":    "INTEGER",
	"INT2":      "INTEGER",
	"INT8":      "INTEGER",

	// REAL affinity aliases
	"DOUBLE":           "REAL",
	"DOUBLE PRECISION": "REAL",
	"FLOAT":            "REAL",

	// TEXT affinity aliases
	"CHARACTER":         "TEXT",
	"VARCHAR":           "TEXT",
	"VARYING CHARACTER": "TEXT",
	"NCHAR":             "TEXT",
	"NATIVE CHARACTER":  "TEXT",
	"NVARCHAR":          "TEXT",
	"CLOB":              "TEXT",
	"CHAR":              "TEXT",

	// NUMERIC affinity alias
	"DECIMAL": "NUMERIC",

	// Boolean alias
	"BOOL": "BOOLEAN",

	// Datetime alias
	"TIMESTAMP": "DATETIME",
}

// NormalizeType converts a SQLite type alias to its canonical form.
func NormalizeType(typeName string) string {
	return common.NormalizeTypeWithMap(typeName, AliasMap)
}
