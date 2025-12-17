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
)

// AliasMap maps ClickHouse type aliases to their canonical names.
// All keys and values are UPPERCASE.
var AliasMap = map[string]string{
	// Integer aliases (SQL standard to ClickHouse)
	"TINYINT":  "INT8",
	"SMALLINT": "INT16",
	"INT":      "INT32",
	"INTEGER":  "INT32",
	"BIGINT":   "INT64",

	// Float aliases
	"FLOAT":  "FLOAT32",
	"DOUBLE": "FLOAT64",

	// Boolean alias
	"BOOLEAN": "BOOL",

	// String aliases
	"TEXT":    "STRING",
	"VARCHAR": "STRING",
	"CHAR":    "STRING",

	// Datetime alias
	"TIMESTAMP": "DATETIME",
}

// NormalizeType converts a ClickHouse type alias to its canonical form.
func NormalizeType(typeName string) string {
	return common.NormalizeTypeWithMap(typeName, AliasMap)
}
