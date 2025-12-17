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
)

// AliasMap maps MySQL type aliases to their canonical names.
// All keys and values are UPPERCASE.
var AliasMap = map[string]string{
	// Integer alias
	"INTEGER": "INT",

	// Boolean aliases (MySQL treats BOOLEAN as TINYINT(1))
	"BOOL": "BOOLEAN",

	// Fixed-point aliases
	"DEC":   "DECIMAL",
	"FIXED": "DECIMAL",

	// Floating-point aliases
	"DOUBLE PRECISION": "DOUBLE",
	"REAL":             "DOUBLE",

	// Character aliases
	"CHARACTER":         "CHAR",
	"CHARACTER VARYING": "VARCHAR",
}

// NormalizeType converts a MySQL type alias to its canonical form.
func NormalizeType(typeName string) string {
	return common.NormalizeTypeWithMap(typeName, AliasMap)
}
