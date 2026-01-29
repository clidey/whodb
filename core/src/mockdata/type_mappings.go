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

	switch normalizedType {
	case "int", "integer", "int2", "int4", "int8", "smallint", "bigint", "tinyint", "mediumint", "serial", "bigserial", "smallserial":
		return genInt(constraints, faker)

	case "uint", "uint8", "uint16", "uint32", "uint64":
		return genUint(constraints, faker)

	case "float", "float4", "float8", "real", "double", "double precision", "decimal", "numeric", "number", "money":
		return genDecimal(constraints, faker)

	case "bool", "boolean", "bit":
		return faker.Bool()

	case "date", "date32":
		return genDate(faker)
	case "datetime", "datetime2", "datetime64", "timestamp", "timestamptz", "smalldatetime":
		return genDateTime(faker)
	case "time", "timetz":
		return genTime(faker)
	case "year":
		return genYear(faker)

	case "uuid":
		return faker.UUID()

	case "json", "jsonb":
		return genJSON(faker)

	case "bytea", "blob", "binary", "varbinary", "image", "tinyblob", "mediumblob", "longblob":
		return genBinary(constraints, faker)

	case "text", "string", "varchar", "char", "character", "character varying", "nvarchar", "nchar", "ntext", "fixedstring", "clob":
		return genText(constraints, faker)

	case "enum":
		return genText(constraints, faker)

	default:
		return genText(constraints, faker)
	}
}

// genInt generates an integer respecting check_min/check_max constraints.
// Uses practical defaults for mock data rather than full INT range.
func genInt(c map[string]any, f *gofakeit.Faker) any {
	minVal := int64(1)
	maxVal := int64(100)

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
	maxVal := uint64(100)

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

// genYear generates a year value (MySQL YEAR type: 1901-2155)
func genYear(f *gofakeit.Faker) any {
	return f.IntRange(1970, time.Now().Year())
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

// genBinary generates binary data as hex string
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
	return bytes
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
