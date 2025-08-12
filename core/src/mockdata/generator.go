/*
 * Copyright 2025 Clidey, Inc.
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
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
)

const (
	// Default configuration values
	DefaultNullProbability = 0.2     // 20% chance of null for nullable columns
	DefaultStringMinLen    = 10      // Minimum string length
	DefaultStringMaxLen    = 24      // Maximum string length
	DefaultFloatMin        = 0.0     // Minimum float value
	DefaultFloatMax        = 10000.0 // Maximum float value
)

// Use shared database type sets
var (
	intTypes      = common.IntTypes
	uintTypes     = common.UintTypes
	floatTypes    = common.FloatTypes
	boolTypes     = common.BoolTypes
	dateTypes     = common.DateTypes
	dateTimeTypes = common.DateTimeTypes
	uuidTypes     = common.UuidTypes
	textTypes     = common.TextTypes
	jsonTypes     = common.JsonTypes
)

type Generator struct {
	faker *gofakeit.Faker
}

func NewGenerator() *Generator {
	return &Generator{
		faker: gofakeit.New(uint64(time.Now().UnixNano())),
	}
}

// detectDatabaseType returns the simplified database type for a column
func detectDatabaseType(columnType string) string {
	upperType := strings.ToUpper(columnType)

	// Handle PostgreSQL arrays first
	if strings.Contains(upperType, "[]") {
		return "array"
	}

	// Remove size specifiers like VARCHAR(255) -> VARCHAR
	if idx := strings.Index(upperType, "("); idx > 0 {
		upperType = upperType[:idx]
	}
	upperType = strings.TrimSpace(upperType)

	switch {
	case intTypes.Contains(upperType):
		return "int"
	case uintTypes.Contains(upperType):
		return "uint"
	case floatTypes.Contains(upperType):
		return "float"
	case boolTypes.Contains(upperType):
		return "bool"
	case dateTypes.Contains(upperType):
		return "date"
	case dateTimeTypes.Contains(upperType):
		return "datetime"
	case uuidTypes.Contains(upperType):
		return "uuid"
	case jsonTypes.Contains(upperType):
		return "json"
	case textTypes.Contains(upperType):
		return "text"
	default:
		return "text" // Default to text for unknown types
	}
}

// parseMaxLen extracts the max length from types like varchar(n)
func parseMaxLen(columnType string) int {
	if strings.Contains(columnType, "(") {
		start := strings.Index(columnType, "(")
		end := strings.Index(columnType, ")")
		if start != -1 && end != -1 && end > start+1 {
			var n int
			if _, err := fmt.Sscanf(columnType[start+1:end], "%d", &n); err == nil {
				return n
			}
		}
	}
	return 0
}

// generateByType generates mock data based on the detected database type
// Returns properly typed values that the database driver can handle

func (g *Generator) generateByType(dbType string, columnType string) any {
	switch dbType {
	case "int":
		// Use Go's explicit int types for SQL integer types
		lowerType := strings.ToLower(columnType)
		switch {
		case strings.Contains(lowerType, "tinyint"):
			return int8(g.faker.Int8())
		case strings.Contains(lowerType, "smallint"):
			return int16(g.faker.Int16())
		case strings.Contains(lowerType, "bigint"):
			return int64(g.faker.Int64())
		case strings.Contains(lowerType, "int32"):
			return int32(g.faker.Int32())
		case strings.Contains(lowerType, "int16"):
			return int16(g.faker.Int16())
		case strings.Contains(lowerType, "int8"):
			return int8(g.faker.Int8())
		default:
			return int(g.faker.Int8())
		}
	case "uint":
		// Use Go's explicit uint types for SQL unsigned integer types
		lowerType := strings.ToLower(columnType)
		switch {
		case strings.Contains(lowerType, "tinyint unsigned") || strings.Contains(lowerType, "uint8"):
			return uint8(g.faker.Uint8())
		case strings.Contains(lowerType, "smallint unsigned") || strings.Contains(lowerType, "uint16"):
			return uint16(g.faker.Uint16())
		case strings.Contains(lowerType, "bigint unsigned") || strings.Contains(lowerType, "uint64"):
			return uint64(g.faker.Uint64())
		case strings.Contains(lowerType, "uint32"):
			return uint32(g.faker.Uint32())
		case strings.Contains(lowerType, "uint16"):
			return uint16(g.faker.Uint16())
		case strings.Contains(lowerType, "uint8"):
			return uint8(g.faker.Uint8())
		default:
			return uint(g.faker.Uint8())
		}
	case "float":
		return g.faker.Float32Range(DefaultFloatMin, DefaultFloatMax)
	case "bool":
		return g.faker.Bool()
	case "date":
		// Generate dates within a reasonable range (last 10 years to avoid timezone issues)
		now := time.Now()
		tenYearsAgo := now.AddDate(-10, 0, 0)
		return g.faker.DateRange(tenYearsAgo, now)
	case "datetime":
		// Generate datetimes within a reasonable range (last 10 years to avoid timezone issues)
		now := time.Now()
		tenYearsAgo := now.AddDate(-10, 0, 0)
		return g.faker.DateRange(tenYearsAgo, now)
	case "uuid":
		return g.faker.UUID()
	case "json":
		data := map[string]any{
			g.faker.Word(): g.faker.Word(),
			g.faker.Word(): g.faker.IntRange(1, 100),
		}
		jsonBytes, _ := json.Marshal(data)
		return string(jsonBytes)
	case "array":
		// PostgreSQL array format - needs to be string representation
		baseType := strings.ReplaceAll(columnType, "[]", "")
		arraySize := g.faker.IntRange(1, 5)
		elements := make([]string, arraySize)
		for i := range arraySize {
			val := g.generateByType(detectDatabaseType(baseType), baseType)
			elements[i] = fmt.Sprintf("%v", val)
		}
		return "{" + strings.Join(elements, ",") + "}"
	case "text":
		fallthrough
	default:
		maxLen := parseMaxLen(columnType)
		if maxLen <= 0 {
			maxLen = g.faker.IntRange(DefaultStringMinLen, DefaultStringMaxLen)
		}
		text := g.faker.LetterN(uint(maxLen))
		if len(text) > maxLen {
			text = text[:maxLen]
		}
		return text
	}
}

// GenerateValue generates a mock value based on column type only
// Returns properly typed values that the database driver can handle
func (g *Generator) GenerateValue(columnName string, columnType string, constraints map[string]any) (any, error) {
	columnTypeLower := strings.ToLower(columnType)

	// Check constraints
	allowNull := false
	requireUnique := false
	if constraints != nil {
		if nullable, ok := constraints["nullable"]; ok {
			allowNull = nullable.(bool)
		}
		if unique, ok := constraints["unique"]; ok {
			requireUnique = unique.(bool)
		}
	}

	// Generate NULL for nullable columns with configured probability
	if allowNull && g.faker.Float64() < DefaultNullProbability {
		return nil, nil
	}

	// Generate value based on database type only
	dbType := detectDatabaseType(columnTypeLower)
	value := g.generateByType(dbType, columnTypeLower)

	// For columns that require uniqueness, use inherently unique generators
	if requireUnique {
		// For unique columns, prefer UUIDs or timestamp-based values
		switch dbType {
		case "text", "uuid":
			value = g.faker.UUID()
		case "int", "uint":
			// Use timestamp + random for unique integers
			value = int32(time.Now().UnixNano()/1000 + int64(g.faker.IntRange(0, 9999)))
		}
		// For other types, rely on random generation being unlikely to collide
	}

	return value, nil
}

// GenerateRowDataWithConstraints generates mock data for a complete row
func (g *Generator) GenerateRowDataWithConstraints(columns []engine.Column, colConstraints map[string]map[string]any) ([]engine.Record, error) {

	records := make([]engine.Record, 0, len(columns))

	for _, col := range columns {
		// Skip serial columns - database generates these
		if strings.Contains(strings.ToLower(col.Type), "serial") {
			continue
		}

		constraints := make(map[string]any)
		if colConstraints != nil {
			if c, ok := colConstraints[col.Name]; ok {
				constraints = c
			}
		}

		value, err := g.GenerateValue(col.Name, col.Type, constraints)
		if err != nil {
			return nil, fmt.Errorf("failed to generate value for column %s: %w", col.Name, err)
		}

		// TODO: Refactor engine.Record to support interface{}/any values instead of strings.
		// This would allow us to pass typed values directly to database plugins,
		// letting each plugin handle formatting according to its specific requirements.
		// Current approach requires converting typed values to strings here, which
		// defeats the purpose of returning typed values from generateByType().

		// Convert typed value to string for the Record
		var valueStr string
		extra := map[string]string{
			"Type": col.Type,
		}

		if value == nil {
			// Mark as NULL in Extra field, leave Value empty
			valueStr = ""
			extra["IsNull"] = "true"
		} else if t, ok := value.(time.Time); ok {
			// Format time values for MySQL compatibility
			if strings.Contains(strings.ToLower(col.Type), "date") && !strings.Contains(strings.ToLower(col.Type), "time") {
				valueStr = t.Format("2006-01-02")
			} else {
				// MySQL datetime format without timezone
				valueStr = t.Format("2006-01-02 15:04:05")
			}
		} else {
			valueStr = fmt.Sprintf("%v", value)
		}

		records = append(records, engine.Record{
			Key:   col.Name,
			Value: valueStr,
			Extra: extra,
		})
	}

	return records, nil
}

// GenerateRowData generates mock data without constraints
func (g *Generator) GenerateRowData(columns []engine.Column) ([]engine.Record, error) {
	return g.GenerateRowDataWithConstraints(columns, nil)
}
