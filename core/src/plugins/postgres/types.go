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

package postgres

import (
	"github.com/clidey/whodb/core/src/common"
)

// AliasMap maps PostgreSQL type aliases to their canonical names.
// All keys and values are UPPERCASE.
var AliasMap = map[string]string{
	// Integer aliases
	"INT":     "INTEGER",
	"INT2":    "SMALLINT",
	"INT4":    "INTEGER",
	"INT8":    "BIGINT",
	"SERIAL2": "SMALLSERIAL",
	"SERIAL4": "SERIAL",
	"SERIAL8": "BIGSERIAL",

	// Float aliases
	"FLOAT":  "DOUBLE PRECISION",
	"FLOAT4": "REAL",
	"FLOAT8": "DOUBLE PRECISION",

	// Boolean alias
	"BOOL": "BOOLEAN",

	// Character aliases
	"VARCHAR": "CHARACTER VARYING",
	"CHAR":    "CHARACTER",

	// Timestamp aliases
	"TIMESTAMP WITHOUT TIME ZONE": "TIMESTAMP",
	"TIMESTAMPTZ":                 "TIMESTAMP WITH TIME ZONE",
	"TIME WITHOUT TIME ZONE":      "TIME",
	"TIMETZ":                      "TIME WITH TIME ZONE",
}

// NormalizeType converts a PostgreSQL type alias to its canonical form.
func NormalizeType(typeName string) string {
	return common.NormalizeTypeWithMap(typeName, AliasMap)
}
