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

package mockdata

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

// TypeGenerator is a function that generates mock data for a database type.
// Returns (value, handled). If handled is false, the caller falls back to default handling.
type TypeGenerator func(dbType string, databaseType string, constraints map[string]any, faker *gofakeit.Faker) (any, bool)

var eeTypeGenerator TypeGenerator

// RegisterEETypeGenerator allows EE to register a handler for EE-specific types
func RegisterEETypeGenerator(gen TypeGenerator) {
	eeTypeGenerator = gen
}

// unwrapTypeModifiers strips ClickHouse wrapper types like Nullable(...) and
// LowCardinality(...) to expose the inner type. Handles nested wrappers like
// LowCardinality(Nullable(String)). The stripping is case-insensitive.
func unwrapTypeModifiers(dbType string) string {
	for {
		lower := strings.ToLower(dbType)
		trimmed := strings.TrimSpace(lower)
		unwrapped := false
		for _, prefix := range []string{"nullable(", "lowcardinality("} {
			if strings.HasPrefix(trimmed, prefix) && strings.HasSuffix(trimmed, ")") {
				// Extract inner type preserving original case from dbType
				inner := strings.TrimSpace(dbType[len(prefix) : len(dbType)-1])
				dbType = inner
				unwrapped = true
				break
			}
		}
		if !unwrapped {
			return dbType
		}
	}
}

// GenerateByType generates a value for a database type, respecting constraints.
// The dbType parameter is the raw column type (e.g., "int", "text[]", "varchar(255)").
// The constraints map contains check_min, check_max, length, scale, check_values, etc.
func GenerateByType(dbType string, databaseType string, constraints map[string]any, faker *gofakeit.Faker) any {
	// Try ee first
	if eeTypeGenerator != nil {
		if value, handled := eeTypeGenerator(dbType, databaseType, constraints, faker); handled {
			return value
		}
	}

	// Unwrap ClickHouse wrapper types (Nullable, LowCardinality) before normalization
	dbType = unwrapTypeModifiers(dbType)

	normalizedType := strings.ToLower(dbType)

	if strings.HasSuffix(normalizedType, "[]") {
		return genArray(dbType, faker)
	}

	// Handle ClickHouse compound types before stripping parentheses
	if strings.HasPrefix(normalizedType, "array(") {
		return genClickHouseArray(dbType, databaseType, constraints, faker)
	}
	if strings.HasPrefix(normalizedType, "map(") {
		return genClickHouseMap(dbType, databaseType, faker)
	}
	if strings.HasPrefix(normalizedType, "tuple(") {
		return genClickHouseTuple(dbType, databaseType, faker)
	}

	if idx := strings.Index(normalizedType, "("); idx > 0 {
		normalizedType = normalizedType[:idx]
	}

	// Handle timestamp/time types with timezone suffixes (e.g., "timestamp with time zone")
	// These need prefix matching because PostgreSQL includes timezone info in the type name
	if strings.HasPrefix(normalizedType, "timestamp") {
		return genDateTime(faker)
	}
	if strings.HasPrefix(normalizedType, "time") && !strings.HasPrefix(normalizedType, "tinyint") {
		return genTime(faker)
	}

	switch normalizedType {
	case "int", "integer", "int2", "int4", "int8", "smallint", "bigint", "tinyint", "mediumint", "serial", "bigserial", "smallserial",
		"int16", "int32", "int64", "int128", "int256":
		return genInt(normalizedType, constraints, faker)

	case "uint", "uint8", "uint16", "uint32", "uint64", "uint128", "uint256",
		"tinyint unsigned", "smallint unsigned", "mediumint unsigned", "int unsigned", "bigint unsigned":
		return genUint(normalizedType, constraints, faker)

	case "float", "float4", "float8", "real", "double", "double precision", "decimal", "numeric", "number", "money",
		"float32", "float64":
		return genDecimal(constraints, faker)

	case "bool", "boolean", "bit":
		return faker.Bool()

	case "date", "date32":
		return genDate(faker)
	case "datetime", "datetime64":
		return genDateTime(faker)
	case "interval":
		return genInterval(faker)
	case "year":
		return genYear(faker)

	case "uuid":
		return faker.UUID()

	case "json", "jsonb":
		return genJSON(faker)

	case "bytea", "blob", "binary", "varbinary", "image", "tinyblob", "mediumblob", "longblob":
		return genBinary(constraints, faker)

	case "text", "string", "varchar", "char", "character", "character varying", "nvarchar", "nchar", "ntext", "fixedstring", "clob", "long",
		"tinytext", "mediumtext", "longtext":
		return genText(constraints, faker)

	case "hstore":
		return genHstore(faker)

	case "enum", "enum8", "enum16":
		return genText(constraints, faker)

	case "set":
		return genText(constraints, faker)

	case "inet", "cidr", "ipv4":
		return genIPv4(faker)
	case "ipv6":
		return genIPv6(faker)
	case "macaddr", "macaddr8":
		return genMACAddr(faker)

	case "point":
		return genPoint(faker)

	case "xml":
		return genXML(faker)

	case "line", "lseg", "box", "path", "polygon", "circle":
		return genGeometry(normalizedType, faker)

	case "geometry", "geography", "linestring", "multipoint", "multilinestring", "multipolygon", "geometrycollection":
		return genSpatial(normalizedType, faker)

	case "decimal32", "decimal64", "decimal128", "decimal256":
		return genClickHouseDecimal(normalizedType, constraints, faker)

	default:
		return genText(constraints, faker)
	}
}

// genInt generates an integer respecting type limits and check_min/check_max constraints.
// Type-specific limits are applied as defaults when no constraints are provided.
func genInt(typeName string, c map[string]any, f *gofakeit.Faker) any {
	// Type-specific default limits (using safe positive ranges for mock data)
	var minVal, maxVal int64
	switch typeName {
	case "tinyint":
		minVal, maxVal = 1, 127
	case "smallint", "int2", "int16", "smallserial":
		minVal, maxVal = 1, 32767
	case "mediumint":
		minVal, maxVal = 1, 8388607
	default:
		// int, integer, int4, int8, bigint, serial, bigserial, int32, int64, int128, int256
		// Conservative max to avoid overflows in triggers/computed columns
		// (e.g., SUM(price * quantity) stored in a NUMERIC(10,2) column)
		minVal, maxVal = 1, 1000
	}

	// Override with explicit constraints if provided
	if c != nil {
		if v, ok := c["check_min"].(float64); ok {
			minVal = int64(v)
		}
		if v, ok := c["check_max"].(float64); ok {
			maxVal = int64(v)
		}
	}

	// Ensure min <= max
	if minVal > maxVal {
		minVal, maxVal = maxVal, minVal
	}

	return f.IntRange(int(minVal), int(maxVal))
}

// genUint generates an unsigned integer respecting type limits and check_min/check_max constraints.
func genUint(typeName string, c map[string]any, f *gofakeit.Faker) any {
	// Type-specific default limits
	var minVal, maxVal uint64
	switch typeName {
	case "uint8", "tinyint unsigned":
		minVal, maxVal = 0, 255
	case "uint16", "smallint unsigned":
		minVal, maxVal = 0, 65535
	case "mediumint unsigned":
		minVal, maxVal = 0, 16777215
	default:
		// uint, uint32, uint64, uint128, uint256, int unsigned, bigint unsigned
		minVal, maxVal = 0, 1000
	}

	// Override with explicit constraints if provided
	if c != nil {
		if v, ok := c["check_min"].(float64); ok && v >= 0 {
			minVal = uint64(v)
		}
		if v, ok := c["check_max"].(float64); ok && v >= 0 {
			maxVal = uint64(v)
		}
	}

	if minVal > maxVal {
		minVal, maxVal = maxVal, minVal
	}

	return f.UintRange(uint(minVal), uint(maxVal))
}

// genDecimal generates a decimal number respecting precision, scale, and check_min/check_max constraints.
// Uses a conservative default max (1000) rather than precision-derived max to avoid overflows
// from triggers, computed columns, or expressions like SUM(price * quantity) that can exceed
// individual column limits.
func genDecimal(c map[string]any, f *gofakeit.Faker) any {
	minVal := 0.0
	maxVal := 1000.0
	scale := 2

	if c != nil {
		if s, ok := c["scale"].(int); ok && s >= 0 {
			scale = s
		}
		// Use precision to TIGHTEN max when the column can't hold the default range.
		// e.g., decimal(5,2) max is 999.99, which is below the default 1000 → cap to 999.99.
		// But decimal(10,2) max is 99999999.99 — DON'T inflate to that, because triggers
		// or computed columns (e.g., SUM(price * quantity)) can overflow other columns.
		var precision int
		if p, ok := c["precision"].(int); ok && p > 0 {
			precision = p
		} else if p, ok := c["precision"].(int64); ok && p > 0 {
			precision = int(p)
		}
		if precision > 0 {
			intDigits := precision - scale
			var precisionMax float64
			if intDigits > 0 {
				precisionMax = math.Pow(10, float64(intDigits)) - math.Pow(10, -float64(scale))
			} else {
				precisionMax = 1 - math.Pow(10, -float64(scale))
			}
			// Only tighten, never inflate
			if precisionMax < maxVal {
				maxVal = precisionMax
			}
		}
		// Explicit constraints override
		if v, ok := c["check_min"].(float64); ok {
			minVal = v
		}
		if v, ok := c["check_max"].(float64); ok {
			maxVal = v
		}
	}

	if minVal > maxVal {
		minVal, maxVal = maxVal, minVal
	}

	val := f.Float64Range(minVal, maxVal)
	multiplier := math.Pow(10, float64(scale))
	rounded := math.Round(val*multiplier) / multiplier
	// Clamp after rounding — rounding can push values just above the boundary
	// (e.g., 99999999.995 rounds to 100000000.00 which overflows NUMERIC(10,2))
	if rounded > maxVal {
		rounded = maxVal
	}
	if rounded < minVal {
		rounded = minVal
	}
	return rounded
}

// genDate generates a date within the last 10 years
func genDate(f *gofakeit.Faker) any {
	start := time.Now().AddDate(-10, 0, 0)
	end := time.Now()
	return f.DateRange(start, end).Format("2006-01-02")
}

// genDateTime generates a datetime within the last 10 years
func genDateTime(f *gofakeit.Faker) any {
	start := time.Now().AddDate(-10, 0, 0)
	end := time.Now()
	return f.DateRange(start, end).Format("2006-01-02 15:04:05")
}

// genTime generates a time value
func genTime(f *gofakeit.Faker) any {
	return f.Date().Format("15:04:05")
}

// genYear generates a year value (MySQL YEAR type: 1901-2155)
func genYear(f *gofakeit.Faker) any {
	return f.IntRange(1970, time.Now().Year())
}

// genInterval generates a PostgreSQL-compatible interval string
func genInterval(f *gofakeit.Faker) any {
	units := []string{"seconds", "minutes", "hours", "days", "weeks", "months", "years"}
	unit := units[f.Number(0, len(units)-1)]
	value := f.Number(1, 30)
	return fmt.Sprintf("%d %s", value, unit)
}

// genIPv4 generates an IPv4 address string
func genIPv4(f *gofakeit.Faker) any {
	return f.IPv4Address()
}

// genIPv6 generates an IPv6 address string
func genIPv6(f *gofakeit.Faker) any {
	return f.IPv6Address()
}

// genMACAddr generates a MAC address string
func genMACAddr(f *gofakeit.Faker) any {
	return f.MacAddress()
}

// genPoint generates a PostgreSQL POINT value as "(x,y)"
func genPoint(f *gofakeit.Faker) any {
	x := f.Float64Range(-180, 180)
	y := f.Float64Range(-90, 90)
	return fmt.Sprintf("(%f,%f)", x, y)
}

// genGeometry generates PostgreSQL native geometry types
func genGeometry(typeName string, f *gofakeit.Faker) any {
	switch typeName {
	case "line":
		return "{1,2,3}"
	case "lseg":
		return "[(0,0),(1,1)]"
	case "box":
		return "(1,1),(0,0)"
	case "path":
		return "[(0,0),(1,1),(2,0)]"
	case "polygon":
		return "((0,0),(1,0),(1,1),(0,1),(0,0))"
	case "circle":
		return "<(0,0),1>"
	default:
		return "(0,0)"
	}
}

// genSpatial generates PostGIS/spatial types in WKT format
func genSpatial(typeName string, f *gofakeit.Faker) any {
	switch typeName {
	case "geometry", "geography":
		return "POINT(0 0)"
	case "linestring":
		return "LINESTRING(0 0, 1 1, 2 2)"
	case "multipoint":
		return "MULTIPOINT((0 0), (1 1))"
	case "multilinestring":
		return "MULTILINESTRING((0 0, 1 1), (2 2, 3 3))"
	case "multipolygon":
		return "MULTIPOLYGON(((0 0, 1 0, 1 1, 0 0)))"
	case "geometrycollection":
		return "GEOMETRYCOLLECTION(POINT(0 0), LINESTRING(0 0, 1 1))"
	default:
		return "POINT(0 0)"
	}
}

// genClickHouseDecimal generates ClickHouse decimal values as strings.
// Uses scale/precision from constraints when available (set by the ClickHouse plugin).
// Decimal32(S) has max precision 9, Decimal64(S) has 18, Decimal128(S) has 38, Decimal256(S) has 76.
func genClickHouseDecimal(typeName string, c map[string]any, f *gofakeit.Faker) any {
	// Default max precisions per type
	maxPrecision := 9
	switch typeName {
	case "decimal64":
		maxPrecision = 18
	case "decimal128":
		maxPrecision = 38
	case "decimal256":
		maxPrecision = 76
	}

	scale := 2 // default
	if c != nil {
		if s, ok := c["scale"].(int); ok && s >= 0 {
			scale = s
		}
		if p, ok := c["precision"].(int); ok && p > 0 {
			maxPrecision = p
		}
	}

	intDigits := maxPrecision - scale
	if intDigits < 0 {
		intDigits = 0
	}
	// Cap integer digits to avoid float64 overflow
	if intDigits > 15 {
		intDigits = 15
	}

	maxVal := math.Pow(10, float64(intDigits)) - math.Pow(10, -float64(scale))
	if maxVal <= 0 {
		maxVal = 1 - math.Pow(10, -float64(scale))
	}
	// Keep values conservative for mock data
	if maxVal > 1000 {
		maxVal = 1000
	}

	val := f.Float64Range(0, maxVal)
	return fmt.Sprintf("%.*f", scale, val)
}

// genXML generates a simple XML element
func genXML(f *gofakeit.Faker) any {
	return fmt.Sprintf("<data><id>%d</id><value>%s</value></data>", f.Number(1, 1000), f.Word())
}

// genJSON generates a simple JSON object
func genJSON(f *gofakeit.Faker) any {
	data := map[string]any{
		"id":     f.Number(1, 1000),
		"name":   f.Name(),
		"email":  f.Email(),
		"active": f.Bool(),
	}
	jsonBytes, _ := json.Marshal(data)
	return string(jsonBytes)
}

// genHstore generates a PostgreSQL hstore value as "key1"=>"value1","key2"=>"value2"
func genHstore(f *gofakeit.Faker) any {
	pairs := make([]string, f.Number(1, 3))
	for i := range pairs {
		key := f.Word()
		value := f.Word()
		pairs[i] = fmt.Sprintf("\"%s\"=>\"%s\"", key, value)
	}
	return strings.Join(pairs, ",")
}

// genBinary generates binary data as hex string (0x prefixed)
// The hex format survives string serialization and can be decoded by ConvertStringValue
func genBinary(c map[string]any, f *gofakeit.Faker) any {
	length := 16
	if c != nil {
		if l, ok := c["length"].(int); ok && l > 0 {
			length = min(l, 256)
		}
	}

	bytes := make([]byte, length)
	for i := range bytes {
		bytes[i] = byte(f.Number(0, 255))
	}
	// Return as hex string with 0x prefix for proper round-trip through string serialization
	return fmt.Sprintf("0x%x", bytes)
}

// genArray generates a PostgreSQL-style array literal like {val1,val2,val3}
func genArray(dbType string, f *gofakeit.Faker) any {
	count := f.Number(1, 5)
	elements := make([]string, count)

	elementType := strings.TrimSuffix(strings.ToLower(dbType), "[]")

	for i := range count {
		if elementType == "int" || elementType == "integer" || elementType == "int4" {
			elements[i] = f.Numerify("###")
		} else {
			elements[i] = f.Word()
		}
	}

	return "{" + strings.Join(elements, ",") + "}"
}

// genClickHouseArray generates a ClickHouse Array literal like [val1, val2, val3]
func genClickHouseArray(dbType string, databaseType string, constraints map[string]any, faker *gofakeit.Faker) any {
	innerType := extractInnerType(dbType)
	count := faker.Number(1, 5)
	elements := make([]string, count)
	for i := range count {
		val := GenerateByType(innerType, databaseType, constraints, faker)
		elements[i] = fmt.Sprintf("%v", val)
	}
	return "[" + strings.Join(elements, ", ") + "]"
}

// genClickHouseMap generates a ClickHouse Map literal like {'k1': v1, 'k2': v2}
func genClickHouseMap(dbType string, databaseType string, faker *gofakeit.Faker) any {
	keyType, valueType := extractMapTypes(dbType)
	count := faker.Number(1, 3)
	pairs := make([]string, count)
	for i := range count {
		key := GenerateByType(keyType, databaseType, nil, faker)
		val := GenerateByType(valueType, databaseType, nil, faker)
		keyStr := strings.ReplaceAll(fmt.Sprintf("%v", key), "'", "''")
		valStr := strings.ReplaceAll(fmt.Sprintf("%v", val), "'", "''")
		pairs[i] = "'" + keyStr + "': '" + valStr + "'"
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}

// genClickHouseTuple generates a ClickHouse Tuple literal like ('text', 42, 3.14).
// String-typed values are single-quoted as required by ClickHouse SQL literal syntax.
func genClickHouseTuple(dbType string, databaseType string, faker *gofakeit.Faker) any {
	innerTypes := extractTupleTypes(dbType)
	elements := make([]string, len(innerTypes))
	for i, t := range innerTypes {
		val := GenerateByType(t, databaseType, nil, faker)
		// Quote string values per ClickHouse tuple literal syntax
		if s, isStr := val.(string); isStr {
			elements[i] = "'" + strings.ReplaceAll(s, "'", "''") + "'"
		} else {
			elements[i] = fmt.Sprintf("%v", val)
		}
	}
	return "(" + strings.Join(elements, ", ") + ")"
}

// extractInnerType extracts the inner type from a wrapper like "Array(Int32)" -> "Int32"
func extractInnerType(dbType string) string {
	start := strings.Index(dbType, "(")
	if start == -1 {
		return "String"
	}
	end := strings.LastIndex(dbType, ")")
	if end == -1 || end <= start {
		return "String"
	}
	return strings.TrimSpace(dbType[start+1 : end])
}

// extractMapTypes extracts key and value types from "Map(String, Int32)" -> ("String", "Int32")
func extractMapTypes(dbType string) (string, string) {
	inner := extractInnerType(dbType)
	// Split on first comma that's not inside nested parentheses
	depth := 0
	for i, ch := range inner {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				return strings.TrimSpace(inner[:i]), strings.TrimSpace(inner[i+1:])
			}
		}
	}
	return "String", "String"
}

// extractTupleTypes extracts element types from "Tuple(String, Int32, Float64)"
func extractTupleTypes(dbType string) []string {
	inner := extractInnerType(dbType)
	var types []string
	depth := 0
	start := 0
	for i, ch := range inner {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				types = append(types, strings.TrimSpace(inner[start:i]))
				start = i + 1
			}
		}
	}
	// Add the last element
	if remaining := strings.TrimSpace(inner[start:]); remaining != "" {
		types = append(types, remaining)
	}
	if len(types) == 0 {
		return []string{"String"}
	}
	return types
}

// genText generates text respecting length and check_values (ENUM) constraints
func genText(c map[string]any, f *gofakeit.Faker) any {
	if c != nil {
		if values, ok := c["check_values"].([]string); ok && len(values) > 0 {
			return values[f.Number(0, len(values)-1)]
		}
		if values, ok := c["check_values"].([]any); ok && len(values) > 0 {
			strValues := make([]string, len(values))
			for i, v := range values {
				strValues[i], _ = v.(string)
			}
			return strValues[f.Number(0, len(strValues)-1)]
		}
	}

	maxLen := 255
	if c != nil {
		if l, ok := c["length"].(int); ok && l > 0 {
			maxLen = l
		}
	}

	var text string
	if maxLen <= 10 {
		text = f.LetterN(uint(maxLen))
	} else if maxLen <= 50 {
		text = f.LoremIpsumSentence(3)
	} else {
		text = f.LoremIpsumSentence(f.Number(3, 10))
	}

	if len(text) > maxLen {
		text = text[:maxLen]
	}

	return text
}
