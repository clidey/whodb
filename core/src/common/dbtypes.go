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

package common

import (
	mapset "github.com/deckarep/golang-set/v2"
)

// Shared database data type classifications across the backend.
// These group engine-visible SQL types into coarse categories.
var (
	IntTypes = mapset.NewSet(
		"INTEGER", "SMALLINT", "BIGINT", "INT", "TINYINT", "MEDIUMINT",
		"INT4", "INT8", "INT16", "INT32", "INT64",
		// Serial types (primarily Postgres)
		"SERIAL", "BIGSERIAL", "SMALLSERIAL",
	)

	UintTypes = mapset.NewSet(
		"TINYINT UNSIGNED", "SMALLINT UNSIGNED", "MEDIUMINT UNSIGNED", "BIGINT UNSIGNED",
		"UINT8", "UINT16", "UINT32", "UINT64",
	)

	FloatTypes = mapset.NewSet(
		"REAL", "NUMERIC", "DOUBLE PRECISION", "FLOAT", "NUMBER", "DOUBLE", "DECIMAL",
		// Money appears in some engines (e.g., Postgres, MSSQL)
		"MONEY",
	)

	BoolTypes = mapset.NewSet(
		"BOOLEAN", "BIT", "BOOL",
	)

	DateTypes = mapset.NewSet(
		"DATE",
	)

	DateTimeTypes = mapset.NewSet(
		"DATETIME", "TIMESTAMP", "TIMESTAMP WITH TIME ZONE", "TIMESTAMP WITHOUT TIME ZONE",
		"DATETIME2", "SMALLDATETIME", "TIMETZ", "TIMESTAMPTZ", "TIME",
	)

	UuidTypes = mapset.NewSet(
		"UUID",
	)

	TextTypes = mapset.NewSet(
		"TEXT", "VARCHAR", "CHAR", "CHARACTER VARYING", "CHARACTER", "STRING",
		"LONGTEXT", "MEDIUMTEXT", "TINYTEXT",
	)

	JsonTypes = mapset.NewSet(
		"JSON", "JSONB",
	)

	BinaryTypes = mapset.NewSet(
		"BLOB", "BYTEA", "VARBINARY", "BINARY", "IMAGE", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB",
	)
)
