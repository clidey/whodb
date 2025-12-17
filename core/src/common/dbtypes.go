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
//
// IMPORTANT: These sets contain CANONICAL types only - no aliases.
// Types are normalized via per-plugin NormalizeType() before classification.
// Each canonical type listed here is the official form for at least one database.
var (
	IntTypes = mapset.NewSet(
		"INTEGER", "SMALLINT", "BIGINT", "INT", "TINYINT", "MEDIUMINT",
		"INT8", "INT16", "INT32", "INT64", "INT128", "INT256",
		"SERIAL", "BIGSERIAL", "SMALLSERIAL",
	)

	UintTypes = mapset.NewSet(
		"TINYINT UNSIGNED", "SMALLINT UNSIGNED", "MEDIUMINT UNSIGNED",
		"INT UNSIGNED", "BIGINT UNSIGNED",
		"UINT8", "UINT16", "UINT32", "UINT64", "UINT128", "UINT256",
	)

	FloatTypes = mapset.NewSet(
		"REAL", "DOUBLE PRECISION", "NUMERIC", "DECIMAL",
		"FLOAT", "DOUBLE",
		"NUMBER",
		"MONEY",
		"FLOAT32", "FLOAT64",
	)

	BoolTypes = mapset.NewSet(
		"BOOLEAN",
		"BIT",
	)

	DateTypes = mapset.NewSet(
		"DATE",
		"DATE32",
	)

	DateTimeTypes = mapset.NewSet(
		"TIMESTAMP", "TIMESTAMP WITH TIME ZONE",
		"TIME", "TIME WITH TIME ZONE",
		"DATETIME", "YEAR",
		"DATETIME2", "SMALLDATETIME",
		"INTERVAL",
		"DATETIME64",
	)

	UuidTypes = mapset.NewSet(
		"UUID",
	)

	TextTypes = mapset.NewSet(
		"CHARACTER VARYING", "CHARACTER", "TEXT",
		"VARCHAR", "CHAR",
		"TINYTEXT", "MEDIUMTEXT", "LONGTEXT",
		"STRING", "FIXEDSTRING",
	)

	JsonTypes = mapset.NewSet(
		"JSON",
		"JSONB",
	)

	BinaryTypes = mapset.NewSet(
		"BYTEA",
		"BINARY", "VARBINARY",
		"TINYBLOB", "BLOB", "MEDIUMBLOB", "LONGBLOB",
		"IMAGE",
	)

	ArrayTypes = mapset.NewSet(
		"ARRAY",
	)

	GeometryTypes = mapset.NewSet(
		"POINT", "LINE", "LSEG", "BOX", "PATH", "POLYGON", "CIRCLE",
		"GEOMETRY", "GEOGRAPHY",
	)

	NetworkTypes = mapset.NewSet(
		"CIDR", "INET", "MACADDR", "MACADDR8",
		"IPV4", "IPV6",
	)

	XMLTypes = mapset.NewSet(
		"XML",
	)
)
