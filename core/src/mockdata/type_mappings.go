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

// GenerateByType generates a value for a database type, respecting constraints.
// The dbType parameter is the raw column type (e.g., "int", "text[]", "varchar(255)").
// The constraints map contains check_min, check_max, length, scale, check_values, etc.
func GenerateByType(dbType string, constraints map[string]any, faker *gofakeit.Faker) any {
	normalizedType := strings.ToLower(dbType)

	if strings.HasSuffix(normalizedType, "[]") {
		return genArray(dbType, faker)
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
		return genInt(constraints, faker)

	case "uint", "uint8", "uint16", "uint32", "uint64", "uint128", "uint256",
		"tinyint unsigned", "smallint unsigned", "mediumint unsigned", "int unsigned", "bigint unsigned":
		return genUint(constraints, faker)

	case "float", "float4", "float8", "real", "double", "double precision", "decimal", "numeric", "number", "money", "smallmoney",
		"binary_float", "binary_double",
		"float32", "float64":
		return genDecimal(constraints, faker)

	case "bool", "boolean", "bit":
		return faker.Bool()

	case "date", "date32":
		return genDate(faker)
	case "datetime", "datetime2", "datetime64", "smalldatetime",
		"timestamp with local time zone":
		return genDateTime(faker)
	case "datetimeoffset":
		return genDateTimeOffset(faker)
	case "interval", "interval year to month", "interval day to second":
		return genInterval(faker)
	case "year":
		return genYear(faker)

	case "uuid", "uniqueidentifier":
		return faker.UUID()

	case "json", "jsonb":
		return genJSON(faker)

	case "bytea", "blob", "binary", "varbinary", "image", "tinyblob", "mediumblob", "longblob",
		"raw", "long raw", "bfile":
		return genBinary(constraints, faker)

	case "text", "string", "varchar", "char", "character", "character varying", "nvarchar", "nchar", "ntext", "fixedstring", "clob", "nclob", "long",
		"varchar2", "nvarchar2",
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

	case "xml", "xmltype":
		return genXML(faker)

	// =============================================================================
	// NOT IMPLEMENTED - These types need proper mock data generators
	// Currently fall through to genText which will likely fail on INSERT
	// =============================================================================

	// MSSQL HIERARCHYID - represents position in a hierarchy tree
	// Format: "/1/2/3/" (path notation)
	// Example: "/", "/1/", "/1/2/", "/1/2/1/"
	case "hierarchyid":
		return genText(constraints, faker) // TODO: implement genHierarchyId

	// PostgreSQL/PostGIS geometry types - need WKT (Well-Known Text) format
	// LINE example: "[(0,0),(1,1)]" or "{0,0,1,1}" (PostgreSQL line)
	// POLYGON example: "POLYGON((0 0, 1 0, 1 1, 0 1, 0 0))" (WKT)
	// BOX example: "(1,1),(0,0)"
	// PATH example: "[(0,0),(1,1),(2,0)]" or "((0,0),(1,1),(2,0))"
	// CIRCLE example: "<(0,0),1>" (center and radius)
	// LSEG example: "[(0,0),(1,1)]" (line segment)
	case "line", "lseg", "box", "path", "polygon", "circle":
		return genText(constraints, faker) // TODO: implement genGeometry

	// PostGIS/MSSQL spatial types - need WKT format
	// GEOMETRY example: "POINT(0 0)" or "LINESTRING(0 0, 1 1)"
	// GEOGRAPHY example: "POINT(-122.34 47.65)" (longitude latitude)
	// LINESTRING example: "LINESTRING(0 0, 1 1, 2 2)"
	// MULTIPOINT example: "MULTIPOINT((0 0), (1 1))"
	// MULTILINESTRING example: "MULTILINESTRING((0 0, 1 1), (2 2, 3 3))"
	// MULTIPOLYGON example: "MULTIPOLYGON(((0 0, 1 0, 1 1, 0 0)))"
	case "geometry", "geography", "linestring", "multipoint", "multilinestring", "multipolygon", "geometrycollection":
		return genText(constraints, faker) // TODO: implement genSpatial

	// ClickHouse big integer types - need big.Int handling
	// ClickHouse decimal types with specific precision
	// DECIMAL32 example: up to 9 digits
	// DECIMAL64 example: up to 18 digits
	// DECIMAL128 example: up to 38 digits
	case "decimal32", "decimal64", "decimal128", "decimal256":
		return genText(constraints, faker) // TODO: implement genClickHouseDecimal

	default:
		return genText(constraints, faker)
	}
}

// genInt generates an integer respecting check_min/check_max constraints.
// Uses practical defaults for mock data rather than full INT range.
func genInt(c map[string]any, f *gofakeit.Faker) any {
	minVal := int64(1)
	maxVal := int64(1000000)

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

// genUint generates an unsigned integer respecting check_min/check_max constraints.
func genUint(c map[string]any, f *gofakeit.Faker) any {
	minVal := uint64(0)
	maxVal := uint64(1000000)

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

// genDecimal generates a decimal number respecting check_min/check_max and scale.
func genDecimal(c map[string]any, f *gofakeit.Faker) any {
	minVal := 0.0
	maxVal := 1000.0
	scale := 2

	if c != nil {
		if v, ok := c["check_min"].(float64); ok {
			minVal = v
		}
		if v, ok := c["check_max"].(float64); ok {
			maxVal = v
		}
		if s, ok := c["scale"].(int); ok && s > 0 {
			scale = s
		}
	}

	if minVal > maxVal {
		minVal, maxVal = maxVal, minVal
	}

	val := f.Float64Range(minVal, maxVal)
	multiplier := math.Pow(10, float64(scale))
	return math.Round(val*multiplier) / multiplier
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

// genDateTimeOffset generates a MSSQL DATETIMEOFFSET value with timezone
func genDateTimeOffset(f *gofakeit.Faker) any {
	start := time.Now().AddDate(-10, 0, 0)
	end := time.Now()
	dt := f.DateRange(start, end)
	// MSSQL DATETIMEOFFSET format: YYYY-MM-DD HH:MM:SS.nnnnnnn +/-HH:MM
	offset := f.IntRange(-12, 12)
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	return fmt.Sprintf("%s %s%02d:00", dt.Format("2006-01-02 15:04:05.0000000"), sign, offset)
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
