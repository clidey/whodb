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

// Package specs exposes side-effect-free source-owned metadata specifications
// that can be shared by sourcecatalog registration and plugin implementations.
package specs

import "github.com/clidey/whodb/core/src/engine"

var PostgreSQLSupportedOperators = map[string]string{
	"=": "=", ">=": ">=", ">": ">", "<=": "<=", "<": "<", "<>": "<>",
	"!=": "!=", opBetween: opBetween, opNotBetween: opNotBetween,
	opLike: opLike, opNotLike: opNotLike, "ILIKE": "ILIKE", "NOT ILIKE": "NOT ILIKE",
	"IN": "IN", opNotIn: opNotIn,
	opIsNull: opIsNull, opIsNotNull: opIsNotNull, "AND": "AND", "OR": "OR", "NOT": "NOT",
}

var PostgresAliasMap = map[string]string{
	"INT":                         typeInteger,
	"INT2":                        typeSmallint,
	"INT4":                        typeInteger,
	"INT8":                        typeBigint,
	"SERIAL2":                     "SMALLSERIAL",
	"SERIAL4":                     "SERIAL",
	"SERIAL8":                     "BIGSERIAL",
	"FLOAT":                       "DOUBLE PRECISION",
	"FLOAT4":                      typeReal,
	"FLOAT8":                      "DOUBLE PRECISION",
	"BOOL":                        typeBoolean,
	typeVarchar:                   "CHARACTER VARYING",
	"CHAR":                        "CHARACTER",
	"BPCHAR":                      "CHARACTER",
	"TIMESTAMP WITHOUT TIME ZONE": typeTimestamp,
	"TIMESTAMPTZ":                 "TIMESTAMP WITH TIME ZONE",
	"TIME WITHOUT TIME ZONE":      "TIME",
	"TIMETZ":                      "TIME WITH TIME ZONE",
}

var PostgresTypeDefinitions = []engine.TypeDefinition{
	{ID: typeSmallint, Label: "smallint", Category: engine.TypeCategoryNumeric},
	{ID: typeInteger, Label: "integer", Category: engine.TypeCategoryNumeric},
	{ID: typeBigint, Label: "bigint", Category: engine.TypeCategoryNumeric},
	{ID: "SERIAL", Label: "serial", Category: engine.TypeCategoryNumeric},
	{ID: "BIGSERIAL", Label: "bigserial", Category: engine.TypeCategoryNumeric},
	{ID: "SMALLSERIAL", Label: "smallserial", Category: engine.TypeCategoryNumeric},
	{ID: typeDecimal, Label: "decimal", HasPrecision: true, DefaultPrecision: new(10), Category: engine.TypeCategoryNumeric},
	{ID: "NUMERIC", Label: "numeric", HasPrecision: true, DefaultPrecision: new(10), Category: engine.TypeCategoryNumeric},
	{ID: typeReal, Label: "real", Category: engine.TypeCategoryNumeric},
	{ID: "DOUBLE PRECISION", Label: "double precision", Category: engine.TypeCategoryNumeric},
	{ID: "MONEY", Label: "money", Category: engine.TypeCategoryNumeric},
	{ID: "CHARACTER VARYING", Label: "varchar", HasLength: true, DefaultLength: new(255), Category: engine.TypeCategoryText},
	{ID: "CHARACTER", Label: "char", HasLength: true, DefaultLength: new(1), Category: engine.TypeCategoryText},
	{ID: typeText, Label: "text", Category: engine.TypeCategoryText},
	{ID: "BYTEA", Label: "bytea", Category: engine.TypeCategoryBinary},
	{ID: typeTimestamp, Label: "timestamp", Category: engine.TypeCategoryDatetime},
	{ID: "TIMESTAMP WITH TIME ZONE", Label: "timestamptz", Category: engine.TypeCategoryDatetime},
	{ID: "DATE", Label: "date", Category: engine.TypeCategoryDatetime},
	{ID: "TIME", Label: "time", Category: engine.TypeCategoryDatetime},
	{ID: "TIME WITH TIME ZONE", Label: "timetz", Category: engine.TypeCategoryDatetime},
	{ID: typeBoolean, Label: "boolean", Category: engine.TypeCategoryBoolean},
	{ID: "JSON", Label: "json", Category: engine.TypeCategoryJSON},
	{ID: "JSONB", Label: "jsonb", Category: engine.TypeCategoryJSON},
	{ID: "UUID", Label: "uuid", Category: engine.TypeCategoryOther},
	{ID: "CIDR", Label: "cidr", Category: engine.TypeCategoryOther},
	{ID: "INET", Label: "inet", Category: engine.TypeCategoryOther},
	{ID: "MACADDR", Label: "macaddr", Category: engine.TypeCategoryOther},
	{ID: "POINT", Label: "point", Category: engine.TypeCategoryOther},
	{ID: "LINE", Label: "line", Category: engine.TypeCategoryOther},
	{ID: "LSEG", Label: "lseg", Category: engine.TypeCategoryOther},
	{ID: "BOX", Label: "box", Category: engine.TypeCategoryOther},
	{ID: "PATH", Label: "path", Category: engine.TypeCategoryOther},
	{ID: "CIRCLE", Label: "circle", Category: engine.TypeCategoryOther},
	{ID: "POLYGON", Label: "polygon", Category: engine.TypeCategoryOther},
	{ID: "XML", Label: "xml", Category: engine.TypeCategoryOther},
	{ID: "ARRAY", Label: "array", Category: engine.TypeCategoryOther},
	{ID: "HSTORE", Label: "hstore", Category: engine.TypeCategoryOther},
}

// QuestDBAliasMap maps common SQL aliases to QuestDB type definition IDs.
var QuestDBAliasMap = map[string]string{
	"BOOL":      typeBoolean,
	typeInteger: "INT",
	typeBigint:  "LONG",
	typeText:    typeVarchar,
}

// QuestDBTypeDefinitions contains the column types exposed for QuestDB table creation.
var QuestDBTypeDefinitions = []engine.TypeDefinition{
	{ID: typeBoolean, Label: "boolean", Category: engine.TypeCategoryBoolean},
	{ID: "BYTE", Label: "byte", Category: engine.TypeCategoryNumeric},
	{ID: "SHORT", Label: "short", Category: engine.TypeCategoryNumeric},
	{ID: "INT", Label: "int", Category: engine.TypeCategoryNumeric},
	{ID: "LONG", Label: "long", Category: engine.TypeCategoryNumeric},
	{ID: "FLOAT", Label: "float", Category: engine.TypeCategoryNumeric},
	{ID: "DOUBLE", Label: "double", Category: engine.TypeCategoryNumeric},
	{ID: typeDecimal, Label: "decimal", HasPrecision: true, DefaultPrecision: new(18), Category: engine.TypeCategoryNumeric},
	{ID: typeVarchar, Label: "varchar", Category: engine.TypeCategoryText},
	{ID: "STRING", Label: "string", Category: engine.TypeCategoryText},
	{ID: "SYMBOL", Label: "symbol", Category: engine.TypeCategoryText},
	{ID: "BINARY", Label: "binary", Category: engine.TypeCategoryBinary},
	{ID: "DATE", Label: "date", Category: engine.TypeCategoryDatetime},
	{ID: typeTimestamp, Label: "timestamp", Category: engine.TypeCategoryDatetime},
	{ID: "TIMESTAMP_NS", Label: "timestamp_ns", Category: engine.TypeCategoryDatetime},
	{ID: "UUID", Label: "uuid", Category: engine.TypeCategoryOther},
	{ID: "IPV4", Label: "ipv4", Category: engine.TypeCategoryOther},
	{ID: "LONG256", Label: "long256", Category: engine.TypeCategoryOther},
}

var CockroachDBTypeDefinitions = []engine.TypeDefinition{
	{ID: typeSmallint, Label: "smallint", Category: engine.TypeCategoryNumeric},
	{ID: typeInteger, Label: "integer", Category: engine.TypeCategoryNumeric},
	{ID: typeBigint, Label: "bigint", Category: engine.TypeCategoryNumeric},
	{ID: "SMALLSERIAL", Label: "smallserial", Category: engine.TypeCategoryNumeric},
	{ID: "SERIAL", Label: "serial", Category: engine.TypeCategoryNumeric},
	{ID: "BIGSERIAL", Label: "bigserial", Category: engine.TypeCategoryNumeric},
	{ID: typeDecimal, Label: "decimal", HasPrecision: true, DefaultPrecision: new(10), Category: engine.TypeCategoryNumeric},
	{ID: "NUMERIC", Label: "numeric", HasPrecision: true, DefaultPrecision: new(10), Category: engine.TypeCategoryNumeric},
	{ID: typeReal, Label: "real", Category: engine.TypeCategoryNumeric},
	{ID: "DOUBLE PRECISION", Label: "double precision", Category: engine.TypeCategoryNumeric},
	{ID: "CHARACTER VARYING", Label: "varchar", HasLength: true, DefaultLength: new(255), Category: engine.TypeCategoryText},
	{ID: "CHARACTER", Label: "char", HasLength: true, DefaultLength: new(1), Category: engine.TypeCategoryText},
	{ID: typeText, Label: "text", Category: engine.TypeCategoryText},
	{ID: "BYTEA", Label: "bytea", Category: engine.TypeCategoryBinary},
	{ID: typeTimestamp, Label: "timestamp", Category: engine.TypeCategoryDatetime},
	{ID: "TIMESTAMP WITH TIME ZONE", Label: "timestamptz", Category: engine.TypeCategoryDatetime},
	{ID: "DATE", Label: "date", Category: engine.TypeCategoryDatetime},
	{ID: "TIME", Label: "time", Category: engine.TypeCategoryDatetime},
	{ID: "TIME WITH TIME ZONE", Label: "timetz", Category: engine.TypeCategoryDatetime},
	{ID: typeBoolean, Label: "boolean", Category: engine.TypeCategoryBoolean},
	{ID: "JSON", Label: "json", Category: engine.TypeCategoryJSON},
	{ID: "JSONB", Label: "jsonb", Category: engine.TypeCategoryJSON},
	{ID: "UUID", Label: "uuid", Category: engine.TypeCategoryOther},
	{ID: "INET", Label: "inet", Category: engine.TypeCategoryOther},
	{ID: "ARRAY", Label: "array", Category: engine.TypeCategoryOther},
}

var MySQLSupportedOperators = map[string]string{
	"=": "=", ">=": ">=", ">": ">", "<=": "<=", "<": "<", "<>": "<>",
	"!=": "!=", opBetween: opBetween, opNotBetween: opNotBetween,
	opLike: opLike, opNotLike: opNotLike, "IN": "IN", opNotIn: opNotIn,
	opIsNull: opIsNull, opIsNotNull: opIsNotNull, "AND": "AND", "OR": "OR", "NOT": "NOT",
}

var MySQLAliasMap = map[string]string{
	typeInteger:         "INT",
	"BOOL":              typeBoolean,
	"DEC":               typeDecimal,
	"FIXED":             typeDecimal,
	"NUMERIC":           typeDecimal,
	"DOUBLE PRECISION":  "DOUBLE",
	typeReal:            "DOUBLE",
	"CHARACTER":         "CHAR",
	"CHARACTER VARYING": typeVarchar,
}

var MySQLTypeDefinitions = []engine.TypeDefinition{
	{ID: "TINYINT", Label: "TINYINT", Category: engine.TypeCategoryNumeric},
	{ID: typeSmallint, Label: typeSmallint, Category: engine.TypeCategoryNumeric},
	{ID: "MEDIUMINT", Label: "MEDIUMINT", Category: engine.TypeCategoryNumeric},
	{ID: "INT", Label: "INT", Category: engine.TypeCategoryNumeric},
	{ID: typeBigint, Label: typeBigint, Category: engine.TypeCategoryNumeric},
	{ID: typeDecimal, Label: typeDecimal, HasPrecision: true, DefaultPrecision: new(10), Category: engine.TypeCategoryNumeric},
	{ID: "FLOAT", Label: "FLOAT", Category: engine.TypeCategoryNumeric},
	{ID: "DOUBLE", Label: "DOUBLE", Category: engine.TypeCategoryNumeric},
	{ID: typeVarchar, Label: typeVarchar, HasLength: true, DefaultLength: new(255), Category: engine.TypeCategoryText},
	{ID: "CHAR", Label: "CHAR", HasLength: true, DefaultLength: new(1), Category: engine.TypeCategoryText},
	{ID: "TINYTEXT", Label: "TINYTEXT", Category: engine.TypeCategoryText},
	{ID: typeText, Label: typeText, Category: engine.TypeCategoryText},
	{ID: "MEDIUMTEXT", Label: "MEDIUMTEXT", Category: engine.TypeCategoryText},
	{ID: "LONGTEXT", Label: "LONGTEXT", Category: engine.TypeCategoryText},
	{ID: "BINARY", Label: "BINARY", HasLength: true, DefaultLength: new(1), Category: engine.TypeCategoryBinary},
	{ID: "VARBINARY", Label: "VARBINARY", HasLength: true, DefaultLength: new(255), Category: engine.TypeCategoryBinary},
	{ID: "TINYBLOB", Label: "TINYBLOB", Category: engine.TypeCategoryBinary},
	{ID: "BLOB", Label: "BLOB", Category: engine.TypeCategoryBinary},
	{ID: "MEDIUMBLOB", Label: "MEDIUMBLOB", Category: engine.TypeCategoryBinary},
	{ID: "LONGBLOB", Label: "LONGBLOB", Category: engine.TypeCategoryBinary},
	{ID: "DATE", Label: "DATE", Category: engine.TypeCategoryDatetime},
	{ID: "TIME", Label: "TIME", Category: engine.TypeCategoryDatetime},
	{ID: "DATETIME", Label: "DATETIME", Category: engine.TypeCategoryDatetime},
	{ID: typeTimestamp, Label: typeTimestamp, Category: engine.TypeCategoryDatetime},
	{ID: "YEAR", Label: "YEAR", Category: engine.TypeCategoryDatetime},
	{ID: typeBoolean, Label: "BOOL", Category: engine.TypeCategoryBoolean},
	{ID: "JSON", Label: "JSON", Category: engine.TypeCategoryJSON},
	{ID: "ENUM", Label: "ENUM", Category: engine.TypeCategoryOther},
	{ID: "SET", Label: "SET", Category: engine.TypeCategoryOther},
}

var ClickHouseSupportedOperators = map[string]string{
	"=": "=", ">=": ">=", ">": ">", "<=": "<=", "<": "<", "!=": "!=", "<>": "<>", "==": "==",
	opLike: opLike, opNotLike: opNotLike, "ILIKE": "ILIKE",
	"IN": "IN", opNotIn: opNotIn, "GLOBAL IN": "GLOBAL IN", "GLOBAL NOT IN": "GLOBAL NOT IN",
	opBetween: opBetween, opNotBetween: opNotBetween,
	opIsNull: opIsNull, opIsNotNull: opIsNotNull,
	"AND": "AND", "OR": "OR", "NOT": "NOT",
}

var ClickHouseAliasMap = map[string]string{
	"TINYINT":     "Int8",
	typeSmallint:  "Int16",
	"INT":         "Int32",
	typeInteger:   "Int32",
	typeBigint:    "Int64",
	"FLOAT":       "Float32",
	"DOUBLE":      "Float64",
	typeBoolean:   "Bool",
	typeText:      "String",
	typeVarchar:   "String",
	"CHAR":        "FixedString",
	typeTimestamp: "DateTime",
}

var ClickHouseTypeDefinitions = []engine.TypeDefinition{
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
	{ID: "Decimal", Label: "Decimal", HasPrecision: true, DefaultPrecision: new(10), Category: engine.TypeCategoryNumeric},
	{ID: "Decimal32", Label: "Decimal32", HasPrecision: true, DefaultPrecision: new(9), Category: engine.TypeCategoryNumeric},
	{ID: "Decimal64", Label: "Decimal64", HasPrecision: true, DefaultPrecision: new(18), Category: engine.TypeCategoryNumeric},
	{ID: "Decimal128", Label: "Decimal128", HasPrecision: true, DefaultPrecision: new(38), Category: engine.TypeCategoryNumeric},
	{ID: "String", Label: "String", Category: engine.TypeCategoryText},
	{ID: "FixedString", Label: "FixedString", HasLength: true, DefaultLength: new(16), Category: engine.TypeCategoryText},
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

var SQLiteSupportedOperators = map[string]string{
	"=": "=", ">=": ">=", ">": ">", "<=": "<=", "<": "<", "<>": "<>", "!=": "!=",
	opBetween: opBetween, opNotBetween: opNotBetween,
	opLike: opLike, opNotLike: opNotLike, "GLOB": "GLOB",
	"IN": "IN", opNotIn: opNotIn, opIsNull: opIsNull, opIsNotNull: opIsNotNull,
	"AND": "AND", "OR": "OR", "NOT": "NOT",
}

var SQLiteAliasMap = map[string]string{
	"INT":               typeInteger,
	"TINYINT":           typeInteger,
	typeSmallint:        typeInteger,
	"MEDIUMINT":         typeInteger,
	typeBigint:          typeInteger,
	"INT2":              typeInteger,
	"INT8":              typeInteger,
	"DOUBLE":            typeReal,
	"DOUBLE PRECISION":  typeReal,
	"FLOAT":             typeReal,
	"CHARACTER":         typeText,
	typeVarchar:         typeText,
	"VARYING CHARACTER": typeText,
	"NCHAR":             typeText,
	"NATIVE CHARACTER":  typeText,
	"NVARCHAR":          typeText,
	"CLOB":              typeText,
	"CHAR":              typeText,
	typeDecimal:         "NUMERIC",
	"BOOL":              typeBoolean,
	typeTimestamp:       "DATETIME",
}

var SQLiteTypeDefinitions = []engine.TypeDefinition{
	{ID: "NULL", Label: "NULL", Category: engine.TypeCategoryOther},
	{ID: typeInteger, Label: typeInteger, Category: engine.TypeCategoryNumeric},
	{ID: typeReal, Label: typeReal, Category: engine.TypeCategoryNumeric},
	{ID: typeText, Label: typeText, Category: engine.TypeCategoryText},
	{ID: "BLOB", Label: "BLOB", Category: engine.TypeCategoryBinary},
	{ID: "NUMERIC", Label: "NUMERIC", Category: engine.TypeCategoryNumeric},
	{ID: typeBoolean, Label: typeBoolean, Category: engine.TypeCategoryBoolean},
	{ID: "DATE", Label: "DATE", Category: engine.TypeCategoryDatetime},
	{ID: "DATETIME", Label: "DATETIME", Category: engine.TypeCategoryDatetime},
}

var DuckDBSupportedOperators = map[string]string{
	"=": "=", ">=": ">=", ">": ">", "<=": "<=", "<": "<", "<>": "<>", "!=": "!=",
	opBetween: opBetween, opNotBetween: opNotBetween,
	opLike: opLike, opNotLike: opNotLike,
	"ILIKE": "ILIKE", "NOT ILIKE": "NOT ILIKE",
	"IN": "IN", opNotIn: opNotIn, opIsNull: opIsNull, opIsNotNull: opIsNotNull,
	"AND": "AND", "OR": "OR", "NOT": "NOT",
}

var DuckDBAliasMap = map[string]string{
	"INT":         typeInteger,
	"INT1":        "TINYINT",
	"INT2":        typeSmallint,
	"INT4":        typeInteger,
	"INT8":        typeBigint,
	"SIGNED":      typeBigint,
	"LONG":        typeBigint,
	"SHORT":       typeSmallint,
	"FLOAT4":      "FLOAT",
	"FLOAT8":      "DOUBLE",
	typeReal:      "FLOAT",
	"NUMERIC":     typeDecimal,
	"BOOL":        typeBoolean,
	"LOGICAL":     typeBoolean,
	"STRING":      typeVarchar,
	typeText:      typeVarchar,
	"CHAR":        typeVarchar,
	"BPCHAR":      typeVarchar,
	"BYTEA":       "BLOB",
	"BINARY":      "BLOB",
	"VARBINARY":   "BLOB",
	"DATETIME":    typeTimestamp,
	"TIMESTAMPTZ": "TIMESTAMP WITH TIME ZONE",
}

var DuckDBTypeDefinitions = []engine.TypeDefinition{
	{ID: typeBoolean, Label: "boolean", Category: engine.TypeCategoryBoolean},
	{ID: "TINYINT", Label: "tinyint", Category: engine.TypeCategoryNumeric},
	{ID: typeSmallint, Label: "smallint", Category: engine.TypeCategoryNumeric},
	{ID: typeInteger, Label: "integer", Category: engine.TypeCategoryNumeric},
	{ID: typeBigint, Label: "bigint", Category: engine.TypeCategoryNumeric},
	{ID: "HUGEINT", Label: "hugeint", Category: engine.TypeCategoryNumeric},
	{ID: "UTINYINT", Label: "utinyint", Category: engine.TypeCategoryNumeric},
	{ID: "USMALLINT", Label: "usmallint", Category: engine.TypeCategoryNumeric},
	{ID: "UINTEGER", Label: "uinteger", Category: engine.TypeCategoryNumeric},
	{ID: "UBIGINT", Label: "ubigint", Category: engine.TypeCategoryNumeric},
	{ID: "FLOAT", Label: "float", Category: engine.TypeCategoryNumeric},
	{ID: "DOUBLE", Label: "double", Category: engine.TypeCategoryNumeric},
	{ID: typeDecimal, Label: "decimal", HasPrecision: true, DefaultPrecision: new(18), Category: engine.TypeCategoryNumeric},
	{ID: typeVarchar, Label: "varchar", HasLength: true, DefaultLength: new(255), Category: engine.TypeCategoryText},
	{ID: "BLOB", Label: "blob", Category: engine.TypeCategoryBinary},
	{ID: "DATE", Label: "date", Category: engine.TypeCategoryDatetime},
	{ID: "TIME", Label: "time", Category: engine.TypeCategoryDatetime},
	{ID: typeTimestamp, Label: "timestamp", Category: engine.TypeCategoryDatetime},
	{ID: "TIMESTAMP WITH TIME ZONE", Label: "timestamptz", Category: engine.TypeCategoryDatetime},
	{ID: "INTERVAL", Label: "interval", Category: engine.TypeCategoryDatetime},
	{ID: "JSON", Label: "json", Category: engine.TypeCategoryJSON},
	{ID: "LIST", Label: "list", Category: engine.TypeCategoryOther},
	{ID: "ARRAY", Label: "array", Category: engine.TypeCategoryOther},
	{ID: "STRUCT", Label: "struct", Category: engine.TypeCategoryOther},
	{ID: "MAP", Label: "map", Category: engine.TypeCategoryOther},
	{ID: "UNION", Label: "union", Category: engine.TypeCategoryOther},
	{ID: "UUID", Label: "uuid", Category: engine.TypeCategoryOther},
}
